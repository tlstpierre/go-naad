package naadaudio

import (
	"bytes"
	"context"
	"fmt"
	log "github.com/sirupsen/logrus"
	"github.com/tlstpierre/go-naad/pkg/naad-filter"
	"github.com/tlstpierre/go-naad/pkg/naad-xml"
	"gitlab.com/timstpierre/mc-audio/pkg/mc-transmit"
	"gitlab.com/timstpierre/mc-audio/pkg/piper-tts"
	"gitlab.com/timstpierre/mc-audio/pkg/resample"
	"gitlab.com/timstpierre/mc-audio/pkg/t140"
	"strings"
	"sync"
	"time"
)

type MulticastDest struct {
	Address string `yaml:"address"`
	// Codec is an audio codec (g722, pcmu, pcma) or a real-time text
	// codec: t140 (plain RFC 4103 text) or t140red (with RFC 2198
	// redundancy, RFC 4103 recommended).
	Codec string `yaml:"codec,omitempty"`
	// PayloadType: for audio codecs this is the legacy way to pick a
	// codec by static RTP payload type (0, 8 or 9) -- use Codec instead.
	// For text codecs it overrides the dynamic payload type (96-127;
	// defaults 98 for t140 and 100 for t140red).
	PayloadType *uint8 `yaml:"payloadtype,omitempty"`
}

var codecPayloadTypes = map[string]uint8{
	"pcmu": 0, "g711u": 0, "ulaw": 0, "mulaw": 0,
	"pcma": 8, "g711a": 8, "alaw": 8,
	"g722": 9,
}

var textCodecPayloadTypes = map[string]uint8{
	"t140":    t140.DefaultPT,
	"t140red": t140.DefaultREDPT,
}

// IsText reports whether this destination carries real-time text.
func (d MulticastDest) IsText() bool {
	_, ok := textCodecPayloadTypes[strings.ToLower(d.Codec)]
	return ok
}

// ResolvePT returns the RTP payload type for this destination.
func (d MulticastDest) ResolvePT() (uint8, error) {
	codec := strings.ToLower(d.Codec)
	if pt, ok := textCodecPayloadTypes[codec]; ok {
		if d.PayloadType != nil {
			pt = *d.PayloadType
			if pt < 96 || pt > 127 {
				return 0, fmt.Errorf("multicast %s: %s payloadtype %d must be dynamic (96-127)", d.Address, codec, pt)
			}
		}
		return pt, nil
	}
	if d.Codec != "" {
		pt, ok := codecPayloadTypes[codec]
		if !ok {
			return 0, fmt.Errorf("multicast %s: unknown codec %q (use g722, pcmu, pcma, t140 or t140red)", d.Address, d.Codec)
		}
		if d.PayloadType != nil && *d.PayloadType != pt {
			return 0, fmt.Errorf("multicast %s: codec %q conflicts with payloadtype %d", d.Address, d.Codec, *d.PayloadType)
		}
		return pt, nil
	}
	if d.PayloadType != nil {
		pt := *d.PayloadType
		if rtptransmit.RatesForPT(pt).AudioRate == 0 {
			return 0, fmt.Errorf("multicast %s: unsupported payloadtype %d", d.Address, pt)
		}
		return pt, nil
	}
	return 0, fmt.Errorf("multicast %s: no codec specified", d.Address)
}

type ChannelConfig struct {
	SpeakContent  bool     `yaml:"speakcontent"`
	SoremOnly     bool     `yaml:"soremonly"`
	StripComments bool     `yaml:"stripcomments"`
	Language      string   `yaml:"language"`
	Voice         string   `yaml:"voice"`
	CAPCodes      []string `yaml:"capcodes"`
	// GeoFences match alerts whose polygons or circles overlap any of these
	// areas. Combined with CAPCodes as OR. If both are empty, all alerts match.
	GeoFences        []naadfilter.GeoFence `yaml:"geofences"`
	Multicast        []MulticastDest       `yaml:"multicast"`
	LocalAudio       bool                  `yaml:"localaudio"`
	LocalAudioDevice string                `yaml:"localaudiodevice"`
	LocalAudioRate   int                   `yaml:"localaudiorate"`
	TTSReplacements  []TTSReplacement      `yaml:"ttsreplacements"`
}

