package runtime

import (
	"afk-hero/internal/domain"
	"afk-hero/internal/state"
	"context"
	"errors"
	"sync"
	"testing"
	"time"
)

type fakeActivityMonitor struct {
	mu   sync.Mutex
	idle time.Duration
	err  error
}

func (f *fakeActivityMonitor) IdleTime() (time.Duration, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.idle, f.err
}

func (f *fakeActivityMonitor) SetIdle(idle time.Duration) {
	f.mu.Lock()
	f.idle = idle
	f.mu.Unlock()
}

type fakePointerController struct {
	mu        sync.Mutex
	position  domain.Point
	moveDelay time.Duration
	moveErr   error
	moveCount int
	started   chan struct{}
	once      sync.Once
	reportFn  func(domain.Point) domain.Point
	moveFn    func(domain.Point, domain.Point) domain.Point
}

func (f *fakePointerController) GetPosition() (domain.Point, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	pos := f.position
	if f.reportFn != nil {
		pos = f.reportFn(pos)
	}
	return pos, nil
}

func (f *fakePointerController) MoveBy(delta domain.Point, extraInfo uintptr) error {
	f.once.Do(func() {
		if f.started != nil {
			close(f.started)
		}
	})

	if f.moveDelay > 0 {
		time.Sleep(f.moveDelay)
	}

	f.mu.Lock()
	defer f.mu.Unlock()
	f.moveCount++
	if f.moveFn != nil {
		f.position = f.moveFn(f.position, delta)
	} else {
		f.position = addPoints(f.position, delta)
	}
	return f.moveErr
}

func (f *fakePointerController) Count() int {
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.moveCount
}

type fakeScreenProvider struct {
	bounds domain.Rect
	err    error
}

func (f *fakeScreenProvider) WorkArea(at domain.Point) (domain.Rect, error) {
	return f.bounds, f.err
}

func (f *fakeScreenProvider) PrimaryWorkArea() (domain.Rect, error) {
	return f.bounds, f.err
}

type fakeWindowManager struct {
	windows             []domain.WindowInfo
	autoInfoByID        map[uintptr]domain.WindowInfo
	windowInfoByID      map[uintptr]domain.WindowInfo
	autoTarget          domain.WindowInfo
	autoTargetErr       error
	activateErr         error
	activateCalls       int
	findFingerprintHits int
	resolveAutoHits     int
	autoInfoCalls       int
	windowInfoCalls     int
	lastActivated       uintptr
	lastFingerprint     domain.WindowFingerprint
	lastAutoBlacklist   []string
	lastListBlacklist   []string
	statuses            map[uintptr]domain.WindowActivationStatus
}

func (f *fakeWindowManager) ListWindows(autoProcessBlacklist []string) ([]domain.WindowInfo, error) {
	f.lastListBlacklist = append([]string(nil), autoProcessBlacklist...)
	return f.windows, nil
}

func (f *fakeWindowManager) ActivateWindow(id uintptr) error {
	f.activateCalls++
	f.lastActivated = id
	return f.activateErr
}

func (f *fakeWindowManager) FindWindowByFingerprint(fingerprint domain.WindowFingerprint) (domain.WindowInfo, error) {
	f.findFingerprintHits++
	f.lastFingerprint = fingerprint
	for _, window := range f.windows {
		if fingerprint.Matches(window) {
			return window, nil
		}
	}
	return domain.WindowInfo{}, errors.New("window not found")
}

func (f *fakeWindowManager) ResolveAutoTarget(autoProcessBlacklist []string) (domain.WindowInfo, error) {
	f.resolveAutoHits++
	f.lastAutoBlacklist = append([]string(nil), autoProcessBlacklist...)
	return f.autoTarget, f.autoTargetErr
}

func (f *fakeWindowManager) AutoTargetInfo(id uintptr, autoProcessBlacklist []string) (domain.WindowInfo, error) {
	f.autoInfoCalls++
	f.lastAutoBlacklist = append([]string(nil), autoProcessBlacklist...)
	if f.autoInfoByID != nil {
		if info, ok := f.autoInfoByID[id]; ok {
			return info, nil
		}
		return domain.WindowInfo{}, errors.New("window not found")
	}
	if f.autoTarget.ID == id {
		return f.autoTarget, nil
	}
	return domain.WindowInfo{}, errors.New("window not found")
}

func (f *fakeWindowManager) WindowInfo(id uintptr) (domain.WindowInfo, error) {
	f.windowInfoCalls++
	if f.windowInfoByID != nil {
		if info, ok := f.windowInfoByID[id]; ok {
			return info, nil
		}
		return domain.WindowInfo{}, errors.New("window not found")
	}
	for _, window := range f.windows {
		if window.ID == id {
			return window, nil
		}
	}
	return domain.WindowInfo{}, errors.New("window not found")
}

func (f *fakeWindowManager) WindowStatus(id uintptr) (domain.WindowActivationStatus, error) {
	if f.statuses != nil {
		if status, ok := f.statuses[id]; ok {
			return status, nil
		}
	}
	if id == 0 {
		return domain.WindowActivationStatus{Exists: false}, nil
	}
	return domain.WindowActivationStatus{Exists: true, Foreground: true}, nil
}

