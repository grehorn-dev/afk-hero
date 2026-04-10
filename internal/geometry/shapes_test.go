package geometry

import (
	"afk-hero/internal/domain"
	"testing"
)

func TestShapeVertices_Triangle(t *testing.T) {
	pts := ShapeVertices(domain.ShapeTriangle, 100)
	if len(pts) != 3 {
		t.Fatalf("expected 3 vertices for triangle, got %d", len(pts))
	}
}

func TestShapeVertices_Rectangle(t *testing.T) {
	pts := ShapeVertices(domain.ShapeRectangle, 100)
	if len(pts) != 4 {
		t.Fatalf("expected 4 vertices for rectangle, got %d", len(pts))
	}
}

func TestShapeVertices_Pentagon(t *testing.T) {
	pts := ShapeVertices(domain.ShapePentagon, 100)
	if len(pts) != 5 {
		t.Fatalf("expected 5 vertices for pentagon, got %d", len(pts))
	}
}

func TestShapeVertices_Segment(t *testing.T) {
	pts := ShapeVertices(domain.ShapeSegment, 50)
	if len(pts) != 2 {
		t.Fatalf("expected 2 vertices for segment, got %d", len(pts))
	}
	if pts[0].X != 0 || pts[0].Y != 0 {
		t.Errorf("segment should start at origin, got (%d,%d)", pts[0].X, pts[0].Y)
	}
}

func TestShapeVertices_OnePixel(t *testing.T) {
	pts := ShapeVertices(domain.ShapeOnePixel, 100)
	if len(pts) != 2 {
		t.Fatalf("expected 2 vertices for one pixel, got %d", len(pts))
	}
	if pts[1].X != 1 {
		t.Errorf("one pixel should move 1px, got %d", pts[1].X)
	}
}

func TestRegularPolygon_VertexCount(t *testing.T) {
	for _, n := range []int{3, 4, 5, 6, 8} {
		pts := regularPolygon(n, 100)
		if len(pts) != n {
			t.Errorf("regularPolygon(%d) returned %d vertices", n, len(pts))
		}
	}
}
