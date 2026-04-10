package geometry

import (
	"afk-hero/internal/domain"
	"math"
)

// ShapeVertices generates the vertices of a shape centered at origin with given radius.
// The returned vertices form a closed path (first vertex == last vertex implied by caller).
func ShapeVertices(shape domain.Shape, radius float64) []domain.Point {
	switch shape {
	case domain.ShapeTriangle:
		return regularPolygon(3, radius)
	case domain.ShapeRectangle:
		return rectangleVertices(radius)
	case domain.ShapePentagon:
		return regularPolygon(5, radius)
	case domain.ShapeSegment:
		return []domain.Point{{X: 0, Y: 0}, {X: int(radius), Y: 0}}
	case domain.ShapeOnePixel:
		return []domain.Point{{X: 0, Y: 0}, {X: 1, Y: 0}}
	default:
		return regularPolygon(3, radius)
	}
}

// regularPolygon generates vertices of a regular n-sided polygon centered at origin.
func regularPolygon(n int, radius float64) []domain.Point {
	points := make([]domain.Point, n)
	for i := range n {
		angle := 2*math.Pi*float64(i)/float64(n) - math.Pi/2 // Start from top
		points[i] = domain.Point{
			X: int(math.Round(radius * math.Cos(angle))),
			Y: int(math.Round(radius * math.Sin(angle))),
		}
	}
	return points
}

// rectangleVertices generates a rectangle with width=2*radius and height=radius.
func rectangleVertices(radius float64) []domain.Point {
	w := int(math.Round(radius))
	h := int(math.Round(radius * 0.6))
	return []domain.Point{
		{X: -w, Y: -h},
		{X: w, Y: -h},
		{X: w, Y: h},
		{X: -w, Y: h},
	}
}
