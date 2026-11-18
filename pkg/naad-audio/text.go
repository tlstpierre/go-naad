package naadaudio

import (
	"context"
	"fmt"
	"strings"
	"sync"

	log "github.com/sirupsen/logrus"
	"github.com/tlstpierre/go-naad/pkg/naad-xml"
	"gitlab.com/timstpierre/mc-audio/pkg/mc-transmit"
	"gitlab.com/timstpierre/mc-audio/pkg/t140"
)

// textOutput is a long-lived RFC 4103 real-time text stream to one
// multicast destination. Messages are queued and sent in order by a
// dedicated goroutine so text never waits on (or delays) audio.
type textOutput struct {
	address string
	rtp     *rtptransmit.Sender
	sender  *t140.Sender
	queue   chan string
}

func newTextOutput(dest MulticastDest, ctx context.Context, wg *sync.WaitGroup) (*textOutput, error) {
	pt, err := dest.ResolvePT()
	if err != nil {
		return nil, err
	}
	// rate/ptime only affect audio timestamps, which text doesn't use
	rtpSender, err := rtptransmit.NewSender([]string{dest.Address}, pt, t140.ClockRate, 20)
	if err != nil {
		return nil, err
	}
	cfg := t140.SenderConfig{}
	if strings.EqualFold(dest.Codec, "t140red") {
		cfg.REDPT = pt
		cfg.Redundancy = t140.DefaultRedundancy
	} else {
		cfg.PT = pt
	}
	out := &textOutput{
		address: dest.Address,
		rtp:     rtpSender,
		sender:  t140.NewSender(rtpSender.SendPacket, cfg),
		queue:   make(chan string, 16),
	}
	log.Infof("Real-time text (%s, PT %d) to %s", strings.ToLower(dest.Codec), pt, dest.Address)
	wg.Add(1)
	go out.run(ctx, wg)
	return out, nil
}

func (o *textOutput) run(ctx context.Context, wg *sync.WaitGroup) {
	defer wg.Done()
	defer o.rtp.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case text := <-o.queue:
			if err := o.sender.Send(ctx, text); err != nil && ctx.Err() == nil {
				log.Errorf("Real-time text to %s failed: %v", o.address, err)
			}
		}
	}
}

// sendText queues the block's text for every text destination. A Cancel
// is a stand-down like an ended block, so it never rings the bell.
func (t *TransmitChannel) sendText(msgType naadxml.MessageType, info *naadxml.AlertInfo) {
	if len(t.textOutputs) == 0 {
		return
	}
	text := AlertText(info, t.Config.StripComments)
	if msgType == naadxml.AlertCancel {
		text = strings.TrimPrefix(text, t140.BEL)
	}
	if text == "" {
		return
	}
	for _, out := range t.textOutputs {
		select {
		case out.queue <- text:
		default:
			log.Errorf("Real-time text queue to %s is full, dropping %q", out.address, info.Headline)
		}
	}
}

// maxDisplayAreas bounds the area line of the fallback display text; some
// alerts list dozens of areas.
const maxDisplayAreas = 5

// AlertText is the display text sent to real-time text endpoints: the SOREM
// broadcast text if present, otherwise the headline, the affected areas and
// the description, cleaned up for display (see DisplayText). It starts with
// BEL for Severe and Extreme alerts in effect (not stand-downs) and ends
// with a T.140 line separator.
func AlertText(info *naadxml.AlertInfo, stripComments bool) string {
	var text string
	if info.SoremLayer != nil {
		text = DisplayText(info.SoremLayer.BroadcastText)
	}
	if text == "" {
		description := info.Description
		if stripComments {
			description = StripComments(description)
		}
		text = DisplayText(strings.Join([]string{info.Headline, areaSummary(info, maxDisplayAreas), description}, "\n"))
	}
	if text == "" {
		return ""
	}
	text = t140.NormalizeNewlines(text) + t140.LineSeparator
	// The bell is for an alert taking effect, not for its stand-down.
	if (info.Severity == naadxml.Extreme || info.Severity == naadxml.Severe) && info.InEffect() {
		text = t140.BEL + text
	}
	return text
}

// areaSummary lists the alert's area descriptions, at most max of them
// ("A, B and 3 more"); max <= 0 lists them all.
func areaSummary(info *naadxml.AlertInfo, max int) string {
	var areas []string
	for _, a := range info.Area {
		if d := strings.TrimSpace(a.Description); d != "" {
			areas = append(areas, d)
		}
	}
	if max > 0 && len(areas) > max {
		return fmt.Sprintf("%s and %d more", strings.Join(areas[:max], ", "), len(areas)-max)
	}
	return strings.Join(areas, ", ")
}
