package filter

import (
	"encoding/xml"
	"flag"
	//	"github.com/davecgh/go-spew/spew"
	log "github.com/sirupsen/logrus"
	"github.com/tlstpierre/go-naad/pkg/naad-xml"
	"os"
	"testing"
)

var (
	testData    = flag.String("testdata", "../../testdata/", "Path to Pelmorex sample XML")
	LogLevel    = flag.String("loglevel", "info", "The logging verbosity")
	CAPLocation = flag.Int("location", 3520005, "The CAP location code to test")
)

func TestMain(m *testing.M) {
	flag.Parse()
	lvl, _ := log.ParseLevel(*LogLevel)
	log.SetLevel(lvl)
	exitVal := m.Run()
	log.Infof("Leaving TestMain() (Session closing) (Exit Status: %d)", exitVal)
}

func TestCAPCode(t *testing.T) {
	data, err := os.ReadFile(*testData + "Sample1_CAPCP_No_Attachment.xml")
	if err != nil {
		log.Fatalf("Could not open test XML file - %v", err)
	}
	var testAlert naadxml.Alert
	err = xml.Unmarshal(data, &testAlert)
	if err != nil {
		t.Fail()
		t.Errorf("Problem decoding sample 1 - %v", err)
		return
	}
	if AlertIsCAPArea(&testAlert, *CAPLocation) {
		log.Infof("Alert matches location %d", *CAPLocation)
	} else {
		t.Fail()
		t.Log("No location matched")
	}
}

func TestPoint(t *testing.T) {
	data, err := os.ReadFile(*testData + "Sample1_CAPCP_No_Attachment.xml")
	if err != nil {
		log.Fatalf("Could not open test XML file - %v", err)
	}
	var testAlert naadxml.Alert
	err = xml.Unmarshal(data, &testAlert)
	if err != nil {
		t.Fail()
		t.Errorf("Problem decoding sample 1 - %v", err)
		return
	}

	place := PointFromLatLon(43.700475, -79.443933)
	if IsPlaceInAlert(&testAlert, place) {
		log.Info("Alert matches area")
	} else {
		t.Fail()
		t.Log("No location matched")
	}
}
