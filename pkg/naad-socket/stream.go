package naadsocket

import (
	"context"
	"encoding/xml"
	"errors"
	"fmt"
	log "github.com/sirupsen/logrus"
	"github.com/tlstpierre/go-naad/pkg/naad-xml"
	"io"
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
	if err := r.dial(); err != nil {
		return err
	}
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

// isConnectionError returns true for errors that indicate the TCP connection
// is lost and needs to be re-established (EOF, network errors), as opposed
// to XML parsing errors from malformed content on an otherwise healthy connection.
func isConnectionError(err error) bool {
	if errors.Is(err, io.EOF) || errors.Is(err, io.ErrUnexpectedEOF) {
		return true
	}
	var netErr *net.OpError
	if errors.As(err, &netErr) {
		return true
	}
	return false
}

func (r *TCPReceiver) listen() {
	defer r.wg.Done()

	backoff := 5 * time.Second
	const maxBackoff = 2 * time.Minute

	for {
		// Check for shutdown before starting a decode cycle
		select {
		case <-r.ctx.Done():
			log.Infof("Closing connection to %s", r.host)
			r.socket.Close()
			return
		default:
		}

		decoder := xml.NewDecoder(r.socket)
		if err := r.decodeLoop(decoder); err != nil {
			r.socket.Close()

			if !isConnectionError(err) {
				log.Errorf("XML parsing error on %s (reconnecting): %v", r.host, err)
			} else {
				log.Warnf("Connection to %s lost: %v", r.host, err)
			}

			// Backoff and reconnect
			log.Infof("Reconnecting to %s in %s", r.host, backoff)
			select {
			case <-time.After(backoff):
			case <-r.ctx.Done():
				return
			}

			if err := r.dial(); err != nil {
				log.Errorf("Failed to reconnect to %s: %v", r.host, err)
				backoff = min(backoff*2, maxBackoff)
				continue
			}
			// Successful reconnect -- reset backoff
			backoff = 5 * time.Second
		}
	}
}

// dial establishes the TCP connection without spawning a new goroutine.
func (r *TCPReceiver) dial() error {
	ctx, cancel := context.WithTimeout(r.ctx, time.Minute)
	defer cancel()
	var d net.Dialer
	log.Infof("Connecting to %s", r.host)
	conn, err := d.DialContext(ctx, "tcp", r.host)
	if err != nil {
		return fmt.Errorf("could not dial %s - %v", r.host, err)
	}
	r.socket = conn
	log.Infof("Connected to %s", r.host)
	return nil
}

// decodeLoop reads and dispatches alerts until a connection-level or parsing
// error occurs, at which point it returns the error.
func (r *TCPReceiver) decodeLoop(decoder *xml.Decoder) error {
	for {
		select {
		case <-r.ctx.Done():
			return nil
		default:
			alert := new(naadxml.Alert)
			err := decoder.Decode(alert)
			if err != nil {
				return err
			}
			log.Debugf("Decoded alert ID %s - type %s", alert.Identifier, alert.MsgType)
			alert.Receiver = r.host
			if r.handler != nil {
				if herr := r.handler(alert); herr != nil {
					log.Errorf("Problem processing alert ID %s - %v", alert.Identifier, herr)
				}
			}
		}
	}
}
