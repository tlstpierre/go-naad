package naadaudio

import (
	"bytes"
	"context"
	log "github.com/sirupsen/logrus"
	"github.com/tlstpierre/go-naad/pkg/naad-xml"
	"github.com/tlstpierre/mc-audio/pkg/g711"
	"github.com/tlstpierre/mc-audio/pkg/mc-transmit"
	"github.com/tlstpierre/mc-audio/pkg/piper-tts"
	"strings"
	"sync"
	"time"
)

type TransmitChannel struct {
	LocalAudio       bool
	Multicast        bool
	Language         string
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

func NewTransmitter(ch chan *naadxml.Alert, language string, ctx context.Context, wg *sync.WaitGroup) (*TransmitChannel, error) {
	tx := &TransmitChannel{
		alertChan: ch,
		ctx:       ctx,
		Language:  language,
		wg:        wg,
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
				if strings.EqualFold(alertInfo.Language, t.Language) {
					audio, samplerate, err := t.GetAudio(alertInfo)
					log.Infof("Got audio at samplerate %d", samplerate)
					if err != nil {
						log.Errorf("Problem getting audio for alert %s language %s - %v", msg.Identifier, alertInfo.Language, err)
					}
					audio = t.addTone(alertInfo, audio, samplerate)
					if t.Multicast {
						log.Info("Channel has multicast output")
						mctx, txerr := rtptransmit.NewSender(t.Address, 0, 8000, 20)
						if txerr != nil {
							log.Errorf("problem creating multicast transmitter - %v", txerr)
							continue
						}
						lowrate := rtptransmit.Downsample(audio, int(samplerate)/8000)
						encoded := g711.ULawEncode(lowrate)
						mctx.SendBuffer(bytes.NewBuffer(encoded), 160, t.ctx)
						mctx.Stop()
					}
					if t.LocalAudio {
						// TODO send to local audio output
					}
				}
			}
		}
	}
}
