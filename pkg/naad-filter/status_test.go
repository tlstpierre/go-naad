package naadfilter

import (
	"encoding/xml"
	"os"
	"testing"

	naadxml "github.com/tlstpierre/go-naad/pkg/naad-xml"
)

func loadECCC(t *testing.T, name string) *naadxml.Alert {
	t.Helper()
	data, err := os.ReadFile("../../testdata/eccc/" + name)
	if err != nil {
		t.Fatal(err)
	}
	var alert naadxml.Alert
	if err := xml.Unmarshal(data, &alert); err != nil {
		t.Fatal(err)
	}
	return &alert
}

// The same wind warning area, active and then ended: only the active
// message matches a fence or CAP code there.
func TestStandDownAreasDontMatch(t *testing.T) {
	fence := GeoFence{Lat: 50.5, Lon: -56.7, RadiusKm: 20} // Northern Peninsula East, NL
	active, ended := loadECCC(t, "wind-active.cap"), loadECCC(t, "wind-ended.cap")
	if !AlertInGeoFence(active, fence) || !AlertIsCAPArea(active, "1009001") {
		t.Error("active wind warning should match")
	}
	if AlertInGeoFence(ended, fence) || AlertIsCAPArea(ended, "1009001") {
		t.Error("ended wind warning should not match")
	}
}

// Drawn threat areas count only while issued or continued.
func TestThreatAreaMatching(t *testing.T) {
	fence := GeoFence{Lat: 44.1077, Lon: -78.9552, RadiusKm: 20}
	poly, err := naadxml.ParsePolygon("44.05,-79.0 44.15,-79.0 44.15,-78.9 44.05,-78.9 44.05,-79.0")
	if err != nil {
		t.Fatal(err)
	}
	for status, want := range map[string]bool{"issued": true, "continued": true, "ended": false, "cancelled": false} {
		alert := &naadxml.Alert{Info: []*naadxml.AlertInfo{{Area: []naadxml.AlertArea{{
			Description: status + " threat area",
			Polygons:    []naadxml.GeoPolygon{poly},
			Geocode:     []naadxml.Parameter{{Name: "layer:EC-MSC-SMC:1.1:DLC", Value: status}},
		}}}}}
		if got := AlertInGeoFence(alert, fence); got != want {
			t.Errorf("%s threat area: match %v, want %v", status, got, want)
		}
	}
}
