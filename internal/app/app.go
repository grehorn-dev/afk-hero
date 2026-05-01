package app

import (
	"afk-hero/internal/appmeta"
	"afk-hero/internal/config"
	"afk-hero/internal/domain"
	"afk-hero/internal/i18n"
	"afk-hero/internal/logging"
	"afk-hero/internal/platform"
	appruntime "afk-hero/internal/runtime"
	"afk-hero/internal/state"
	"context"
	"fmt"
	"sync"

	wailsruntime "github.com/wailsapp/wails/v2/pkg/runtime"
)

// traySignature captures the settings fields that affect the tray render.
// Only fields consumed by tray.StateToMenuState should be compared here so
// we do not trigger expensive tray refreshes on unrelated settings edits.
type traySignature struct {
	enabled  bool
	language string
}

func traySignatureFor(s domain.Settings) traySignature {
	return traySignature{
		enabled:  s.Enabled,
		language: s.Language,
	}
}

type engineController interface {
	Start(context.Context)
	Stop()
	Enable()
	Disable()
	SetActivationStateCallback(func(domain.WindowActivationState))
	ActivationState() domain.WindowActivationState
}

// App is the main application struct bound to Wails.
type App struct {
	ctx        context.Context
	store      *state.Store
	engine     engineController
	sm         *domain.StateMachine
	hideWindow func(context.Context)
	emit       func(ctx context.Context, name string, payload ...interface{})

	// Platform adapters
	activityMonitor platform.UserActivityMonitor
	pointer         platform.PointerController
	screen          platform.ScreenProvider
	windowMgr       platform.WindowManager

	// Tray update callback
	onTrayUpdate func(settings domain.Settings, state domain.State)

	newEngine func(
		store *state.Store,
		sm *domain.StateMachine,
		activity platform.UserActivityMonitor,
		pointer platform.PointerController,
		screen platform.ScreenProvider,
		windowMgr platform.WindowManager,
	) engineController

	trayMu            sync.Mutex
	lastTraySignature traySignature
	lastTrayState     domain.State
	trayPrimed        bool

	closeMu       sync.Mutex
	quitRequested bool
}

// NewApp creates a new application instance.
func NewApp(
	activity platform.UserActivityMonitor,
	pointer platform.PointerController,
	screen platform.ScreenProvider,
	windowMgr platform.WindowManager,
) *App {
	return &App{
		activityMonitor: activity,
		pointer:         pointer,
		screen:          screen,
		windowMgr:       windowMgr,
		hideWindow:      wailsruntime.WindowHide,
		emit:            wailsruntime.EventsEmit,
		newEngine: func(
			store *state.Store,
			sm *domain.StateMachine,
			activity platform.UserActivityMonitor,
			pointer platform.PointerController,
			screen platform.ScreenProvider,
			windowMgr platform.WindowManager,
		) engineController {
			return appruntime.NewEngine(store, sm, activity, pointer, screen, windowMgr)
		},
	}
}

// SetTrayCallback sets the callback for tray menu updates.
func (a *App) SetTrayCallback(cb func(domain.Settings, domain.State)) {
	a.onTrayUpdate = cb
}

// Startup is called when the Wails app starts.
func (a *App) Startup(ctx context.Context) {
	a.ctx = ctx

	// Load config
	settings, err := config.Load()

	// Initialize logging BEFORE first Get() call
	if initErr := logging.Init(settings.Effective().Logging); initErr != nil {
		// Can't log this — Init failed
		_ = initErr
	}

	log := logging.Get()

	if err != nil {
		log.Warn("config load issue, using defaults", "error", err)
	}

	// Preload English translations as fallback
	if _, err := i18n.LoadTranslations("eng"); err != nil {
		log.Warn("failed to preload english translations", "error", err)
	}
	if _, err := i18n.LoadTranslations(settings.Language); err != nil {
		log.Warn("failed to preload selected translations", "language", settings.Language, "error", err)
	}

	// Create state store with change callback
	a.store = state.NewStore(settings, a.handleStoreChange)

	// Create state machine — use logging.Get() in callback to pick up runtime changes
	a.sm = domain.NewStateMachine(domain.StateDisabled, func(from, to domain.State, event domain.Event) {
		logging.Get().Info("state transition", "from", from, "to", to, "event", event)
		a.store.SetState(to)
	})

	// Create and start engine
	a.engine = a.newEngine(a.store, a.sm, a.activityMonitor, a.pointer, a.screen, a.windowMgr)
	a.engine.SetActivationStateCallback(a.handleActivationStateChange)
	a.engine.Start(ctx)

	progress, statusKey := a.store.Progress()
	a.handleStoreChange(settings, a.store.State(), progress, statusKey)

	// Enable if settings say so
	if settings.Enabled {
		a.engine.Enable()
	}

	log.Info("application started", "language", settings.Language, "enabled", settings.Enabled)
}

