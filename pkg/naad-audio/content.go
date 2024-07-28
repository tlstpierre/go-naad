package naadaudio

import (
	"bytes"
	"encoding/binary"
	"errors"
	"fmt"
	"github.com/hajimehoshi/go-mp3"
	log "github.com/sirupsen/logrus"
	"github.com/tlstpierre/go-naad/pkg/naad-xml"
	"io"
	//	"github.com/tlstpierre/mc-audio/pkg/mc-transmit"
	"strings"
	"time"
)

const (
	AlertCooldown    = 1 * time.Minute
	AnnounceCooldown = 10 * time.Second
)

func (t *TransmitChannel) GetAudio(info *naadxml.AlertInfo) ([]int16, uint16, error) {
	var audio []int16
	var rate uint16
	var err error

	for _, resource := range info.Resources {
		log.Infof("Found resource %s %s %+v", resource.Description, resource.MimeType, resource)
		if resource.Description == "Broadcast Audio" || strings.HasSuffix(resource.Description, ".mp3") {
			log.Info("Found broadcast audio")
			if strings.EqualFold(resource.MimeType, "application/x-url") || (strings.EqualFold(resource.MimeType, "audio/mpeg") && len(resource.Content) == 0 && strings.HasPrefix(resource.URI, "http")) {
				log.Info("Broadcast audio is URL")
				err = resource.Fetch()
				if err != nil {
					log.Errorf("Problem fetching resource - %v", err)
					continue
				}
			}
			audio, rate, err = uncompressAudio(&resource)
			if err != nil {
				log.Errorf("Problem uncompressing audio - %v", err)
				continue
			}
			return audio, rate, nil
		}
	}
	log.Warnf("No audio content found - using TTS")
	var text string
	if info.SoremLayer != nil {
		if info.SoremLayer.BroadcastText != "" {
			text = info.SoremLayer.BroadcastText
		}
	}

	if text == "" {
		text = TTSText(info)
	}
	log.Infof("Spoken text will be %s", text)
	audio, err = t.speaker.Speak(text)
	if err != nil {
		return nil, 16000, err
	}
	return audio, t.speaker.Config.Samplerate, nil

}

func uncompressAudio(resource *naadxml.Resource) ([]int16, uint16, error) {
	log.Infof("Audio resource is %s length %d / %d", resource.MimeType, len(resource.Content), resource.Size)
	switch resource.MimeType {
	case "audio/mpeg":
		log.Info("Got MP3 file")
		buffer := bytes.NewBuffer([]byte(resource.Content))
		decoder, err := mp3.NewDecoder(buffer)
		if err != nil {
			return nil, 16000, fmt.Errorf("Problem creating MP3 decoder - %v", err)
		}
		samplerate := uint16(decoder.SampleRate())
		log.Infof("Set up MP3 decoder at samplerate %d length %d", samplerate, decoder.Length())
		highrate := make([]int16, samplerate*30)
		sample := make([]int16, 2)
		for {
			err := binary.Read(decoder, binary.LittleEndian, &sample)
			if err != nil {
				if errors.Is(err, io.EOF) {
					break
				}
				log.Errorf("Binary read error - %v", err)
				break
				//return nil, samplerate, fmt.Errorf("Problem reading audio data - %v", err)
			}
			highrate = append(highrate, sample[0])
		}
		//		mono := rtptransmit.MixMono(highrate)
		return highrate, samplerate, nil
	default:
		return nil, 16000, fmt.Errorf("Don't know how to decode audio of type %s", resource.MimeType)
	}
}

func (t *TransmitChannel) addTone(info *naadxml.AlertInfo, audio []int16, rate uint16) []int16 {
	var emergency bool
	if info.SoremLayer != nil {
		if info.SoremLayer.BroadcastImmediate {
			log.Info("Setting emergency to true due to Sorem Broadcast Immediate")
			emergency = true
		}
	}
	if info.Urgency != naadxml.Past && (info.Severity == naadxml.Extreme || info.Severity == naadxml.Severe) {
		log.Infof("Setting emergency to true due to Urgency %s and Severity %s", info.Urgency, info.Severity)
		emergency = true
	}

	if emergency {
		if time.Since(t.LastAlertTone) > AlertCooldown {
			audio := append(GenerateCAAS(uint32(rate)), audio...)
			t.LastAlertTone = time.Now()
			return audio
		} else {
			log.Warnf("Sent last CAAS tone at %s - not sending it again", t.LastAlertTone)
			return audio
		}
	}

	if time.Since(t.LastAnnounceTone) > AnnounceCooldown {
		t.LastAnnounceTone = time.Now()
		audio = append(AnnounceChime(uint32(rate)), audio...)
	}
	return audio
}
