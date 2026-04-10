package geometry

import (
	"afk-hero/internal/domain"
	"math"
	"time"
)

// AnimationFrame represents a single frame of the mouse animation.
type AnimationFrame struct {
	Position domain.Point
	Time     time.Duration
}

// RelativeStep represents a timed relative mouse movement delta.
type RelativeStep struct {
	Delta domain.Point
	Time  time.Duration
}

// Interpolate generates smooth animation frames along a path of waypoints
// over the given total duration. Uses linear interpolation between waypoints
// with equal time distribution based on segment length.
func Interpolate(waypoints []domain.Point, totalDuration time.Duration, tickInterval time.Duration) []AnimationFrame {
	if len(waypoints) < 2 {
		return nil
	}

	// Calculate total path length
	totalLength := 0.0
	segments := make([]float64, len(waypoints)-1)
	for i := 0; i < len(waypoints)-1; i++ {
		dx := float64(waypoints[i+1].X - waypoints[i].X)
		dy := float64(waypoints[i+1].Y - waypoints[i].Y)
		seg := math.Sqrt(dx*dx + dy*dy)
		segments[i] = seg
		totalLength += seg
	}

	if totalLength == 0 {
		return nil
	}

	// Calculate cumulative normalized distances for each waypoint
	cumDist := make([]float64, len(waypoints))
	cumDist[0] = 0
	for i := 0; i < len(segments); i++ {
		cumDist[i+1] = cumDist[i] + segments[i]/totalLength
	}

	// Generate frames at tick intervals
	numTicks := int(totalDuration / tickInterval)
	if numTicks < 1 {
		numTicks = 1
	}

	frames := make([]AnimationFrame, 0, numTicks+1)

	for tick := 0; tick <= numTicks; tick++ {
		t := float64(tick) / float64(numTicks) // 0.0 to 1.0
		elapsed := time.Duration(float64(totalDuration) * t)

		// Find which segment we're in
		pos := interpolateAtT(waypoints, cumDist, t)
		frames = append(frames, AnimationFrame{
			Position: pos,
			Time:     elapsed,
		})
	}

	return frames
}

// RelativeSteps converts absolute animation frames into timed unit-step relative deltas.
// Large frame deltas are decomposed into 1px substeps to avoid Windows pointer acceleration
// from distorting the final closed path.
func RelativeSteps(frames []AnimationFrame) []RelativeStep {
	if len(frames) < 2 {
		return nil
	}

	steps := make([]RelativeStep, 0, len(frames)-1)
	prevFrame := frames[0]

	for _, frame := range frames[1:] {
		delta := domain.Point{
			X: frame.Position.X - prevFrame.Position.X,
			Y: frame.Position.Y - prevFrame.Position.Y,
		}
		unitSteps := expandDelta(delta)
		if len(unitSteps) == 0 {
			prevFrame = frame
			continue
		}

		interval := frame.Time - prevFrame.Time
		for i, step := range unitSteps {
			steps = append(steps, RelativeStep{
				Delta: step,
				Time:  prevFrame.Time + time.Duration((int64(interval)*int64(i+1))/int64(len(unitSteps))),
			})
		}

		prevFrame = frame
	}

	return steps
}

// interpolateAtT returns the interpolated position at normalized time t (0..1).
func interpolateAtT(waypoints []domain.Point, cumDist []float64, t float64) domain.Point {
	if t <= 0 {
		return waypoints[0]
	}
	if t >= 1 {
		return waypoints[len(waypoints)-1]
	}

	// Find the segment containing t
	for i := 0; i < len(cumDist)-1; i++ {
		if t >= cumDist[i] && t <= cumDist[i+1] {
			segLen := cumDist[i+1] - cumDist[i]
			if segLen == 0 {
				return waypoints[i]
			}
			localT := (t - cumDist[i]) / segLen
			return lerpPoint(waypoints[i], waypoints[i+1], localT)
		}
	}

	return waypoints[len(waypoints)-1]
}

// lerpPoint linearly interpolates between two points.
func lerpPoint(a, b domain.Point, t float64) domain.Point {
	return domain.Point{
		X: a.X + int(math.Round(float64(b.X-a.X)*t)),
		Y: a.Y + int(math.Round(float64(b.Y-a.Y)*t)),
	}
}

func expandDelta(delta domain.Point) []domain.Point {
	steps := max(abs(delta.X), abs(delta.Y))
	if steps == 0 {
		return nil
	}

	expanded := make([]domain.Point, 0, steps)
	prev := domain.Point{}
	for i := 1; i <= steps; i++ {
		target := domain.Point{
			X: int(math.Round(float64(i*delta.X) / float64(steps))),
			Y: int(math.Round(float64(i*delta.Y) / float64(steps))),
		}
		expanded = append(expanded, domain.Point{
			X: target.X - prev.X,
			Y: target.Y - prev.Y,
		})
		prev = target
	}

	return expanded
}
