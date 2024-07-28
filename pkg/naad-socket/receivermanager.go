package naadsocket

import (
	"context"
	//	"github.com/davecgh/go-spew/spew"
	"fmt"
	log "github.com/sirupsen/logrus"
	"github.com/tlstpierre/go-naad/pkg/naad-filter"
	"github.com/tlstpierre/go-naad/pkg/naad-xml"
	"net/http"
	"strings"
	"sync"
	"time"
)

type ReceiverGroup struct {
	ctx            context.Context
	wg             *sync.WaitGroup
	Receivers      map[string]Receiver
	ArchiveServers []string
	lastHeartBeat  time.Time
	msgChannel     chan *naadxml.Alert
	outChannel     chan *naadxml.Alert
	fetchChannel   chan naadxml.Reference
	deDuplicator   *naadfilter.DeDuplicate
}

func NewReceiverGroup(sources []string, archiveServers []string, delivery chan *naadxml.Alert, ctx context.Context) (*ReceiverGroup, error) {
	rxg := &ReceiverGroup{
		ctx:            ctx,
		wg:             &sync.WaitGroup{},
		outChannel:     delivery,
		msgChannel:     make(chan *naadxml.Alert, len(sources)*2),
		fetchChannel:   make(chan naadxml.Reference, 12),
		deDuplicator:   naadfilter.NewDeDuplicate(),
		Receivers:      make(map[string]Receiver, len(sources)),
		ArchiveServers: archiveServers,
	}

	msgHandler := func(msg *naadxml.Alert) error {
		rxg.msgChannel <- msg
		return nil
	}

	for _, src := range sources {
		if strings.HasPrefix(src, "tcp://") {
			log.Infof("Adding TCP socket receiver %s", src)
			addr := strings.TrimPrefix(src, "tcp://")
			rx, err := NewTCPReceiver(addr)
			if err != nil {
				return nil, fmt.Errorf("Could not create TCP receiver on %s - %v", src, err)
			}
			rx.SetHandler(msgHandler)
			rxg.Receivers[src] = rx

		} else if strings.HasPrefix(src, "udp://") {
			log.Infof("Adding UDP listener %s", src)
			addr := strings.TrimPrefix(src, "udp://")
			rx, err := NewListener(addr)
			if err != nil {
				return nil, fmt.Errorf("Could not create UDP listener on %s - %v", src, err)
			}
			rx.SetHandler(msgHandler)
			rxg.Receivers[src] = rx
		}
	}

	return rxg, nil
}

func (g *ReceiverGroup) Start() error {
	g.wg.Add(1)
	go g.receiverHandler()
	go g.referenceFetcher()

	return nil
}

func (r *ReceiverGroup) receiverHandler() {
	log.Info("Starting receiver manager")
	defer r.wg.Done()
	cleanTicker := time.NewTicker(30 * time.Minute)

	for rxname, rx := range r.Receivers {
		err := rx.Connect()
		if err != nil {
			log.Errorf("Problem connecting to streaming server %s - %v", rxname, err)
		}
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
			r.deDuplicator.CleanOld(24 * time.Hour)
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
						r.fetchChannel <- ref
					}
				}
				continue
			}

			// If not, send it to the message handler
			if r.deDuplicator.HasMessage(message.Identifier) {
				log.Infof("Message %s is a duplicate", message.Identifier)
				continue
			}
			r.deDuplicator.MarkMessage(message.Identifier)
			// spew.Dump(message)
			r.outChannel <- message
		}
	}
}

func (r *ReceiverGroup) referenceFetcher() {
	httpclient := &http.Client{
		Timeout: 10 * time.Second,
	}

	for {
		select {
		case <-r.ctx.Done():
			return
		case ref := <-r.fetchChannel:
			log.Infof("Fetching referenced message %s", ref.Identifier)
			var refMessage *naadxml.Alert
			for _, server := range r.ArchiveServers {
				var err error
				refMessage, err = ref.Fetch(httpclient, server)
				if err != nil {
					log.Error(err)
				} else {
					continue
				}
			}
			r.deDuplicator.MarkReference(ref.Identifier)
			r.deDuplicator.MarkMessage(refMessage.Identifier)
			// Send the message onward
			r.outChannel <- refMessage
		}
	}
}
