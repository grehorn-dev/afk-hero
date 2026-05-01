package runtime

import (
	"afk-hero/internal/domain"
	"afk-hero/internal/geometry"
	"afk-hero/internal/logging"
	"context"
	"math/rand/v2"
	"time"
)

// handleWaitingForInactivity advances the WaitingForInactivity state: it reads
// the current idle time, publishes progress relative to the sampled threshold,
// and triggers the animation cycle when the threshold is reached.
func (e *Engine) handleWaitingForInactivity(ctx context.Context, currentState domain.State, settings domain.Settings) {
	idle, err := e.activityMonitor.IdleTime()
	if err != nil {
		logging.Get().Error("idle time check failed", "error", err)
		return
	}

	threshold := e.sampledInactivityThreshold(currentState, settings)
	if threshold < 1 {
		threshold = 1
	}

	e.store.SetProgress(
		clampProgress(float64(idle.Seconds())/float64(threshold)),
		"status.WaitingForInactivity",
	)

	if idle < time.Duration(threshold)*time.Second {
		return
	}

	e.resetInactivityThreshold()
	if !e.send(domain.EventInactivityReached) {
		return
	}

	cycleCtx, cycleID := e.startCycle(ctx)
	go e.runAnimation(cycleCtx, cycleID, settings)
}

// runAnimation plans a trajectory, decomposes it into unit-step relative
// deltas and drives the pointer through them. Exits early on user input,
// platform errors or context cancellation. When the cycle completes
// successfully, it hands off to waitInterval for the idle phase.
func (e *Engine) runAnimation(ctx context.Context, cycleID uint64, settings domain.Settings) {
	defer logging.RecoverPanic(
		"runtime.Engine.runAnimation",
		"cycle_id", cycleID,
		"shape", settings.Shape,
		"direction", settings.Direction,
	)

	clearCycle := true
	defer func() {
		if clearCycle {
			e.clearCycle(cycleID)
		}
	}()

	origin, err := e.pointer.GetPosition()
	if err != nil {
		logging.Get().Error("get cursor position failed", "error", err)
		if e.send(domain.EventError) {
			e.store.SetProgress(0, "status.Error")
		}
		return
	}

	if e.shouldSkipAnimation(settings) {
		logging.Get().Debug("animation skipped: target window is not foreground")
		if e.send(domain.EventUserInput) {
			e.store.SetProgress(0, "status.WaitingForInactivity")
		}
		return
	}

	bounds, err := e.screen.WorkArea(origin)
	if err != nil {
		logging.Get().Error("get work area failed", "error", err)
		if e.send(domain.EventError) {
			e.store.SetProgress(0, "status.Error")
		}
		return
	}

	shape := settings.Shape
	if shape == domain.ShapeRandom {
		shapes := domain.RandomShapes()
		shape = shapes[rand.IntN(len(shapes))]
	}

	direction := settings.Direction
	if direction == domain.DirectionRandom {
		dirs := domain.RandomDirections()
		direction = dirs[rand.IntN(len(dirs))]
	}

	distance := e.pickValue(settings.Distance)
	duration := time.Duration(e.pickValue(settings.Speed)) * time.Second
	waypoints := geometry.PlanTrajectory(origin, shape, direction, distance, bounds)
	if len(waypoints) < 2 {
		logging.Get().Warn("trajectory planning produced no movement")
		if e.send(domain.EventAnimationCompleted) {
			e.store.SetProgress(0, "status.WaitingForInterval")
		}
		return
	}

	frames := geometry.Interpolate(waypoints, duration, animationTickRate)
	if len(frames) == 0 {
		if e.send(domain.EventAnimationCompleted) {
			e.store.SetProgress(0, "status.WaitingForInterval")
		}
		return
	}
	steps := geometry.RelativeSteps(frames)
	if len(steps) == 0 {
		if e.send(domain.EventAnimationCompleted) {
			e.store.SetProgress(0, "status.WaitingForInterval")
		}
		return
	}

	logging.Get().Info("animation started",
		"shape", shape,
		"direction", direction,
		"distance", distance,
		"duration", duration,
		"frames", len(frames),
		"steps", len(steps),
	)

	startTime := time.Now()
	interrupted := false
	failed := false
	lastObserved := origin
	totalSteps := len(steps)

	for i, step := range steps {
		select {
		case <-ctx.Done():
			return
		default:
		}

		if e.checkUserInput(lastObserved) {
			logging.Get().Info("animation interrupted by user input")
			interrupted = true
			break
		}

		if err := e.pointer.MoveBy(step.Delta, platformInputSignature); err != nil {
			logging.Get().Error("move cursor failed", "error", err)
			failed = true
			break
		}

		e.store.SetProgress(
			clampProgress(float64(i+1)/float64(totalSteps)),
			"status.Animating",
		)
		observedPosition, err := e.pointer.GetPosition()
		if err != nil {
			logging.Get().Debug("get cursor position after move failed", "error", err)
			lastObserved = addPoints(lastObserved, step.Delta)
		} else {
			lastObserved = observedPosition
		}

		targetTime := startTime.Add(step.Time)
		sleepDuration := time.Until(targetTime)
		if sleepDuration <= 0 {
			continue
		}

		timer := time.NewTimer(sleepDuration)
		select {
		case <-ctx.Done():
			timer.Stop()
			return
		case <-timer.C:
		}
	}

	if ctx.Err() != nil {
		return
	}

	switch {
	case failed:
		if e.send(domain.EventError) {
			e.store.SetProgress(0, "status.Error")
		}
	case interrupted:
		if e.send(domain.EventUserInput) {
			e.store.SetProgress(0, "status.WaitingForInactivity")
		}
	default:
		if !e.send(domain.EventAnimationCompleted) {
			return
		}

		e.store.SetProgress(0, "status.WaitingForInterval")
		interval := e.pickValue(settings.Interval)
		clearCycle = false
		go e.waitInterval(ctx, cycleID, time.Duration(interval)*time.Second, settings)
	}
}

func (e *Engine) shouldSkipAnimation(settings domain.Settings) bool {
	if !settings.ActivationEnabled || !settings.ActivationTargetWindowOnly || e.windowMgr == nil {
		return false
	}

	target, ok := e.activationTarget(settings)
	if !ok {
		return true
	}

	status, err := e.windowMgr.WindowStatus(target.ID)
	if err != nil || !status.Exists {
		return true
	}

	return status.Minimized || !status.Foreground
}

func (e *Engine) activationTarget(settings domain.Settings) (domain.WindowInfo, bool) {
	switch settings.ActivationMode {
	case domain.ActivationModeAuto:
		return e.autoActivationCache()
	case domain.ActivationModeManual:
		return e.manualActivationCache(settings.ActivationFingerprint())
	default:
		return domain.WindowInfo{}, false
	}
}

// checkUserInput reports whether the cursor has drifted from the position
// the runtime last observed after its own MoveBy call. Any deviation from
// lastObserved is treated as external (user) input. This avoids the false
// negative of the earlier trace-based check, where a user could move the
// cursor back onto a recorded position and be ignored.
func (e *Engine) checkUserInput(lastObserved domain.Point) bool {
	current, err := e.pointer.GetPosition()
	if err != nil {
		return false
	}

	return current != lastObserved
}

func addPoints(base, delta domain.Point) domain.Point {
	return domain.Point{
		X: base.X + delta.X,
		Y: base.Y + delta.Y,
	}
}
