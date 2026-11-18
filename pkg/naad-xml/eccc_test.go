package naadxml

import (
	"encoding/xml"
	"os"
	"strings"
	"testing"
)

func loadECCC(t *testing.T, name string) *Alert {
	t.Helper()
	data, err := os.ReadFile("../../testdata/eccc/" + name)
	if err != nil {
		t.Fatal(err)
	}
	var alert Alert
	if err := xml.Unmarshal(data, &alert); err != nil {
		t.Fatalf("%s: %v", name, err)
	}
	return &alert
}

// The risk-tiered (colour) parameters in live ECCC alerts, 2026-11-18.
func TestECRiskParameters(t *testing.T) {
	found := 0
	for _, name := range []string{"wind-active.cap", "wind-ended.cap", "snowfall-active-and-ended.cap"} {
		alert := loadECCC(t, name)
		if err := alert.ProcessAlert(); err != nil {
			t.Fatalf("%s: %v", name, err)
		}
		for _, info := range alert.Info {
			ec := info.ECLayer
			if ec == nil || ec.Colour == "" || info.Language != "en-CA" {
				continue
			}
			found++
			if ec.Colour != "yellow" || ec.Impact != "moderate" || ec.Confidence != "high" || ec.RiskRanking != 8 {
				t.Errorf("%s: colour %q impact %q confidence %q ranking %d", name, ec.Colour, ec.Impact, ec.Confidence, ec.RiskRanking)
			}
		}
	}
	if found == 0 {
		t.Fatal("no info block with a colour")
	}

	alert := loadECCC(t, "wind-active.cap")
	alert.ProcessAlert()
	ec := alert.Info[0].ECLayer
	if ec.EventEndTime.IsZero() && ec.EventEndTimeDescription == "" {
		t.Errorf("no event end time: %+v", ec)
	}
}

func TestECStormParameters(t *testing.T) {
	ec := &EC{}
	for name, value := range map[string]string{
		"Storm_Speed":                "40 km/h",
		"Storm_Direction":            "90.12841",
		"Storm_Geometry_Type":        "isolated_cell",
		"Storm_Point":                "51.863808,-115.032158",
		"Storm_Time":                 "20250808153000000",
		"Motion_Description":         "east at 40 km/h",
		"Storm_Position_Description": "about 25 km southwest of Burnstick Lake",
		"Reference_Location_Points":  "Water Valley, Cremona Bergen",
		"Previous_Colour":            "yellow",
	} {
		ECParam(ec, "1.1", name, value)
	}
	s := ec.Storm
	if s.Speed != "40 km/h" || s.Direction != 90.12841 || s.GeometryType != "isolated_cell" || len(s.Points) != 1 ||
		s.Time.Format("2006-01-02 15:04") != "2025-08-08 15:30" || s.MotionDescription != "east at 40 km/h" ||
		s.PositionDescription == "" || s.ReferenceLocationPoints == "" || ec.PreviousColour != "yellow" {
		t.Errorf("storm %+v previous colour %q", s, ec.PreviousColour)
	}
}

func TestLocationStatus(t *testing.T) {
	alert := loadECCC(t, "snowfall-active-and-ended.cap")
	var active, ended int
	for _, info := range alert.Info {
		switch info.LocationStatus() {
		case "active":
			active++
			if !info.InEffect() {
				t.Error("active block not in effect")
			}
		case "ended":
			ended++
			if info.InEffect() {
				t.Error("ended block in effect")
			}
		default:
			t.Errorf("status %q", info.LocationStatus())
		}
	}
	if active != 2 || ended != 2 {
		t.Errorf("%d active and %d ended blocks, want 2 and 2 (en and fr)", active, ended)
	}

	ended1 := loadECCC(t, "wind-ended.cap")
	if !ended1.RefersBack() {
		t.Error("ended update should refer back")
	}
	refersToActive := false
	for _, r := range ended1.References.References {
		refersToActive = refersToActive || r.Identifier == "urn:oid:2.49.0.1.124.3144277405.2026"
	}
	if !refersToActive {
		t.Error("wind-ended.cap should reference wind-active.cap")
	}
	for _, info := range ended1.Info {
		if info.InEffect() {
			t.Errorf("%s block of an all-clear is in effect", info.Language)
		}
	}

	// Any issuer's AllClear is a stand-down.
	if (&AlertInfo{Response: []ResponseType{"AllClear"}}).InEffect() {
		t.Error("AllClear in effect")
	}
	if !(&AlertInfo{}).InEffect() {
		t.Error("a plain block should be in effect")
	}
}

func TestThreatAreaStatus(t *testing.T) {
	for status, want := range map[string]bool{"": true, "issued": true, "continued": true, "ended": false, "cancelled": false} {
		area := AlertArea{}
		if status != "" {
			area.Geocode = []Parameter{{Name: "layer:EC-MSC-SMC:1.1:DLC", Value: status}}
		}
		if area.InEffect() != want {
			t.Errorf("DLC %q: in effect %v, want %v", status, area.InEffect(), want)
		}
	}
}

func TestReferencesAcrossLines(t *testing.T) {
	doc := `<alert><msgType>Cancel</msgType><references>a@b,id-1,2026-11-17T08:40:07-00:00
	  a@b,id-2,2026-11-17T14:57:58-00:00 </references></alert>`
	var alert Alert
	if err := xml.NewDecoder(strings.NewReader(doc)).Decode(&alert); err != nil {
		t.Fatal(err)
	}
	refs := alert.References.References
	if len(refs) != 2 || refs[1].Identifier != "id-2" || !alert.RefersBack() {
		t.Errorf("refs %+v", refs)
	}
}
