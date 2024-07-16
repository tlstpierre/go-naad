package naadsocket

import (
	"context"
	"encoding/xml"
	"fmt"
	log "github.com/sirupsen/logrus"
	"github.com/tlstpierre/go-naad/pkg/naad-xml"
	"net"
	"sync"
	// "time"
)

type Listener struct {
	ctx        context.Context
	cancelFunc context.CancelFunc
	socket     *net.UDPConn
	addr       *net.UDPAddr
	handler    func(*naadxml.Alert) error
	wg         *sync.WaitGroup
}

func NewListener(address string) (*Listener, error) {
	if address == "" {
		return nil, fmt.Errorf("Need host and port for UDP socket")
	}
	listener := &Listener{
		ctx: context.Background(),
		wg:  new(sync.WaitGroup),
	}
	var err error
	listener.addr, err = net.ResolveUDPAddr("udp", address)
	if err != nil {
		return nil, err
	}

	return listener, nil
}

func (r *Listener) Connect() error {

	l, err := net.ListenMulticastUDP("udp", nil, r.addr)
	if err != nil {
		return err
	}
	r.socket = l

	log.Infof("Connected to %s", r.addr)
	r.wg.Add(1)
	go r.listen()
	return nil
}

func (r *Listener) Disconnect() {
	r.socket.Close()
	r.wg.Wait()
}

func (r *Listener) SetHandler(handler func(*naadxml.Alert) error) {
	r.handler = handler
}

func (r *Listener) listen() {
	defer r.wg.Done()
	decoder := xml.NewDecoder(r.socket)
	var err error
	for {
		select {
		case <-r.ctx.Done():
			log.Infof("Closing connection to %s", r.addr)
			r.socket.Close()
			return
		default:
			alert := new(naadxml.Alert)
			err = decoder.Decode(alert)
			if err != nil {
				log.Error(err)
				if err.Error() == "EOF" {
					log.Warnf("Got EOF on UDP listener %s", r.addr)
				}
			} else {
				log.Debugf("Decoded alert ID %s - type %s", alert.Identifier, alert.MsgType)
				alert.Receiver = r.addr.String()
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
