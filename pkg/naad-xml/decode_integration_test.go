//go:build integration

// Integration tests: these talk to live NAAD/Pelmorex services, send real
// audio, or run for minutes. Run with: go test -tags integration ./...
package naadxml

import (
	"encoding/xml"
	log "github.com/sirupsen/logrus"
	"os"
	"testing"
)

func TestSample4Fetch(t *testing.T) {
	data, err := os.ReadFile(*testData + "Sample4_CAPCP_with_External_Large_Audio_File.XML")
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
	for _, alert := range testAlert.Info {
		for _, resource := range alert.Resources {
			if resource.Content == nil {
				err := resource.Fetch()
				if err != nil {
					t.Fail()
					t.Logf("Problem fetching resource %s - %v", resource.Description, err)
					continue
				}
				err = resource.SaveFile("./")
				if err != nil {
					t.Fail()
					t.Logf("Problem saving resource %s - %v", resource.Description, err)
				} else {
					t.Logf("Decoded %s", resource.Description)
				}
			}
		}
	}
	//spew.Dump(testAlert)
}
