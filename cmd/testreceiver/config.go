package main

import (
	"gopkg.in/yaml.v2"
	"os"
)

type Config struct {
	StreamServers  []string `yaml:"streamservers"`
	ArchiveServers []string `yaml:"archiveservers"`
	CAPCodes       []int    `yaml:"capcodes"`
	Lat            float64  `yaml:"lat"`
	Lon            float64  `yaml:"lon"`
}

// Initialize a config object with default values
func (c *Config) Initialize() {
	*c = Config{
		StreamServers: []string{
			"streaming1.naad-adna.pelmorex.com:8080",
		},
		ArchiveServers: []string{
			"capcp1.naad-adna.pelmorex.com",
		},
		CAPCodes: []int{
			3518020, // Scugog
			3518029, // Uxbridge
		},
		Lat: 44.10747,
		Lon: -78.95514,
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
