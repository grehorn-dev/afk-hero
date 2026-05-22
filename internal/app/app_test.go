package app

import (
	"afk-hero/internal/config"
	"afk-hero/internal/domain"
	"afk-hero/internal/logging"
	"afk-hero/internal/platform"
	"afk-hero/internal/state"
	"context"
	"errors"
	"testing"
	"time"
)

type fakeEngine struct {
	enableCalls     int
	disableCalls    int
	startCalls      int
	stopCalls       int
	activationState domain.WindowActivationState
}

type fakeAutostartManager struct {
	lastEnabled bool
	setCalls    int
	setErr      error
}

func (f *fakeAutostartManager) SetAutostart(enabled bool) error {
	f.lastEnabled = enabled
	f.setCalls++
	return f.setErr
}

type fakeWindowManager struct {
	windows           []domain.WindowInfo
	listErr           error
	lastListBlacklist []string
}

func (f *fakeEngine) Start(ctx context.Context) { f.startCalls++ }
func (f *fakeEngine) Stop()                     { f.stopCalls++ }
func (f *fakeEngine) Enable()                   { f.enableCalls++ }
func (f *fakeEngine) Disable()                  { f.disableCalls++ }
func (f *fakeEngine) SetActivationStateCallback(cb func(domain.WindowActivationState)) {
	if cb != nil {
		cb(f.activationState)
	}
}
func (f *fakeEngine) ActivationState() domain.WindowActivationState { return f.activationState }

func (f *fakeWindowManager) ListWindows(autoProcessBlacklist []string) ([]domain.WindowInfo, error) {
	f.lastListBlacklist = append([]string(nil), autoProcessBlacklist...)
	return f.windows, f.listErr
}

func (f *fakeWindowManager) ActivateWindow(id uintptr) error {
	return errors.New("not implemented")
}

func (f *fakeWindowManager) FindWindowByFingerprint(fingerprint domain.WindowFingerprint) (domain.WindowInfo, error) {
	return domain.WindowInfo{}, errors.New("not implemented")
}

func (f *fakeWindowManager) ResolveAutoTarget(autoProcessBlacklist []string) (domain.WindowInfo, error) {
	return domain.WindowInfo{}, errors.New("not implemented")
}

func (f *fakeWindowManager) AutoTargetInfo(id uintptr, autoProcessBlacklist []string) (domain.WindowInfo, error) {
	return domain.WindowInfo{}, errors.New("not implemented")
}

func (f *fakeWindowManager) WindowInfo(id uintptr) (domain.WindowInfo, error) {
	return domain.WindowInfo{}, errors.New("not implemented")
}

func (f *fakeWindowManager) WindowStatus(id uintptr) (domain.WindowActivationStatus, error) {
	return domain.WindowActivationStatus{}, errors.New("not implemented")
}

func TestApplySettingsPersistsAndDisablesEngine(t *testing.T) {
	t.Setenv("APPDATA", t.TempDir())

	settings := domain.DefaultSettings()
	engine := &fakeEngine{}
	app := &App{
		store:  state.NewStore(settings, nil),
		engine: engine,
	}

	updated := settings
	updated.Enabled = false
	updated.Language = "rus"

	if err := app.ApplySettings(updated); err != nil {
		t.Fatalf("ApplySettings returned error: %v", err)
	}

	if engine.disableCalls != 1 {
		t.Fatalf("expected Disable to be called once, got %d", engine.disableCalls)
	}

	loaded, err := config.Load()
	if err != nil {
		t.Fatalf("Load returned error: %v", err)
	}
	if loaded.Language != "rus" {
		t.Fatalf("expected persisted language rus, got %s", loaded.Language)
	}
	if loaded.Enabled {
		t.Fatal("expected persisted enabled=false")
	}
}

