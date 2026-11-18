package naadsocket

import (
	"flag"
	log "github.com/sirupsen/logrus"
	"testing"
)

var LogLevel = flag.String("loglevel", "info", "The logging verbosity")

func TestMain(m *testing.M) {
	flag.Parse()
	lvl, _ := log.ParseLevel(*LogLevel)
	log.SetLevel(lvl)

	exitVal := m.Run()
	log.Infof("Leaving TestMain() (Session closing) (Exit Status: %d)", exitVal)
}
