package main

import (
	"fmt"
	log "github.com/sirupsen/logrus"
	"github.com/tlstpierre/go-naad/pkg/naad-filter"
	"github.com/tlstpierre/go-naad/pkg/naad-xml"
)

var (
	thisPlace naadfilter.Place
)

func initFilter() {
	thisPlace = naadfilter.PointFromLatLon(configData.Lat, configData.Lon)
}

func displayInfo(alert *naadxml.AlertInfo) {
	/*
			fmt.Printf("\nNew Alert\n")
			fmt.Printf("Alert ID %s", msg.Identifier)
			fmt.Printf("Sender:\t%s\n", msg.Sender)
			fmt.Printf("Status:\t%s\n", msg.Status)
		fmt.Printf("Type:\t%s\n", msg.MsgType)
	*/
	var matchesCAP bool
	var matchesLocation bool
	var areaDescription string
	for _, code := range configData.CAPCodes {
		for _, area := range alert.Area {
			if naadfilter.IsCAPArea(area, code) {
				matchesCAP = true
				areaDescription = area.Description
			}

			if naadfilter.IsPlaceInArea(area, thisPlace) {
				matchesLocation = true
				areaDescription = area.Description
			}
		}
	}

	localAlert := matchesCAP || matchesLocation

	if localAlert {
		log.Warnf("\n\nThis alert is local\nCAP Code match: %v Lat/Lon match: %v\n\n", matchesCAP, matchesLocation)
	}

	fmt.Printf("\nAlert in %s\n", alert.Language)
	fmt.Printf("Event\t%s\n", alert.Event)
	fmt.Printf("Urgency\t%s\n", alert.Urgency)
	fmt.Printf("Severity\t%s\n", alert.Severity)
	fmt.Printf("Certainty\t%s\n", alert.Certainty)
	fmt.Printf("Headline\t%s\n", alert.Headline)
	fmt.Printf("Location\t%s\n", areaDescription)
	fmt.Printf("Description\t%s\n", alert.Description)
	if alert.SoremLayer != nil {
		log.Infof("SoremLayer is %+v", *alert.SoremLayer)
	}
	if alert.ECLayer != nil {
		log.Infof("EC Layer is %+v", *alert.ECLayer)
	}
	if alert.CAPLayer != nil {
		log.Infof("CAP layer is %+v", *alert.CAPLayer)
	}
	fmt.Printf("\n\n")

	fmt.Printf("\nEND OF ALERT\n\n")

}