func newEngineForTest(
	t *testing.T,
	initialState domain.State,
	settings domain.Settings,
	activity *fakeActivityMonitor,
	pointer *fakePointerController,
	screen *fakeScreenProvider,
	windowMgr *fakeWindowManager,
) (*Engine, *state.Store, *domain.StateMachine) {
	t.Helper()

	store := state.NewStore(settings, nil)
	sm := domain.NewStateMachine(initialState, func(from, to domain.State, event domain.Event) {
		store.SetState(to)
	})

	engine := NewEngine(store, sm, activity, pointer, screen, windowMgr)
	return engine, store, sm
}

func TestRunAnimation_MoveFailureTransitionsToError(t *testing.T) {
	settings := domain.DefaultSettings()
	settings.Shape = domain.ShapeSegment
	settings.Direction = domain.DirectionRight
	settings.Distance = domain.NumericRange{Mode: domain.RangeModeFixed, Value: 10}
	settings.Speed = domain.NumericRange{Mode: domain.RangeModeFixed, Value: 1}

	engine, store, sm := newEngineForTest(
		t,
		domain.StateAnimating,
		settings,
		&fakeActivityMonitor{idle: time.Hour},
		&fakePointerController{position: domain.Point{X: 50, Y: 50}, moveErr: errors.New("move failed")},
		&fakeScreenProvider{bounds: domain.Rect{Left: 0, Top: 0, Right: 200, Bottom: 200}},
		&fakeWindowManager{},
	)

	engine.runAnimation(context.Background(), 1, settings)

	if got := sm.Current(); got != domain.StateError {
		t.Fatalf("expected state Error, got %s", got)
	}

	progress, statusKey := store.Progress()
	if progress != 0 {
		t.Fatalf("expected progress reset to 0, got %f", progress)
	}
	if statusKey != "status.Error" {
		t.Fatalf("expected status.Error, got %s", statusKey)
	}
}

func TestHandleWaitingForInactivitySamplesThresholdOncePerWaitingState(t *testing.T) {
	settings := domain.DefaultSettings()
	settings.Inactivity = domain.NumericRange{Mode: domain.RangeModeRandom, MinVal: 1, MaxVal: 3}

	engine, _, sm := newEngineForTest(
		t,
		domain.StateWaitingForInactivity,
		settings,
		&fakeActivityMonitor{idle: 2 * time.Second},
		&fakePointerController{position: domain.Point{X: 50, Y: 50}},
		&fakeScreenProvider{bounds: domain.Rect{Left: 0, Top: 0, Right: 200, Bottom: 200}},
		&fakeWindowManager{},
	)

	var pickCalls int
	engine.pickValueFn = func(nr domain.NumericRange) int {
		pickCalls++
		if pickCalls == 1 {
			return 3
		}
		return 1
	}

	engine.handleWaitingForInactivity(context.Background(), domain.StateWaitingForInactivity, settings)
	engine.handleWaitingForInactivity(context.Background(), domain.StateWaitingForInactivity, settings)

	if got := sm.Current(); got != domain.StateWaitingForInactivity {
		t.Fatalf("expected to stay in WaitingForInactivity, got %s", got)
	}
	if pickCalls != 1 {
		t.Fatalf("expected inactivity threshold to be sampled once, got %d", pickCalls)
	}
}

func TestHandleWaitingForInactivityResamplesThresholdWhenSettingsChange(t *testing.T) {
	settings := domain.DefaultSettings()
	settings.Shape = domain.ShapeSegment
	settings.Direction = domain.DirectionRight
	settings.Distance = domain.NumericRange{Mode: domain.RangeModeFixed, Value: 10}
	settings.Speed = domain.NumericRange{Mode: domain.RangeModeFixed, Value: 1}
	settings.Inactivity = domain.NumericRange{Mode: domain.RangeModeRandom, MinVal: 1, MaxVal: 3}

	pointer := &fakePointerController{
		position:  domain.Point{X: 50, Y: 50},
		moveDelay: 100 * time.Millisecond,
	}

	engine, _, sm := newEngineForTest(
		t,
		domain.StateWaitingForInactivity,
		settings,
		&fakeActivityMonitor{idle: 2 * time.Second},
		pointer,
		&fakeScreenProvider{bounds: domain.Rect{Left: 0, Top: 0, Right: 200, Bottom: 200}},
		&fakeWindowManager{},
	)

	var pickCalls int
	engine.pickValueFn = func(nr domain.NumericRange) int {
		pickCalls++
		if pickCalls == 1 {
			return 3
		}
		return 1
	}

	parentCtx, cancel := context.WithCancel(context.Background())
	defer cancel()

	engine.handleWaitingForInactivity(parentCtx, domain.StateWaitingForInactivity, settings)

	updatedSettings := settings
	updatedSettings.Inactivity = domain.NumericRange{Mode: domain.RangeModeRandom, MinVal: 1, MaxVal: 2}
	engine.handleWaitingForInactivity(parentCtx, domain.StateWaitingForInactivity, updatedSettings)

	if pickCalls != 2 {
		t.Fatalf("expected inactivity threshold to be re-sampled after settings change, got %d calls", pickCalls)
	}

	if got := sm.Current(); got == domain.StateWaitingForInactivity {
		t.Fatalf("expected updated inactivity threshold to apply immediately, state remained %s", got)
	}
}

