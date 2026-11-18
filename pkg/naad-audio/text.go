package naadaudio

import (
	"context"
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
			if err := o.sender.Send(ctx, text); err != nil {
				log.Errorf("Real-time text to %s failed: %v", o.address, err)
			}
		}
	}
}

func (t *TransmitChannel) sendText(info *naadxml.AlertInfo) {
	if len(t.textOutputs) == 0 {
		return
	}
	text := AlertText(info, t.Config.StripComments)
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

// AlertText is the display text sent to real-time text endpoints: the SOREM
// broadcast text if present, otherwise the headline and description. It
// starts with BEL for Severe and Extreme alerts and ends with a T.140 line
// separator.
func AlertText(info *naadxml.AlertInfo, stripComments bool) string {
	var text string
	if info.SoremLayer != nil && strings.TrimSpace(info.SoremLayer.BroadcastText) != "" {
		text = info.SoremLayer.BroadcastText
	} else {
		description := info.Description
		if stripComments {
			description, _, _ = strings.Cut(description, "###")
		}
		parts := []string{}
		for _, p := range []string{info.Headline, description} {
			if p = strings.TrimSpace(p); p != "" {
				parts = append(parts, p)
			}
		}
		text = strings.Join(parts, "\n")
	}
	text = strings.TrimSpace(text)
	if text == "" {
		return ""
	}
	text = t140.NormalizeNewlines(text) + t140.LineSeparator
	if info.Severity == naadxml.Extreme || info.Severity == naadxml.Severe {
		text = t140.BEL + text
	}
	return text
}
