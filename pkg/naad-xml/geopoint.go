package naadxml

import (
	"encoding/xml"
	"fmt"
	"strconv"
	"strings"

	log "github.com/sirupsen/logrus"
)

// GeoPolygon is a CAP <polygon>: a whitespace-separated list of "lat,lon"
// pairs. CAP requires the first and last points to be equal, but does not
// specify a winding order.
type GeoPolygon struct {
	Perimeter []GeoPoint
}

type GeoPoint struct {
	Lat float64
	Lon float64
}

func parseLatLon(s string) (GeoPoint, error) {
	latStr, lonStr, found := strings.Cut(s, ",")
	if !found {
		return GeoPoint{}, fmt.Errorf("expected lat,lon but got %q", s)
	}
	lat, err := strconv.ParseFloat(strings.TrimSpace(latStr), 64)
	if err != nil {
		return GeoPoint{}, fmt.Errorf("problem parsing lat value %q - %v", latStr, err)
	}
	lon, err := strconv.ParseFloat(strings.TrimSpace(lonStr), 64)
	if err != nil {
		return GeoPoint{}, fmt.Errorf("problem parsing lon value %q - %v", lonStr, err)
	}
	if lat < -90 || lat > 90 || lon < -180 || lon > 180 {
		return GeoPoint{}, fmt.Errorf("lat,lon %q out of range", s)
	}
	return GeoPoint{Lat: lat, Lon: lon}, nil
}

// ParsePolygon parses CAP polygon text ("lat,lon lat,lon ...").
func ParsePolygon(text string) (GeoPolygon, error) {
	fields := strings.Fields(text)
	g := GeoPolygon{Perimeter: make([]GeoPoint, 0, len(fields))}
	for _, f := range fields {
		pt, err := parseLatLon(f)
		if err != nil {
			return GeoPolygon{}, err
		}
		g.Perimeter = append(g.Perimeter, pt)
	}
	return g, nil
}

func (g *GeoPolygon) UnmarshalXML(d *xml.Decoder, start xml.StartElement) error {
	var text string
	if err := d.DecodeElement(&text, &start); err != nil {
		return err
	}
	parsed, err := ParsePolygon(text)
	if err != nil {
		// Don't reject the whole alert over bad geometry -- CAP code
		// matching can still route it. Leave the polygon empty.
		log.Warnf("Ignoring malformed polygon - %v", err)
		*g = GeoPolygon{}
		return nil
	}
	*g = parsed
	return nil
}

// GeoCircle is a CAP <circle>: "lat,lon radius" with radius in kilometres.
type GeoCircle struct {
	Lat    float64
	Lon    float64
	Radius float64
}

// ParseCircle parses CAP circle text ("lat,lon radius").
func ParseCircle(text string) (GeoCircle, error) {
	fields := strings.Fields(text)
	if len(fields) != 2 {
		return GeoCircle{}, fmt.Errorf("expected \"lat,lon radius\" but got %q", text)
	}
	pt, err := parseLatLon(fields[0])
	if err != nil {
		return GeoCircle{}, err
	}
	radius, err := strconv.ParseFloat(fields[1], 64)
	if err != nil {
		return GeoCircle{}, fmt.Errorf("problem parsing radius value %q - %v", fields[1], err)
	}
	if radius < 0 {
		return GeoCircle{}, fmt.Errorf("negative radius %v", radius)
	}
	return GeoCircle{Lat: pt.Lat, Lon: pt.Lon, Radius: radius}, nil
}

func (c *GeoCircle) UnmarshalXML(d *xml.Decoder, start xml.StartElement) error {
	var text string
	if err := d.DecodeElement(&text, &start); err != nil {
		return err
	}
	parsed, err := ParseCircle(text)
	if err != nil {
		// See GeoPolygon.UnmarshalXML. A zero radius circle at 0,0 would
		// be a valid shape, so mark it invalid with a negative radius.
		log.Warnf("Ignoring malformed circle - %v", err)
		*c = GeoCircle{Radius: -1}
		return nil
	}
	*c = parsed
	return nil
}