func TestDisableCancelsAnimation(t *testing.T) {
	settings := domain.DefaultSettings()
	settings.Shape = domain.ShapeSegment
	settings.Direction = domain.DirectionRight
	settings.Distance = domain.NumericRange{Mode: domain.RangeModeFixed, Value: 40}
	settings.Speed = domain.NumericRange{Mode: domain.RangeModeFixed, Value: 1}

	pointer := &fakePointerController{
		position:  domain.Point{X: 50, Y: 50},
		moveDelay: 20 * time.Millisecond,
		started:   make(chan struct{}),
	}

	engine, _, sm := newEngineForTest(
		t,
		domain.StateAnimating,
		settings,
		&fakeActivityMonitor{idle: time.Hour},
		pointer,
		&fakeScreenProvider{bounds: domain.Rect{Left: 0, Top: 0, Right: 200, Bottom: 200}},
		&fakeWindowManager{},
	)

	parentCtx, cancel := context.WithCancel(context.Background())
	defer cancel()

	cycleCtx, cycleID := engine.startCycle(parentCtx)
	done := make(chan struct{})
	go func() {
		engine.runAnimation(cycleCtx, cycleID, settings)
		close(done)
	}()

	select {
	case <-pointer.started:
	case <-time.After(time.Second):
		t.Fatal("animation did not start")
	}

	engine.Disable()

	select {
	case <-done:
	case <-time.After(time.Second):
		t.Fatal("animation was not canceled by Disable")
	}

	if got := sm.Current(); got != domain.StateDisabled {
		t.Fatalf("expected Disabled after cancel, got %s", got)
	}

	moveCount := pointer.Count()
	time.Sleep(100 * time.Millisecond)
	if pointer.Count() != moveCount {
		t.Fatalf("expected cursor movement to stop after Disable, got %d -> %d", moveCount, pointer.Count())
	}
}

func TestDisableCancelsWaitInterval(t *testing.T) {
	settings := domain.DefaultSettings()
	engine, _, sm := newEngineForTest(
		t,
		domain.StateWaitingForInterval,
		settings,
		&fakeActivityMonitor{idle: time.Hour},
		&fakePointerController{position: domain.Point{X: 50, Y: 50}},
		&fakeScreenProvider{bounds: domain.Rect{Left: 0, Top: 0, Right: 200, Bottom: 200}},
		&fakeWindowManager{},
	)

	parentCtx, cancel := context.WithCancel(context.Background())
	defer cancel()

	cycleCtx, cycleID := engine.startCycle(parentCtx)
	done := make(chan struct{})
	go func() {
		engine.waitInterval(cycleCtx, cycleID, 2*time.Second, settings)
		close(done)
	}()

	time.Sleep(100 * time.Millisecond)
	engine.Disable()

	select {
	case <-done:
	case <-time.After(time.Second):
		t.Fatal("interval wait was not canceled by Disable")
	}

	if got := sm.Current(); got != domain.StateDisabled {
		t.Fatalf("expected Disabled after interval cancel, got %s", got)
	}
}

func TestResolveActivationWindowUsesResolvedAutoTarget(t *testing.T) {
	settings := domain.DefaultSettings()
	settings.ActivationEnabled = true
	settings.ActivationMode = domain.ActivationModeAuto
	settings.ActivationAutoProcessBlacklist = []string{"chrome.exe", "firefox.exe"}

	windowMgr := &fakeWindowManager{
		autoTarget: domain.WindowInfo{
			ID:          1234,
			Title:       "Game",
			ProcessName: "game.exe",
			ClassName:   "UnrealWindow",
		},
	}

	engine, _, _ := newEngineForTest(
		t,
		domain.StateAnimating,
		settings,
		&fakeActivityMonitor{idle: time.Hour},
		&fakePointerController{position: domain.Point{X: 50, Y: 50}},
		&fakeScreenProvider{bounds: domain.Rect{Left: 0, Top: 0, Right: 200, Bottom: 200}},
		windowMgr,
	)

	target, err := engine.resolveActivationWindow(settings)
	if err != nil {
		t.Fatalf("expected auto target resolution to succeed, got %v", err)
	}
	if len(windowMgr.lastAutoBlacklist) != 2 || windowMgr.lastAutoBlacklist[0] != "chrome.exe" || windowMgr.lastAutoBlacklist[1] != "firefox.exe" {
		t.Fatalf("expected auto blacklist to be forwarded, got %#v", windowMgr.lastAutoBlacklist)
	}
	if target.ID != 1234 {
		t.Fatalf("expected initial auto target 1234, got %d", target.ID)
	}

	windowMgr.autoTarget = domain.WindowInfo{
		ID:          5678,
		Title:       "Game",
		ProcessName: "game.exe",
		ClassName:   "UnrealWindow",
	}

	target, err = engine.resolveActivationWindow(settings)
	if err != nil {
		t.Fatalf("expected current auto candidate resolution to succeed, got %v", err)
	}
	if target.ID != 5678 {
		t.Fatalf("expected updated auto target 5678, got %d", target.ID)
	}
}

