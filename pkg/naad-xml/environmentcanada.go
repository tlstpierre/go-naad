package naadxml

import (
	"strconv"
	"strings"
	"sync"
	"time"

	log "github.com/sirupsen/logrus"
)

// EC holds an info block's Environment Canada (EC-MSC-SMC layer)
// parameters. The layer is documented in ECCC's CAP release notes
// (https://collaboration.cmc.ec.gc.ca/cmc/cmos/doc/CAP/CAP-Release-Note_EN.pdf);
// the risk-tiered (colour) parameters aren't, beyond the public colour
// matrix of impact against confidence.
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
	CAPCount                    CAPCounter

	// Risk-tiered ("colour-coded") alerting, since November 2025.
	Colour         string // yellow, orange, red (jaune, orange, rouge)
	PreviousColour string // the colour before this update; empty on the first message
	Impact         string // MSC_Impact: low, moderate, high, extreme (localized)
	Confidence     string // MSC_Confidence: low, moderate, high, very high (localized)
	// RiskRanking is MSC_Risk_Tiered_Ranking. ECCC doesn't document the
	// scale; 8 has been seen with moderate impact, high confidence, yellow.
	RiskRanking             int
	EventEndTime            time.Time
	EventEndTimeDescription string // e.g. "Until further notice"

	// Storm details, on tornado and severe thunderstorm warnings with a
	// drawn threat area (active blocks only).
	Storm StormInfo
}

// StormInfo describes the storm behind a tornado or severe thunderstorm
// warning.
type StormInfo struct {
	Speed                   string    // "40 km/h" over land, knots over water
	Direction               float64   // degrees
	GeometryType            string    // isolated_cell, area or line
	Points                  []string  // 1-3 "lat,lon" centre reference points
	Time                    time.Time // when the forecaster last placed it
	MotionDescription       string    // "east at 40 km/h"
	PositionDescription     string    // "about 25 km southwest of Burnstick Lake"
	ReferenceLocationPoints string    // places affected, e.g. "Water Valley, Cremona"
}

// unknownECParams remembers the EC parameters already reported, so a new
// one ECCC adds is logged once rather than for every alert.
var unknownECParams sync.Map

type CAPCounter struct {
	A uint64
	M uint64
	C uint64
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
		ec.CAPCount = parseECCapCount(value)
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
	case "Colour":
		ec.Colour = value
	case "Previous_Colour":
		ec.PreviousColour = value
	case "MSC_Impact":
		ec.Impact = value
	case "MSC_Confidence":
		ec.Confidence = value
	case "MSC_Risk_Tiered_Ranking":
		n, err := strconv.Atoi(strings.TrimSpace(value))
		if err != nil {
			log.Debugf("EC MSC_Risk_Tiered_Ranking %q is not a number", value)
		}
		ec.RiskRanking = n
	case "Event_End_Time":
		t, err := time.Parse(time.RFC3339, strings.TrimSpace(value))
		if err != nil {
			log.Debugf("EC Event_End_Time %q: %v", value, err)
		}
		ec.EventEndTime = t
	case "Event_End_Time_Description":
		ec.EventEndTimeDescription = value
	case "Storm_Speed":
		ec.Storm.Speed = value
	case "Storm_Direction":
		ec.Storm.Direction, _ = strconv.ParseFloat(strings.TrimSpace(value), 64)
	case "Storm_Geometry_Type":
		ec.Storm.GeometryType = value
	case "Storm_Point":
		ec.Storm.Points = append(ec.Storm.Points, value)
	case "Storm_Time":
		// yyyyMMddHHmmssSSS, UTC
		if v := strings.TrimSpace(value); len(v) >= 14 {
			ec.Storm.Time, _ = time.Parse("20060102150405", v[:14])
		}
	case "Motion_Description":
		ec.Storm.MotionDescription = value
	case "Storm_Position_Description":
		ec.Storm.PositionDescription = value
	case "Reference_Location_Points":
		ec.Storm.ReferenceLocationPoints = value
	default:
		if _, seen := unknownECParams.LoadOrStore(parameter, true); !seen {
			log.Warnf("Unknown EC parameter %s (layer version %s) with value %q; further ones are logged at debug level", parameter, version, value)
		} else {
			log.Debugf("Unknown EC parameter %s with value %q", parameter, value)
		}
	}
	return nil
}

func parseECCapCount(value string) CAPCounter {
	list := strings.Split(value, " ")
	var counter CAPCounter
	for _, entry := range list {
		entryFields := strings.Split(entry, ":")
		if len(entryFields) == 2 {
			count, _ := strconv.ParseInt(entryFields[1], 10, 64)
			switch entryFields[0] {
			case "A":
				counter.A = uint64(count)
			case "M":
				counter.M = uint64(count)
			case "C":
				counter.C = uint64(count)
			default:
				log.Warnf("Unknown EC CAP Count value of %s", entryFields[0])
			}
		}
	}
	return counter
}
