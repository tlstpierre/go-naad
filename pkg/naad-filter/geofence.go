package naadfilter

import (
	"fmt"

	"github.com/tlstpierre/go-naad/pkg/naad-xml"
)

// GeoFence is a configured area of interest: a centre point and a radius.
// An alert matches if any of its polygons or circles overlaps the fence.
type GeoFence struct {
	Lat      float64 `yaml:"lat"`
	Lon      float64 `yaml:"lon"`
	RadiusKm float64 `yaml:"radiuskm"`
}

func (g GeoFence) Validate() error {
	if g.Lat < -90 || g.Lat > 90 {
		return fmt.Errorf("geofence lat %v out of range", g.Lat)
	}
	if g.Lon < -180 || g.Lon > 180 {
		return fmt.Errorf("geofence lon %v out of range", g.Lon)
	}
	if g.RadiusKm <= 0 {
		return fmt.Errorf("geofence radiuskm must be greater than zero")
	}
	return nil
}

func (g GeoFence) Place() Place {
	return CircleFromLLRadius(g.Lat, g.Lon, g.RadiusKm)
}

func (g GeoFence) String() string {
	return fmt.Sprintf("%.5f,%.5f r=%gkm", g.Lat, g.Lon, g.RadiusKm)
}

// AlertInGeoFence reports whether any area geometry in the alert overlaps
// the fence.
func AlertInGeoFence(alert *naadxml.Alert, fence GeoFence) bool {
	return IsPlaceInAlert(alert, fence.Place())
}
