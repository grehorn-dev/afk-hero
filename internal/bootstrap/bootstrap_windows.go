//go:build windows

package bootstrap

import (
	"afk-hero/internal/platform"
	winplatform "afk-hero/internal/platform/windows"

	"github.com/wailsapp/wails/v2/pkg/options"
	woptions "github.com/wailsapp/wails/v2/pkg/options/windows"
)

// Services bundles platform-specific adapters for app startup wiring.
type Services struct {
	Activity      platform.UserActivityMonitor
	Pointer       platform.PointerController
	Screen        platform.ScreenProvider
	WindowManager platform.WindowManager
}

// NewServices builds the platform-specific service bundle for Windows.
func NewServices() Services {
	return Services{
		Activity:      winplatform.NewActivityMonitor(),
		Pointer:       winplatform.NewPointerController(),
		Screen:        winplatform.NewScreenProvider(),
		WindowManager: winplatform.NewWindowManager(),
	}
}

// ConfigurePlatformOptions applies Windows-specific Wails options.
func ConfigurePlatformOptions(appOptions *options.App) {
	appOptions.Windows = &woptions.Options{
		WebviewIsTransparent: false,
		WindowIsTranslucent:  false,
		DisableWindowIcon:    false,
	}
}