type TransmitChannel struct {
	Config           ChannelConfig
	LastAlertTone    time.Time
	LastAnnounceTone time.Time
	speaker          *pipertts.Speaker
	playbackDev      LocalPlayback
	alertChan        chan *naadxml.Alert
	textOutputs      []*textOutput
	ctx              context.Context
	wg               *sync.WaitGroup
}

// Validate checks the channel's multicast and geofence settings.
func (c ChannelConfig) Validate() error {
	for _, dest := range c.Multicast {
		if _, err := dest.ResolvePT(); err != nil {
			return err
		}
	}
	for _, fence := range c.GeoFences {
		if err := fence.Validate(); err != nil {
			return err
		}
	}
	return nil
}

func NewTransmitter(ch chan *naadxml.Alert, config ChannelConfig, ctx context.Context, wg *sync.WaitGroup, audioBackend AudioBackend) (*TransmitChannel, error) {
	if err := config.Validate(); err != nil {
		return nil, err
	}
	if config.LocalAudioRate == 0 {
		config.LocalAudioRate = 48000
	}
	tx := &TransmitChannel{
		alertChan: ch,
		ctx:       ctx,
		wg:        wg,
		Config:    config,
	}
	if config.LocalAudio {
		if audioBackend == nil {
			log.Warn("Local audio requested but no audio backend available -- skipping")
		} else {
			frameSize := config.LocalAudioRate / 50 // 20ms frames
			dev, err := audioBackend.OpenPlayback(config.LocalAudioDevice, config.LocalAudioRate, 1, frameSize)
			if err != nil {
				return nil, err
			}
			if err := dev.Start(); err != nil {
				dev.Close()
				return nil, err
			}
			tx.playbackDev = dev
		}
	}
	for _, dest := range config.Multicast {
		if !dest.IsText() {
			continue
		}
		out, err := newTextOutput(dest, ctx, wg)
		if err != nil {
			return nil, err
		}
		tx.textOutputs = append(tx.textOutputs, out)
	}
	wg.Add(1)
	go tx.handleMessages()
	return tx, nil
}

func (t *TransmitChannel) AddTTS(config pipertts.PiperConfig) {
	t.speaker = pipertts.NewSpeaker(config, t.ctx)
}

func (t *TransmitChannel) handleMessages() {
	defer t.wg.Done()
	defer func() {
		if t.playbackDev != nil {
			t.playbackDev.Stop()
			t.playbackDev.Close()
		}
	}()
	for {
		select {
		case <-t.ctx.Done():
			return
		case msg := <-t.alertChan:
			for _, alertInfo := range msg.Info {
				if strings.EqualFold(alertInfo.Language, t.Config.Language) {
					if t.Config.SoremOnly {
						if alertInfo.SoremLayer == nil {
							log.Infof("Skipping announcement for %s - missing Sorem layer", alertInfo.Headline)
							continue
						}
						if alertInfo.SoremLayer.BroadcastText == "" {
							log.Infof("Skipping announcement for %s - no broadcast text", alertInfo.Headline)
							continue
						}
					}
					// Text goes out immediately and independently of
					// audio, so displays update even if TTS fails.
					t.sendText(msg.MsgType, alertInfo)

					audio, samplerate, err := t.GetAudio(alertInfo)
					if err != nil {
						log.Errorf("Problem getting audio for alert %s language %s - %v", msg.Identifier, alertInfo.Language, err)
						continue
					}
					log.Infof("Got audio at samplerate %d", samplerate)
					audio = t.addTone(alertInfo, audio, samplerate)

					t.sendToMulticast(audio, samplerate)
					t.sendToLocalAudio(audio, samplerate)
				}
			}
		}
	}
}

