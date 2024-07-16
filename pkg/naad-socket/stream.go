package naadsocket

import (
	"context"
	"encoding/xml"
	"fmt"
	log "github.com/sirupsen/logrus"
	"github.com/tlstpierre/go-naad/pkg/naad-xml"
	"net"
	"sync"
	"time"
)

type TCPReceiver struct {
	ctx        context.Context
	cancelFunc context.CancelFunc
	socket     net.Conn
	host       string
	handler    func(*naadxml.Alert) error
	wg         *sync.WaitGroup
}

func NewTCPReceiver(host string) (*TCPReceiver, error) {
	if host == "" {
		return nil, fmt.Errorf("Need host and port for TCP connection")
	}
	receiver := &TCPReceiver{
		host: host,
		ctx:  context.Background(),
		wg:   new(sync.WaitGroup),
	}
	receiver.ctx, receiver.cancelFunc = context.WithCancel(receiver.ctx)

	return receiver, nil
}

func (r *TCPReceiver) Connect() error {
	ctx, _ := context.WithTimeout(r.ctx, time.Minute)
	var err error
	var d net.Dialer
	log.Infof("Connecting to %s", r.host)
	r.socket, err = d.DialContext(ctx, "tcp", r.host)
	if err != nil {
		return fmt.Errorf("Could not dial %s - %v", r.host, err)
	}
	log.Infof("Connected to %s", r.host)
	r.wg.Add(1)
	go r.listen()
	return nil
}

func (r *TCPReceiver) Disconnect() {
	r.socket.Close()
	r.cancelFunc()
	r.wg.Wait()
}

func (r *TCPReceiver) SetHandler(handler func(*naadxml.Alert) error) {
	r.handler = handler
}

func (r *TCPReceiver) listen() {
	defer r.wg.Done()
	decoder := xml.NewDecoder(r.socket)
	var err error
	for {
		select {
		case <-r.ctx.Done():
			log.Infof("Closing connection to %s", r.host)
			r.socket.Close()
			return
		default:
			alert := new(naadxml.Alert)
			err = decoder.Decode(alert)
			if err != nil {
				log.Error(err)
				//				if err.Error() == "EOF" {
				r.socket.Close()
				log.Warnf("Connection to %s closed", r.host)
				time.Sleep(5 * time.Second)
				log.Infof("Attempting to re-connect to host %s", r.host)
				r.Connect()
				return
				//				}
			} else {
				log.Debugf("Decoded alert ID %s - type %s", alert.Identifier, alert.MsgType)
				alert.Receiver = r.host
				if r.handler != nil {
					err = r.handler(alert)
					if err != nil {
						log.Errorf("Problem processing alert ID %s - %v", alert.Identifier, err)
					}
				}
			}
		}
	}
}
