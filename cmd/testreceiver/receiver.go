package main

import (
	"context"
	//	"github.com/davecgh/go-spew/spew"
	log "github.com/sirupsen/logrus"
	"github.com/tlstpierre/go-naad/pkg/naad-filter"
	"github.com/tlstpierre/go-naad/pkg/naad-tcp"
	"github.com/tlstpierre/go-naad/pkg/naad-xml"
	"net/http"
	"sync"
	"time"
)

var (
	thisPlace naadfilter.Place
)

type Receiver struct {
	ctx           context.Context
	wg            *sync.WaitGroup
	Receivers     map[string]*naadtcp.Receiver
	lastHeartBeat time.Time
	msgChannel    chan *naadxml.Alert
	outChannel    chan *naadxml.Alert
	fetchChannel  chan naadxml.Reference
	deDuplicator  *naadfilter.DeDuplicate
}

func StartReceiver(ctx context.Context, wg *sync.WaitGroup, outChan chan *naadxml.Alert) (*Receiver, error) {
	thisPlace = naadfilter.PointFromLatLon(configData.Lat, configData.Lon)

	messageChannel := make(chan *naadxml.Alert, 4)
	msgHandler := func(msg *naadxml.Alert) error {
		messageChannel <- msg
		return nil
	}

	receiver := &Receiver{
		ctx:          ctx,
		wg:           wg,
		msgChannel:   messageChannel,
		outChannel:   outChan,
		fetchChannel: make(chan naadxml.Reference, 10), // Leave room for up to 10 references
		Receivers:    make(map[string]*naadtcp.Receiver, len(configData.StreamServers)),
		deDuplicator: naadfilter.NewDeDuplicate(),
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
	go receiver.referenceFetcher()
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

	for {
		select {
		case <-r.ctx.Done():
			log.Info("Stopping receivers")
			for _, rx := range r.Receivers {
				rx.Disconnect()
			}
			return
		case <-cleanTicker.C:
			r.deDuplicator.CleanOld(time.Hour)

		case message := <-r.msgChannel:
			// Handle message

			// Check to see if message is a heartbeat
			if message.Sender == "NAADS-Heartbeat" {
				log.Infof("Received heartbeat from %s", message.Receiver)
				r.lastHeartBeat = time.Now()
				for _, ref := range message.References.References {
					if r.deDuplicator.HasReference(ref.Identifier) {
						continue
					} else {
						log.Infof("Sending ref %s to be fetched", ref.Identifier)
						r.fetchChannel <- ref
					}
				}
				continue
			}

			// If not, send it to the message handler
			if r.deDuplicator.HasMessage(message.Identifier) {
				log.Infof("Message %s is a duplicate", message.Identifier)
			}
			r.deDuplicator.MarkMessage(message.Identifier)
			// spew.Dump(message)
			log.Infof("Sending message %s for processing", message.Identifier)
			r.outChannel <- message
		}
	}
}

func (r *Receiver) referenceFetcher() {
	httpclient := &http.Client{
		Timeout: 10 * time.Second,
	}

	for {
		select {
		case <-r.ctx.Done():
			return
		case ref := <-r.fetchChannel:
			log.Infof("Fetching referenced message %s", ref.Identifier)
			refMessage, err := ref.Fetch(httpclient, configData.ArchiveServers[0])
			if err != nil {
				log.Error(err)
				continue
			}
			r.deDuplicator.MarkReference(ref.Identifier)
			r.deDuplicator.MarkMessage(refMessage.Identifier)
			// Send the message onward
			r.outChannel <- refMessage
		}
	}
}
