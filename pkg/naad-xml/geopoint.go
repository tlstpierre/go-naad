package naadxml

import (
	"encoding/xml"
	"fmt"
	"io"
	"strconv"
	"strings"
)

type GeoPolygon struct {
	Perimeter []GeoPoint
}

type GeoPoint struct {
	Lat float64
	Lon float64
}

func (g *GeoPolygon) UnmarshalXML(d *xml.Decoder, start xml.StartElement) error {
	for {
		t, err := d.Token()
		if err == io.EOF {
			break
		}
		//		log.Infof("XML start element is %+v and token is %+v", start, t)
		switch t.(type) {
		case xml.CharData:
			points := strings.Split(string(t.(xml.CharData)), " ")
			g.Perimeter = make([]GeoPoint, len(points))
			for i, point := range points {
				latStr, lonStr, found := strings.Cut(point, ",")
				if !found {
					return fmt.Errorf("Parse error separating points - expecting , as lat/lon separator")
				}
				var (
					lat float64
					lon float64
					err error
				)
				if lat, err = strconv.ParseFloat(latStr, 64); err != nil {
					return fmt.Errorf("Problem parsing lat value - %v", err)
				}
				if lon, err = strconv.ParseFloat(lonStr, 64); err != nil {
					return fmt.Errorf("Problem parsing lon value - %v", err)
				}

				g.Perimeter[i] = GeoPoint{
					Lat: lat,
					Lon: lon,
				}
			}
		}
	}
	return nil
}

type GeoCircle struct {
	Lat    float64
	Lon    float64
	Radius float64
}

func (c *GeoCircle) UnmarshalXML(d *xml.Decoder, start xml.StartElement) error {
	for {
		t, err := d.Token()
		if err == io.EOF {
			break
		}
		//		log.Infof("XML start element is %+v and token is %+v", start, t)
		switch t.(type) {
		case xml.CharData:
			point, radiusStr, foundRadius := strings.Cut(string(t.(xml.CharData)), " ")
			if !foundRadius {
				return fmt.Errorf("Could not find radius in %s", string(t.(xml.CharData)))
			}
			latStr, lonStr, foundLatLon := strings.Cut(point, ",")
			if !foundLatLon {
				return fmt.Errorf("Could not split lat/lon in %s", string(t.(xml.CharData)))
			}
			var err error

			if c.Radius, err = strconv.ParseFloat(radiusStr, 64); err != nil {
				return fmt.Errorf("Problem parsing radius value - %v", err)
			}

			if c.Lat, err = strconv.ParseFloat(latStr, 64); err != nil {
				return fmt.Errorf("Problem parsing lat value - %v", err)
			}

			if c.Lon, err = strconv.ParseFloat(lonStr, 64); err != nil {
				return fmt.Errorf("Problem parsing lon value - %v", err)
			}
		}
	}
	return nil
}
