//go:build !localaudio

package main

import "github.com/tlstpierre/go-naad/pkg/naad-audio"

func newAudioBackend() naadaudio.AudioBackend {
	return nil
}
