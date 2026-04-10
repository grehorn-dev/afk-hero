package runtime

import (
	"afk-hero/internal/domain"
	"afk-hero/internal/logging"
	"fmt"
	"time"
)

// observeWindowActivation runs the activation monitor for a single tick.
// It resolves (or re-resolves) the target, checks its status, publishes the
// runtime activation state and finally triggers ActivateWindow when the
// configured inactivity timeout is reached.
func (e *Engine) observeWindowActivation(settings domain.Settings) {
	if e.windowMgr == nil || !settings.Enabled || !settings.ActivationEnabled {
		e.clearAutoActivationCache()
		e.clearManualActivationCache()
		e.resetActivationSchedule()
		e.resetActivationMonitor(
			"disabled",
			"settings_enabled", settings.Enabled,
			"activation_enabled", settings.ActivationEnabled,
			"window_manager_available", e.windowMgr != nil,
		)
		return
	}

	target, err := e.resolveActivationWindow(settings)
	if err != nil {
		if settings.ActivationMode == domain.ActivationModeAuto {
			e.clearAutoActivationCache()
		}
		if settings.ActivationMode == domain.ActivationModeManual {
			e.clearManualActivationCache()
		}
		attrs := []any{
			"mode", settings.ActivationMode,
			"error", err,
		}
		if settings.ActivationMode == domain.ActivationModeManual {
			attrs = append(attrs,
				"target_process", settings.TargetProcessName,
				"target_class", settings.TargetWindowClass,
				"target_title", settings.TargetWindowTitle,
			)
		}
		e.resetActivationMonitor("target_resolution_failed", attrs...)
		return
	}
	status, err := e.windowMgr.WindowStatus(target.ID)
	if err != nil {
		if settings.ActivationMode == domain.ActivationModeAuto {
			e.clearAutoActivationCache()
		}
		if settings.ActivationMode == domain.ActivationModeManual {
			e.clearManualActivationCache()
		}
		e.resetActivationMonitor(
			"window_status_failed",
			"window_id", target.ID,
			"process", target.ProcessName,
			"class", target.ClassName,
			"error", err,
		)
		return
	}
	if !status.Exists {
		if settings.ActivationMode == domain.ActivationModeAuto {
			e.clearAutoActivationCache()
		}
		if settings.ActivationMode == domain.ActivationModeManual {
			e.clearManualActivationCache()
		}
		e.resetActivationMonitor(
			"target_disappeared",
			"window_id", target.ID,
			"process", target.ProcessName,
			"class", target.ClassName,
			"title", target.Title,
		)
		return
	}

	if !status.Minimized && status.Foreground {
		e.resetActivationDeadline()
		e.noteActivationDiagnostic(
			activationDiagnosticKey("active", target),
			"activation target is already active",
			"mode", settings.ActivationMode,
			"window_id", target.ID,
			"process", target.ProcessName,
			"class", target.ClassName,
			"title", target.Title,
		)
		e.publishActivationState(domain.WindowActivationState{
			TargetFound: true,
			Tracking:    false,
			Progress:    0,
			ProcessName: target.ProcessName,
			WindowTitle: target.Title,
		})
		return
	}

	inactiveSince := e.markActivationInactive()
	timeout := time.Duration(settings.ActivationTimeout) * time.Second
	if timeout <= 0 {
		timeout = time.Second
	}
	progress := clampProgress(time.Since(inactiveSince).Seconds() / timeout.Seconds())
	e.noteActivationDiagnostic(
		activationDiagnosticKey("inactive", target),
		"activation target became inactive",
		"mode", settings.ActivationMode,
		"window_id", target.ID,
		"process", target.ProcessName,
		"class", target.ClassName,
		"title", target.Title,
		"foreground", status.Foreground,
		"minimized", status.Minimized,
		"timeout", timeout,
	)
	e.publishActivationState(domain.WindowActivationState{
		TargetFound: true,
		Tracking:    true,
		Progress:    progress,
		ProcessName: target.ProcessName,
		WindowTitle: target.Title,
	})
	if progress < 1 {
		return
	}

	logging.Get().Debug(
		"activation timeout reached, requesting window activation",
		"mode", settings.ActivationMode,
		"window_id", target.ID,
		"process", target.ProcessName,
		"class", target.ClassName,
		"title", target.Title,
		"timeout", timeout,
	)
	if err := e.windowMgr.ActivateWindow(target.ID); err != nil {
		logging.Get().Debug(
			"window activation failed",
			"mode", settings.ActivationMode,
			"window_id", target.ID,
			"process", target.ProcessName,
			"class", target.ClassName,
			"title", target.Title,
			"error", err,
		)
	} else {
		logging.Get().Debug(
			"window activation request completed",
			"mode", settings.ActivationMode,
			"window_id", target.ID,
			"process", target.ProcessName,
			"class", target.ClassName,
			"title", target.Title,
		)
	}

	e.bumpActivationDeadline()
}

