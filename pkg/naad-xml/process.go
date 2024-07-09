package naadxml

import (
	log "github.com/sirupsen/logrus"
	"strings"
)

func (a *Alert) ProcessAlert() error {
	for index, info := range a.Info {
		err := info.ProcessInfo()
		if err != nil {
			log.Errorf("Problem processing alert %d from %s", index, a.Identifier)
		}
	}
	return nil
}

func (i *AlertInfo) ProcessInfo() error {
	if err := i.ProcessParams(); err != nil {
		log.Errorf("Problem processing param for %s - %v", i.Description, err)
	}

	for _, resource := range i.Resources {
		if resource.MimeType == "application/x-url" && strings.EqualFold(resource.Description, "Broadcast Audio") {
			err := resource.Fetch()
			if err != nil {
				log.Errorf("Could not fetch resource %s - %v", resource.Description, err)
			}
		}
	}
	return nil
}