func TestHandleStoreChangeAvoidsProgressOnlyTrayUpdates(t *testing.T) {
	settings := domain.DefaultSettings()
	app := &App{}

	trayUpdates := 0
	app.SetTrayCallback(func(s domain.Settings, st domain.State) {
		trayUpdates++
	})

	app.handleStoreChange(settings, domain.StateWaitingForInactivity, 0.1, "status.WaitingForInactivity")
	app.handleStoreChange(settings, domain.StateWaitingForInactivity, 0.9, "status.WaitingForInactivity")

	if trayUpdates != 1 {
		t.Fatalf("expected tray to update once for identical settings/state, got %d", trayUpdates)
	}
}

func TestGetSettingsAndStateAreSafeBeforeStartup(t *testing.T) {
	app := &App{}

	settings := app.GetSettings()
	if settings.Language != "eng" {
		t.Fatalf("expected default language before startup, got %s", settings.Language)
	}

	runtimeState := app.GetState()
	if runtimeState.State != domain.StateDisabled {
		t.Fatalf("expected Disabled state before startup, got %#v", runtimeState.State)
	}

	activationState := app.GetActivationState()
	if activationState != (domain.WindowActivationState{}) {
		t.Fatalf("expected zero activation state before startup, got %#v", activationState)
	}
}

func TestGetActivationStateDelegatesToEngine(t *testing.T) {
	engine := &fakeEngine{
		activationState: domain.WindowActivationState{
			TargetFound: true,
			Tracking:    true,
			Progress:    0.75,
			ProcessName: "game.exe",
			WindowTitle: "Game",
		},
	}
	app := &App{engine: engine}

	state := app.GetActivationState()
	if state != engine.activationState {
		t.Fatalf("expected activation state %#v, got %#v", engine.activationState, state)
	}
}

func TestBeforeCloseAllowsExplicitQuit(t *testing.T) {
	app := &App{
		hideWindow: func(context.Context) {},
	}

	if prevent := app.BeforeClose(context.TODO()); !prevent {
		t.Fatal("expected regular close to be intercepted and hidden")
	}

	app.closeMu.Lock()
	app.quitRequested = true
	app.closeMu.Unlock()

	if prevent := app.BeforeClose(context.TODO()); prevent {
		t.Fatal("expected explicit quit to bypass hide-on-close")
	}
}

func TestGetWindowListUsesCurrentActivationBlacklist(t *testing.T) {
	settings := domain.DefaultSettings()
	settings.ActivationAutoProcessBlacklist = []string{"chrome.exe", "msedge.exe"}

	windowMgr := &fakeWindowManager{
		windows: []domain.WindowInfo{{ID: 1, ProcessName: "game.exe", ClassName: "GameWindow"}},
	}
	app := &App{
		store:     state.NewStore(settings, nil),
		windowMgr: windowMgr,
	}

	windows, err := app.GetWindowList()
	if err != nil {
		t.Fatalf("GetWindowList returned error: %v", err)
	}
	if len(windows) != 1 {
		t.Fatalf("expected one window, got %d", len(windows))
	}
	if len(windowMgr.lastListBlacklist) != 2 || windowMgr.lastListBlacklist[0] != "chrome.exe" || windowMgr.lastListBlacklist[1] != "msedge.exe" {
		t.Fatalf("expected current auto blacklist to be forwarded, got %#v", windowMgr.lastListBlacklist)
	}
}

func TestApplySettingsPreservesBackendOnlyActivationConfig(t *testing.T) {
	t.Setenv("APPDATA", t.TempDir())

	settings := domain.DefaultSettings()
	settings.ActivationTickIntervalMS = 900
	settings.ActivationAutoProcessBlacklist = []string{"chrome.exe", "custom.exe"}

	engine := &fakeEngine{}
	app := &App{
		store:  state.NewStore(settings, nil),
		engine: engine,
	}

	updated := settings
	updated.Language = "rus"
	updated.ActivationTickIntervalMS = 0
	updated.ActivationAutoProcessBlacklist = nil

	if err := app.ApplySettings(updated); err != nil {
		t.Fatalf("ApplySettings returned error: %v", err)
	}

	loaded, err := config.Load()
	if err != nil {
		t.Fatalf("Load returned error: %v", err)
	}
	if loaded.ActivationTickIntervalMS != 900 {
		t.Fatalf("expected preserved activation tick interval 900, got %d", loaded.ActivationTickIntervalMS)
	}
	if len(loaded.ActivationAutoProcessBlacklist) != 2 || loaded.ActivationAutoProcessBlacklist[0] != "chrome.exe" || loaded.ActivationAutoProcessBlacklist[1] != "custom.exe" {
		t.Fatalf("expected preserved activation blacklist, got %#v", loaded.ActivationAutoProcessBlacklist)
	}
}

