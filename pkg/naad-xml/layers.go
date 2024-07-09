package naadxml

import (
	"fmt"
	log "github.com/sirupsen/logrus"
	"strings"
)

type Layer struct {
	Format  string
	Version string
}

type LayperParam struct {
	Format  string
	Version string
	Value   string
}

type Profile struct {
	Profile string
	Version string
}

func (a *Alert) GetLayers() {
	for _, code := range a.Code {
		log.Debugf("Code is %s", code)
		parts := strings.Split(code, ":")
		log.Debugf("Parts are %+v", parts)
		if len(parts) == 3 {
			log.Debugf("Found element %s", parts[0])
			switch parts[0] {
			case "profile":
				a.Profiles = append(a.Profiles, Profile{
					Profile: parts[1],
					Version: parts[2],
				})
			case "layer":
				a.Layers = append(a.Layers, Layer{
					Format:  parts[1],
					Version: parts[2],
				})
			}
		}
	}
}

func (a *AlertInfo) ProcessLayerParam(param Parameter) error {
	parts := strings.Split(param.Name, ":")
	log.Debugf("Parts are %+v", parts)
	if len(parts) == 4 {
		log.Debugf("Found element %s", parts[0])
		switch parts[1] {
		case "SOREM", "sorem":
			if a.SoremLayer == nil {
				log.Debug("Adding SOREM layer")
				a.SoremLayer = &Sorem{}
			}
			return SoremParam(a.SoremLayer, parts[2], parts[3], param.Value)

		case "EC-MSC-SMC", "ec-msc-smc":
			if a.ECLayer == nil {
				log.Debug("Adding EC layer")
				a.ECLayer = &EC{}
			}
			return ECParam(a.ECLayer, parts[2], parts[3], param.Value)

		}
	}
	return fmt.Errorf("Expecting four parts in layer param name - got %s", param.Name)
}

func (a *AlertInfo) ProcessProfileParam(param Parameter) error {
	parts := strings.Split(param.Name, ":")
	log.Debugf("Parts are %+v", parts)
	if len(parts) == 4 {
		log.Debugf("Found element %s", parts[0])
		switch parts[1] {
		case "CAP-CP", "cap-cp":
			if a.CAPLayer == nil {
				log.Debug("Adding CAP-CP layer")
				a.CAPLayer = &CAP{}
			}
			return CAPParam(a.CAPLayer, parts[2], parts[3], param.Value)

		}
	}
	return fmt.Errorf("Expecting four parts in layer param name - got %s", param.Name)
}
