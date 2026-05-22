package appmeta

import "afk-hero/internal/domain"

const (
	WindowWidth            = 590
	WindowBaseHeight       = 640
	WindowAdvancedHeight   = 160
	WindowActivationHeight = 110
)

// WindowHeight returns the fixed settings window height for the given settings.
func WindowHeight(settings domain.Settings) int {
	height := WindowBaseHeight
	if settings.Advanced {
		height += WindowAdvancedHeight
	}
	if settings.ActivationEnabled {
		height += WindowActivationHeight
	}
	return height
}
