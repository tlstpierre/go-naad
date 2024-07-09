package naadxml

import (
	log "github.com/sirupsen/logrus"
	"strconv"
	"strings"
)

type EC struct {
	BroadcastIntrusive          bool
	Event                       string
	AlertName                   string
	AlertType                   string
	AlertLocationStatus         string
	AlertCoverage               string
	DesignationCode             string
	NewlyActiveAreas            []string
	ParentURI                   string
	AdditionalAlertingAuthority string
	CAPCount                    uint64
}

func ECParam(ec *EC, version, parameter, value string) error {
	log.Debugf("EC version is %s parameter is %s value is %s", version, parameter, value)
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
			log.Errorf("Problem parsing CAP_Count value %s - %v", value, err)
			return err
		}
		ec.CAPCount = uint64(count)
	case "Alert_Location_Status":
		ec.AlertLocationStatus = value
	case "Alert_Name":
		ec.AlertName = value
	case "Alert_Coverage":
		ec.AlertCoverage = value
	case "Designation_Code":
		ec.DesignationCode = value
	case "Newly_Active_Areas":
		ec.NewlyActiveAreas = append(ec.NewlyActiveAreas, value)
	case "Additional_Alerting_Authority":
		ec.AdditionalAlertingAuthority = value
	default:
		log.Warnf("Unknown EC parameter %s with value %s", parameter, value)

	}
	return nil
}
