package main

import (
	"context"
	"fmt"
	log "github.com/sirupsen/logrus"
	"github.com/tlstpierre/go-naad/pkg/naad-xml"
	"time"
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

			// Cache the alert
			NaadCache.Add(alert)

			fmt.Printf("\nProcessing New Alert\n")
			fmt.Printf("Alert ID %s", alert.Identifier)
			fmt.Printf("Sender:\t%s\n", alert.Sender)
			fmt.Printf("Status:\t%s\n", alert.Status)
			fmt.Printf("Type:\t%s\n", alert.MsgType)
			err := alert.ProcessAlert()
			if err != nil {
				log.Warnf("Problem processing alert - %v", err)
			}
			AnnounceMessage(alert, func(ref naadxml.Reference) *naadxml.Alert {
				ctx, cancel := context.WithTimeout(p.ctx, 45*time.Second)
				defer cancel()
				original, err := NaadCache.Lookup(ctx, ref)
				if err != nil {
					log.Warnf("Could not get %s, referenced by %s: %v", ref.Identifier, alert.Identifier, err)
					return nil
				}
				return original
			})
			for _, info := range alert.Info {
				if info.Language != "en-CA" {
					continue
				}
				p.outChannel <- info
			}
		}
	}
}
