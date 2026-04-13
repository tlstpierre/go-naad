package naadfilter

import (
	"github.com/golang/geo/s1"
	"github.com/golang/geo/s2"
	// log "github.com/sirupsen/logrus"
)

const (
	Earth float64 = 6371
)

type Place interface {
	InPolygon(*s2.Loop) bool
	InCircle(s2.Cap) bool
}

// A single point
type Point s2.Point

func PointFromLatLon(lat, lng float64) Point {
	ll := s2.LatLngFromDegrees(lat, lng)
	return Point(s2.PointFromLatLng(ll))
}

func (p Point) InPolygon(polygon *s2.Loop) bool {
	return polygon.ContainsPoint(s2.Point(p))
}

func (p Point) InCircle(circle s2.Cap) bool {
	return circle.ContainsPoint(s2.Point(p))
}

// A boundary polygon
type Boundary s2.Loop

func BoundaryFromPoints(points []Point) Boundary {
	s2points := make([]s2.Point, len(points))
	for index, point := range points {
		s2points[index] = s2.Point(point)
	}
	loop := s2.LoopFromPoints(s2points)
	return Boundary(*loop)
}

func (b Boundary) InPolygon(polygon *s2.Loop) bool {
	boundaryLoop := s2.Loop(b)
	return polygon.Intersects(&boundaryLoop)
}

func (b Boundary) InCircle(circle s2.Cap) bool {
	loop := s2.Loop(b)
	return capIntersectsLoop(circle, &loop)
}

// A circular area defined by a point and a radius
type Circle s2.Cap

func KmToAngle(km float64) s1.Angle {
	return s1.Angle(km / Earth)
}

func AngleToKm(angle s1.ChordAngle) float64 {
	return float64(angle.Angle()) * Earth
}

func CircleFromLLRadius(lat, lng, radius float64) Circle {
	centre := pointFromLatLon(lat, lng)
	angle := KmToAngle(radius)
	s2cap := s2.CapFromCenterAngle(centre, angle)
	return Circle(s2cap)
}

func (c Circle) InPolygon(polygon *s2.Loop) bool {
	return capIntersectsLoop(s2.Cap(c), polygon)
}

func (c Circle) InCircle(circle s2.Cap) bool {
	return circle.Intersects(s2.Cap(c))
}

// capIntersectsLoop checks whether an s2.Cap (circle) intersects an s2.Loop
// (polygon). They intersect if:
//   - the cap center is inside the loop, or
//   - any vertex of the loop is inside the cap, or
//   - any edge of the loop passes through the cap
func capIntersectsLoop(cap s2.Cap, loop *s2.Loop) bool {
	if loop.ContainsPoint(cap.Center()) {
		return true
	}
	numVertices := loop.NumVertices()
	for i := 0; i < numVertices; i++ {
		if cap.ContainsPoint(loop.Vertex(i)) {
			return true
		}
	}
	capRadius := cap.Radius()
	for i := 0; i < numVertices; i++ {
		edge := loop.Edge(i)
		if s2.DistanceFromSegment(cap.Center(), edge.V0, edge.V1) <= capRadius {
			return true
		}
	}
	return false
}
