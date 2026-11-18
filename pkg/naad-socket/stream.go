package naadsocket

import (
	"context"
	"encoding/xml"
	"errors"
	"fmt"
	log "github.com/sirupsen/logrus"
	"github.com/tlstpierre/go-naad/pkg/naad-xml"
	"io"
	"math/rand"
	"net"
	"sync"
	"sync/atomic"
	"time"
)

type TCPReceiverOptions struct {
	ConnectTimeout time.Duration `yaml:"connecttimeout"`
	// ReadTimeout is how long the connection may go without receiving any
	// data before it is considered dead. NAAD sends a heartbeat every
	// minute, so the default of 150s tolerates one missed heartbeat.
	ReadTimeout time.Duration `yaml:"readtimeout"`
	// MinBackoff and MaxBackoff bound the reconnect delay. The delay
	// doubles on every connection that fails or drops before becoming
	// stable, and resets once a connection is stable.
	MinBackoff time.Duration `yaml:"minbackoff"`
	MaxBackoff time.Duration `yaml:"maxbackoff"`
	// StableAfter is how long a connection must stay up, having delivered
	// at least one message, before the backoff resets.
	StableAfter time.Duration `yaml:"stableafter"`
}

func (o *TCPReceiverOptions) setDefaults() {
	if o.ConnectTimeout == 0 {
		o.ConnectTimeout = time.Minute
	}
	if o.ReadTimeout == 0 {
		o.ReadTimeout = 150 * time.Second
	}
	if o.MinBackoff == 0 {
		o.MinBackoff = 5 * time.Second
	}
	if o.MaxBackoff == 0 {
		o.MaxBackoff = 5 * time.Minute
	}
	if o.StableAfter == 0 {
		o.StableAfter = time.Minute
	}
}

// deadlineReader refreshes the read deadline before every read so that a
// silent (half-open) connection eventually errors out instead of blocking
// forever. It also records whether any data has arrived.
type deadlineReader struct {
	conn    net.Conn
	timeout time.Duration
	gotData *atomic.Bool
}

func (d deadlineReader) Read(p []byte) (int, error) {
	if err := d.conn.SetReadDeadline(time.Now().Add(d.timeout)); err != nil {
		return 0, err
	}
	n, err := d.conn.Read(p)
	if n > 0 {
		d.gotData.Store(true)
	}
	return n, err
}

// nextBackoff doubles the delay up to max.
func nextBackoff(current, min, max time.Duration) time.Duration {
	if current < min {
		return min
	}
	next := current * 2
	if next > max {
		next = max
	}
	return next
}

// jitter spreads the delay by +/-20% so multiple receivers don't retry in
// lockstep.
func jitter(d time.Duration) time.Duration {
	spread := int64(d) / 5
	if spread <= 0 {
		return d
	}
	return d + time.Duration(rand.Int63n(2*spread)-spread)
}

type TCPReceiver struct {
	ctx        context.Context
	cancelFunc context.CancelFunc
	socketMu   sync.Mutex
	socket     net.Conn
	host       string
	handler    func(*naadxml.Alert) error
	wg         *sync.WaitGroup
	options    TCPReceiverOptions
}

func NewTCPReceiver(host string, opts ...TCPReceiverOptions) (*TCPReceiver, error) {
	if host == "" {
		return nil, fmt.Errorf("Need host and port for TCP connection")
	}
	var options TCPReceiverOptions
	if len(opts) > 0 {
		options = opts[0]
	}
	options.setDefaults()
	receiver := &TCPReceiver{
		host:    host,
		ctx:     context.Background(),
		wg:      new(sync.WaitGroup),
		options: options,
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
	r.cancelFunc()
	r.closeSocket() // unblock any pending read
	r.wg.Wait()
}

func (r *TCPReceiver) currentSocket() net.Conn {
	r.socketMu.Lock()
	defer r.socketMu.Unlock()
	return r.socket
}

func (r *TCPReceiver) closeSocket() {
	r.socketMu.Lock()
	defer r.socketMu.Unlock()
	if r.socket != nil {
		r.socket.Close()
	}
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

	backoff := r.options.MinBackoff
	failures := 0

	for {
		// Check for shutdown before starting a decode cycle
		select {
		case <-r.ctx.Done():
			log.Infof("Closing connection to %s", r.host)
			r.closeSocket()
			return
		default:
		}

		connectedAt := time.Now()
		var gotData atomic.Bool
		conn := r.currentSocket()
		decoder := xml.NewDecoder(deadlineReader{conn: conn, timeout: r.options.ReadTimeout, gotData: &gotData})
		err := r.decodeLoop(decoder)
		conn.Close()
		if err == nil { // context cancelled
			continue
		}
		uptime := time.Since(connectedAt).Round(time.Millisecond)

		// A connection that was up for a while and delivered data was
		// healthy: start over at the minimum delay. Anything else (e.g. the
		// server accepting then immediately closing) counts as a failure so
		// we back off instead of hammering the server.
		if gotData.Load() && uptime >= r.options.StableAfter {
			backoff = r.options.MinBackoff
			failures = 0
		} else {
			failures++
		}

		var netErr net.Error
		switch {
		case errors.As(err, &netErr) && netErr.Timeout():
			log.Warnf("No data from %s for %s, connection presumed dead (up %s)", r.host, r.options.ReadTimeout, uptime)
		case isConnectionError(err):
			log.Warnf("Connection to %s lost after %s: %v", r.host, uptime, err)
		default:
			log.Errorf("XML parsing error on %s after %s (reconnecting): %v", r.host, uptime, err)
		}

		// Keep retrying (with backoff) until a dial succeeds
		for {
			delay := jitter(backoff)
			if failures > 0 {
				log.Infof("Reconnecting to %s in %s (%d consecutive failures)", r.host, delay.Round(time.Second), failures)
			} else {
				log.Infof("Reconnecting to %s in %s", r.host, delay.Round(time.Second))
			}
			select {
			case <-time.After(delay):
			case <-r.ctx.Done():
				return
			}
			if failures > 0 {
				backoff = nextBackoff(backoff, r.options.MinBackoff, r.options.MaxBackoff)
			}
			if err := r.dial(); err != nil {
				log.Errorf("Failed to reconnect to %s: %v", r.host, err)
				failures++
				continue
			}
			break
		}
	}
}

// dial establishes the TCP connection without spawning a new goroutine.
func (r *TCPReceiver) dial() error {
	ctx, cancel := context.WithTimeout(r.ctx, r.options.ConnectTimeout)
	defer cancel()
	var d net.Dialer
	log.Infof("Connecting to %s", r.host)
	conn, err := d.DialContext(ctx, "tcp", r.host)
	if err != nil {
		return fmt.Errorf("could not dial %s - %v", r.host, err)
	}
	r.socketMu.Lock()
	r.socket = conn
	r.socketMu.Unlock()
	// Disconnect may have run while we were dialling
	if r.ctx.Err() != nil {
		conn.Close()
		return r.ctx.Err()
	}
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