func TestResolveActivationWindowReusesAutoHandleCache(t *testing.T) {
	settings := domain.DefaultSettings()
	settings.ActivationEnabled = true
	settings.ActivationMode = domain.ActivationModeAuto
	settings.ActivationAutoProcessBlacklist = []string{"chrome.exe"}

	window := domain.WindowInfo{
		ID:          1234,
		Title:       "Game",
		ProcessName: "game.exe",
		ClassName:   "UnrealWindow",
	}
	windowMgr := &fakeWindowManager{
		autoTarget:   window,
		autoInfoByID: map[uintptr]domain.WindowInfo{window.ID: window},
	}

	engine, _, _ := newEngineForTest(
		t,
		domain.StateAnimating,
		settings,
		&fakeActivityMonitor{idle: time.Hour},
		&fakePointerController{position: domain.Point{X: 50, Y: 50}},
		&fakeScreenProvider{bounds: domain.Rect{Left: 0, Top: 0, Right: 200, Bottom: 200}},
		windowMgr,
	)

	first, err := engine.resolveActivationWindow(settings)
	if err != nil {
		t.Fatalf("first auto target resolution failed: %v", err)
	}
	second, err := engine.resolveActivationWindow(settings)
	if err != nil {
		t.Fatalf("second auto target resolution failed: %v", err)
	}

	if first.ID != window.ID || second.ID != window.ID {
		t.Fatalf("expected cached auto target %d, got %d then %d", window.ID, first.ID, second.ID)
	}
	if windowMgr.resolveAutoHits != 1 {
		t.Fatalf("expected full auto resolver to run once, got %d", windowMgr.resolveAutoHits)
	}
	if windowMgr.autoInfoCalls != 1 {
		t.Fatalf("expected cached auto target validation to run once, got %d", windowMgr.autoInfoCalls)
	}
}

func TestResolveActivationWindowReResolvesAutoTargetWhenCachedHandleInvalidates(t *testing.T) {
	settings := domain.DefaultSettings()
	settings.ActivationEnabled = true
	settings.ActivationMode = domain.ActivationModeAuto

	firstWindow := domain.WindowInfo{
		ID:          1234,
		Title:       "Game",
		ProcessName: "game.exe",
		ClassName:   "UnrealWindow",
	}
	secondWindow := domain.WindowInfo{
		ID:          5678,
		Title:       "Game 2",
		ProcessName: "game2.exe",
		ClassName:   "CryENGINE",
	}
	windowMgr := &fakeWindowManager{
		autoTarget:   firstWindow,
		autoInfoByID: map[uintptr]domain.WindowInfo{firstWindow.ID: firstWindow},
	}

	engine, _, _ := newEngineForTest(
		t,
		domain.StateAnimating,
		settings,
		&fakeActivityMonitor{idle: time.Hour},
		&fakePointerController{position: domain.Point{X: 50, Y: 50}},
		&fakeScreenProvider{bounds: domain.Rect{Left: 0, Top: 0, Right: 200, Bottom: 200}},
		windowMgr,
	)

	target, err := engine.resolveActivationWindow(settings)
	if err != nil {
		t.Fatalf("initial auto target resolution failed: %v", err)
	}
	if target.ID != firstWindow.ID {
		t.Fatalf("expected initial auto target %d, got %d", firstWindow.ID, target.ID)
	}

	windowMgr.autoTarget = secondWindow
	windowMgr.autoInfoByID = map[uintptr]domain.WindowInfo{secondWindow.ID: secondWindow}

	target, err = engine.resolveActivationWindow(settings)
	if err != nil {
		t.Fatalf("auto target re-resolution failed: %v", err)
	}
	if target.ID != secondWindow.ID {
		t.Fatalf("expected re-resolved auto target %d, got %d", secondWindow.ID, target.ID)
	}
	if windowMgr.resolveAutoHits != 2 {
		t.Fatalf("expected full auto resolver to rerun after cache invalidation, got %d", windowMgr.resolveAutoHits)
	}
	if windowMgr.autoInfoCalls != 1 {
		t.Fatalf("expected one cached auto target validation before invalidation, got %d", windowMgr.autoInfoCalls)
	}
}

func TestResolveActivationWindowReturnsAutoResolverError(t *testing.T) {
	settings := domain.DefaultSettings()
	settings.ActivationEnabled = true
	settings.ActivationMode = domain.ActivationModeAuto

	windowMgr := &fakeWindowManager{
		autoTargetErr: errors.New("target window not found"),
	}

	engine, _, _ := newEngineForTest(
		t,
		domain.StateAnimating,
		settings,
		&fakeActivityMonitor{idle: time.Hour},
		&fakePointerController{position: domain.Point{X: 50, Y: 50}},
		&fakeScreenProvider{bounds: domain.Rect{Left: 0, Top: 0, Right: 200, Bottom: 200}},
		windowMgr,
	)

	if _, err := engine.resolveActivationWindow(settings); err == nil {
		t.Fatal("expected auto target resolution to surface resolver failure")
	}
}

