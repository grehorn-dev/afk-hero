package runtime

import (
	"afk-hero/internal/domain"
	"context"
)

// startCycle creates a new cancelable animation cycle and cancels any
// previously active cycle. The returned context is cancelled when the cycle
// is disabled or replaced; the returned cycleID must be passed to clearCycle
// when the cycle owner finishes to release the bookkeeping slot.
func (e *Engine) startCycle(parent context.Context) (context.Context, uint64) {
	e.mu.Lock()
	defer e.mu.Unlock()

	if e.activeCycleCancel != nil {
		e.activeCycleCancel()
	}

	ctx, cancel := context.WithCancel(parent)
	e.activeCycleCancel = cancel
	e.activeCycleID++

	return ctx, e.activeCycleID
}

// clearCycle releases the active cycle slot if the given cycleID still owns it.
// This is idempotent: if a newer cycle has already taken over, clearCycle is a
// no-op so late-arriving cleanups cannot stomp on an unrelated cycle.
func (e *Engine) clearCycle(cycleID uint64) {
	e.mu.Lock()
	defer e.mu.Unlock()

	if e.activeCycleID != cycleID {
		return
	}

	e.activeCycleCancel = nil
}

// cancelActiveCycle cancels any in-flight cycle. Safe to call at any time;
// safe to call concurrently with startCycle.
func (e *Engine) cancelActiveCycle() {
	e.mu.Lock()
	cancel := e.activeCycleCancel
	e.activeCycleCancel = nil
	e.mu.Unlock()

	if cancel != nil {
		cancel()
	}
}

// sampledInactivityThreshold returns a cached inactivity threshold for the
// current state and config. It resamples the threshold whenever the state or
// the underlying NumericRange changes, which keeps Random-mode thresholds
// stable within a single waiting window but adapts promptly to settings edits.
func (e *Engine) sampledInactivityThreshold(currentState domain.State, settings domain.Settings) int {
	e.mu.Lock()
	defer e.mu.Unlock()

	if e.inactivityThreshold == 0 || e.inactivityState != currentState || e.inactivityConfig != settings.Inactivity {
		e.inactivityThreshold = e.pickValue(settings.Inactivity)
		e.inactivityState = currentState
		e.inactivityConfig = settings.Inactivity
	}

	return e.inactivityThreshold
}

// resetInactivityThreshold clears the cached inactivity threshold so the next
// WaitingForInactivity iteration will resample it.
func (e *Engine) resetInactivityThreshold() {
	e.mu.Lock()
	e.inactivityThreshold = 0
	e.inactivityState = ""
	e.inactivityConfig = domain.NumericRange{}
	e.mu.Unlock()
}
