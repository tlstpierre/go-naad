package naadsocket

import (
	"net"
	"sync/atomic"
	"testing"
	"time"
)

// fakeServer accepts connections and hands each to fn. It returns the
// address and a counter of accepted connections.
func fakeServer(t *testing.T, fn func(net.Conn)) (string, *atomic.Int32) {
	t.Helper()
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { ln.Close() })
	var count atomic.Int32
	go func() {
		for {
			c, err := ln.Accept()
			if err != nil {
				return
			}
			count.Add(1)
			go fn(c)
		}
	}()
	return ln.Addr().String(), &count
}

// Server accepts then immediately closes (what streaming1 is doing). With
// 20ms min backoff, retries should be 20, 40, 80, 160, 320... so ~6
// connections in 1s rather than ~50 at a fixed 20ms.
func TestBackoffGrowsOnImmediateClose(t *testing.T) {
	addr, count := fakeServer(t, func(c net.Conn) { c.Close() })
	rx, err := NewTCPReceiver(addr, TCPReceiverOptions{
		MinBackoff: 20 * time.Millisecond, MaxBackoff: 10 * time.Second, StableAfter: time.Minute,
	})
	if err != nil {
		t.Fatal(err)
	}
	if err := rx.Connect(); err != nil {
		t.Fatal(err)
	}
	time.Sleep(time.Second)
	rx.Disconnect()
	if n := count.Load(); n < 3 || n > 9 {
		t.Errorf("got %d connections in 1s, expected exponential backoff (~6)", n)
	}
}

// Server accepts but never sends anything: the read deadline should fire and
// we should reconnect.
func TestReadTimeoutReconnects(t *testing.T) {
	addr, count := fakeServer(t, func(c net.Conn) {
		time.Sleep(5 * time.Second)
		c.Close()
	})
	rx, err := NewTCPReceiver(addr, TCPReceiverOptions{
		ReadTimeout: 100 * time.Millisecond, MinBackoff: 10 * time.Millisecond,
	})
	if err != nil {
		t.Fatal(err)
	}
	if err := rx.Connect(); err != nil {
		t.Fatal(err)
	}
	time.Sleep(600 * time.Millisecond)
	rx.Disconnect()
	if n := count.Load(); n < 2 {
		t.Errorf("got %d connections, expected a reconnect after the read timeout", n)
	}
}

func TestNextBackoff(t *testing.T) {
	min, max := 5*time.Second, 5*time.Minute
	b := min
	for i := 0; i < 10; i++ {
		b = nextBackoff(b, min, max)
	}
	if b != max {
		t.Errorf("backoff should cap at %s, got %s", max, b)
	}
	for i := 0; i < 1000; i++ {
		if j := jitter(10 * time.Second); j < 8*time.Second || j > 12*time.Second {
			t.Fatalf("jitter out of range: %s", j)
		}
	}
}