func TestObserveWindowActivationActivatesManualTargetAfterTimeout(t *testing.T) {
	settings := domain.DefaultSettings()
	settings.ActivationEnabled = true
	settings.ActivationMode = domain.ActivationModeManual
	settings.ActivationTimeout = 1
	settings.TargetProcessName = "game.exe"
	settings.TargetWindowClass = "UnrealWindow"
	settings.TargetWindowTitle = "Game"

	window := domain.WindowInfo{
		ID:          4321,
		Title:       "Game",
		ProcessName: "game.exe",
		ClassName:   "UnrealWindow",
	}
	windowMgr := &fakeWindowManager{
		windows: []domain.WindowInfo{window},
		statuses: map[uintptr]domain.WindowActivationStatus{
			window.ID: {Exists: true, Minimized: true, Foreground: false},
		},
	}

	engine, _, _ := newEngineForTest(
		t,
		domain.StateWaitingForInactivity,
		settings,
		&fakeActivityMonitor{idle: time.Hour},
		&fakePointerController{position: domain.Point{X: 50, Y: 50}},
		&fakeScreenProvider{bounds: domain.Rect{Left: 0, Top: 0, Right: 200, Bottom: 200}},
		windowMgr,
	)

	engine.mu.Lock()
	engine.activationInactive = time.Now().Add(-2 * time.Second)
	engine.mu.Unlock()

	engine.observeWindowActivation(settings)

	if windowMgr.activateCalls != 1 {
		t.Fatalf("expected one activation attempt, got %d", windowMgr.activateCalls)
	}
	if windowMgr.lastActivated != window.ID {
		t.Fatalf("expected manual activation for %d, got %d", window.ID, windowMgr.lastActivated)
	}
	if !windowMgr.lastFingerprint.Matches(window) {
		t.Fatalf("expected manual lookup by process/class fingerprint, got %+v", windowMgr.lastFingerprint)
	}
}

func TestObserveWindowActivationPublishesProgressWhileWindowIsInactive(t *testing.T) {
	settings := domain.DefaultSettings()
	settings.Enabled = true
	settings.ActivationEnabled = true
	settings.ActivationMode = domain.ActivationModeManual
	settings.ActivationTimeout = 10
	settings.TargetProcessName = "game.exe"
	settings.TargetWindowClass = "UnrealWindow"
	settings.TargetWindowTitle = "Game"

	window := domain.WindowInfo{
		ID:          4321,
		Title:       "Game",
		ProcessName: "game.exe",
		ClassName:   "UnrealWindow",
	}
	windowMgr := &fakeWindowManager{
		windows: []domain.WindowInfo{window},
		statuses: map[uintptr]domain.WindowActivationStatus{
			window.ID: {Exists: true, Minimized: true, Foreground: false},
		},
	}

	engine, _, _ := newEngineForTest(
		t,
		domain.StateWaitingForInactivity,
		settings,
		&fakeActivityMonitor{idle: time.Hour},
		&fakePointerController{position: domain.Point{X: 50, Y: 50}},
		&fakeScreenProvider{bounds: domain.Rect{Left: 0, Top: 0, Right: 200, Bottom: 200}},
		windowMgr,
	)

	engine.mu.Lock()
	engine.activationInactive = time.Now().Add(-5 * time.Second)
	engine.mu.Unlock()

	engine.observeWindowActivation(settings)

	state := engine.ActivationState()
	if !state.TargetFound {
		t.Fatal("expected activation state to report a resolved target")
	}
	if !state.Tracking {
		t.Fatal("expected activation state to report active timeout tracking")
	}
	if state.ProcessName != window.ProcessName || state.WindowTitle != window.Title {
		t.Fatalf("expected activation state to expose target metadata, got %+v", state)
	}
	if state.Progress < 0.49 || state.Progress > 0.51 {
		t.Fatalf("expected progress near 0.5, got %f", state.Progress)
	}
}

func TestObserveWindowActivationResetsProgressWhenWindowReturnsForeground(t *testing.T) {
	settings := domain.DefaultSettings()
	settings.Enabled = true
	settings.ActivationEnabled = true
	settings.ActivationMode = domain.ActivationModeManual
	settings.ActivationTimeout = 10
	settings.TargetProcessName = "game.exe"
	settings.TargetWindowClass = "UnrealWindow"

	window := domain.WindowInfo{
		ID:          4321,
		Title:       "Game",
		ProcessName: "game.exe",
		ClassName:   "UnrealWindow",
	}
	windowMgr := &fakeWindowManager{
		windows: []domain.WindowInfo{window},
		statuses: map[uintptr]domain.WindowActivationStatus{
			window.ID: {Exists: true, Minimized: false, Foreground: true},
		},
	}

	engine, _, _ := newEngineForTest(
		t,
		domain.StateWaitingForInactivity,
		settings,
		&fakeActivityMonitor{idle: time.Hour},
		&fakePointerController{position: domain.Point{X: 50, Y: 50}},
		&fakeScreenProvider{bounds: domain.Rect{Left: 0, Top: 0, Right: 200, Bottom: 200}},
		windowMgr,
	)

	engine.mu.Lock()
	engine.activationState = domain.WindowActivationState{
		TargetFound: true,
		Tracking:    true,
		Progress:    0.8,
	}
	engine.activationInactive = time.Now().Add(-8 * time.Second)
	engine.mu.Unlock()

	engine.observeWindowActivation(settings)

	state := engine.ActivationState()
	if !state.TargetFound {
		t.Fatal("expected activation state to keep resolved target information")
	}
	if state.Tracking {
		t.Fatal("expected tracking to stop once the window is foreground again")
	}
	if state.ProcessName != window.ProcessName || state.WindowTitle != window.Title {
		t.Fatalf("expected activation state to keep target metadata, got %+v", state)
	}
	if state.Progress != 0 {
		t.Fatalf("expected activation progress reset to 0, got %f", state.Progress)
	}
}

