package main

import (
	"context"
	log "github.com/sirupsen/logrus"
	"github.com/tlstpierre/go-naad/pkg/naad-audio"
	"github.com/tlstpierre/go-naad/pkg/naad-filter"
	"github.com/tlstpierre/go-naad/pkg/naad-xml"
	//	"github.com/tlstpierre/mc-audio/pkg/piper-tts"
	"sync"
)

var (
	//announcerWG     *sync.WaitGroup
	//announcerCancel context.CancelFunc
	announcers map[string]*Announcer
)

type Announcer struct {
	Config      naadaudio.ChannelConfig
	transmitter *naadaudio.TransmitChannel
	msgChan     chan *naadxml.Alert
}

func AnnouncerInit(ctx context.Context, wg *sync.WaitGroup) error {
	announcers = make(map[string]*Announcer, len(configData.Channels))

	for channel, channelConfig := range configData.Channels {
		log.Infof("Starting announcer %s", channel)
		announcer := &Announcer{
			Config:  channelConfig,
			msgChan: make(chan *naadxml.Alert, 10),
		}
		var err error
		announcer.transmitter, err = naadaudio.NewTransmitter(announcer.msgChan, channelConfig, ctx, wg)
		if err != nil {
			return err
		}
		piperConfig := configData.TTSConfig
		if channelConfig.Voice != "" {
			piperConfig.Voice = channelConfig.Voice
		}
		announcer.transmitter.AddTTS(piperConfig)
		announcer.transmitter.AddMulticast(channelConfig.Addresses)
		announcers[channel] = announcer
	}
	return nil
}

func AnnounceMessage(msg *naadxml.Alert) {
	for channel, announcer := range announcers {

		var matched bool
		if len(announcer.Config.CAPCodes) > 0 {
			for _, code := range announcer.Config.CAPCodes {
				if naadfilter.AlertIsCAPArea(msg, code) {
					matched = true
					log.Infof("Alert for channel %s matched on CAP code %s", channel, code)
					break
				}
			}
		} else {
			log.Infof("Announcing all CAP codes to %s", channel)
			matched = true
		}
		if !matched {
			continue
		}
		log.Infof("Announcing message %s to channel %s", msg.Identifier, channel)
		announcer.msgChan <- msg
	}
}
