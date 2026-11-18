package naadfilter

import (
	"fmt"

	"github.com/golang/geo/s2"
	log "github.com/sirupsen/logrus"

	"github.com/tlstpierre/go-naad/pkg/naad-xml"
)

// LoopFromPolygon converts a CAP polygon into a normalized s2 loop.
//
// CAP polygons repeat the first point as the last point and have no defined
// winding order. s2 loops must not repeat vertices and treat the interior as
// the region to the left of the edges, so a clockwise polygon would otherwise
// describe the whole Earth *except* the alert area. Normalize() flips any loop
// covering more than a hemisphere, which no real alert polygon does.
func LoopFromPolygon(polygon naadxml.GeoPolygon) (*s2.Loop, error) {
	points := make([]s2.Point, 0, len(polygon.Perimeter))
	for _, gp := range polygon.Perimeter {
		p := pointFromLatLon(gp.Lat, gp.Lon)
		// Drop consecutive duplicates (s2 rejects zero-length edges)
		if n := len(points); n > 0 && points[n-1].ApproxEqual(p) {
			continue
		}
		points = append(points, p)
	}
	// Drop the closing vertex
	if n := len(points); n > 1 && points[0].ApproxEqual(points[n-1]) {
		points = points[:n-1]
	}
	if len(points) < 3 {
		return nil, fmt.Errorf("polygon has %d distinct vertices, need at least 3", len(points))
	}
	loop := s2.LoopFromPoints(points)
	loop.Normalize()
	if err := loop.Validate(); err != nil {
		// Typically a self-intersecting hand-drawn polygon. The vertex and
		// edge distance tests still work, so use it on a best-effort basis.
		log.Warnf("Alert polygon is not a valid simple loop, matching on best effort - %v", err)
	}
	return loop, nil
}

func IsPlaceInPolygon(polygon naadxml.GeoPolygon, place Place) bool {
	loop, err := LoopFromPolygon(polygon)
	if err != nil {
		log.Warnf("Skipping alert polygon - %v", err)
		return false
	}
	return place.InPolygon(loop)
}

func IsPlaceInCircle(circle naadxml.GeoCircle, place Place) bool {
	if circle.Radius < 0 {
		return false // malformed, see naadxml.GeoCircle.UnmarshalXML
	}
	centre := pointFromLatLon(circle.Lat, circle.Lon)
	angle := KmToAngle(circle.Radius)
	s2cap := s2.CapFromCenterAngle(centre, angle)
	return place.InCircle(s2cap)
}

// IsPlaceInArea reports whether place overlaps any polygon or circle in the
// alert area. Areas with only geocodes never match here.
func IsPlaceInArea(area naadxml.AlertArea, place Place) (matched bool) {
	// Geometry comes from the network; never let a degenerate shape crash
	// the receiver.
	defer func() {
		if r := recover(); r != nil {
			log.Errorf("Recovered from panic matching area %q - %v", area.Description, r)
			matched = false
		}
	}()
	for _, polygon := range area.Polygons {
		if IsPlaceInPolygon(polygon, place) {
			return true
		}
	}
	for _, circle := range area.Circles {
		if IsPlaceInCircle(circle, place) {
			return true
		}
	}
	return false
}

// IsPlaceInAlert reports whether place overlaps an area the alert is in
// effect for. Stand-down blocks and ended or cancelled threat areas are
// skipped (see naadxml.AlertInfo.InEffect and AlertArea.InEffect): they
// say where the alert no longer applies, so they never start an alert. A
// stand-down reaches those who heard the original through its references.
func IsPlaceInAlert(alert *naadxml.Alert, place Place) bool {
	for _, info := range alert.Info {
		if !info.InEffect() {
			continue
		}
		for _, area := range info.Area {
			if !area.InEffect() {
				continue
			}
			if IsPlaceInArea(area, place) {
				return true
			}
		}
	}
	return false
}

func pointFromLatLon(lat, lng float64) s2.Point {
	ll := s2.LatLngFromDegrees(lat, lng)
	return s2.PointFromLatLng(ll)
}
