package main

import (
	"context"
	"fmt"
	//	"github.com/davecgh/go-spew/spew"
	log "github.com/sirupsen/logrus"
	"github.com/tlstpierre/go-naad/pkg/filter"
	"github.com/tlstpierre/go-naad/pkg/naad-tcp"
	"github.com/tlstpierre/go-naad/pkg/naad-xml"
	"net/http"
	"sync"
	"time"
)

var (
	thisPlace filter.Place
)

type Receiver struct {
	ctx           context.Context
	wg            *sync.WaitGroup
	Receivers     map[string]*naadtcp.Receiver
	lastHeartBeat time.Time
	msgChannel    chan *naadxml.Alert
}

func StartReceiver(ctx context.Context, wg *sync.WaitGroup) (*Receiver, error) {
	thisPlace = filter.PointFromLatLon(configData.Lat, configData.Lon)

	messageChannel := make(chan *naadxml.Alert, 4)
	msgHandler := func(msg *naadxml.Alert) error {
		messageChannel <- msg
		return nil
	}

	receiver := &Receiver{
		ctx:        ctx,
		wg:         wg,
		msgChannel: messageChannel,
		Receivers:  make(map[string]*naadtcp.Receiver, len(configData.StreamServers)),
	}

	for _, server := range configData.StreamServers {
		rx, err := naadtcp.NewReceiver(server)
		if err != nil {
			return nil, err
		}
		rx.AddHandler(msgHandler)
		receiver.Receivers[server] = rx
	}
	wg.Add(1)
	go receiver.receiverHandler()

	for _, rx := range receiver.Receivers {
		err := rx.Connect()
		if err != nil {
			return nil, err
		}
	}
	return receiver, nil
}

func (r *Receiver) receiverHandler() {
	log.Info("Starting receiver manager")
	defer r.wg.Done()
	cleanTicker := time.NewTicker(30 * time.Minute)
	deDuplicator := NewDeDuplicate()
	httpclient := &http.Client{
		Timeout: 10 * time.Second,
	}
	for {
		select {
		case <-r.ctx.Done():
			log.Info("Stopping receivers")
			for _, rx := range r.Receivers {
				rx.Disconnect()
			}
			return
		case <-cleanTicker.C:
			deDuplicator.CleanOld(time.Hour)
		case message := <-r.msgChannel:
			// Handle message

			// Check to see if message is a heartbeat
			if message.Sender == "NAADS-Heartbeat" {
				log.Infof("Received heartbeat from %s", message.Receiver)
				r.lastHeartBeat = time.Now()
				for _, ref := range message.References.References {
					// TODO run this as a worker
					if deDuplicator.HasReference(ref.Identifier) {
						continue
					}
					go func() {
						log.Infof("Fetching referenced message %s", ref.Identifier)
						refMessage, err := ref.Fetch(httpclient, configData.ArchiveServers[0])
						if err != nil {
							log.Error(err)
							return
						}
						deDuplicator.MarkReference(ref.Identifier)
						deDuplicator.MarkMessage(refMessage.Identifier)
						// Send the message onward
						displayMessage(refMessage)
					}()
				}
				continue
			}
			// If not, send it to the message handler
			if deDuplicator.HasMessage(message.Identifier) {
				log.Infof("Message %s is a duplicate", message.Identifier)
			}
			deDuplicator.MarkMessage(message.Identifier)
			// spew.Dump(message)
			displayMessage(message)
		}
	}
}

func displayMessage(msg *naadxml.Alert) {
	fmt.Printf("\nNew Alert\n")
	fmt.Printf("Alert ID %s", msg.Identifier)
	fmt.Printf("Sender:\t%s\n", msg.Sender)
	fmt.Printf("Status:\t%s\n", msg.Status)
	fmt.Printf("Type:\t%s\n", msg.MsgType)
	var matchesCAP bool
	for _, code := range configData.CAPCodes {
		matchesCAP = filter.AlertIsCAPArea(msg, code)
		if matchesCAP {
			break
		}
	}
	matchesLocation := filter.IsPlaceInAlert(msg, thisPlace)

	localAlert := matchesCAP || matchesLocation

	if localAlert {
		log.Warnf("\n\nThis alert is local\nCAP Code match: %v Lat/Lon match: %v\n\n", matchesCAP, matchesLocation)
	}

	msg.ProcessAlert()
	for _, alert := range msg.Info {
		if filter.IsPlaceInArea(alert.Area, thisPlace) {
			log.Warnf("Our place is %+v", thisPlace)
			log.Warnf("Alert area is %+v", *alert.Area.Polygon)
		}
		fmt.Printf("\nAlert in %s\n", alert.Language)
		fmt.Printf("Event\t%s\n", alert.Event)
		fmt.Printf("Urgency\t%s\n", alert.Urgency)
		fmt.Printf("Severity\t%s\n", alert.Severity)
		fmt.Printf("Certainty\t%s\n", alert.Certainty)
		fmt.Printf("Headline\t%s\n", alert.Headline)
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
	}

	fmt.Printf("\nEND OF ALERT\n\n")

}
