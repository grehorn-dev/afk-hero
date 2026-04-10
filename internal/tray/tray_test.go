package tray

import (
	"afk-hero/internal/domain"
	"testing"
)

func TestGetMenuLabelsFallsBackToEnglish(t *testing.T) {
	labels := GetMenuLabels("")

	if labels.Settings == "" || labels.Enable == "" || labels.Exit == "" {
		t.Fatalf("expected non-empty english fallback labels, got %+v", labels)
	}
}

func TestStateToMenuStateUsesAppliedSettings(t *testing.T) {
	settings := domain.DefaultSettings()
	settings.Enabled = false
	settings.Language = "rus"

	menuState := StateToMenuState(settings, domain.StateDisabled)
	if menuState.Enabled {
		t.Fatal("expected tray state to reflect disabled settings")
	}
	if menuState.Language != "rus" {
		t.Fatalf("expected language rus, got %s", menuState.Language)
	}
}
