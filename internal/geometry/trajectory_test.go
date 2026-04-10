package geometry

import (
	"afk-hero/internal/domain"
	"testing"
)

var testBounds = domain.Rect{Left: 0, Top: 0, Right: 1920, Bottom: 1080}

func withTrajectoryPickers(t *testing.T, direction domain.Direction, startIndex int, clockwise bool) {
	t.Helper()

	prevDirection := pickPolygonDirection
	prevStart := pickCenteredStartVertex
	prevClockwise := pickCenteredClockwise

	pickPolygonDirection = func() domain.Direction {
		return direction
	}
	pickCenteredStartVertex = func(numVertices int) int {
		return startIndex
	}
	pickCenteredClockwise = func() bool {
		return clockwise
	}

	t.Cleanup(func() {
		pickPolygonDirection = prevDirection
		pickCenteredStartVertex = prevStart
		pickCenteredClockwise = prevClockwise
	})
}

func withOnePixelDirectionPicker(t *testing.T, direction domain.Direction) {
	t.Helper()

	prevPicker := pickOnePixelDirection
	pickOnePixelDirection = func() domain.Direction {
		return direction
	}

	t.Cleanup(func() {
		pickOnePixelDirection = prevPicker
	})
}

func assertPointsEqual(t *testing.T, got, want []domain.Point) {
	t.Helper()

	if len(got) != len(want) {
		t.Fatalf("expected %d points, got %d", len(want), len(got))
	}

	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("point %d mismatch: got %v, want %v", i, got[i], want[i])
		}
	}
}

func TestPlanTrajectory_ReturnsToOrigin(t *testing.T) {
	shapes := []domain.Shape{domain.ShapeTriangle, domain.ShapeRectangle, domain.ShapePentagon, domain.ShapeSegment}
	dirs := []domain.Direction{domain.DirectionCenter, domain.DirectionUp, domain.DirectionDown, domain.DirectionLeft, domain.DirectionRight}
	origin := domain.Point{X: 960, Y: 540}

	for _, shape := range shapes {
		for _, dir := range dirs {
			traj := PlanTrajectory(origin, shape, dir, 100, testBounds)
			if len(traj) < 2 {
				t.Errorf("shape=%s dir=%s: trajectory too short (%d points)", shape, dir, len(traj))
				continue
			}
			last := traj[len(traj)-1]
			if last != origin {
				t.Errorf("shape=%s dir=%s: expected return to origin (%d,%d), got (%d,%d)",
					shape, dir, origin.X, origin.Y, last.X, last.Y)
			}
		}
	}
}

func TestPlanTrajectory_WithinBounds(t *testing.T) {
	shapes := []domain.Shape{domain.ShapeTriangle, domain.ShapeRectangle, domain.ShapePentagon}
	origin := domain.Point{X: 960, Y: 540}

	for _, shape := range shapes {
		traj := PlanTrajectory(origin, shape, domain.DirectionCenter, 200, testBounds)
		for i, p := range traj {
			if p.X < testBounds.Left || p.X >= testBounds.Right || p.Y < testBounds.Top || p.Y >= testBounds.Bottom {
				t.Errorf("shape=%s: waypoint %d (%d,%d) is outside bounds", shape, i, p.X, p.Y)
			}
		}
	}
}

func TestPlanTrajectory_OnePixel(t *testing.T) {
	origin := domain.Point{X: 100, Y: 100}
	tests := []struct {
		name string
		dir  domain.Direction
		want domain.Point
	}{
		{name: "center becomes right", dir: domain.DirectionCenter, want: domain.Point{X: 101, Y: 100}},
		{name: "up", dir: domain.DirectionUp, want: domain.Point{X: 100, Y: 99}},
		{name: "down", dir: domain.DirectionDown, want: domain.Point{X: 100, Y: 101}},
		{name: "left", dir: domain.DirectionLeft, want: domain.Point{X: 99, Y: 100}},
		{name: "right", dir: domain.DirectionRight, want: domain.Point{X: 101, Y: 100}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			traj := PlanTrajectory(origin, domain.ShapeOnePixel, tt.dir, 0, testBounds)
			want := []domain.Point{origin, tt.want, origin}
			assertPointsEqual(t, traj, want)
		})
	}
}

