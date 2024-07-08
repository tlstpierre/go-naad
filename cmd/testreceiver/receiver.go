package main

import (
	"context"
	"github.com/davecgh/go-spew/spew"
	log "github.com/sirupsen/logrus"
	"github.com/tlstpierre/go-naad/pkg/naad-tcp"
	"github.com/tlstpierre/go-naad/pkg/naad-xml"
	"sync"

	"time"
)

type Receiver struct {
	ctx           context.Context
	wg            *sync.WaitGroup
	Receivers     []*naadtcp.Receiver
	lastHeartBeat time.Time
	msgChannel    chan *naadxml.Alert
}

func StartReceiver(ctx context.Context, wg *sync.WaitGroup) (*Receiver, error) {
	messageChannel := make(chan *naadxml.Alert, 4)

	msgHandler := func(msg *naadxml.Alert) error {
		messageChannel <- msg
		return nil
	}

	receiver := &Receiver{
		ctx:        ctx,
		wg:         wg,
		msgChannel: messageChannel,
		Receivers:  make([]*naadtcp.Receiver, 0, len(configData.Servers)),
	}

	for _, server := range configData.Servers {
		rx, err := naadtcp.NewReceiver(server)
		if err != nil {
			return nil, err
		}
		rx.AddHandler(msgHandler)
		receiver.Receivers = append(receiver.Receivers, rx)
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
	for {
		select {
		case <-r.ctx.Done():
			log.Info("Stopping receivers")
			for _, rx := range r.Receivers {
				rx.Disconnect()
			}
			return
		case message := <-r.msgChannel:
			// Handle message

			// Check to see if message is a heartbeat

			// If not, send it to the message handler
			spew.Dump(message)
		}
	}
}

type DeDuplicate struct {
	sync.RWLock
	messageList         map[uuid.UUID]struct{}
	retrievedReferences map[string]struct{}
}

func (d *DeDuplicate) HasMessage(msg uuid.UUID) bool {

}

func (d *DeDuplicate) MarkMessage(msg uuid.UUID) {

}
