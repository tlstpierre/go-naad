package naadaudio

import (
	"context"
	"encoding/binary"
	"net"
	"testing"
	"time"
)

// Captures what sendMulticast puts on the wire for each codec.
func TestSendMulticastRTPParameters(t *testing.T) {
	cases := []struct {
		codec       string
		wantPT      uint8
		wantTSStep  uint32 // RTP clock ticks per 20 ms packet
		wantPayload int
	}{
		{"g722", 9, 160, 160}, // 8 kHz RTP clock per RFC 3551, 64 kbit/s
		{"pcmu", 0, 160, 160},
		{"pcma", 8, 160, 160},
	}
	for _, c := range cases {
		t.Run(c.codec, func(t *testing.T) {
			conn, err := net.ListenUDP("udp", &net.UDPAddr{IP: net.IPv4(127, 0, 0, 1)})
			if err != nil {
				t.Fatal(err)
			}
			defer conn.Close()

			pt, err := MulticastDest{Address: conn.LocalAddr().String(), Codec: c.codec}.ResolvePT()
			if err != nil {
				t.Fatal(err)
			}
			pcm := make([]int16, 22050/5) // 200 ms at an odd TTS rate
			for i := range pcm {
				pcm[i] = int16(i * 37)
			}
			go sendMulticast([]string{conn.LocalAddr().String()}, pt, pcm, 22050, context.Background())

			buf := make([]byte, 1500)
			var lastTS uint32
			for i := 0; i < 5; i++ {
				conn.SetReadDeadline(time.Now().Add(2 * time.Second))
				n, err := conn.Read(buf)
				if err != nil {
					t.Fatal(err)
				}
				gotPT := buf[1] & 0x7f
				ts := binary.BigEndian.Uint32(buf[4:8])
				if gotPT != c.wantPT {
					t.Errorf("PT %d want %d", gotPT, c.wantPT)
				}
				if n-12 != c.wantPayload {
					t.Errorf("payload %d bytes want %d", n-12, c.wantPayload)
				}
				if i > 0 && ts-lastTS != c.wantTSStep {
					t.Errorf("timestamp step %d want %d", ts-lastTS, c.wantTSStep)
				}
				lastTS = ts
			}
		})
	}
}

func TestResolvePT(t *testing.T) {
	nine, zero := uint8(9), uint8(0)
	good := map[string]MulticastDest{
		"codec":       {Codec: "G722"},
		"legacy pt 0": {PayloadType: &zero},
		"legacy pt 9": {PayloadType: &nine},
		"both agree":  {Codec: "g722", PayloadType: &nine},
		"alias g711u": {Codec: "g711u"},
	}
	for name, d := range good {
		if _, err := d.ResolvePT(); err != nil {
			t.Errorf("%s: %v", name, err)
		}
	}
	bad := map[string]MulticastDest{
		"none":     {},
		"unknown":  {Codec: "opus"},
		"conflict": {Codec: "pcmu", PayloadType: &nine},
	}
	for name, d := range bad {
		if _, err := d.ResolvePT(); err == nil {
			t.Errorf("%s: expected error", name)
		}
	}
}