func TestPlanTrajectory_OnePixelRandomUsesPickedDirection(t *testing.T) {
	withOnePixelDirectionPicker(t, domain.DirectionDown)

	origin := domain.Point{X: 100, Y: 100}
	traj := PlanTrajectory(origin, domain.ShapeOnePixel, domain.DirectionRandom, 0, testBounds)

	want := []domain.Point{
		origin,
		{X: 100, Y: 101},
		origin,
	}
	assertPointsEqual(t, traj, want)
}

func TestPlanTrajectory_OnePixelBoundaryOverridesDirection(t *testing.T) {
	bounds := domain.Rect{Left: 0, Top: 0, Right: 100, Bottom: 100}
	directions := []domain.Direction{
		domain.DirectionCenter,
		domain.DirectionUp,
		domain.DirectionDown,
		domain.DirectionLeft,
		domain.DirectionRight,
		domain.DirectionRandom,
	}

	tests := []struct {
		name   string
		origin domain.Point
		want   domain.Point
	}{
		{name: "top left corner", origin: domain.Point{X: 0, Y: 0}, want: domain.Point{X: 1, Y: 0}},
		{name: "top right corner", origin: domain.Point{X: 99, Y: 0}, want: domain.Point{X: 98, Y: 0}},
		{name: "bottom right corner", origin: domain.Point{X: 99, Y: 99}, want: domain.Point{X: 98, Y: 99}},
		{name: "bottom left corner", origin: domain.Point{X: 0, Y: 99}, want: domain.Point{X: 1, Y: 99}},
		{name: "left edge", origin: domain.Point{X: 0, Y: 50}, want: domain.Point{X: 1, Y: 50}},
		{name: "top edge", origin: domain.Point{X: 50, Y: 0}, want: domain.Point{X: 50, Y: 1}},
		{name: "right edge", origin: domain.Point{X: 99, Y: 50}, want: domain.Point{X: 98, Y: 50}},
		{name: "bottom edge", origin: domain.Point{X: 50, Y: 99}, want: domain.Point{X: 50, Y: 98}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			withOnePixelDirectionPicker(t, domain.DirectionDown)
			for _, dir := range directions {
				traj := PlanTrajectory(tt.origin, domain.ShapeOnePixel, dir, 0, bounds)
				want := []domain.Point{tt.origin, tt.want, tt.origin}
				assertPointsEqual(t, traj, want)
			}
		})
	}
}

func TestPlanTrajectory_OnePixelOutsideWorkAreaUsesNearestBoundaryRule(t *testing.T) {
	bounds := domain.Rect{Left: 0, Top: 0, Right: 100, Bottom: 100}
	tests := []struct {
		name   string
		origin domain.Point
		dir    domain.Direction
		want   domain.Point
	}{
		{
			name:   "below bottom edge goes up",
			origin: domain.Point{X: 50, Y: 100},
			dir:    domain.DirectionLeft,
			want:   domain.Point{X: 50, Y: 99},
		},
		{
			name:   "outside top left follows top-left corner rule",
			origin: domain.Point{X: -1, Y: -1},
			dir:    domain.DirectionUp,
			want:   domain.Point{X: 0, Y: -1},
		},
		{
			name:   "outside right edge goes left",
			origin: domain.Point{X: 100, Y: 50},
			dir:    domain.DirectionDown,
			want:   domain.Point{X: 99, Y: 50},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			traj := PlanTrajectory(tt.origin, domain.ShapeOnePixel, tt.dir, 0, bounds)
			want := []domain.Point{tt.origin, tt.want, tt.origin}
			assertPointsEqual(t, traj, want)
		})
	}
}

func TestReorderVerticesClockwise(t *testing.T) {
	vertices := []domain.Point{
		{X: 1, Y: 1},
		{X: 2, Y: 2},
		{X: 3, Y: 3},
		{X: 4, Y: 4},
	}

	got := reorderVertices(vertices, 2, true)
	want := []domain.Point{
		{X: 3, Y: 3},
		{X: 4, Y: 4},
		{X: 1, Y: 1},
		{X: 2, Y: 2},
	}

	assertPointsEqual(t, got, want)
}

