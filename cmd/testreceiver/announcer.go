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

// referenceLookup finds a message that an Update or Cancel refers to,
// from the cache or the archive; nil if it can't be had.
type referenceLookup func(naadxml.Reference) *naadxml.Alert

// matchChannel decides whether a message goes to a channel, why, and which
// of its info blocks the channel should announce.
//
// A channel with neither CAP codes nor geofences takes every message whole.
// Otherwise a message matches on its own areas in effect (stand-down blocks
// and ended or cancelled threat areas never start an alert), and the
// channel announces its in-effect blocks. Failing that, an Update or Cancel
// matches if any message it refers to matched on its own areas: that is a
// stand-down or amendment for an alert the channel announced (or would
// have, had it been running), so the channel gets it whatever its own
// geometry, and announces its stand-down blocks if it has any.
func matchChannel(config naadaudio.ChannelConfig, msg *naadxml.Alert, lookup referenceLookup) (bool, string, []*naadxml.AlertInfo) {
	if len(config.CAPCodes) == 0 && len(config.GeoFences) == 0 {
		return true, "no location filter", msg.Info
	}
	if reason := matchAreas(config, msg); reason != "" {
		var active []*naadxml.AlertInfo
		for _, info := range msg.Info {
			if info.InEffect() {
				active = append(active, info)
			}
		}
		return true, reason, active
	}
	if !msg.RefersBack() || lookup == nil {
		return false, "", nil
	}
	for _, ref := range msg.References.References {
		original := lookup(ref)
		if original == nil {
			continue
		}
		if reason := matchAreas(config, original); reason != "" {
			var standDown []*naadxml.AlertInfo
			for _, info := range msg.Info {
				if !info.InEffect() {
					standDown = append(standDown, info)
				}
			}
			if len(standDown) == 0 {
				standDown = msg.Info
			}
			return true, "reference to " + ref.Identifier + ", which matched on " + reason, standDown
		}
	}
	return false, "", nil
}

// matchAreas says how a message's areas in effect match the channel's CAP
// codes or geofences, or "" if they don't.
func matchAreas(config naadaudio.ChannelConfig, msg *naadxml.Alert) string {
	for _, code := range config.CAPCodes {
		if naadfilter.AlertIsCAPArea(msg, code) {
			return "CAP code " + code
		}
	}
	for _, fence := range config.GeoFences {
		if naadfilter.AlertInGeoFence(msg, fence) {
			return "geofence " + fence.String()
		}
	}
	return ""
}

// AnnounceMessage sends a message to every channel it matches, each
// getting a copy holding only the info blocks that concern it.
func AnnounceMessage(msg *naadxml.Alert, lookup referenceLookup) {
	for channel, announcer := range announcers {
		matched, reason, infos := matchChannel(announcer.Config, msg, lookup)
		if !matched {
			continue
		}
		log.Infof("Alert %s for channel %s matched on %s", msg.Identifier, channel, reason)
		if len(infos) == 0 {
			continue
		}
		channelMsg := *msg
		channelMsg.Info = infos
		if announcer.dupSuppressor.IsDuplicate(&channelMsg) {
			log.Infof("Skipping message %s to channel %s because it hasn't changed enough", msg.Identifier, channel)
			continue
		}
		announcer.dupSuppressor.Insert(&channelMsg)
		log.Infof("Announcing message %s to channel %s", msg.Identifier, channel)
		announcer.msgChan <- &channelMsg
		announcer.dupSuppressor.Clean()
	}
}