func (e *Engine) resolveActivationWindow(settings domain.Settings) (domain.WindowInfo, error) {
	switch settings.ActivationMode {
	case domain.ActivationModeAuto:
		e.clearManualActivationCache()
		return e.resolveAutoActivationWindow(settings.ActivationAutoProcessBlacklist)
	case domain.ActivationModeManual:
		e.clearAutoActivationCache()
		return e.resolveManualActivationWindow(settings.ActivationFingerprint())
	default:
		e.clearAutoActivationCache()
		e.clearManualActivationCache()
		return domain.WindowInfo{}, fmt.Errorf("unsupported activation mode: %s", settings.ActivationMode)
	}
}

func (e *Engine) resolveAutoActivationWindow(autoProcessBlacklist []string) (domain.WindowInfo, error) {
	if cached, ok := e.autoActivationCache(); ok {
		info, err := e.windowMgr.AutoTargetInfo(cached.ID, autoProcessBlacklist)
		if err == nil {
			e.setAutoActivationCache(info)
			return info, nil
		}

		logging.Get().Debug(
			"auto activation cache invalidated",
			"window_id", cached.ID,
			"process", cached.ProcessName,
			"class", cached.ClassName,
			"error", err,
		)
		e.clearAutoActivationCache()
	}

	target, err := e.windowMgr.ResolveAutoTarget(autoProcessBlacklist)
	if err != nil {
		return domain.WindowInfo{}, err
	}

	e.setAutoActivationCache(target)
	return target, nil
}

func (e *Engine) resolveManualActivationWindow(fingerprint domain.WindowFingerprint) (domain.WindowInfo, error) {
	if !fingerprint.Valid() {
		e.clearManualActivationCache()
		return domain.WindowInfo{}, fmt.Errorf("manual activation requires process name and window class")
	}

	if cached, ok := e.manualActivationCache(fingerprint); ok {
		info, err := e.windowMgr.WindowInfo(cached.ID)
		if err == nil && fingerprint.Matches(info) {
			e.setManualActivationCache(fingerprint, info)
			return info, nil
		}

		logging.Get().Debug(
			"manual activation cache invalidated",
			"window_id", cached.ID,
			"target_process", fingerprint.ProcessName,
			"target_class", fingerprint.WindowClass,
			"error", err,
		)
		e.clearManualActivationCache()
	}

	target, err := e.windowMgr.FindWindowByFingerprint(fingerprint)
	if err != nil {
		return domain.WindowInfo{}, err
	}

	e.setManualActivationCache(fingerprint, target)
	return target, nil
}

func (e *Engine) manualActivationCache(fingerprint domain.WindowFingerprint) (domain.WindowInfo, bool) {
	e.mu.Lock()
	defer e.mu.Unlock()

	if !e.manualActivationSet || e.manualActivationFP != fingerprint {
		return domain.WindowInfo{}, false
	}

	return e.manualActivationWnd, true
}

func (e *Engine) setManualActivationCache(fingerprint domain.WindowFingerprint, target domain.WindowInfo) {
	e.mu.Lock()
	e.manualActivationFP = fingerprint
	e.manualActivationWnd = target
	e.manualActivationSet = true
	e.mu.Unlock()
}

