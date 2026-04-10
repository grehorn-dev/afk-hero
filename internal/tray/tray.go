package tray

import (
	"afk-hero/internal/domain"
	"afk-hero/internal/i18n"
	_ "embed"
)

//go:embed icons/enabled.ico
var EnabledIcon []byte

//go:embed icons/disabled.ico
var DisabledIcon []byte

// Callbacks holds the functions called by tray menu actions.
type Callbacks struct {
	OnShowSettings func()
	OnEnable       func()
	OnDisable      func()
	OnExit         func()
}

// MenuState represents the current state for tray menu rendering.
type MenuState struct {
	Enabled  bool
	Language string
}

// MenuLabels holds localized strings for the tray context menu.
type MenuLabels struct {
	Settings string
	Enable   string
	Disable  string
	Exit     string
}

// GetMenuLabels returns localized menu labels for the given language code.
func GetMenuLabels(lang string) MenuLabels {
	if lang == "" {
		lang = "eng"
	}

	return MenuLabels{
		Settings: i18n.Translate(lang, "tray.settings"),
		Enable:   i18n.Translate(lang, "tray.enable"),
		Disable:  i18n.Translate(lang, "tray.disable"),
		Exit:     i18n.Translate(lang, "tray.exit"),
	}
}

// GetIcon returns the appropriate tray icon for the given enabled state.
func GetIcon(enabled bool) []byte {
	if enabled {
		return EnabledIcon
	}
	return DisabledIcon
}

// StateToMenuState converts domain state to tray menu state.
func StateToMenuState(settings domain.Settings, _ domain.State) MenuState {
	return MenuState{
		Enabled:  settings.Enabled,
		Language: settings.Language,
	}
}