// Shutdown is called when the app is closing.
func (a *App) Shutdown(ctx context.Context) {
	log := logging.Get()
	log.Info("application shutting down")

	if a.engine != nil {
		a.engine.Stop()
	}

	// Persist the latest applied settings.
	if a.store != nil {
		settings := a.store.Settings()
		if err := config.Save(settings); err != nil {
			log.Error("config save failed during shutdown", "error", err)
		}
	}

	logging.Close()
}

// BeforeClose is called before the window closes. Returns true to prevent close (hide instead).
func (a *App) BeforeClose(ctx context.Context) bool {
	a.closeMu.Lock()
	quitRequested := a.quitRequested
	a.closeMu.Unlock()

	if quitRequested {
		return false
	}

	if ctx == nil {
		return true
	}

	if a.hideWindow != nil {
		a.hideWindow(ctx)
	}
	return true
}

// --- Wails Bindings ---

// GetDefaultSettings returns the canonical default settings.
func (a *App) GetDefaultSettings() domain.Settings {
	return domain.DefaultSettings()
}

// GetSettings returns the current applied settings.
func (a *App) GetSettings() domain.Settings {
	if a.store == nil {
		return domain.DefaultSettings()
	}
	return a.store.Settings()
}

// ApplySettings applies new settings from the frontend.
func (a *App) ApplySettings(settings domain.Settings) error {
	if a.store == nil || a.engine == nil {
		return fmt.Errorf("application is not ready")
	}

	log := logging.Get()
	current := a.store.Settings()
	settings.ActivationTickIntervalMS = current.ActivationTickIntervalMS
	settings.ActivationAutoProcessBlacklist = append([]string(nil), current.ActivationAutoProcessBlacklist...)

	// Normalize settings
	settings = config.NormalizeSettings(settings)

	// Handle enabled state change
	wasEnabled := a.store.Enabled()
	a.store.UpdateSettings(settings)

	if settings.Enabled && !wasEnabled {
		a.engine.Enable()
	} else if !settings.Enabled && wasEnabled {
		a.engine.Disable()
	}

	// Handle logging change
	currentEffective := current.Effective()
	nextEffective := settings.Effective()
	if currentEffective.Logging != nextEffective.Logging {
		if err := logging.SetEnabled(nextEffective.Logging); err != nil {
			log.Error("logging state change failed", "error", err)
		}
	}

	// Save config
	if err := config.Save(settings); err != nil {
		log.Error("config save failed", "error", err)
		return err
	}

	// Notify frontend
	a.emitEvent("settings:changed", settings)

	log.Info("settings applied", "language", settings.Language, "enabled", settings.Enabled)
	return nil
}

// SetEnabled toggles the enabled state (used from tray menu).
func (a *App) SetEnabled(enabled bool) {
	if a.store == nil || a.engine == nil {
		return
	}

	settings := a.store.Settings()
	settings.Enabled = enabled
	a.store.UpdateSettings(settings)

	if enabled {
		a.engine.Enable()
	} else {
		a.engine.Disable()
	}

	if err := config.Save(settings); err != nil {
		logging.Get().Error("config save failed", "error", err)
	}

	a.emitEvent("settings:changed", settings)
}

// GetTranslations returns translations for the given language code.
func (a *App) GetTranslations(code string) (map[string]string, error) {
	return i18n.LoadTranslations(code)
}

