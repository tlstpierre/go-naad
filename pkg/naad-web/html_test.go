package naadweb

import (
	"encoding/xml"
	"flag"
	"fmt"
	log "github.com/sirupsen/logrus"
	"github.com/tlstpierre/go-naad/pkg/naad-cache"
	"github.com/tlstpierre/go-naad/pkg/naad-xml"
	"os"
	"testing"
	"time"
)

var (
	testData = flag.String("testdata", "../../testdata/", "Path to Pelmorex sample XML")
	LogLevel = flag.String("loglevel", "info", "The logging verbosity")
)

func TestMain(m *testing.M) {
	flag.Parse()
	lvl, _ := log.ParseLevel(*LogLevel)
	log.SetLevel(lvl)
	exitVal := m.Run()
	log.Infof("Leaving TestMain() (Session closing) (Exit Status: %d)", exitVal)
}

func TestWebserver(t *testing.T) {
	cache := naadcache.NewCache()
	samples := getSamples()
	for _, sample := range samples {
		cache.Add(sample)
	}
	server, err := NewServer(":8080", cache)
	if err != nil {
		t.Fail()
		t.Logf("Problem starting web server - %v", err)
		return
	}
	time.Sleep(10 * time.Minute)
	server.Shutdown()
}

func getSamples() []*naadxml.Alert {
	directory, err := os.ReadDir(*testData)
	if err != nil {
		log.Fatal(err)
	}
	sampleList := make([]*naadxml.Alert, 0, len(directory))
	for _, entry := range directory {
		if entry.Type().IsDir() {
			continue
		}
		data, err := os.ReadFile(fmt.Sprintf("%s/%s", *testData, entry.Name()))
		if err != nil {
			log.Fatalf("Could not open test XML file - %v", err)
		}
		var testAlert naadxml.Alert
		err = xml.Unmarshal(data, &testAlert)
		if err != nil {
			log.Fatalf("Problem decoding sample %s", entry.Name())
		}
		testAlert.ProcessAlert()
		sampleList = append(sampleList, &testAlert)
	}
	return sampleList
}