func TestReorderVerticesCounterClockwise(t *testing.T) {
	vertices := []domain.Point{
		{X: 1, Y: 1},
		{X: 2, Y: 2},
		{X: 3, Y: 3},
		{X: 4, Y: 4},
	}

	got := reorderVertices(vertices, 2, false)
	want := []domain.Point{
		{X: 3, Y: 3},
		{X: 2, Y: 2},
		{X: 1, Y: 1},
		{X: 4, Y: 4},
	}

	assertPointsEqual(t, got, want)
}

func TestPlanTrajectory_CenterStartsFromPickedVertexAndDirection(t *testing.T) {
	withTrajectoryPickers(t, domain.DirectionCenter, 2, false)

	origin := domain.Point{X: 960, Y: 540}
	traj := PlanTrajectory(origin, domain.ShapeRectangle, domain.DirectionCenter, 200, testBounds)

	vertices := ShapeVertices(domain.ShapeRectangle, 100)
	positioned := positionShape(origin, vertices, domain.DirectionCenter)
	positioned = clampToBounds(positioned, testBounds)
	positioned = reorderVertices(positioned, 2, false)

	want := append([]domain.Point{origin}, positioned...)
	want = append(want, positioned[0], origin)

	assertPointsEqual(t, traj, want)
}

func TestPlanTrajectory_RandomDirectionCenterMatchesExplicitCenterTraversal(t *testing.T) {
	withTrajectoryPickers(t, domain.DirectionCenter, 1, true)

	origin := domain.Point{X: 960, Y: 540}
	got := PlanTrajectory(origin, domain.ShapeTriangle, domain.DirectionRandom, 200, testBounds)

	vertices := ShapeVertices(domain.ShapeTriangle, 100)
	positioned := positionShape(origin, vertices, domain.DirectionCenter)
	positioned = clampToBounds(positioned, testBounds)
	positioned = reorderVertices(positioned, 1, true)

	want := append([]domain.Point{origin}, positioned...)
	want = append(want, positioned[0], origin)

	assertPointsEqual(t, got, want)
}

func TestPlanTrajectory_SegmentCenterUsesOppositeEndpointsAroundOrigin(t *testing.T) {
	origin := domain.Point{X: 960, Y: 540}
	traj := PlanTrajectory(origin, domain.ShapeSegment, domain.DirectionCenter, 100, testBounds)

	if len(traj) != 4 {
		t.Fatalf("expected 4 waypoints for centered segment, got %d", len(traj))
	}
	if traj[0] != origin || traj[len(traj)-1] != origin {
		t.Fatalf("expected centered segment to start and end at origin, got %v ... %v", traj[0], traj[len(traj)-1])
	}

	first := traj[1]
	second := traj[2]
	if first == origin || second == origin {
		t.Fatalf("expected both segment endpoints to differ from origin in spacious bounds, got %v and %v", first, second)
	}

	if first.X+second.X != origin.X*2 || first.Y+second.Y != origin.Y*2 {
		t.Fatalf("expected centered segment endpoints to be opposite around origin, got %v and %v", first, second)
	}

	firstDistance := abs(first.X-origin.X) + abs(first.Y-origin.Y)
	secondDistance := abs(second.X-origin.X) + abs(second.Y-origin.Y)
	if firstDistance != 50 || secondDistance != 50 {
		t.Fatalf("expected centered segment endpoints to be half-distance from origin, got %d and %d", firstDistance, secondDistance)
	}
}

func TestPlanTrajectory_CornerPositions(t *testing.T) {
	corners := []domain.Point{
		{X: 0, Y: 0},                                        // Top-left
		{X: testBounds.Right - 1, Y: 0},                     // Top-right
		{X: 0, Y: testBounds.Bottom - 1},                    // Bottom-left
		{X: testBounds.Right - 1, Y: testBounds.Bottom - 1}, // Bottom-right
	}

	for _, corner := range corners {
		traj := PlanTrajectory(corner, domain.ShapeTriangle, domain.DirectionCenter, 100, testBounds)
		if len(traj) < 2 {
			t.Errorf("corner (%d,%d): no movement produced", corner.X, corner.Y)
			continue
		}

		// Verify movement happened (not all points are the same)
		moved := false
		for _, p := range traj {
			if p != corner {
				moved = true
				break
			}
		}
		if !moved {
			t.Errorf("corner (%d,%d): no actual movement in trajectory", corner.X, corner.Y)
		}

		// Verify all points are in bounds
		for i, p := range traj {
			if !testBounds.Contains(p) && p != corner {
				t.Errorf("corner (%d,%d): waypoint %d (%d,%d) out of bounds",
					corner.X, corner.Y, i, p.X, p.Y)
			}
		}
	}
}