func TestResolveActivationWindowReusesManualHandleCache(t *testing.T) {
	settings := domain.DefaultSettings()
	settings.ActivationEnabled = true
	settings.ActivationMode = domain.ActivationModeManual
	settings.TargetProcessName = "game.exe"
	settings.TargetWindowClass = "UnrealWindow"
	settings.TargetWindowTitle = "Game"

	window := domain.WindowInfo{
		ID:          4321,
		Title:       "Game",
		ProcessName: "game.exe",
		ClassName:   "UnrealWindow",
	}
	windowMgr := &fakeWindowManager{
		windows:        []domain.WindowInfo{window},
		windowInfoByID: map[uintptr]domain.WindowInfo{window.ID: window},
	}

	engine, _, _ := newEngineForTest(
		t,
		domain.StateWaitingForInactivity,
		settings,
		&fakeActivityMonitor{idle: time.Hour},
		&fakePointerController{position: domain.Point{X: 50, Y: 50}},
		&fakeScreenProvider{bounds: domain.Rect{Left: 0, Top: 0, Right: 200, Bottom: 200}},
		windowMgr,
	)

	first, err := engine.resolveActivationWindow(settings)
	if err != nil {
		t.Fatalf("first manual target resolution failed: %v", err)
	}
	second, err := engine.resolveActivationWindow(settings)
	if err != nil {
		t.Fatalf("second manual target resolution failed: %v", err)
	}

	if first.ID != window.ID || second.ID != window.ID {
		t.Fatalf("expected cached manual target %d, got %d then %d", window.ID, first.ID, second.ID)
	}
	if windowMgr.findFingerprintHits != 1 {
		t.Fatalf("expected fingerprint lookup to run once, got %d", windowMgr.findFingerprintHits)
	}
	if windowMgr.windowInfoCalls != 1 {
		t.Fatalf("expected cached window info lookup to run once, got %d", windowMgr.windowInfoCalls)
	}
}

func TestResolveActivationWindowReResolvesManualTargetWhenCachedHandleInvalidates(t *testing.T) {
	settings := domain.DefaultSettings()
	settings.ActivationEnabled = true
	settings.ActivationMode = domain.ActivationModeManual
	settings.TargetProcessName = "game.exe"
	settings.TargetWindowClass = "UnrealWindow"
	settings.TargetWindowTitle = "Game"

	firstWindow := domain.WindowInfo{
		ID:          4321,
		Title:       "Game",
		ProcessName: "game.exe",
		ClassName:   "UnrealWindow",
	}
	secondWindow := domain.WindowInfo{
		ID:          6789,
		Title:       "Game",
		ProcessName: "game.exe",
		ClassName:   "UnrealWindow",
	}
	windowMgr := &fakeWindowManager{
		windows:        []domain.WindowInfo{firstWindow},
		windowInfoByID: map[uintptr]domain.WindowInfo{firstWindow.ID: firstWindow},
	}

	engine, _, _ := newEngineForTest(
		t,
		domain.StateWaitingForInactivity,
		settings,
		&fakeActivityMonitor{idle: time.Hour},
		&fakePointerController{position: domain.Point{X: 50, Y: 50}},
		&fakeScreenProvider{bounds: domain.Rect{Left: 0, Top: 0, Right: 200, Bottom: 200}},
		windowMgr,
	)

	target, err := engine.resolveActivationWindow(settings)
	if err != nil {
		t.Fatalf("initial manual target resolution failed: %v", err)
	}
	if target.ID != firstWindow.ID {
		t.Fatalf("expected initial manual target %d, got %d", firstWindow.ID, target.ID)
	}

	windowMgr.windows = []domain.WindowInfo{secondWindow}
	windowMgr.windowInfoByID = map[uintptr]domain.WindowInfo{secondWindow.ID: secondWindow}

	target, err = engine.resolveActivationWindow(settings)
	if err != nil {
		t.Fatalf("manual target re-resolution failed: %v", err)
	}
	if target.ID != secondWindow.ID {
		t.Fatalf("expected re-resolved manual target %d, got %d", secondWindow.ID, target.ID)
	}
	if windowMgr.findFingerprintHits != 2 {
		t.Fatalf("expected fingerprint lookup to rerun after cache invalidation, got %d", windowMgr.findFingerprintHits)
	}
	if windowMgr.windowInfoCalls != 1 {
		t.Fatalf("expected one cached window info lookup before invalidation, got %d", windowMgr.windowInfoCalls)
	}
}

