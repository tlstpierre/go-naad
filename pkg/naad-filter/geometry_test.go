package naadfilter

import (
	"encoding/xml"
	"os"
	"testing"

	"github.com/tlstpierre/go-naad/pkg/naad-xml"
)

// Our deployment fence: 20 km around the receiver.
var homeFence = GeoFence{Lat: 44.1077013162553, Lon: -78.95523247860514, RadiusKm: 20}

func poly(t *testing.T, text string) naadxml.GeoPolygon {
	t.Helper()
	p, err := naadxml.ParsePolygon(text)
	if err != nil {
		t.Fatal(err)
	}
	return p
}

func TestPolygonWindingOrder(t *testing.T) {
	home := PointFromLatLon(44.10747, -78.95514)
	vancouver := PointFromLatLon(49.28, -123.12)
	ccw := poly(t, "44.0,-79.1 44.0,-78.8 44.2,-78.8 44.2,-79.1 44.0,-79.1")
	cw := poly(t, "44.0,-79.1 44.2,-79.1 44.2,-78.8 44.0,-78.8 44.0,-79.1")
	for name, p := range map[string]naadxml.GeoPolygon{"ccw": ccw, "cw": cw} {
		if !IsPlaceInPolygon(p, home) {
			t.Errorf("%s: home should be inside", name)
		}
		if IsPlaceInPolygon(p, vancouver) {
			t.Errorf("%s: vancouver should be outside", name)
		}
	}
}

func TestGeoFenceOverlap(t *testing.T) {
	cases := []struct {
		name string
		area naadxml.AlertArea
		want bool
	}{
		// Polygon entirely contains the fence centre
		{"polygon containing centre", naadxml.AlertArea{Polygons: []naadxml.GeoPolygon{
			poly(t, "43.9,-79.3 44.3,-79.3 44.3,-78.6 43.9,-78.6 43.9,-79.3")}}, true},
		// Small polygon ~15 km east of centre, entirely inside the fence
		{"polygon inside fence", naadxml.AlertArea{Polygons: []naadxml.GeoPolygon{
			poly(t, "44.10,-78.78 44.12,-78.78 44.12,-78.76 44.10,-78.76 44.10,-78.78")}}, true},
		// Large polygon whose west edge passes ~17 km east of centre;
		// no vertex inside the fence, only the edge crosses it
		{"edge crosses fence", naadxml.AlertArea{Polygons: []naadxml.GeoPolygon{
			poly(t, "43.0,-78.74 45.0,-78.74 45.0,-77.0 43.0,-77.0 43.0,-78.74")}}, true},
		// Toronto free-drawn polygon (~50 km away)
		{"toronto polygon", naadxml.AlertArea{Polygons: []naadxml.GeoPolygon{
			poly(t, "43.745743,-79.471744 43.700089,-79.496463 43.679236,-79.436038 43.703068,-79.350894 43.783430,-79.335788 43.790370,-79.422305 43.745743,-79.471744")}}, false},
		// Circle 30 km away with 15 km radius -> overlaps 20 km fence
		{"circle overlapping", naadxml.AlertArea{Circles: []naadxml.GeoCircle{{Lat: 44.1077, Lon: -78.58, Radius: 15}}}, true},
		// Circle 30 km away with 5 km radius -> 5 km gap
		{"circle near miss", naadxml.AlertArea{Circles: []naadxml.GeoCircle{{Lat: 44.1077, Lon: -78.58, Radius: 5}}}, false},
		// Second polygon in the area is the one that matches
		{"multiple polygons", naadxml.AlertArea{Polygons: []naadxml.GeoPolygon{
			poly(t, "49.2,-123.2 49.4,-123.2 49.4,-123.0 49.2,-123.0 49.2,-123.2"),
			poly(t, "44.10,-78.96 44.11,-78.96 44.11,-78.95 44.10,-78.95 44.10,-78.96")}}, true},
		// Geocode-only area never matches geometrically
		{"geocode only", naadxml.AlertArea{}, false},
		// Degenerate polygon and malformed circle are ignored, not fatal
		{"degenerate", naadxml.AlertArea{
			Polygons: []naadxml.GeoPolygon{poly(t, "44.1,-78.9 44.1,-78.9")},
			Circles:  []naadxml.GeoCircle{{Radius: -1}}}, false},
	}
	place := homeFence.Place()
	for _, c := range cases {
		if got := IsPlaceInArea(c.area, place); got != c.want {
			t.Errorf("%s: got %v want %v", c.name, got, c.want)
		}
	}
}

func TestSampleAlertsAgainstFence(t *testing.T) {
	// The Pelmorex samples are all in Toronto, well outside our fence,
	// but a 60 km fence reaches them.
	wide := GeoFence{Lat: homeFence.Lat, Lon: homeFence.Lon, RadiusKm: 60}
	for _, f := range []string{"Sample6_CAPCP_with_free_drawn_polygon.xml", "Sample7_CAPCP_with_free_drawn_circle.xml", "Sample8_CAPCP_with_Event_location.xml"} {
		data, err := os.ReadFile(*testData + f)
		if err != nil {
			t.Fatal(err)
		}
		var alert naadxml.Alert
		if err := xml.Unmarshal(data, &alert); err != nil {
			t.Fatalf("%s: %v", f, err)
		}
		if AlertInGeoFence(&alert, homeFence) {
			t.Errorf("%s: should not match 20 km fence", f)
		}
		if !AlertInGeoFence(&alert, wide) {
			t.Errorf("%s: should match 60 km fence", f)
		}
	}
}

func TestMalformedGeometryDoesNotRejectAlert(t *testing.T) {
	doc := `<alert xmlns="urn:oasis:names:tc:emergency:cap:1.2"><identifier>x</identifier><info><area><areaDesc>A</areaDesc><polygon>garbage</polygon><circle>44.1,-78.9</circle><polygon>
	  44.0,-79.1  44.0,-78.8
	  44.2,-78.8 44.2,-79.1 44.0,-79.1 </polygon></area></info></alert>`
	var alert naadxml.Alert
	if err := xml.Unmarshal([]byte(doc), &alert); err != nil {
		t.Fatalf("alert rejected: %v", err)
	}
	if !AlertInGeoFence(&alert, homeFence) {
		t.Error("valid polygon after malformed ones should still match")
	}
}
