//go:build windows

package windows

import (
	"afk-hero/internal/appmeta"
	"fmt"
	"os"
	"syscall"

	"golang.org/x/sys/windows/registry"
)

const registryRunPath = `SOFTWARE\Microsoft\Windows\CurrentVersion\Run`

// AutostartManager manages autostart via the Windows registry.
type AutostartManager struct{}

func NewAutostartManager() *AutostartManager { return &AutostartManager{} }

func (m *AutostartManager) SetAutostart(enabled bool) error {
	key, err := registry.OpenKey(registry.CURRENT_USER, registryRunPath, registry.SET_VALUE)
	if err != nil {
		return fmt.Errorf("open registry run key: %w", err)
	}
	defer key.Close()

	if enabled {
		exePath, err := os.Executable()
		if err != nil {
			return fmt.Errorf("get executable path: %w", err)
		}
		return key.SetStringValue(appmeta.DisplayName, `"`+exePath+`"`)
	}

	if err := key.DeleteValue(appmeta.DisplayName); err != nil {
		if errno, ok := err.(syscall.Errno); ok && errno == 2 {
			return nil
		}
		return fmt.Errorf("remove autostart entry: %w", err)
	}
	return nil
}
