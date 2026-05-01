//go:build !windows

package stubs

import (
	"afk-hero/internal/apperrors"
	"afk-hero/internal/domain"
	"time"
)

// ActivityMonitor is a stub for non-Windows platforms.
type ActivityMonitor struct{}

func NewActivityMonitor() *ActivityMonitor { return &ActivityMonitor{} }

func (m *ActivityMonitor) IdleTime() (time.Duration, error) {
	return 0, apperrors.ErrNotImplemented
}

// PointerController is a stub for non-Windows platforms.
type PointerController struct{}

func NewPointerController() *PointerController { return &PointerController{} }

func (pc *PointerController) GetPosition() (domain.Point, error) {
	return domain.Point{}, apperrors.ErrNotImplemented
}

func (pc *PointerController) MoveBy(delta domain.Point, extraInfo uintptr) error {
	return apperrors.ErrNotImplemented
}

// ScreenProvider is a stub for non-Windows platforms.
type ScreenProvider struct{}

func NewScreenProvider() *ScreenProvider { return &ScreenProvider{} }

func (sp *ScreenProvider) WorkArea(at domain.Point) (domain.Rect, error) {
	return domain.Rect{}, apperrors.ErrNotImplemented
}

func (sp *ScreenProvider) PrimaryWorkArea() (domain.Rect, error) {
	return domain.Rect{}, apperrors.ErrNotImplemented
}

// WindowManager is a stub for non-Windows platforms.
type WindowManager struct{}

func NewWindowManager() *WindowManager { return &WindowManager{} }

func (wm *WindowManager) ListWindows(autoProcessBlacklist []string) ([]domain.WindowInfo, error) {
	return nil, apperrors.ErrNotImplemented
}

func (wm *WindowManager) ActivateWindow(id uintptr) error {
	return apperrors.ErrNotImplemented
}

func (wm *WindowManager) FindWindowByFingerprint(fingerprint domain.WindowFingerprint) (domain.WindowInfo, error) {
	return domain.WindowInfo{}, apperrors.ErrNotImplemented
}

func (wm *WindowManager) ResolveAutoTarget(autoProcessBlacklist []string) (domain.WindowInfo, error) {
	return domain.WindowInfo{}, apperrors.ErrNotImplemented
}

func (wm *WindowManager) AutoTargetInfo(id uintptr, autoProcessBlacklist []string) (domain.WindowInfo, error) {
	return domain.WindowInfo{}, apperrors.ErrNotImplemented
}

func (wm *WindowManager) WindowInfo(id uintptr) (domain.WindowInfo, error) {
	return domain.WindowInfo{}, apperrors.ErrNotImplemented
}

func (wm *WindowManager) WindowStatus(id uintptr) (domain.WindowActivationStatus, error) {
	return domain.WindowActivationStatus{}, apperrors.ErrNotImplemented
}

