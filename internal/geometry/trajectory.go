package geometry

import (
	"afk-hero/internal/domain"
	"math"
	"math/rand/v2"
)

var (
	pickOnePixelDirection = func() domain.Direction {
		dirs := []domain.Direction{
			domain.DirectionUp,
			domain.DirectionDown,
			domain.DirectionLeft,
			domain.DirectionRight,
		}
		return dirs[rand.IntN(len(dirs))]
	}
	pickPolygonDirection = func() domain.Direction {
		dirs := domain.RandomDirections()
		return dirs[rand.IntN(len(dirs))]
	}
	pickCenteredStartVertex = rand.IntN
	pickCenteredClockwise   = func() bool {
		return rand.IntN(2) == 0
	}
)

// PlanTrajectory computes the full animation waypoints for a movement cycle.
// It returns the sequence of points the cursor should visit, starting and ending
// at the current cursor position (origin).
func PlanTrajectory(
	origin domain.Point,
	shape domain.Shape,
	direction domain.Direction,
	distance int,
	bounds domain.Rect,
) []domain.Point {
	// Handle OnePixel special case
	if shape == domain.ShapeOnePixel {
		return planOnePixel(origin, direction, bounds)
	}

	// Handle Segment
	if shape == domain.ShapeSegment {
		return planSegment(origin, direction, distance, bounds)
	}

	// Generate shape vertices centered at origin
	radius := float64(distance) / 2
	if radius < 1 {
		radius = 1
	}
	vertices := ShapeVertices(shape, radius)
	resolvedDirection := resolvePolygonDirection(direction)

	// Position vertices relative to cursor based on direction.
	// Bounds clamping happens in clampToBounds below.
	positioned := positionShape(origin, vertices, resolvedDirection)

	// Ensure all points are within bounds, apply fallback if needed
	positioned = clampToBounds(positioned, bounds)

	if len(positioned) == 0 {
		return fallbackMovement(origin, bounds)
	}
	if resolvedDirection == domain.DirectionCenter {
		positioned = reorderVertices(positioned, pickCenteredStartVertex(len(positioned)), pickCenteredClockwise())
	}

	// Build full trajectory: origin -> first vertex -> shape path -> first vertex -> origin
	trajectory := make([]domain.Point, 0, len(positioned)+2)
	trajectory = append(trajectory, origin)
	trajectory = append(trajectory, positioned...)
	// Close the shape by returning to first vertex
	trajectory = append(trajectory, positioned[0])
	// Return to origin
	trajectory = append(trajectory, origin)

	return trajectory
}

func resolvePolygonDirection(dir domain.Direction) domain.Direction {
	if dir == domain.DirectionRandom {
		return pickPolygonDirection()
	}
	return dir
}

func planOnePixel(origin domain.Point, dir domain.Direction, bounds domain.Rect) []domain.Point {
	target := segmentTarget(origin, onePixelStepDirection(origin, resolveOnePixelDirection(dir), bounds), 1)
	if target == origin {
		return fallbackMovement(origin, bounds)
	}

	return []domain.Point{origin, target, origin}
}

func resolveOnePixelDirection(dir domain.Direction) domain.Direction {
	switch dir {
	case domain.DirectionCenter:
		return domain.DirectionRight
	case domain.DirectionRandom:
		return pickOnePixelDirection()
	default:
		return dir
	}
}

func onePixelStepDirection(origin domain.Point, preferred domain.Direction, bounds domain.Rect) domain.Direction {
	left := origin.X <= bounds.Left
	right := origin.X >= bounds.Right-1
	top := origin.Y <= bounds.Top
	bottom := origin.Y >= bounds.Bottom-1

	switch {
	case left && top:
		return domain.DirectionRight
	case right && top:
		return domain.DirectionLeft
	case right && bottom:
		return domain.DirectionLeft
	case left && bottom:
		return domain.DirectionRight
	case left:
		return domain.DirectionRight
	case top:
		return domain.DirectionDown
	case right:
		return domain.DirectionLeft
	case bottom:
		return domain.DirectionUp
	default:
		return preferred
	}
}

func planSegment(origin domain.Point, dir domain.Direction, distance int, bounds domain.Rect) []domain.Point {
	if distance < 1 {
		distance = 1
	}

	switch dir {
	case domain.DirectionCenter:
		return planCenteredSegment(origin, distance, bounds)
	case domain.DirectionRandom:
		picked := domain.RandomDirections()
		d := picked[rand.IntN(len(picked))]
		return planSegment(origin, d, distance, bounds)
	}

	target := clampPoint(segmentTarget(origin, dir, distance), bounds)
	if target == origin {
		return fallbackMovement(origin, bounds)
	}

	return []domain.Point{origin, target, origin}
}

