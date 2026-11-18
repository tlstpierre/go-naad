package main

import (
	"context"
	log "github.com/sirupsen/logrus"
	"github.com/tlstpierre/go-naad/pkg/naad-audio"
	"github.com/tlstpierre/go-naad/pkg/naad-filter"
	"github.com/tlstpierre/go-naad/pkg/naad-xml"
	"sync"
	"time"
)

var (
	announcers map[string]*Announcer
)

type Announcer struct {
	Config        naadaudio.ChannelConfig
	transmitter   *naadaudio.TransmitChannel
	dupSuppressor *naadfilter.UpdateSuppressor
	msgChan       chan *naadxml.Alert
}

func AnnouncerInit(ctx context.Context, wg *sync.WaitGroup) error {
	announcers = make(map[string]*Announcer, len(configData.Channels))

	// Only initialize the audio backend if any channel needs local audio
	var audioBackend naadaudio.AudioBackend
	for _, channelConfig := range configData.Channels {
		if channelConfig.LocalAudio {
			audioBackend = newAudioBackend()
			break
		}
	}

	for channel, channelConfig := range configData.Channels {
		log.Infof("Starting announcer %s", channel)
		announcer := &Announcer{
			Config:  channelConfig,
			msgChan: make(chan *naadxml.Alert, 10),
		}
		var err error
		announcer.transmitter, err = naadaudio.NewTransmitter(announcer.msgChan, channelConfig, ctx, wg, audioBackend)
		if err != nil {
			return err
		}
		piperConfig := configData.TTSConfig
		if channelConfig.Voice != "" {
			piperConfig.Voice = channelConfig.Voice
		}
		announcer.transmitter.AddTTS(piperConfig)
		announcer.dupSuppressor = naadfilter.NewUpdateSuppressor(2 * time.Hour)
		announcers[channel] = announcer
	}
	return nil
}

// matchChannel reports whether an alert should be announced on a channel and
// why. A channel with neither CAP codes nor geofences takes every alert.
func matchChannel(config naadaudio.ChannelConfig, msg *naadxml.Alert) (bool, string) {
	if len(config.CAPCodes) == 0 && len(config.GeoFences) == 0 {
		return true, "no location filter"
	}
	for _, code := range config.CAPCodes {
		if naadfilter.AlertIsCAPArea(msg, code) {
			return true, "CAP code " + code
		}
	}
	for _, fence := range config.GeoFences {
		if naadfilter.AlertInGeoFence(msg, fence) {
			return true, "geofence " + fence.String()
		}
	}
	return false, ""
}

func AnnounceMessage(msg *naadxml.Alert) {
	for channel, announcer := range announcers {
		matched, reason := matchChannel(announcer.Config, msg)
		if !matched {
			continue
		}
		log.Infof("Alert %s for channel %s matched on %s", msg.Identifier, channel, reason)
		if announcer.dupSuppressor.IsDuplicate(msg) {
			log.Infof("Skipping message %s to channel %s because it hasn't changed enough", msg.Identifier, channel)
			continue
		}
		announcer.dupSuppressor.Insert(msg)
		log.Infof("Announcing message %s to channel %s", msg.Identifier, channel)
		announcer.msgChan <- msg
		announcer.dupSuppressor.Clean()
	}
}
