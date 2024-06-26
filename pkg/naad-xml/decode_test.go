package naadxml

import (
	"encoding/xml"
	"flag"
	"github.com/davecgh/go-spew/spew"
	log "github.com/sirupsen/logrus"
	"os"
	"testing"
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

func TestSample1(t *testing.T) {
	data, err := os.ReadFile(*testData + "Sample1_CAPCP_No_Attachment.xml")
	if err != nil {
		log.Fatalf("Could not open test XML file - %v", err)
	}
	var testAlert Alert
	err = xml.Unmarshal(data, &testAlert)
	if err != nil {
		t.Fail()
		t.Errorf("Problem decoding sample 1 - %v", err)
		return
	}
	spew.Dump(testAlert)
	for _, reference := range testAlert.References.References {
		log.Infof("Reference can be fetched at %s", reference.URL("testhost"))
	}
}

func TestSample2(t *testing.T) {
	data, err := os.ReadFile(*testData + "Sample2_CAPCP_with_Embedded_Large_Audio_File.xml")
	if err != nil {
		log.Fatalf("Could not open test XML file - %v", err)
	}
	var testAlert Alert
	err = xml.Unmarshal(data, &testAlert)
	if err != nil {
		t.Fail()
		t.Errorf("Problem decoding sample 2 - %v", err)
		return
	}
	spew.Dump(testAlert)
}

func TestSample7(t *testing.T) {
	data, err := os.ReadFile(*testData + "Sample7_CAPCP_with_free_drawn_circle.xml")
	if err != nil {
		log.Fatalf("Could not open test XML file - %v", err)
	}
	var testAlert Alert
	err = xml.Unmarshal(data, &testAlert)
	if err != nil {
		t.Fail()
		t.Errorf("Problem decoding sample 7 - %v", err)
		return
	}
	spew.Dump(testAlert)
}