// GetSupportedLanguages returns the list of supported languages.
func (a *App) GetSupportedLanguages() []i18n.Language {
	return i18n.SupportedLanguages
}

// GetWindowList returns the list of available windows.
func (a *App) GetWindowList() ([]domain.WindowInfo, error) {
	if a.windowMgr == nil {
		return nil, fmt.Errorf("window manager is not available")
	}

	settings := domain.DefaultSettings()
	if a.store != nil {
		settings = a.store.Settings()
	}

	return a.windowMgr.ListWindows(settings.ActivationAutoProcessBlacklist)
}

// GetState returns the current runtime state info.
func (a *App) GetState() domain.RuntimeState {
	if a.store == nil {
		return domain.RuntimeState{
			State:     domain.StateDisabled,
			Enabled:   false,
			Progress:  0.0,
			StatusKey: "status.Disabled",
		}
	}

	progress, statusKey := a.store.Progress()
	return domain.RuntimeState{
		State:     a.store.State(),
		Enabled:   a.store.Enabled(),
		Progress:  progress,
		StatusKey: statusKey,
	}
}

// GetActivationState returns the current window activation monitor state.
func (a *App) GetActivationState() domain.WindowActivationState {
	if a.engine == nil {
		return domain.WindowActivationState{}
	}

	return a.engine.ActivationState()
}

// ShowWindow shows and focuses the settings window.
func (a *App) ShowWindow() {
	if a.ctx == nil {
		return
	}
	wailsruntime.WindowShow(a.ctx)
	wailsruntime.WindowUnminimise(a.ctx)
	wailsruntime.WindowSetAlwaysOnTop(a.ctx, true)
	wailsruntime.WindowSetAlwaysOnTop(a.ctx, false)
}

// SyncWindowLayout updates the fixed window size for the current draft layout.
func (a *App) SyncWindowLayout(advanced bool, activationEnabled bool) {
	if a.ctx == nil {
		return
	}

	height := appmeta.WindowHeight(domain.Settings{
		Advanced:          advanced,
		ActivationEnabled: activationEnabled,
	})

	wailsruntime.WindowSetMinSize(a.ctx, appmeta.WindowWidth, height)
	wailsruntime.WindowSetMaxSize(a.ctx, appmeta.WindowWidth, height)
	wailsruntime.WindowSetSize(a.ctx, appmeta.WindowWidth, height)
}

// RequestQuit requests the native application shutdown.
func (a *App) RequestQuit() {
	if a.ctx == nil {
		return
	}

	a.closeMu.Lock()
	a.quitRequested = true
	a.closeMu.Unlock()

	wailsruntime.Quit(a.ctx)
}

func (a *App) handleStoreChange(settings domain.Settings, st domain.State, progress float64, statusKey string) {
	defer logging.RecoverPanic(
		"app.handleStoreChange",
		"state", st,
		"enabled", settings.Enabled,
		"status_key", statusKey,
	)

	a.emitEvent("state:changed", domain.RuntimeState{
		State:     st,
		Enabled:   settings.Enabled,
		Progress:  progress,
		StatusKey: statusKey,
	})

	if a.onTrayUpdate == nil {
		return
	}

	signature := traySignatureFor(settings)
	a.trayMu.Lock()
	shouldUpdateTray := !a.trayPrimed || signature != a.lastTraySignature || st != a.lastTrayState
	if shouldUpdateTray {
		a.lastTraySignature = signature
		a.lastTrayState = st
		a.trayPrimed = true
	}
	a.trayMu.Unlock()

	if shouldUpdateTray {
		a.onTrayUpdate(settings, st)
	}
}

func (a *App) handleActivationStateChange(state domain.WindowActivationState) {
	defer logging.RecoverPanic(
		"app.handleActivationStateChange",
		"target_found", state.TargetFound,
		"tracking", state.Tracking,
		"process", state.ProcessName,
		"title", state.WindowTitle,
	)

	a.emitEvent("activation:changed", state)
}

func (a *App) emitEvent(name string, payload interface{}) {
	defer logging.RecoverPanic("app.emitEvent", "event", name)

	if a.ctx == nil || a.emit == nil {
		return
	}

	a.emit(a.ctx, name, payload)
}
