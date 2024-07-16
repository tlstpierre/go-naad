package naadsocket

import (
	"flag"
	"github.com/davecgh/go-spew/spew"
	log "github.com/sirupsen/logrus"
	"github.com/tlstpierre/go-naad/pkg/naad-xml"
	"testing"
	"time"
)

var (
	testHost     = flag.String("testhost1", "streaming1.naad-adna.pelmorex.com:8080", "TCP streaming server 1")
	testHost2    = flag.String("testhost2", "streaming2.naad-adna.pelmorex.com:8080", "TCP streaming server 2")
	UDPListen    = flag.String("udplisten", "224.0.10.10:25555", "UDP listen address and port")
	archive1     = flag.String("archivehost1", "capcp1.naad-adna.pelmorex.com", "Archive host 1")
	LogLevel     = flag.String("loglevel", "info", "The logging verbosity")
	testReceiver *Receiver

	dumpAlert = func(alert *naadxml.Alert) error {
		spew.Dump(alert)
		for _, reference := range alert.References.References {
			log.Infof("Reference can be fetched at %s", reference.URL("capcp1.naad-adna.pelmorex.com/"))
		}
		return nil
	}
)

func TestMain(m *testing.M) {
	flag.Parse()
	lvl, _ := log.ParseLevel(*LogLevel)
	log.SetLevel(lvl)

	exitVal := m.Run()
	log.Infof("Leaving TestMain() (Session closing) (Exit Status: %d)", exitVal)
}

func TestTCP(t *testing.T) {
	var err error
	testReceiver, err = NewReceiver(*testHost)
	if err != nil {
		log.Fatal(err)
	}

	testReceiver.AddHandler(dumpAlert)
	testReceiver.Connect()
	time.Sleep(5 * time.Minute)
	testReceiver.Disconnect()
}

func TestUDP(t *testing.T) {
	testListener, err := NewListener(*UDPListen)
	if err != nil {
		t.Fail()
		t.Logf("Problem opening listener - %v", err)
		return
	}
	testListener.AddHandler(dumpAlert)
	testListener.Connect()
	time.Sleep(5 * time.Minute)
	testReceiver.Disconnect()
}