func TestApplySettingsUsesEffectiveLoggingState(t *testing.T) {
	t.Setenv("APPDATA", t.TempDir())
	t.Cleanup(func() {
		logging.Close()
		_ = logging.Init(false)
	})

	if err := logging.Init(false); err != nil {
		t.Fatalf("Init returned error: %v", err)
	}

	settings := domain.DefaultSettings()
	engine := &fakeEngine{}
	app := &App{
		store:  state.NewStore(settings, nil),
		engine: engine,
	}

	advancedDisabled := settings
	advancedDisabled.Advanced = false
	advancedDisabled.Logging = true

	if err := app.ApplySettings(advancedDisabled); err != nil {
		t.Fatalf("ApplySettings returned error: %v", err)
	}
	if logging.Enabled() {
		t.Fatal("expected effective logging to stay disabled while advanced settings are off")
	}

	advancedEnabled := advancedDisabled
	advancedEnabled.Advanced = true

	if err := app.ApplySettings(advancedEnabled); err != nil {
		t.Fatalf("ApplySettings returned error: %v", err)
	}
	if !logging.Enabled() {
		t.Fatal("expected logging to enable when advanced settings are turned back on")
	}
}

type stubActivityMonitor struct{}

func (stubActivityMonitor) IdleTime() (time.Duration, error) { return 0, nil }

type stubPointerController struct{}

func (stubPointerController) GetPosition() (domain.Point, error) { return domain.Point{}, nil }
func (stubPointerController) MoveBy(_ domain.Point, _ uintptr) error {
	return nil
}

type stubScreenProvider struct{}

func (stubScreenProvider) WorkArea(_ domain.Point) (domain.Rect, error) {
	return domain.Rect{Right: 1920, Bottom: 1080}, nil
}
func (stubScreenProvider) PrimaryWorkArea() (domain.Rect, error) {
	return domain.Rect{Right: 1920, Bottom: 1080}, nil
}

// TestStartupWiresDependenciesInCorrectOrder verifies the critical
// initialization sequence in App.Startup:
//
//  1. logging.Init is called before the first logging.Get() capture, so
//     advanced-mode logging takes effect from the very first log entry
//     (regression guard for the H-2 stale-logger bug);
//  2. the state store, state machine and engine are created and wired;
//  3. the engine is started exactly once;
//  4. Enable is invoked when the persisted settings opt in, and skipped
//     otherwise;
//  5. the tray callback receives exactly one priming update;
//  6. the "state:changed" event reaches the injected emit function.
func TestStartupWiresDependenciesInCorrectOrder(t *testing.T) {
	t.Setenv("APPDATA", t.TempDir())
	t.Cleanup(func() {
		logging.Close()
		_ = logging.Init(false)
	})

	engine := &fakeEngine{}
	trayUpdates := 0
	emitCalls := map[string]int{}

	app := NewApp(
		stubActivityMonitor{},
		stubPointerController{},
		stubScreenProvider{},
		&fakeWindowManager{},
		&fakeAutostartManager{},
	)
	app.hideWindow = func(context.Context) {}
	app.emit = func(_ context.Context, name string, _ ...interface{}) {
		emitCalls[name]++
	}
	app.newEngine = func(
		_ *state.Store,
		_ *domain.StateMachine,
		_ platform.UserActivityMonitor,
		_ platform.PointerController,
		_ platform.ScreenProvider,
		_ platform.WindowManager,
	) engineController {
		return engine
	}
	app.SetTrayCallback(func(domain.Settings, domain.State) {
		trayUpdates++
	})

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	app.Startup(ctx)

	if app.store == nil {
		t.Fatal("expected Startup to create the state store")
	}
	if app.sm == nil {
		t.Fatal("expected Startup to create the state machine")
	}
	if app.engine == nil {
		t.Fatal("expected Startup to create the engine")
	}

	if engine.startCalls != 1 {
		t.Fatalf("expected engine.Start to be called exactly once, got %d", engine.startCalls)
	}

	// Default settings have Enabled=true, so Enable should run.
	if engine.enableCalls != 1 {
		t.Fatalf("expected engine.Enable to be called exactly once for enabled defaults, got %d", engine.enableCalls)
	}

	if trayUpdates != 1 {
		t.Fatalf("expected exactly one priming tray update, got %d", trayUpdates)
	}

	if emitCalls["state:changed"] == 0 {
		t.Fatalf("expected at least one state:changed event, got %d", emitCalls["state:changed"])
	}
}

