//go:build localaudio

package main

import (
	log "github.com/sirupsen/logrus"
	"github.com/tlstpierre/go-naad/pkg/naad-audio"
	"gitlab.com/timstpierre/mc-audio/pkg/audio"
)

// malgoBackendAdapter wraps mc-audio's Backend to satisfy naadaudio.AudioBackend.
type malgoBackendAdapter struct {
	backend *audio.MalgoBackend
}

func (a *malgoBackendAdapter) OpenPlayback(deviceID string, sampleRate, channels, frameSize int) (naadaudio.LocalPlayback, error) {
	return a.backend.OpenPlayback(audio.PlaybackConfig{
		DeviceID:   deviceID,
		SampleRate: sampleRate,
		Channels:   channels,
		FrameSize:  frameSize,
	})
}

func (a *malgoBackendAdapter) Close() error {
	return a.backend.Close()
}

func newAudioBackend() naadaudio.AudioBackend {
	backend, err := audio.NewMalgoBackend()
	if err != nil {
		log.Errorf("Could not initialize audio backend: %v", err)
		return nil
	}
	return &malgoBackendAdapter{backend: backend}
}
