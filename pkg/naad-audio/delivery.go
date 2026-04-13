package naadaudio

import (
	"bytes"
	"context"
	"encoding/binary"
	"github.com/shenjinti/go722"
	log "github.com/sirupsen/logrus"
	"github.com/tlstpierre/go-naad/pkg/naad-xml"
	"gitlab.com/timstpierre/mc-audio/pkg/g711"
	"gitlab.com/timstpierre/mc-audio/pkg/mc-transmit"
	"gitlab.com/timstpierre/mc-audio/pkg/piper-tts"
	"gitlab.com/timstpierre/mc-audio/pkg/resample"
	"strings"
	"sync"
	"time"
)

type MulticastDest struct {
	Address     string `yaml:"address"`
	PayloadType uint8  `yaml:"payloadtype"`
}

type ChannelConfig struct {
	SpeakContent     bool             `yaml:"speakcontent"`
	SoremOnly        bool             `yaml:"soremonly"`
	StripComments    bool             `yaml:"stripcomments"`
	Language         string           `yaml:"language"`
	Voice            string           `yaml:"voice"`
	CAPCodes         []string         `yaml:"capcodes"`
	Multicast        []MulticastDest  `yaml:"multicast"`
	LocalAudio       bool             `yaml:"localaudio"`
	LocalAudioDevice string           `yaml:"localaudiodevice"`
	LocalAudioRate   int              `yaml:"localaudiorate"`
	TTSReplacements  []TTSReplacement `yaml:"ttsreplacements"`
}

type TransmitChannel struct {
	Config           ChannelConfig
	LastAlertTone    time.Time
	LastAnnounceTone time.Time
	speaker          *pipertts.Speaker
	playbackDev      LocalPlayback
	alertChan        chan *naadxml.Alert
	ctx              context.Context
	wg               *sync.WaitGroup
}

func NewTransmitter(ch chan *naadxml.Alert, config ChannelConfig, ctx context.Context, wg *sync.WaitGroup, audioBackend AudioBackend) (*TransmitChannel, error) {
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

// sendToMulticast groups multicast destinations by payload type and sends
// encoded audio to each group.
func (t *TransmitChannel) sendToMulticast(pcm []int16, samplerate uint16) {
	if len(t.Config.Multicast) == 0 {
		return
	}

	// Group destinations by payload type
	groups := make(map[uint8][]string)
	for _, dest := range t.Config.Multicast {
		groups[dest.PayloadType] = append(groups[dest.PayloadType], dest.Address)
	}

	for pt, addrs := range groups {
		if err := sendMulticast(addrs, pt, pcm, samplerate, t.ctx); err != nil {
			log.Errorf("Multicast send failed for PT %d: %v", pt, err)
		}
	}
}

// sendMulticast encodes audio for the given payload type and transmits via RTP.
func sendMulticast(addresses []string, pt uint8, pcm []int16, samplerate uint16, ctx context.Context) error {
	switch pt {
	case 9: // G.722
		mctx, err := rtptransmit.NewSender(addresses, 9, 16000, 20)
		if err != nil {
			return err
		}
		lowrate := resample.Downsample(pcm, int(samplerate)/16000)
		encoder := go722.NewG722Encoder(go722.Rate64000, go722.G722_DEFAULT)
		g722Buf := new(bytes.Buffer)
		if err := binary.Write(g722Buf, binary.LittleEndian, lowrate); err != nil {
			mctx.Stop()
			return err
		}
		encoded := encoder.Encode(g722Buf.Bytes())
		mctx.SendBuffer(bytes.NewBuffer(encoded), 160, ctx)
		mctx.Stop()

	case 0: // mu-law (G.711)
		mctx, err := rtptransmit.NewSender(addresses, 0, 8000, 20)
		if err != nil {
			return err
		}
		lowrate := resample.Downsample(pcm, int(samplerate)/8000)
		encoded := g711.ULawEncode(lowrate)
		mctx.SendBuffer(bytes.NewBuffer(encoded), 160, ctx)
		mctx.Stop()

	default:
		log.Warnf("Unsupported payload type %d -- skipping", pt)
	}
	return nil
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
