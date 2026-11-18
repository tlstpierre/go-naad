package main

import (
	"encoding/xml"
	"os"
	"strings"
	"testing"

	"github.com/tlstpierre/go-naad/pkg/naad-audio"
	"github.com/tlstpierre/go-naad/pkg/naad-filter"
	"github.com/tlstpierre/go-naad/pkg/naad-xml"
)

func TestMatchChannel(t *testing.T) {
	fence := naadfilter.GeoFence{Lat: 44.1077013162553, Lon: -78.95523247860514, RadiusKm: 20}
	local := naadaudio.ChannelConfig{CAPCodes: []string{"3518020"}, GeoFences: []naadfilter.GeoFence{fence}}

	capOnly := &naadxml.Alert{Info: []*naadxml.AlertInfo{{Area: []naadxml.AlertArea{{
		Geocode: []naadxml.Parameter{{Name: "profile:CAP-CP:Location:0.3", Value: "3518020"}}}}}}}
	nearCircle := &naadxml.Alert{Info: []*naadxml.AlertInfo{{Area: []naadxml.AlertArea{{
		Circles: []naadxml.GeoCircle{{Lat: 44.0, Lon: -79.1, Radius: 2}}}}}}}
	farCircle := &naadxml.Alert{Info: []*naadxml.AlertInfo{{Area: []naadxml.AlertArea{{
		Circles: []naadxml.GeoCircle{{Lat: 45.4, Lon: -75.7, Radius: 10}}}}}}}

	for name, c := range map[string]struct {
		cfg   naadaudio.ChannelConfig
		alert *naadxml.Alert
		want  bool
	}{
		"cap code":         {local, capOnly, true},
		"geofence":         {local, nearCircle, true},
		"neither":          {local, farCircle, false},
		"unfiltered":       {naadaudio.ChannelConfig{}, farCircle, true},
		"fence only, miss": {naadaudio.ChannelConfig{GeoFences: local.GeoFences}, capOnly, false},
	} {
		if got, reason, _ := matchChannel(c.cfg, c.alert, nil); got != c.want {
			t.Errorf("%s: got %v (%s) want %v", name, got, reason, c.want)
		}
	}
}

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

// A stand-down reaches the channels that had the alert through its
// references, whatever its own areas, and carries only its stand-down
// blocks. Live ECCC wind warning chain, 2026-11-18.
func TestMatchChannelStandDown(t *testing.T) {
	active, ended := loadECCC(t, "wind-active.cap"), loadECCC(t, "wind-ended.cap")
	cache := map[string]*naadxml.Alert{active.Identifier: active}
	lookup := func(ref naadxml.Reference) *naadxml.Alert { return cache[ref.Identifier] }

	fenced := naadaudio.ChannelConfig{GeoFences: []naadfilter.GeoFence{{Lat: 50.5, Lon: -56.7, RadiusKm: 20}}}
	coded := naadaudio.ChannelConfig{CAPCodes: []string{"1009"}}
	elsewhere := naadaudio.ChannelConfig{GeoFences: []naadfilter.GeoFence{{Lat: 44.1077, Lon: -78.9552, RadiusKm: 20}}}

	for name, cfg := range map[string]naadaudio.ChannelConfig{"fence": fenced, "cap code": coded} {
		ok, reason, infos := matchChannel(cfg, active, lookup)
		if !ok || len(infos) != 2 {
			t.Errorf("%s: active warning matched %v (%s) with %d blocks", name, ok, reason, len(infos))
		}
		ok, reason, infos = matchChannel(cfg, ended, lookup)
		if !ok || !strings.Contains(reason, active.Identifier) {
			t.Errorf("%s: stand-down matched %v (%s)", name, ok, reason)
		}
		for _, info := range infos {
			if info.InEffect() {
				t.Errorf("%s: stand-down carries an in-effect block", name)
			}
		}
		// Without the original (cache and archive both failing) the
		// stand-down's own ended areas don't match.
		if ok, _, _ := matchChannel(cfg, ended, func(naadxml.Reference) *naadxml.Alert { return nil }); ok {
			t.Errorf("%s: stand-down matched without its original", name)
		}
	}

	if ok, reason, _ := matchChannel(elsewhere, ended, lookup); ok {
		t.Errorf("a channel elsewhere took the stand-down: %s", reason)
	}
	if ok, _, infos := matchChannel(naadaudio.ChannelConfig{}, ended, lookup); !ok || len(infos) != len(ended.Info) {
		t.Errorf("unfiltered channel should take the whole message")
	}
}

// An update with blocks for both the areas still under the alert and the
// areas it has left: a channel inside the active areas announces only
// those.
func TestMatchChannelActiveBlocksOnly(t *testing.T) {
	msg := loadECCC(t, "snowfall-active-and-ended.cap")
	unfiltered := naadaudio.ChannelConfig{}
	if _, _, infos := matchChannel(unfiltered, msg, nil); len(infos) != 4 {
		t.Fatalf("want all 4 blocks unfiltered, got %d", len(infos))
	}
	var code string
	for _, info := range msg.Info {
		if info.InEffect() {
			for _, g := range info.Area[0].Geocode {
				if strings.HasPrefix(g.Name, "profile:CAP-CP:Location") {
					code = g.Value
				}
			}
			break
		}
	}
	ok, reason, infos := matchChannel(naadaudio.ChannelConfig{CAPCodes: []string{code}}, msg, nil)
	if !ok || len(infos) != 2 {
		t.Fatalf("matched %v (%s) with %d blocks, want the 2 active", ok, reason, len(infos))
	}
	for _, info := range infos {
		if !info.InEffect() {
			t.Error("ended block announced to a channel in the active area")
		}
	}
}
