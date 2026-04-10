// Package runtime owns the main engine that drives AFK-Hero: the idle
// detection loop, the animation scheduler and the window activation monitor.
// The engine is deliberately platform-agnostic — it speaks only to the
// platform interfaces (PointerController, UserActivityMonitor, ScreenProvider,
// WindowManager) and keeps all Win32-specific code behind adapters.
//
// The package is split into focused files:
//
//	engine.go      - Engine struct, lifecycle (Start/Stop/Enable/Disable),
//	                 run/tick dispatch, shared helpers.
//	cycle.go       - per-cycle bookkeeping and cached inactivity threshold.
//	animation.go   - WaitingForInactivity handling and the animation loop.
//	interval.go    - cool-down between animation cycles.
//	activation.go  - window activation monitor (Auto/Manual modes, caches,
//	                 diagnostics and publishing).
package runtime

import (
	"afk-hero/internal/domain"
	"afk-hero/internal/logging"
	"afk-hero/internal/platform"
	"afk-hero/internal/state"
	"context"
	"math/rand/v2"
	"sync"
	"time"
)

const (
	idlePollInterval       = 250 * time.Millisecond
	animationTickRate      = 16 * time.Millisecond // ~60fps
	platformInputSignature = 0xAF4D0E12
)

// Engine drives the main application loop.
type Engine struct {
	store           *state.Store
	sm              *domain.StateMachine
	activityMonitor platform.UserActivityMonitor
	pointer         platform.PointerController
	screen          platform.ScreenProvider
	windowMgr       platform.WindowManager

	pickValueFn func(domain.NumericRange) int

	// mu guards every field below, including the lifecycle cancel function
	// and per-cycle cancellation state. Callers must NOT hold mu while
	// invoking user-provided callbacks (onActivationChange) or platform
	// adapters to avoid unbounded contention and re-entrancy.
	mu                  sync.Mutex
	cancel              context.CancelFunc
	activeCycleCancel   context.CancelFunc
	activeCycleID       uint64
	inactivityThreshold int
	inactivityState     domain.State
	inactivityConfig    domain.NumericRange
	activationInactive  time.Time
	activationLastTick  time.Time
	activationState     domain.WindowActivationState
	activationLogKey    string
	autoActivationWnd   domain.WindowInfo
	autoActivationSet   bool
	manualActivationFP  domain.WindowFingerprint
	manualActivationWnd domain.WindowInfo
	manualActivationSet bool
	onActivationChange  func(domain.WindowActivationState)
}

// NewEngine creates a new runtime engine.
func NewEngine(
	store *state.Store,
	sm *domain.StateMachine,
	activity platform.UserActivityMonitor,
	pointer platform.PointerController,
	screen platform.ScreenProvider,
	windowMgr platform.WindowManager,
) *Engine {
	return &Engine{
		store:           store,
		sm:              sm,
		activityMonitor: activity,
		pointer:         pointer,
		screen:          screen,
		windowMgr:       windowMgr,
		pickValueFn:     pickValue,
	}
}

// Start begins the main engine loop in a goroutine.
// Calling Start more than once cancels the previous run loop before
// starting a new one. Safe for concurrent use with Stop.
func (e *Engine) Start(ctx context.Context) {
	runCtx, cancel := context.WithCancel(ctx)

	e.mu.Lock()
	if e.cancel != nil {
		e.cancel()
	}
	e.cancel = cancel
	e.mu.Unlock()

	go e.run(runCtx)
}

// Stop gracefully stops the engine and cancels any in-flight animation cycle.
// Safe to call multiple times and safe for concurrent use with Start.
func (e *Engine) Stop() {
	e.cancelActiveCycle()

	e.mu.Lock()
	cancel := e.cancel
	e.cancel = nil
	e.mu.Unlock()

	if cancel != nil {
		cancel()
	}
}

// SetActivationStateCallback registers a callback for activation monitor
// state updates. The callback fires once synchronously with the current
// activation state so fresh subscribers can render an initial snapshot.
func (e *Engine) SetActivationStateCallback(cb func(domain.WindowActivationState)) {
	e.mu.Lock()
	e.onActivationChange = cb
	state := e.activationState
	e.mu.Unlock()

	if cb != nil {
		cb(state)
	}
}

// ActivationState returns the latest activation monitor state.
func (e *Engine) ActivationState() domain.WindowActivationState {
	e.mu.Lock()
	defer e.mu.Unlock()
	return e.activationState
}

// Enable transitions the engine from Disabled to WaitingForInactivity.
// It is a no-op if the engine is already enabled.
func (e *Engine) Enable() {
	e.resetInactivityThreshold()
	if e.sm.Current() == domain.StateDisabled {
		e.send(domain.EventEnable)
	}
}

// Disable cancels any in-flight animation cycle and transitions the engine
// back to the Disabled state. It is a no-op if the engine is already
// disabled.
func (e *Engine) Disable() {
	e.resetInactivityThreshold()
	e.cancelActiveCycle()
	if e.sm.Current() != domain.StateDisabled {
		e.send(domain.EventDisable)
	}
	e.store.SetProgress(0, "status.Disabled")
}

func (e *Engine) run(ctx context.Context) {
	defer logging.RecoverPanic("runtime.Engine.run")

	// Logger is resolved per call so runtime toggles of logging.SetEnabled
	// are reflected immediately instead of being pinned to the state at
	// goroutine start.
	logging.Get().Info("engine started")
	defer logging.Get().Info("engine stopped")

	ticker := time.NewTicker(idlePollInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			e.tick(ctx)
		}
	}
}

func (e *Engine) tick(ctx context.Context) {
	defer logging.RecoverPanic("runtime.Engine.tick", "state", e.sm.Current())

	settings := e.store.Settings().Effective()
	if e.shouldObserveWindowActivation(settings) {
		e.observeWindowActivation(settings)
	}
	currentState := e.sm.Current()

	switch currentState {
	case domain.StateDisabled:
		e.resetInactivityThreshold()
		e.store.SetProgress(0, "status.Disabled")

	case domain.StateWaitingForInactivity:
		e.handleWaitingForInactivity(ctx, currentState, settings)

	case domain.StateWaitingForInterval, domain.StateAnimating:
		e.resetInactivityThreshold()

	case domain.StateError:
		e.resetInactivityThreshold()
		e.send(domain.EventRecover)
	}
}

// send routes a state machine event through the engine, logging any
// transition that is not valid in the current state but never treating it
// as a hard error. Returns true when the transition actually happened.
func (e *Engine) send(event domain.Event) bool {
	if _, err := e.sm.Send(event); err != nil {
		logging.Get().Debug("state transition ignored", "event", event, "state", e.sm.Current(), "error", err)
		return false
	}

	return true
}

func (e *Engine) pickValue(nr domain.NumericRange) int {
	if e.pickValueFn == nil {
		return pickValue(nr)
	}

	return e.pickValueFn(nr)
}

func pickValue(nr domain.NumericRange) int {
	if nr.Mode == domain.RangeModeRandom {
		if nr.MaxVal <= nr.MinVal {
			return nr.MinVal
		}
		return nr.MinVal + rand.IntN(nr.MaxVal-nr.MinVal+1)
	}
	return nr.Value
}

func clampProgress(v float64) float64 {
	if v < 0 {
		return 0
	}
	if v > 1 {
		return 1
	}
	return v
}
