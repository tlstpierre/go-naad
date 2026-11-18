package main

import (
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
		if got, reason := matchChannel(c.cfg, c.alert); got != c.want {
			t.Errorf("%s: got %v (%s) want %v", name, got, reason, c.want)
		}
	}
}
