package naadfilter

import (
	"github.com/golang/geo/s2"
	//	log "github.com/sirupsen/logrus"

	"github.com/tlstpierre/go-naad/pkg/naad-xml"
)

func IsPlaceInPolygon(polygon naadxml.GeoPolygon, place Place) bool {
	points := make([]s2.Point, len(polygon.Perimeter))
	for index, geopoint := range polygon.Perimeter {
		points[index] = pointFromLatLon(geopoint.Lat, geopoint.Lon)
	}
	loop := s2.LoopFromPoints(points)
	return place.InPolygon(loop)

}

func IsPlaceInCircle(circle naadxml.GeoCircle, place Place) bool {
	centre := pointFromLatLon(circle.Lat, circle.Lon)
	angle := KmToAngle(circle.Radius)
	s2cap := s2.CapFromCenterAngle(centre, angle)
	return place.InCircle(s2cap)
}

func IsPlaceInArea(area naadxml.AlertArea, place Place) bool {
	if area.Polygon != nil {
		if IsPlaceInPolygon(*area.Polygon, place) {
			return true
		}
	}
	if area.Circle != nil {
		if IsPlaceInCircle(*area.Circle, place) {
			return true
		}
	}
	return false
}

func IsPlaceInAlert(alert *naadxml.Alert, place Place) bool {
	for _, info := range alert.Info {
		for _, area := range info.Area {
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
