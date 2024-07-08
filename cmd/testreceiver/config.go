package main

import (
	"gopkg.in/yaml.v2"
	"os"
)

type Config struct {
	Servers  []string `yaml:"servers"`
	CAPCodes []int    `yaml:"capcodes"`
}

// Initialize a config object with default values
func (c *Config) Initialize() {
	*c = Config{
		Servers: []string{
			"streaming1.naad-adna.pelmorex.com:8080",
		},
		CAPCodes: []int{},
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
