package platform

import (
	"afk-hero/internal/domain"
	"time"
)

// UserActivityMonitor detects user input activity.
type UserActivityMonitor interface {
	// IdleTime returns how long since the last real user input event.
	IdleTime() (time.Duration, error)
}

// PointerController controls the mouse cursor.
type PointerController interface {
	// GetPosition returns the current cursor position.
	GetPosition() (domain.Point, error)

	// MoveBy moves the cursor by a relative delta.
	// The extraInfo parameter is used to tag injected input.
	MoveBy(delta domain.Point, extraInfo uintptr) error
}

// ScreenProvider provides screen geometry information.
type ScreenProvider interface {
	// WorkArea returns the usable desktop area for the monitor containing the given point.
	WorkArea(at domain.Point) (domain.Rect, error)

	// PrimaryWorkArea returns the primary monitor's work area.
	PrimaryWorkArea() (domain.Rect, error)
}

// WindowManager lists and activates windows.
type WindowManager interface {
	// ListWindows returns visible top-level windows.
	ListWindows(autoProcessBlacklist []string) ([]domain.WindowInfo, error)

	// ActivateWindow brings the specified window to the foreground.
	ActivateWindow(id uintptr) error

	// FindWindowByFingerprint resolves a stable process/class matcher to a live window.
	FindWindowByFingerprint(fingerprint domain.WindowFingerprint) (domain.WindowInfo, error)

	// ResolveAutoTarget returns the current best-effort activation target for Auto mode.
	ResolveAutoTarget(autoProcessBlacklist []string) (domain.WindowInfo, error)

	// AutoTargetInfo reports whether the specified window handle still qualifies as the current Auto target.
	AutoTargetInfo(id uintptr, autoProcessBlacklist []string) (domain.WindowInfo, error)

	// WindowInfo returns the current identity fields for the specified window handle.
	WindowInfo(id uintptr) (domain.WindowInfo, error)

	// WindowStatus reports whether the window exists and is currently foreground/minimized.
	WindowStatus(id uintptr) (domain.WindowActivationStatus, error)
}