func TestPlanTrajectory_SegmentCenterStillMovesAtScreenCorner(t *testing.T) {
	origin := domain.Point{X: 0, Y: 0}
	traj := PlanTrajectory(origin, domain.ShapeSegment, domain.DirectionCenter, 100, testBounds)

	if len(traj) < 2 {
		t.Fatalf("expected centered segment to produce movement at corner, got %d waypoints", len(traj))
	}
	if traj[len(traj)-1] != origin {
		t.Fatalf("expected centered segment to return to origin, got %v", traj[len(traj)-1])
	}

	moved := false
	for _, p := range traj {
		if p != origin {
			moved = true
		}
		if !testBounds.Contains(p) {
			t.Fatalf("expected all waypoints to stay in bounds, got %v", p)
		}
	}

	if !moved {
		t.Fatal("expected centered segment to produce real movement at corner")
	}
}

func TestPlanTrajectory_EdgePositions(t *testing.T) {
	edges := []domain.Point{
		{X: 960, Y: 0},                     // Top edge center
		{X: 960, Y: testBounds.Bottom - 1}, // Bottom edge center
		{X: 0, Y: 540},                     // Left edge center
		{X: testBounds.Right - 1, Y: 540},  // Right edge center
	}

	for _, edge := range edges {
		traj := PlanTrajectory(edge, domain.ShapeRectangle, domain.DirectionCenter, 150, testBounds)
		if len(traj) < 2 {
			t.Errorf("edge (%d,%d): no movement produced", edge.X, edge.Y)
		}
	}
}

func TestPlanTrajectory_AlwaysProducesMovement(t *testing.T) {
	// Property: for any position within bounds, trajectory should have movement
	positions := []domain.Point{
		{X: 0, Y: 0}, {X: 1919, Y: 1079}, {X: 960, Y: 540},
		{X: 0, Y: 540}, {X: 1919, Y: 0},
	}
	shapes := domain.AllShapes()
	shapes = shapes[1:] // Skip Random

	for _, pos := range positions {
		for _, shape := range shapes {
			traj := PlanTrajectory(pos, shape, domain.DirectionCenter, 50, testBounds)
			if len(traj) < 2 {
				t.Errorf("pos=(%d,%d) shape=%s: trajectory has fewer than 2 points",
					pos.X, pos.Y, shape)
			}
		}
	}
}

func TestFallbackMovement(t *testing.T) {
	corner := domain.Point{X: 0, Y: 0}
	traj := fallbackMovement(corner, testBounds)
	if len(traj) < 2 {
		t.Fatal("fallback should produce at least 2 points")
	}
	if traj[0] != corner {
		t.Error("fallback should start at origin")
	}
}

func TestClampPoint(t *testing.T) {
	bounds := domain.Rect{Left: 10, Top: 20, Right: 100, Bottom: 200}
	tests := []struct {
		in  domain.Point
		out domain.Point
	}{
		{domain.Point{X: 50, Y: 50}, domain.Point{X: 50, Y: 50}},
		{domain.Point{X: 5, Y: 50}, domain.Point{X: 10, Y: 50}},
		{domain.Point{X: 200, Y: 50}, domain.Point{X: 99, Y: 50}},
		{domain.Point{X: 50, Y: 10}, domain.Point{X: 50, Y: 20}},
		{domain.Point{X: 50, Y: 300}, domain.Point{X: 50, Y: 199}},
	}
	for _, tt := range tests {
		got := clampPoint(tt.in, bounds)
		if got != tt.out {
			t.Errorf("clampPoint(%v) = %v, want %v", tt.in, got, tt.out)
		}
	}
}
