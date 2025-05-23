package naadaudio

import (
	"bytes"
	"context"
	"encoding/binary"
	"github.com/shenjinti/go722"
	log "github.com/sirupsen/logrus"
	"github.com/tlstpierre/go-naad/pkg/naad-xml"
	"github.com/tlstpierre/mc-audio/pkg/g711"
	"github.com/tlstpierre/mc-audio/pkg/mc-transmit"
	"github.com/tlstpierre/mc-audio/pkg/piper-tts"
	"strings"
	"sync"
	"time"
)

type ChannelConfig struct {
	SpeakContent  bool     `yaml:"speakcontent"`
	SoremOnly     bool     `yaml:"soremonly"`
	StripComments bool     `yaml:"stripcomments"`
	G722          bool     `yaml:"g722"`
	Addresses     []string `yaml:"addresses"`
	Language      string   `yaml:"language"`
	Voice         string   `yaml:"voice"`
	CAPCodes      []string `yaml:"capcodes"`
}

type TransmitChannel struct {
	LocalAudio       bool
	Multicast        bool
	Config           ChannelConfig
	TonesPath        string
	Address          []string
	TTSConfig        pipertts.PiperConfig
	LastAlertTone    time.Time
	LastAnnounceTone time.Time
	speaker          *pipertts.Speaker
	alertChan        chan *naadxml.Alert
	ctx              context.Context
	wg               *sync.WaitGroup
}

func NewTransmitter(ch chan *naadxml.Alert, config ChannelConfig, ctx context.Context, wg *sync.WaitGroup) (*TransmitChannel, error) {
	tx := &TransmitChannel{
		alertChan: ch,
		ctx:       ctx,
		wg:        wg,
		Config:    config,
	}
	wg.Add(1)
	go tx.handleMessages()
	return tx, nil
}

func (t *TransmitChannel) AddTTS(config pipertts.PiperConfig) {
	t.speaker = pipertts.NewSpeaker(config, t.ctx)
}

func (t *TransmitChannel) AddMulticast(addresses []string) {
	t.Multicast = true
	t.Address = append(t.Address, addresses...)
}

func (t *TransmitChannel) handleMessages() {
	defer t.wg.Done()
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
					log.Infof("Got audio at samplerate %d", samplerate)
					if err != nil {
						log.Errorf("Problem getting audio for alert %s language %s - %v", msg.Identifier, alertInfo.Language, err)
					}
					audio = t.addTone(alertInfo, audio, samplerate)
					if t.Multicast {
						log.Info("Channel has multicast output")
						if t.Config.G722 {
							mctx, txerr := rtptransmit.NewSender(t.Address, 9, 16000, 20) // G722 actually uses 8k timestamps
							if txerr != nil {
								log.Errorf("problem creating multicast transmitter - %v", txerr)
								continue
							}
							lowrate := rtptransmit.Downsample(audio, int(samplerate)/16000)
							encoder := go722.NewG722Encoder(go722.Rate64000, go722.G722_DEFAULT)
							g722Buf := new(bytes.Buffer)
							err = binary.Write(g722Buf, binary.LittleEndian, lowrate)
							if err != nil {
								log.Errorf("Problem converting audio to little-endian for g722 encoder - %v", err)
								continue
							}
							encoded := encoder.Encode(g722Buf.Bytes())
							mctx.SendBuffer(bytes.NewBuffer(encoded), 160, t.ctx)
							mctx.Stop()
						} else {
							mctx, txerr := rtptransmit.NewSender(t.Address, 0, 8000, 20) // G722 actually uses 8k timestamps
							if txerr != nil {
								log.Errorf("problem creating multicast transmitter - %v", txerr)
								continue
							}
							lowrate := rtptransmit.Downsample(audio, int(samplerate)/8000)
							encoded := g711.ULawEncode(lowrate)
							mctx.SendBuffer(bytes.NewBuffer(encoded), 160, t.ctx)
							mctx.Stop()
						}

					}
					if t.LocalAudio {
						// TODO send to local audio output
					}
				}
			}
		}
	}
}