func planCenteredSegment(origin domain.Point, distance int, bounds domain.Rect) []domain.Point {
	halfDistance := int(math.Round(float64(distance) / 2))
	if halfDistance < 1 {
		halfDistance = 1
	}

	directions := []domain.Direction{
		domain.DirectionUp,
		domain.DirectionDown,
		domain.DirectionLeft,
		domain.DirectionRight,
	}

	start := rand.IntN(len(directions))
	var best []domain.Point
	bestScore := 0
	fullScore := halfDistance * 4

	for i := 0; i < len(directions); i++ {
		firstDir := directions[(start+i)%len(directions)]
		candidate := []domain.Point{
			origin,
			clampPoint(segmentTarget(origin, firstDir, halfDistance), bounds),
			clampPoint(segmentTarget(origin, oppositeDirection(firstDir), halfDistance), bounds),
			origin,
		}

		score := pathLength(candidate)
		if score > bestScore {
			best = candidate
			bestScore = score
		}
		if score >= fullScore {
			return candidate
		}
	}

	if bestScore > 0 {
		return best
	}

	return fallbackMovement(origin, bounds)
}

func reorderVertices(vertices []domain.Point, startIndex int, clockwise bool) []domain.Point {
	if len(vertices) == 0 {
		return nil
	}

	startIndex %= len(vertices)
	if startIndex < 0 {
		startIndex += len(vertices)
	}

	reordered := make([]domain.Point, 0, len(vertices))
	for i := 0; i < len(vertices); i++ {
		idx := startIndex + i
		if !clockwise {
			idx = startIndex - i
		}
		for idx < 0 {
			idx += len(vertices)
		}
		idx %= len(vertices)
		reordered = append(reordered, vertices[idx])
	}

	return reordered
}

func segmentTarget(origin domain.Point, dir domain.Direction, distance int) domain.Point {
	target := origin
	switch dir {
	case domain.DirectionUp:
		target.Y -= distance
	case domain.DirectionDown:
		target.Y += distance
	case domain.DirectionLeft:
		target.X -= distance
	case domain.DirectionRight:
		target.X += distance
	}

	return target
}

func oppositeDirection(dir domain.Direction) domain.Direction {
	switch dir {
	case domain.DirectionUp:
		return domain.DirectionDown
	case domain.DirectionDown:
		return domain.DirectionUp
	case domain.DirectionLeft:
		return domain.DirectionRight
	case domain.DirectionRight:
		return domain.DirectionLeft
	default:
		return dir
	}
}

func pathLength(points []domain.Point) int {
	if len(points) < 2 {
		return 0
	}

	total := 0
	for i := 0; i < len(points)-1; i++ {
		total += abs(points[i+1].X-points[i].X) + abs(points[i+1].Y-points[i].Y)
	}
	return total
}

func abs(v int) int {
	if v < 0 {
		return -v
	}
	return v
}

// positionShape translates shape vertices relative to origin based on direction.
func positionShape(origin domain.Point, vertices []domain.Point, dir domain.Direction) []domain.Point {
	if len(vertices) == 0 {
		return nil
	}

	result := make([]domain.Point, len(vertices))

	switch dir {
	case domain.DirectionCenter:
		// Center the shape on origin
		for i, v := range vertices {
			result[i] = domain.Point{X: origin.X + v.X, Y: origin.Y + v.Y}
		}
	case domain.DirectionUp:
		// Shape drawn above cursor: offset vertices so bottom is at origin
		_, maxY := vertexBounds(vertices)
		for i, v := range vertices {
			result[i] = domain.Point{X: origin.X + v.X, Y: origin.Y + v.Y - maxY.Y}
		}
	case domain.DirectionDown:
		minY, _ := vertexBounds(vertices)
		for i, v := range vertices {
			result[i] = domain.Point{X: origin.X + v.X, Y: origin.Y + v.Y - minY.Y}
		}
	case domain.DirectionLeft:
		_, maxX := vertexBoundsX(vertices)
		for i, v := range vertices {
			result[i] = domain.Point{X: origin.X + v.X - maxX, Y: origin.Y + v.Y}
		}
	case domain.DirectionRight:
		minX, _ := vertexBoundsX(vertices)
		for i, v := range vertices {
			result[i] = domain.Point{X: origin.X + v.X - minX, Y: origin.Y + v.Y}
		}
	case domain.DirectionRandom:
		dirs := []domain.Direction{domain.DirectionCenter, domain.DirectionUp, domain.DirectionDown, domain.DirectionLeft, domain.DirectionRight}
		picked := dirs[rand.IntN(len(dirs))]
		return positionShape(origin, vertices, picked)
	}

	return result
}

func vertexBounds(pts []domain.Point) (min, max domain.Point) {
	min = pts[0]
	max = pts[0]
	for _, p := range pts[1:] {
		if p.Y < min.Y {
			min = p
		}
		if p.Y > max.Y {
			max = p
		}
	}
	return
}

