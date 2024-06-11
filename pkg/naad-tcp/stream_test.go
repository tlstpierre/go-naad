package naadtcp

import (
	"flag"
	"github.com/davecgh/go-spew/spew"
	log "github.com/sirupsen/logrus"
	"github.com/tlstpierre/go-naad/pkg/naad-xml"
	"testing"
	"time"
)

var (
	testHost     = flag.String("testhost", "streaming1.naad-adna.pelmorex.com:8080", "Path to Pelmorex sample XML")
	LogLevel     = flag.String("loglevel", "info", "The logging verbosity")
	testReceiver *Receiver
)

func TestMain(m *testing.M) {
	flag.Parse()
	lvl, _ := log.ParseLevel(*LogLevel)
	log.SetLevel(lvl)
	var err error
	testReceiver, err = NewReceiver(*testHost)
	if err != nil {
		log.Fatal(err)
	}

	exitVal := m.Run()
	log.Infof("Leaving TestMain() (Session closing) (Exit Status: %d)", exitVal)
}

func TestListen(t *testing.T) {
	dumpAlert := func(alert *naadxml.Alert) error {
		spew.Dump(alert)
		for _, reference := range alert.References.References {
			log.Infof("Reference can be fetched at %s", reference.URL("capcp1.naad-adna.pelmorex.com/"))
		}
		return nil
	}
	testReceiver.AddHandler(dumpAlert)
	testReceiver.Connect()
	time.Sleep(5 * time.Minute)
	testReceiver.Disconnect()
}
