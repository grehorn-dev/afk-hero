package geometry

import (
	"afk-hero/internal/domain"
	"testing"
	"time"
)

func TestInterpolate_BasicPath(t *testing.T) {
	waypoints := []domain.Point{
		{X: 0, Y: 0},
		{X: 100, Y: 0},
		{X: 0, Y: 0},
	}

	frames := Interpolate(waypoints, 1*time.Second, 16*time.Millisecond)
	if len(frames) == 0 {
		t.Fatal("expected frames to be generated")
	}

	// First frame should be at start
	if frames[0].Position != waypoints[0] {
		t.Errorf("first frame should be at start, got %v", frames[0].Position)
	}

	// Last frame should be at end
	last := frames[len(frames)-1]
	if last.Position != waypoints[len(waypoints)-1] {
		t.Errorf("last frame should be at end, got %v", last.Position)
	}
}

func TestInterpolate_SinglePoint(t *testing.T) {
	frames := Interpolate([]domain.Point{{X: 50, Y: 50}}, time.Second, 16*time.Millisecond)
	if frames != nil {
		t.Error("single point should return nil")
	}
}

func TestInterpolate_FrameTimings(t *testing.T) {
	waypoints := []domain.Point{
		{X: 0, Y: 0},
		{X: 100, Y: 100},
	}

	duration := 1 * time.Second
	tick := 100 * time.Millisecond
	frames := Interpolate(waypoints, duration, tick)

	// Should have approximately duration/tick + 1 frames
	expected := int(duration/tick) + 1
	if len(frames) != expected {
		t.Errorf("expected %d frames, got %d", expected, len(frames))
	}

	// Verify timing is monotonically increasing
	for i := 1; i < len(frames); i++ {
		if frames[i].Time <= frames[i-1].Time {
			t.Errorf("frame %d time %v is not after frame %d time %v",
				i, frames[i].Time, i-1, frames[i-1].Time)
		}
	}
}

func TestLerpPoint(t *testing.T) {
	a := domain.Point{X: 0, Y: 0}
	b := domain.Point{X: 100, Y: 200}

	mid := lerpPoint(a, b, 0.5)
	if mid.X != 50 || mid.Y != 100 {
		t.Errorf("expected (50,100), got (%d,%d)", mid.X, mid.Y)
	}

	start := lerpPoint(a, b, 0.0)
	if start != a {
		t.Errorf("expected start, got %v", start)
	}

	end := lerpPoint(a, b, 1.0)
	if end != b {
		t.Errorf("expected end, got %v", end)
	}
}

func TestRelativeSteps_ExpandsLargeDeltaIntoUnitSteps(t *testing.T) {
	frames := []AnimationFrame{
		{Position: domain.Point{X: 0, Y: 0}, Time: 0},
		{Position: domain.Point{X: 3, Y: 1}, Time: 30 * time.Millisecond},
	}

	steps := RelativeSteps(frames)
	want := []RelativeStep{
		{Delta: domain.Point{X: 1, Y: 0}, Time: 10 * time.Millisecond},
		{Delta: domain.Point{X: 1, Y: 1}, Time: 20 * time.Millisecond},
		{Delta: domain.Point{X: 1, Y: 0}, Time: 30 * time.Millisecond},
	}

	if len(steps) != len(want) {
		t.Fatalf("expected %d relative steps, got %d", len(want), len(steps))
	}

	for i := range want {
		if steps[i] != want[i] {
			t.Fatalf("step %d: got %+v, want %+v", i, steps[i], want[i])
		}
	}
}

func TestRelativeSteps_PreservesClosedPathSum(t *testing.T) {
	waypoints := []domain.Point{
		{X: 10, Y: 10},
		{X: 40, Y: 10},
		{X: 10, Y: 10},
	}

	frames := Interpolate(waypoints, time.Second, 100*time.Millisecond)
	steps := RelativeSteps(frames)
	if len(steps) == 0 {
		t.Fatal("expected relative steps to be generated")
	}

	sum := domain.Point{}
	for _, step := range steps {
		if step.Delta.X < -1 || step.Delta.X > 1 || step.Delta.Y < -1 || step.Delta.Y > 1 {
			t.Fatalf("expected unit-step deltas, got %+v", step.Delta)
		}
		sum.X += step.Delta.X
		sum.Y += step.Delta.Y
	}

	if sum != (domain.Point{}) {
		t.Fatalf("expected closed path to sum to zero delta, got %+v", sum)
	}

	if last := steps[len(steps)-1].Time; last != frames[len(frames)-1].Time {
		t.Fatalf("expected last relative step time %v, got %v", frames[len(frames)-1].Time, last)
	}
}