func vertexBoundsX(pts []domain.Point) (minX, maxX int) {
	minX = pts[0].X
	maxX = pts[0].X
	for _, p := range pts[1:] {
		if p.X < minX {
			minX = p.X
		}
		if p.X > maxX {
			maxX = p.X
		}
	}
	return
}

// clampToBounds attempts to fit all points within bounds.
// If points are outside, it shifts the entire shape to fit.
func clampToBounds(pts []domain.Point, bounds domain.Rect) []domain.Point {
	if len(pts) == 0 {
		return nil
	}

	// Find the bounding box of the shape
	minX, maxX := pts[0].X, pts[0].X
	minY, maxY := pts[0].Y, pts[0].Y
	for _, p := range pts[1:] {
		if p.X < minX {
			minX = p.X
		}
		if p.X > maxX {
			maxX = p.X
		}
		if p.Y < minY {
			minY = p.Y
		}
		if p.Y > maxY {
			maxY = p.Y
		}
	}

	// Calculate required shift to fit within bounds
	dx, dy := 0, 0
	if minX < bounds.Left {
		dx = bounds.Left - minX
	} else if maxX >= bounds.Right {
		dx = bounds.Right - 1 - maxX
	}
	if minY < bounds.Top {
		dy = bounds.Top - minY
	} else if maxY >= bounds.Bottom {
		dy = bounds.Bottom - 1 - maxY
	}

	// Check if shape fits at all (even after shifting)
	shapeW := maxX - minX
	shapeH := maxY - minY
	if shapeW >= bounds.Width() || shapeH >= bounds.Height() {
		// Shape too large; scale it down
		return scaleToFit(pts, bounds)
	}

	if dx == 0 && dy == 0 {
		return pts
	}

	result := make([]domain.Point, len(pts))
	for i, p := range pts {
		result[i] = domain.Point{X: p.X + dx, Y: p.Y + dy}
	}
	return result
}

// scaleToFit scales down the shape to fit within bounds while preserving center.
func scaleToFit(pts []domain.Point, bounds domain.Rect) []domain.Point {
	if len(pts) == 0 {
		return nil
	}

	// Find center of the shape
	cx, cy := 0.0, 0.0
	for _, p := range pts {
		cx += float64(p.X)
		cy += float64(p.Y)
	}
	cx /= float64(len(pts))
	cy /= float64(len(pts))

	// Calculate scale factor
	maxW := float64(bounds.Width()) / 2
	maxH := float64(bounds.Height()) / 2
	maxAllowed := maxW
	if maxH < maxAllowed {
		maxAllowed = maxH
	}

	scale := 1.0
	currentMax := 0.0
	for _, p := range pts {
		dx := float64(p.X) - cx
		dy := float64(p.Y) - cy
		d := math.Sqrt(dx*dx + dy*dy)
		if d > currentMax {
			currentMax = d
		}
	}
	if currentMax > maxAllowed && currentMax > 0 {
		scale = maxAllowed / currentMax * 0.9 // 90% to leave margin
	}

	// Scale and recenter within bounds
	bcx := float64(bounds.Left+bounds.Right) / 2
	bcy := float64(bounds.Top+bounds.Bottom) / 2

	result := make([]domain.Point, len(pts))
	for i, p := range pts {
		dx := float64(p.X) - cx
		dy := float64(p.Y) - cy
		result[i] = domain.Point{
			X: int(bcx + dx*scale),
			Y: int(bcy + dy*scale),
		}
	}
	return result
}

// clampPoint constrains a point to be within the given rectangle.
func clampPoint(p domain.Point, bounds domain.Rect) domain.Point {
	if p.X < bounds.Left {
		p.X = bounds.Left
	}
	if p.X >= bounds.Right {
		p.X = bounds.Right - 1
	}
	if p.Y < bounds.Top {
		p.Y = bounds.Top
	}
	if p.Y >= bounds.Bottom {
		p.Y = bounds.Bottom - 1
	}
	return p
}

// fallbackMovement provides a guaranteed minimal movement when normal trajectory fails.
func fallbackMovement(origin domain.Point, bounds domain.Rect) []domain.Point {
	cx := (bounds.Left + bounds.Right) / 2
	cy := (bounds.Top + bounds.Bottom) / 2

	// Try 4 directions, 1 pixel each
	candidates := []domain.Point{
		{X: origin.X + 1, Y: origin.Y},
		{X: origin.X - 1, Y: origin.Y},
		{X: origin.X, Y: origin.Y + 1},
		{X: origin.X, Y: origin.Y - 1},
	}

	for _, c := range candidates {
		if bounds.Contains(c) {
			return []domain.Point{origin, c, origin}
		}
	}

	// Last resort: move to center of screen
	center := domain.Point{X: cx, Y: cy}
	return []domain.Point{origin, center, origin}
}
