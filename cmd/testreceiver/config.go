package main

import (
	"fmt"
	log "github.com/sirupsen/logrus"
	"github.com/tlstpierre/go-naad/pkg/naad-audio"
	"github.com/tlstpierre/mc-audio/pkg/piper-tts"
	"gopkg.in/yaml.v2"
	"os"
)

type Config struct {
	StreamServers  []string                           `yaml:"streamservers"`
	ArchiveServers []string                           `yaml:"archiveservers"`
	CAPCodes       []string                           `yaml:"capcodes"`
	Lat            float64                            `yaml:"lat"`
	Lon            float64                            `yaml:"lon"`
	WebListen      string                             `yaml:"weblisten"`
	TTSConfig      pipertts.PiperConfig               `yaml:"ttsconfig"`
	Channels       map[string]naadaudio.ChannelConfig `yaml:"channels"`
}

// Initialize a config object with default values
func (c *Config) Initialize() {
	*c = Config{
		StreamServers: []string{
			"tcp://streaming1.naad-adna.pelmorex.com:8080",
			"tcp://streaming2.naad-adna.pelmorex.com:8080",
			"udp://224.0.10.10:25555",
		},
		ArchiveServers: []string{
			"capcp1.naad-adna.pelmorex.com",
			"capcp2.naad-adna.pelmorex.com",
		},
		CAPCodes: []string{
			"3518020", // Scugog
			"3518029", // Uxbridge
		},
		Lat:       44.10747,
		Lon:       -78.95514,
		WebListen: ":8081",
		TTSConfig: pipertts.PiperConfig{
			Samplerate: 16000,
			Command:    "/opt/piper/piper",
			VoicePath:  "/opt/piper",
			Voice:      "en_GB-alan-low",
		},
		Channels: map[string]naadaudio.ChannelConfig{
			"test": naadaudio.ChannelConfig{
				SpeakContent:  true,
				SoremOnly:     false,
				StripComments: true,
				Addresses: []string{
					"[FF05:0:0:0:0:0:1:1010]:5004",
				},
				Language: "en-CA",
			},
		},
	}
}

// Load any YAML values into the config object from a file.
func (c *Config) LoadFile(filename string) error {
	// Open config file
	file, err := os.Open(filename)
	if err != nil {
		return err
	}
	defer file.Close()

	// Init new YAML decode
	d := yaml.NewDecoder(file)

	// Start YAML decoding from file
	if err := d.Decode(c); err != nil {
		return err
	}
	return nil
}

func (c *Config) SaveFile(filename string) error {
	file, err := os.OpenFile(filename, os.O_RDWR|os.O_CREATE|os.O_TRUNC, 0644)
	if err != nil {
		return err
	}
	defer file.Close()

	e := yaml.NewEncoder(file)
	// Start YAML encoding to file
	if err := e.Encode(c); err != nil {
		return err
	}
	return nil
}

func (c *Config) OutputDefault() {
	output, err := yaml.Marshal(c)
	if err != nil {
		log.Errorf("Problem marshalling config - %v", err)
		return
	}
	fmt.Print(string(output))
}
