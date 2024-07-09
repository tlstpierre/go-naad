package main

import (
	"context"
	"fmt"
	log "github.com/sirupsen/logrus"
	"github.com/tlstpierre/go-naad/pkg/naad-xml"
)

type Processor struct {
	ctx        context.Context
	inChannel  chan *naadxml.Alert
	outChannel chan *naadxml.AlertInfo
}

func NewProcessor(in chan *naadxml.Alert, out chan *naadxml.AlertInfo, ctx context.Context) *Processor {
	p := &Processor{
		ctx:        ctx,
		inChannel:  in,
		outChannel: out,
	}
	go p.run()
	return p
}

func (p *Processor) run() {
	log.Info("Starting message processor")
	for {
		select {
		case <-p.ctx.Done():
			log.Infof("Stopping processor")
			return
		case alert := <-p.inChannel:
			// Process the alert here

			fmt.Printf("\nProcessing New Alert\n")
			fmt.Printf("Alert ID %s", alert.Identifier)
			fmt.Printf("Sender:\t%s\n", alert.Sender)
			fmt.Printf("Status:\t%s\n", alert.Status)
			fmt.Printf("Type:\t%s\n", alert.MsgType)
			alert.ProcessAlert()
			for _, info := range alert.Info {
				if info.Language != "en-CA" {
					continue
				}
				p.outChannel <- info
			}
		}
	}
}
