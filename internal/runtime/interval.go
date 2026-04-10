package runtime

import (
	"afk-hero/internal/domain"
	"afk-hero/internal/logging"
	"context"
	"time"
)

// waitInterval waits for the configured cool-down between animation cycles.
// It watches for fresh user input (idle time going backwards) and transitions
// back to WaitingForInactivity when such input is detected, or starts the next
// animation cycle when the interval elapses.
func (e *Engine) waitInterval(ctx context.Context, cycleID uint64, duration time.Duration, settings domain.Settings) {
	defer logging.RecoverPanic(
		"runtime.Engine.waitInterval",
		"cycle_id", cycleID,
		"duration", duration,
	)

	clearCycle := true
	defer func() {
		if clearCycle {
			e.clearCycle(cycleID)
		}
	}()

	if duration <= 0 {
		if e.send(domain.EventIntervalElapsed) {
			e.store.SetProgress(0, "status.Animating")
			clearCycle = false
			go e.runAnimation(ctx, cycleID, settings)
		}
		return
	}

	start := time.Now()
	ticker := time.NewTicker(idlePollInterval)
	defer ticker.Stop()
	lastIdle, idleErr := e.activityMonitor.IdleTime()
	if idleErr != nil {
		logging.Get().Debug("idle baseline unavailable during interval wait", "error", idleErr)
		lastIdle = 0
	}

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			idle, err := e.activityMonitor.IdleTime()
			if err != nil {
				logging.Get().Debug("idle check unavailable during interval wait", "error", err)
			} else {
				if idle < lastIdle {
					if e.send(domain.EventUserInput) {
						e.store.SetProgress(0, "status.WaitingForInactivity")
					}
					return
				}
				lastIdle = idle
			}

			elapsed := time.Since(start)
			if elapsed >= duration {
				if e.send(domain.EventIntervalElapsed) {
					e.store.SetProgress(0, "status.Animating")
					clearCycle = false
					go e.runAnimation(ctx, cycleID, settings)
				}
				return
			}

			e.store.SetProgress(
				clampProgress(elapsed.Seconds()/duration.Seconds()),
				"status.WaitingForInterval",
			)
		}
	}
}
