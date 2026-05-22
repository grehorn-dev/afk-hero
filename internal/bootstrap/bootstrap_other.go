//go:build !windows

package bootstrap

import (
	"afk-hero/internal/platform"
	"afk-hero/internal/platform/stubs"

	"github.com/wailsapp/wails/v2/pkg/options"
)

// Services bundles platform-specific adapters for app startup wiring.
type Services struct {
	Activity      platform.UserActivityMonitor
	Pointer       platform.PointerController
	Screen        platform.ScreenProvider
	WindowManager platform.WindowManager
	Autostart     platform.AutostartManager
}

// NewServices builds the platform-specific service bundle for non-Windows stubs.
func NewServices() Services {
	return Services{
		Activity:      stubs.NewActivityMonitor(),
		Pointer:       stubs.NewPointerController(),
		Screen:        stubs.NewScreenProvider(),
		WindowManager: stubs.NewWindowManager(),
		Autostart:     stubs.NewAutostartManager(),
	}
}

// ConfigurePlatformOptions keeps non-Windows startup wiring compile-safe.
func ConfigurePlatformOptions(_ *options.App) {}