func TestTickAppliesConfiguredActivationTickInterval(t *testing.T) {
	settings := domain.DefaultSettings()
	settings.Enabled = true
	settings.ActivationEnabled = true
	settings.ActivationMode = domain.ActivationModeAuto
	settings.ActivationTickIntervalMS = 500

	window := domain.WindowInfo{
		ID:          1234,
		Title:       "Game",
		ProcessName: "game.exe",
		ClassName:   "UnrealWindow",
	}
	windowMgr := &fakeWindowManager{
		autoTarget:   window,
		autoInfoByID: map[uintptr]domain.WindowInfo{window.ID: window},
		statuses: map[uintptr]domain.WindowActivationStatus{
			window.ID: {Exists: true, Minimized: false, Foreground: true},
		},
	}

	engine, _, _ := newEngineForTest(
		t,
		domain.StateWaitingForInactivity,
		settings,
		&fakeActivityMonitor{idle: 0},
		&fakePointerController{position: domain.Point{X: 50, Y: 50}},
		&fakeScreenProvider{bounds: domain.Rect{Left: 0, Top: 0, Right: 200, Bottom: 200}},
		windowMgr,
	)

	engine.tick(context.Background())
	engine.tick(context.Background())

	if windowMgr.resolveAutoHits != 1 {
		t.Fatalf("expected activation polling to skip immediate second tick, got %d full auto resolves", windowMgr.resolveAutoHits)
	}
	if windowMgr.autoInfoCalls != 0 {
		t.Fatalf("expected no cached auto validation before interval elapsed, got %d", windowMgr.autoInfoCalls)
	}

	engine.mu.Lock()
	engine.activationLastTick = time.Now().Add(-600 * time.Millisecond)
	engine.mu.Unlock()

	engine.tick(context.Background())

	if windowMgr.resolveAutoHits != 1 {
		t.Fatalf("expected cached auto target validation after interval, got %d full auto resolves", windowMgr.resolveAutoHits)
	}
	if windowMgr.autoInfoCalls != 1 {
		t.Fatalf("expected one cached auto validation after interval, got %d", windowMgr.autoInfoCalls)
	}
}

func TestRunAnimationDoesNotTreatInjectedInputAsUserInput(t *testing.T) {
	settings := domain.DefaultSettings()
	settings.Shape = domain.ShapeSegment
	settings.Direction = domain.DirectionRight
	settings.Distance = domain.NumericRange{Mode: domain.RangeModeFixed, Value: 20}
	settings.Speed = domain.NumericRange{Mode: domain.RangeModeFixed, Value: 1}
	settings.Interval = domain.NumericRange{Mode: domain.RangeModeFixed, Value: 1}

	pointer := &fakePointerController{position: domain.Point{X: 50, Y: 50}}
	engine, _, sm := newEngineForTest(
		t,
		domain.StateAnimating,
		settings,
		&fakeActivityMonitor{idle: 0},
		pointer,
		&fakeScreenProvider{bounds: domain.Rect{Left: 0, Top: 0, Right: 200, Bottom: 200}},
		&fakeWindowManager{},
	)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	engine.runAnimation(ctx, 1, settings)

	if got := sm.Current(); got != domain.StateWaitingForInterval {
		t.Fatalf("expected animation to complete instead of self-interrupting, got %s", got)
	}

	if pointer.Count() < 2 {
		t.Fatalf("expected multiple cursor moves, got %d", pointer.Count())
	}
}

func TestRunAnimationDoesNotSelfInterruptWhenCursorPositionIsLocked(t *testing.T) {
	settings := domain.DefaultSettings()
	settings.Shape = domain.ShapeSegment
	settings.Direction = domain.DirectionRight
	settings.Distance = domain.NumericRange{Mode: domain.RangeModeFixed, Value: 20}
	settings.Speed = domain.NumericRange{Mode: domain.RangeModeFixed, Value: 1}

	pointer := &fakePointerController{
		position: domain.Point{X: 50, Y: 50},
		moveFn: func(current, delta domain.Point) domain.Point {
			// Simulate a first-person game that keeps the visible cursor pinned.
			return current
		},
	}

	engine, _, sm := newEngineForTest(
		t,
		domain.StateAnimating,
		settings,
		&fakeActivityMonitor{idle: time.Hour},
		pointer,
		&fakeScreenProvider{bounds: domain.Rect{Left: 0, Top: 0, Right: 200, Bottom: 200}},
		&fakeWindowManager{},
	)

	engine.runAnimation(context.Background(), 1, settings)

	if got := sm.Current(); got != domain.StateWaitingForInterval {
		t.Fatalf("expected relative animation to complete with locked cursor, got %s", got)
	}
	if pointer.Count() < 2 {
		t.Fatalf("expected multiple relative cursor moves, got %d", pointer.Count())
	}
}

func TestRunAnimationDoesNotFalseInterruptAtScreenEdges(t *testing.T) {
	settings := domain.DefaultSettings()
	settings.Shape = domain.ShapeSegment
	settings.Direction = domain.DirectionRight
	settings.Distance = domain.NumericRange{Mode: domain.RangeModeFixed, Value: 20}
	settings.Speed = domain.NumericRange{Mode: domain.RangeModeFixed, Value: 1}

	pointer := &fakePointerController{
		position: domain.Point{X: 0, Y: 0},
		reportFn: func(pos domain.Point) domain.Point {
			// Simulate edge-clamped observation that differs from the commanded point.
			if pos.X == 0 {
				pos.X = 1
			}
			if pos.Y == 0 {
				pos.Y = 1
			}
			return pos
		},
	}

	engine, _, sm := newEngineForTest(
		t,
		domain.StateAnimating,
		settings,
		&fakeActivityMonitor{idle: time.Hour},
		pointer,
		&fakeScreenProvider{bounds: domain.Rect{Left: 0, Top: 0, Right: 200, Bottom: 200}},
		&fakeWindowManager{},
	)

	engine.runAnimation(context.Background(), 1, settings)

	if got := sm.Current(); got != domain.StateWaitingForInterval {
		t.Fatalf("expected edge animation to complete instead of self-interrupting, got %s", got)
	}

	if pointer.Count() < 2 {
		t.Fatalf("expected multiple cursor moves on edge trajectory, got %d", pointer.Count())
	}
}

