package naadaudio

import (
	"context"
	"encoding/xml"
	"net"
	"os"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/tlstpierre/go-naad/pkg/naad-xml"
	"gitlab.com/timstpierre/mc-audio/pkg/rtp"
	"gitlab.com/timstpierre/mc-audio/pkg/t140"
)

func TestAlertText(t *testing.T) {
	info := &naadxml.AlertInfo{
		Headline:    "tornado warning",
		Description: "Take cover now.\nStay away from windows.### forecaster note",
		Severity:    naadxml.Extreme,
	}
	got := AlertText(info, true)
	want := t140.BEL + "tornado warning" + t140.LineSeparator + "Take cover now." + t140.LineSeparator + "Stay away from windows." + t140.LineSeparator
	if got != want {
		t.Errorf("fallback text\n got %q\nwant %q", got, want)
	}

	info.SoremLayer = &naadxml.Sorem{BroadcastText: "  Broadcast this.  "}
	info.Severity = "Moderate"
	if got := AlertText(info, true); got != "Broadcast this."+t140.LineSeparator {
		t.Errorf("SOREM text %q", got)
	}

	if got := AlertText(&naadxml.AlertInfo{}, true); got != "" {
		t.Errorf("empty alert gave %q", got)
	}
}

func TestTextDestinationsResolve(t *testing.T) {
	p99, p9 := uint8(99), uint8(9)
	for _, c := range []struct {
		dest MulticastDest
		pt   uint8
		ok   bool
	}{
		{MulticastDest{Codec: "t140"}, 98, true},
		{MulticastDest{Codec: "T140RED"}, 100, true},
		{MulticastDest{Codec: "t140", PayloadType: &p99}, 99, true},
		{MulticastDest{Codec: "t140red", PayloadType: &p9}, 0, false},
	} {
		pt, err := c.dest.ResolvePT()
		if (err == nil) != c.ok || (c.ok && pt != c.pt) {
			t.Errorf("%+v: pt=%d err=%v", c.dest, pt, err)
		}
		if !c.dest.IsText() {
			t.Errorf("%+v should be text", c.dest)
		}
	}
	if (MulticastDest{Codec: "g722"}).IsText() {
		t.Error("g722 is not text")
	}
}

// Sends a sample alert's text through a real TransmitChannel to loopback
// and decodes it with the mc-audio T.140 decoder.
func TestTextEndToEnd(t *testing.T) {
	for _, codec := range []string{"t140", "t140red"} {
		t.Run(codec, func(t *testing.T) {
			conn, err := net.ListenUDP("udp", &net.UDPAddr{IP: net.IPv4(127, 0, 0, 1)})
			if err != nil {
				t.Fatal(err)
			}
			defer conn.Close()

			ctx, cancel := context.WithCancel(context.Background())
			wg := &sync.WaitGroup{}
			defer func() { cancel(); wg.Wait() }()
			tx, err := NewTransmitter(make(chan *naadxml.Alert), ChannelConfig{
				Language:      "en-CA",
				StripComments: true,
				Multicast:     []MulticastDest{{Address: conn.LocalAddr().String(), Codec: codec}},
			}, ctx, wg, nil)
			if err != nil {
				t.Fatal(err)
			}

			data, err := os.ReadFile("../../testdata/Sample10_CAPCP_with_TTS.XML")
			if err != nil {
				t.Fatal(err)
			}
			var alert naadxml.Alert
			if err := xml.Unmarshal(data, &alert); err != nil {
				t.Fatal(err)
			}
			alert.ProcessAlert() // populates the SOREM layer, as the receiver does
			info := alert.Info[0]
			tx.sendText(naadxml.AlertMessage, info)
			want := t140.BOM + t140.BEL + "This is a test" + t140.LineSeparator

			dec := t140.NewDecoder()
			var got strings.Builder
			buf := make([]byte, 2048)
			for !strings.HasSuffix(got.String(), t140.LineSeparator) {
				conn.SetReadDeadline(time.Now().Add(5 * time.Second))
				n, err := conn.Read(buf)
				if err != nil {
					t.Fatalf("after %q: %v", got.String(), err)
				}
				text, err := dec.Decode(rtp.DecodeRTP(buf[:n]))
				if err != nil {
					t.Fatal(err)
				}
				got.WriteString(text)
			}
			if got.String() != want {
				t.Errorf("\n got %q\nwant %q", got.String(), want)
			}
		})
	}
}

// A stand-down of a Severe alert doesn't ring the bell.
func TestAlertTextStandDownNoBell(t *testing.T) {
	ended := &naadxml.AlertInfo{
		Headline:   "tornado warning ended",
		Severity:   naadxml.Severe,
		Parameters: []naadxml.Parameter{{Name: "layer:EC-MSC-SMC:1.1:Alert_Location_Status", Value: "ended"}},
	}
	allClear := &naadxml.AlertInfo{Headline: "all clear", Severity: naadxml.Extreme, Response: []naadxml.ResponseType{"AllClear"}}
	for _, info := range []*naadxml.AlertInfo{ended, allClear} {
		if got := AlertText(info, true); strings.HasPrefix(got, t140.BEL) {
			t.Errorf("%q rings the bell", got)
		}
	}
	active := &naadxml.AlertInfo{Headline: "tornado warning", Severity: naadxml.Severe}
	if got := AlertText(active, true); !strings.HasPrefix(got, t140.BEL) {
		t.Errorf("active Severe alert %q should ring the bell", got)
	}
}