func (e *Engine) clearManualActivationCache() {
	e.mu.Lock()
	e.manualActivationFP = domain.WindowFingerprint{}
	e.manualActivationWnd = domain.WindowInfo{}
	e.manualActivationSet = false
	e.mu.Unlock()
}

func (e *Engine) autoActivationCache() (domain.WindowInfo, bool) {
	e.mu.Lock()
	defer e.mu.Unlock()

	if !e.autoActivationSet {
		return domain.WindowInfo{}, false
	}

	return e.autoActivationWnd, true
}

func (e *Engine) setAutoActivationCache(target domain.WindowInfo) {
	e.mu.Lock()
	e.autoActivationWnd = target
	e.autoActivationSet = true
	e.mu.Unlock()
}

func (e *Engine) clearAutoActivationCache() {
	e.mu.Lock()
	e.autoActivationWnd = domain.WindowInfo{}
	e.autoActivationSet = false
	e.mu.Unlock()
}

// shouldObserveWindowActivation implements cadence gating for activation
// monitoring. Even though the main engine tick runs at 250 ms, activation
// monitoring can be slower (configurable via [activation].tick_interval_ms).
// When the feature is disabled or the window manager is missing, we return
// true so the caller can perform an immediate cleanup pass.
func (e *Engine) shouldObserveWindowActivation(settings domain.Settings) bool {
	if e.windowMgr == nil || !settings.Enabled || !settings.ActivationEnabled {
		e.resetActivationSchedule()
		return true
	}

	interval := time.Duration(settings.ActivationTickIntervalMS) * time.Millisecond
	if interval <= 0 {
		interval = time.Duration(domain.DefaultActivationTickIntervalMS()) * time.Millisecond
	}

	e.mu.Lock()
	defer e.mu.Unlock()

	now := time.Now()
	if e.activationLastTick.IsZero() || now.Sub(e.activationLastTick) >= interval {
		e.activationLastTick = now
		return true
	}

	return false
}

func (e *Engine) resetActivationSchedule() {
	e.mu.Lock()
	e.activationLastTick = time.Time{}
	e.mu.Unlock()
}

func (e *Engine) markActivationInactive() time.Time {
	e.mu.Lock()
	defer e.mu.Unlock()

	if e.activationInactive.IsZero() {
		e.activationInactive = time.Now()
	}

	return e.activationInactive
}

func (e *Engine) bumpActivationDeadline() {
	e.mu.Lock()
	e.activationInactive = time.Now()
	e.mu.Unlock()
}

func (e *Engine) resetActivationDeadline() {
	e.mu.Lock()
	e.activationInactive = time.Time{}
	e.mu.Unlock()
}

func (e *Engine) resetActivationMonitor(reason string, attrs ...any) {
	e.mu.Lock()
	e.activationInactive = time.Time{}
	e.mu.Unlock()

	fields := make([]any, 0, len(attrs)+2)
	fields = append(fields, "reason", reason)
	fields = append(fields, attrs...)
	e.noteActivationDiagnostic("reset:"+reason, "activation monitor reset", fields...)

	e.publishActivationState(domain.WindowActivationState{})
}

// noteActivationDiagnostic emits a debug log entry only when the diagnostic
// key has changed from the previous call. This avoids flooding the log with
// the same "target active" line on every tick while activation monitoring
// is running against a stable target.
func (e *Engine) noteActivationDiagnostic(key, message string, attrs ...any) {
	e.mu.Lock()
	if e.activationLogKey == key {
		e.mu.Unlock()
		return
	}
	e.activationLogKey = key
	e.mu.Unlock()

	logging.Get().Debug(message, attrs...)
}

func activationDiagnosticKey(prefix string, target domain.WindowInfo) string {
	return fmt.Sprintf("%s:%s:%s", prefix, target.ProcessName, target.ClassName)
}

func (e *Engine) publishActivationState(state domain.WindowActivationState) {
	e.mu.Lock()
	if e.activationState == state {
		e.mu.Unlock()
		return
	}
	e.activationState = state
	cb := e.onActivationChange
	e.mu.Unlock()

	if cb != nil {
		cb(state)
	}
}
