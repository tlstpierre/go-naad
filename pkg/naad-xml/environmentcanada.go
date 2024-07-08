package naadxml

import (
	log "github.com/sirupsen/logrus"
	"strconv"
	"strings"
)

type EC struct {
	BroadcastIntrusive bool
	Event              string
	AlertType          string
	ParentURI          string
	CAPCount           uint64
}

func ECParam(ec *EC, version, parameter, value string) error {
	log.Infof("EC version is %s parameter is %s value is %s", version, parameter, value)
	switch parameter {
	case "Parent_URI":
		ec.ParentURI = value
	case "Alert_Type":
		ec.AlertType = value
	case "Broadcast_Intrusive":
		ec.BroadcastIntrusive = strings.EqualFold(value, "Yes")
	case "CAP_count":
		count, err := strconv.ParseInt(value, 10, 64)
		if err != nil {
			return err
		}
		ec.CAPCount = uint64(count)
	}
	return nil
}