// sendToMulticast groups multicast destinations by codec and sends encoded
// audio to every group concurrently, so all destinations hear the alert at
// the same time regardless of codec.
func (t *TransmitChannel) sendToMulticast(pcm []int16, samplerate uint16) {
	if len(t.Config.Multicast) == 0 {
		return
	}

	groups := make(map[uint8][]string)
	for _, dest := range t.Config.Multicast {
		if dest.IsText() {
			continue
		}
		pt, err := dest.ResolvePT()
		if err != nil { // already validated at startup
			log.Errorf("Skipping multicast destination - %v", err)
			continue
		}
		groups[pt] = append(groups[pt], dest.Address)
	}

	var wg sync.WaitGroup
	for pt, addrs := range groups {
		wg.Add(1)
		go func(pt uint8, addrs []string) {
			defer wg.Done()
			if err := sendMulticast(addrs, pt, pcm, samplerate, t.ctx); err != nil {
				log.Errorf("Multicast send failed for PT %d to %v: %v", pt, addrs, err)
			}
		}(pt, addrs)
	}
	wg.Wait()
}

const ptimeMs = 20

// sendMulticast encodes audio for the given payload type and transmits via
// RTP. Codec sample rate and RTP clock rate come from mc-audio, which knows
// that G.722 samples at 16 kHz but uses an 8 kHz RTP clock (RFC 3551).
func sendMulticast(addresses []string, pt uint8, pcm []int16, samplerate uint16, ctx context.Context) error {
	if len(pcm) == 0 {
		return nil
	}
	encoder, audioRate, err := rtptransmit.SelectEncoder(pt)
	if err != nil {
		return err
	}
	rates := rtptransmit.RatesForPT(pt)

	codecPCM, err := resampleTo(pcm, int(samplerate), int(audioRate))
	if err != nil {
		return err
	}
	encoded := encoder.Encode(codecPCM)

	// Bytes per packet = encoded bytes for one ptime worth of samples
	samplesPerPacket := int(audioRate) * ptimeMs / 1000
	packetBytes := samplesPerPacket * len(encoded) / len(codecPCM)

	mctx, err := rtptransmit.NewSender(addresses, pt, uint16(rates.RTPRate), ptimeMs)
	if err != nil {
		return err
	}
	defer mctx.Stop()
	return mctx.SendBuffer(bytes.NewBuffer(encoded), packetBytes, ctx)
}

// resampleTo converts PCM from one sample rate to another.
func resampleTo(pcm []int16, from, to int) ([]int16, error) {
	if from == to {
		return pcm, nil
	}
	rc, err := resample.NewRateConverter(float64(from), float64(to))
	if err != nil {
		return nil, fmt.Errorf("could not resample %d -> %d: %v", from, to, err)
	}
	return rc.Process(pcm), nil
}

// sendToLocalAudio resamples PCM audio to the local playback rate and writes
// it to the playback device.
func (t *TransmitChannel) sendToLocalAudio(pcm []int16, samplerate uint16) {
	if t.playbackDev == nil {
		return
	}
	targetRate := t.Config.LocalAudioRate
	rc, err := resample.NewRateConverter(float64(samplerate), float64(targetRate))
	if err != nil {
		log.Errorf("Could not create rate converter for local audio: %v", err)
		return
	}
	resampled := rc.Process(pcm)

	frameSize := targetRate / 50 // 20ms frames
	for len(resampled) >= frameSize {
		if err := t.playbackDev.Write(resampled[:frameSize]); err != nil {
			log.Errorf("Local audio write error: %v", err)
			return
		}
		resampled = resampled[frameSize:]
	}
	// Write any remaining partial frame padded with silence
	if len(resampled) > 0 {
		frame := make([]int16, frameSize)
		copy(frame, resampled)
		t.playbackDev.Write(frame)
	}
}