func TestRunAnimationPreservesClosedPathWithThresholdSensitiveRelativeMoves(t *testing.T) {
	settings := domain.DefaultSettings()
	settings.Shape = domain.ShapeRectangle
	settings.Direction = domain.DirectionCenter
	settings.Distance = domain.NumericRange{Mode: domain.RangeModeFixed, Value: 30}
	settings.Speed = domain.NumericRange{Mode: domain.RangeModeFixed, Value: 1}
	settings.Interval = domain.NumericRange{Mode: domain.RangeModeFixed, Value: 1}

	origin := domain.Point{X: 80, Y: 80}
	pointer := &fakePointerController{
		position: origin,
		moveFn: func(current, delta domain.Point) domain.Point {
			amplified := delta
			if abs(delta.X) > 1 || abs(delta.Y) > 1 {
				amplified.X *= 2
				amplified.Y *= 2
			}
			return addPoints(current, amplified)
		},
	}

	engine, _, sm := newEngineForTest(
		t,
		domain.StateAnimating,
		settings,
		&fakeActivityMonitor{idle: time.Hour},
		pointer,
		&fakeScreenProvider{bounds: domain.Rect{Left: 0, Top: 0, Right: 200, Bottom: 200}},
		&fakeWindowManager{},
	)

	ctx, cancel := context.WithCancel(context.Background())
	engine.runAnimation(ctx, 1, settings)
	cancel()

	if got := sm.Current(); got != domain.StateWaitingForInterval {
		t.Fatalf("expected animation to complete under threshold-sensitive relative moves, got %s", got)
	}

	finalPosition, err := pointer.GetPosition()
	if err != nil {
		t.Fatalf("get final position: %v", err)
	}
	if finalPosition != origin {
		t.Fatalf("expected closed relative path to return to origin, got %+v want %+v", finalPosition, origin)
	}
}

func abs(v int) int {
	if v < 0 {
		return -v
	}
	return v
}

func TestWaitIntervalTransitionsBackToWaitingForInactivityOnUserInput(t *testing.T) {
	settings := domain.DefaultSettings()
	activity := &fakeActivityMonitor{idle: 10 * time.Second}
	engine, store, sm := newEngineForTest(
		t,
		domain.StateWaitingForInterval,
		settings,
		activity,
		&fakePointerController{position: domain.Point{X: 50, Y: 50}},
		&fakeScreenProvider{bounds: domain.Rect{Left: 0, Top: 0, Right: 200, Bottom: 200}},
		&fakeWindowManager{},
	)

	parentCtx, cancel := context.WithCancel(context.Background())
	defer cancel()

	cycleCtx, cycleID := engine.startCycle(parentCtx)
	done := make(chan struct{})
	go func() {
		engine.waitInterval(cycleCtx, cycleID, 2*time.Second, settings)
		close(done)
	}()

	time.Sleep(100 * time.Millisecond)
	activity.SetIdle(10 * time.Millisecond)

	select {
	case <-done:
	case <-time.After(time.Second):
		t.Fatal("interval wait did not stop after user input")
	}

	if got := sm.Current(); got != domain.StateWaitingForInactivity {
		t.Fatalf("expected WaitingForInactivity after user input, got %s", got)
	}

	progress, statusKey := store.Progress()
	if progress != 0 {
		t.Fatalf("expected progress reset to 0, got %f", progress)
	}
	if statusKey != "status.WaitingForInactivity" {
		t.Fatalf("expected status.WaitingForInactivity, got %s", statusKey)
	}
}

func TestWaitIntervalStartsNextAnimationWithoutReenteringWaitingForInactivity(t *testing.T) {
	settings := domain.DefaultSettings()
	settings.Shape = domain.ShapeSegment
	settings.Direction = domain.DirectionRight
	settings.Distance = domain.NumericRange{Mode: domain.RangeModeFixed, Value: 20}
	settings.Speed = domain.NumericRange{Mode: domain.RangeModeFixed, Value: 1}

	pointer := &fakePointerController{
		position: domain.Point{X: 50, Y: 50},
		started:  make(chan struct{}),
	}
	engine, _, sm := newEngineForTest(
		t,
		domain.StateWaitingForInterval,
		settings,
		&fakeActivityMonitor{idle: 10 * time.Second},
		pointer,
		&fakeScreenProvider{bounds: domain.Rect{Left: 0, Top: 0, Right: 200, Bottom: 200}},
		&fakeWindowManager{},
	)

	parentCtx, cancel := context.WithCancel(context.Background())
	defer cancel()

	cycleCtx, cycleID := engine.startCycle(parentCtx)
	done := make(chan struct{})
	go func() {
		engine.waitInterval(cycleCtx, cycleID, 100*time.Millisecond, settings)
		close(done)
	}()

	select {
	case <-pointer.started:
	case <-time.After(2 * time.Second):
		t.Fatal("next animation did not start after interval elapsed")
	}

	select {
	case <-done:
	case <-time.After(time.Second):
		t.Fatal("interval wait did not return after launching next animation")
	}

	if got := sm.Current(); got != domain.StateAnimating {
		t.Fatalf("expected state to advance directly to Animating, got %s", got)
	}
}
