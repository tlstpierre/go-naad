package naadxml

import (
	log "github.com/sirupsen/logrus"
	"strings"
)

type Sorem struct {
	BroadcastImmediate bool
	WirelessImmediate  bool
	BroadcastText      string
	WirelessText       string
}

func SoremParam(sorem *Sorem, version, parameter, value string) error {
	log.Infof("Sorem version is %s parameter is %s value is %s", version, parameter, value)
	switch parameter {
	case "Broadcast_Immediately":
		sorem.BroadcastImmediate = strings.EqualFold(value, "yes")
	case "WirelessImmediate":
		sorem.WirelessImmediate = strings.EqualFold(value, "yes")
	case "Broadcast_Text":
		sorem.BroadcastText = value
	case "WirelessText":
		sorem.WirelessText = value
	}
	return nil
}