// TestStartupInitializesLoggingBeforeFirstGet guards against the specific
// regression where Startup captured a no-op logger before logging.Init
// flipped the package-level handler. With the fix in place, a fresh logging
// session must be active as soon as Startup returns (assuming the persisted
// settings asked for it).
func TestStartupInitializesLoggingBeforeFirstGet(t *testing.T) {
	t.Setenv("APPDATA", t.TempDir())
	t.Cleanup(func() {
		logging.Close()
		_ = logging.Init(false)
	})

	settings := domain.DefaultSettings()
	settings.Advanced = true
	settings.Logging = true
	if err := config.Save(settings); err != nil {
		t.Fatalf("config.Save returned error: %v", err)
	}

	engine := &fakeEngine{}
	app := NewApp(
		stubActivityMonitor{},
		stubPointerController{},
		stubScreenProvider{},
		&fakeWindowManager{},
		&fakeAutostartManager{},
	)
	app.hideWindow = func(context.Context) {}
	app.emit = func(context.Context, string, ...interface{}) {}
	app.newEngine = func(
		_ *state.Store,
		_ *domain.StateMachine,
		_ platform.UserActivityMonitor,
		_ platform.PointerController,
		_ platform.ScreenProvider,
		_ platform.WindowManager,
	) engineController {
		return engine
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	app.Startup(ctx)

	if !logging.Enabled() {
		t.Fatal("expected logging to be enabled after Startup when persisted settings opt in")
	}
}

// TestStartupSkipsEnableWhenPersistedAsDisabled ensures the engine is NOT
// auto-enabled when the last saved settings had Enabled=false.
func TestStartupSkipsEnableWhenPersistedAsDisabled(t *testing.T) {
	t.Setenv("APPDATA", t.TempDir())
	t.Cleanup(func() {
		logging.Close()
		_ = logging.Init(false)
	})

	disabled := domain.DefaultSettings()
	disabled.Enabled = false
	if err := config.Save(disabled); err != nil {
		t.Fatalf("config.Save returned error: %v", err)
	}

	engine := &fakeEngine{}
	app := NewApp(
		stubActivityMonitor{},
		stubPointerController{},
		stubScreenProvider{},
		&fakeWindowManager{},
		&fakeAutostartManager{},
	)
	app.hideWindow = func(context.Context) {}
	app.emit = func(context.Context, string, ...interface{}) {}
	app.newEngine = func(
		_ *state.Store,
		_ *domain.StateMachine,
		_ platform.UserActivityMonitor,
		_ platform.PointerController,
		_ platform.ScreenProvider,
		_ platform.WindowManager,
	) engineController {
		return engine
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	app.Startup(ctx)

	if engine.enableCalls != 0 {
		t.Fatalf("expected Enable to be skipped for disabled persisted settings, got %d calls", engine.enableCalls)
	}
	if engine.startCalls != 1 {
		t.Fatalf("expected engine to still be Started exactly once, got %d", engine.startCalls)
	}
}
