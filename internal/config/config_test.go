package config

import (
	"afk-hero/internal/appmeta"
	"afk-hero/internal/domain"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestSaveAndLoad(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("APPDATA", dir)

	// Ensure config dir exists
	configDir := filepath.Join(dir, appmeta.DirName)
	if err := os.MkdirAll(configDir, 0o755); err != nil {
		t.Fatalf("MkdirAll failed: %v", err)
	}

	settings := domain.DefaultSettings()
	settings.Language = "rus"
	settings.Theme = domain.ThemeLight
	settings.Advanced = false
	settings.Shape = domain.ShapeTriangle
	settings.Enabled = false
	settings.ActivationEnabled = true
	settings.ActivationMode = domain.ActivationModeManual
	settings.TargetProcessName = "game.exe"
	settings.TargetWindowClass = "UnrealWindow"
	settings.TargetWindowTitle = "Game"
	settings.ActivationTickIntervalMS = 750
	settings.ActivationAutoProcessBlacklist = []string{"chrome.exe", "msedge.exe", "custom-game-launcher.exe"}

	err := Save(settings)
	if err != nil {
		t.Fatalf("Save failed: %v", err)
	}

	loaded, err := Load()
	if err != nil {
		t.Fatalf("Load failed: %v", err)
	}

	if loaded.Language != "rus" {
		t.Errorf("expected language rus, got %s", loaded.Language)
	}
	if loaded.Theme != domain.ThemeLight {
		t.Errorf("expected theme Light, got %s", loaded.Theme)
	}
	if loaded.Advanced {
		t.Error("expected advanced to be false")
	}
	if loaded.Shape != domain.ShapeTriangle {
		t.Errorf("expected shape Triangle, got %s", loaded.Shape)
	}
	if loaded.Enabled {
		t.Error("expected enabled to be false")
	}
	if loaded.TargetProcessName != "game.exe" {
		t.Errorf("expected process name game.exe, got %s", loaded.TargetProcessName)
	}
	if loaded.TargetWindowClass != "UnrealWindow" {
		t.Errorf("expected window class UnrealWindow, got %s", loaded.TargetWindowClass)
	}
	if loaded.TargetWindowTitle != "Game" {
		t.Errorf("expected window title Game, got %s", loaded.TargetWindowTitle)
	}
	if loaded.ActivationTickIntervalMS != 750 {
		t.Errorf("expected activation tick interval 750, got %d", loaded.ActivationTickIntervalMS)
	}
	if got := strings.Join(loaded.ActivationAutoProcessBlacklist, ","); got != "chrome.exe,msedge.exe,custom-game-launcher.exe" {
		t.Errorf("expected persisted activation blacklist, got %s", got)
	}

	configPath := filepath.Join(configDir, configFile)
	data, err := os.ReadFile(configPath)
	if err != nil {
		t.Fatalf("ReadFile failed: %v", err)
	}
	if strings.Contains(string(data), "[window]") {
		t.Fatal("saved config should not contain a [window] section")
	}
	if strings.Contains(string(data), "[localization]") {
		t.Fatal("saved config should not contain a [localization] section")
	}
	if !strings.Contains(string(data), "[appearance]") {
		t.Fatal("saved config should contain an [appearance] section")
	}
	if !strings.Contains(string(data), "advanced = false") {
		t.Fatal("saved config should contain application.advanced")
	}
}

func TestLoad_MissingFile(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("APPDATA", dir)

	settings, err := Load()
	if err != nil {
		t.Fatalf("Load failed for missing file: %v", err)
	}

	defaults := domain.DefaultSettings()
	if settings.Shape != defaults.Shape {
		t.Errorf("expected default shape, got %s", settings.Shape)
	}
}

func TestLoad_CorruptedFile(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("APPDATA", dir)

	configDir := filepath.Join(dir, appmeta.DirName)
	if err := os.MkdirAll(configDir, 0o755); err != nil {
		t.Fatalf("MkdirAll failed: %v", err)
	}
	configPath := filepath.Join(configDir, configFile)
	if err := os.WriteFile(configPath, []byte("{{invalid toml"), 0o644); err != nil {
		t.Fatalf("WriteFile failed: %v", err)
	}

	settings, err := Load()
	if err == nil {
		t.Fatal("expected error for corrupted config")
	}

	// Should still return usable defaults
	defaults := domain.DefaultSettings()
	if settings.Shape != defaults.Shape {
		t.Errorf("expected default shape on corruption, got %s", settings.Shape)
	}
}

func TestLoad_LegacyWindowSectionIsIgnored(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("APPDATA", dir)

	configDir := filepath.Join(dir, appmeta.DirName)
	if err := os.MkdirAll(configDir, 0o755); err != nil {
		t.Fatalf("MkdirAll failed: %v", err)
	}

	configPath := filepath.Join(configDir, configFile)
	legacy := `
[application]
version = "1.0.0"
enabled = true
advanced = false
logging = false

[window]
last_x = 321
last_y = 654

[appearance]
language = "rus"
theme = "Light"

[pattern]
shape = "Triangle"
direction = "Left"
`
	if err := os.WriteFile(configPath, []byte(legacy), 0o644); err != nil {
		t.Fatalf("WriteFile failed: %v", err)
	}

	settings, err := Load()
	if err != nil {
		t.Fatalf("Load failed for legacy config: %v", err)
	}

	if settings.Language != "rus" {
		t.Fatalf("expected language rus, got %s", settings.Language)
	}
	if settings.Advanced {
		t.Fatal("expected explicit advanced=false to be preserved")
	}
	if settings.Theme != domain.ThemeLight {
		t.Fatalf("expected theme Light, got %s", settings.Theme)
	}
	if settings.Shape != domain.ShapeTriangle {
		t.Fatalf("expected shape Triangle, got %s", settings.Shape)
	}
	if settings.Direction != domain.DirectionLeft {
		t.Fatalf("expected direction Left, got %s", settings.Direction)
	}
}

func TestDelete(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("APPDATA", dir)

	configDir := filepath.Join(dir, appmeta.DirName)
	if err := os.MkdirAll(configDir, 0o755); err != nil {
		t.Fatalf("MkdirAll failed: %v", err)
	}

	if err := Save(domain.DefaultSettings()); err != nil {
		t.Fatalf("Save failed: %v", err)
	}

	err := Delete()
	if err != nil {
		t.Fatalf("Delete failed: %v", err)
	}

	configPath := filepath.Join(configDir, configFile)
	if _, err := os.Stat(configPath); !os.IsNotExist(err) {
		t.Error("config file still exists after delete")
	}
}

func TestSaveDoesNotLeaveTemporaryFiles(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("APPDATA", dir)

	settings := domain.DefaultSettings()
	settings.Language = "rus"

	if err := Save(settings); err != nil {
		t.Fatalf("Save failed: %v", err)
	}

	configDir := filepath.Join(dir, appmeta.DirName)
	entries, err := os.ReadDir(configDir)
	if err != nil {
		t.Fatalf("ReadDir failed: %v", err)
	}

	if len(entries) != 1 {
		t.Fatalf("expected only config.toml in config dir, got %d entries", len(entries))
	}

	if entries[0].Name() != configFile {
		t.Fatalf("expected only %s, got %s", configFile, entries[0].Name())
	}
}

func TestNormalizeSettings(t *testing.T) {
	s := domain.Settings{
		Shape:     "InvalidShape",
		Direction: "InvalidDir",
		Language:  "x",
		Theme:     "BadTheme",
		Distance: domain.NumericRange{
			Mode: domain.RangeModeRandom, MinVal: 5000, MaxVal: 1,
		},
		Speed: domain.NumericRange{
			Mode: "BadMode", Value: 999,
		},
		ActivationTimeout:        -5,
		ActivationTickIntervalMS: 50,
		ActivationMode:           "BadMode",
	}

	n := NormalizeSettings(s)

	if n.Shape != domain.ShapeRandom {
		t.Errorf("expected shape normalized to Random, got %s", n.Shape)
	}
	if n.Direction != domain.DirectionRandom {
		t.Errorf("expected direction normalized to Random, got %s", n.Direction)
	}
	if n.Language != "eng" {
		t.Errorf("expected language normalized to eng, got %s", n.Language)
	}
	if n.Theme != domain.ThemeDark {
		t.Errorf("expected theme normalized to Dark, got %s", n.Theme)
	}
	if n.Distance.MinVal > n.Distance.MaxVal {
		t.Error("distance min > max after normalization")
	}
	if n.Speed.Mode != domain.RangeModeFixed {
		t.Errorf("expected speed mode normalized to Fixed, got %s", n.Speed.Mode)
	}
	if n.Speed.Value > MaxSpeed {
		t.Errorf("expected speed value clamped to %d, got %d", MaxSpeed, n.Speed.Value)
	}
	if n.ActivationTimeout < MinActivationTimeout {
		t.Errorf("expected activation timeout >= %d, got %d", MinActivationTimeout, n.ActivationTimeout)
	}
	if n.ActivationMode != domain.ActivationModeAuto {
		t.Errorf("expected activation mode normalized to Auto, got %s", n.ActivationMode)
	}
	if n.ActivationTickIntervalMS != MinActivationTickIntervalMS {
		t.Errorf("expected activation tick interval clamped to %d, got %d", MinActivationTickIntervalMS, n.ActivationTickIntervalMS)
	}
}

func TestNormalizeSettings_DropsInvalidManualWindowFingerprint(t *testing.T) {
	s := domain.DefaultSettings()
	s.ActivationEnabled = true
	s.ActivationMode = domain.ActivationModeManual
	s.TargetProcessName = "game.exe"

	n := NormalizeSettings(s)

	if n.ActivationMode != domain.ActivationModeAuto {
		t.Fatalf("expected invalid manual activation to normalize to Auto, got %s", n.ActivationMode)
	}
	if n.TargetProcessName != "" || n.TargetWindowClass != "" || n.TargetWindowTitle != "" {
		t.Fatalf("expected invalid manual activation fields to be cleared, got %+v", n)
	}
}

func TestNormalizeSettings_NormalizesActivationAutoProcessBlacklist(t *testing.T) {
	s := domain.DefaultSettings()
	s.ActivationAutoProcessBlacklist = []string{
		` "Chrome.exe" `,
		`C:\Program Files\Mozilla Firefox\firefox.exe`,
		"chrome.exe",
		"",
		" custom.exe ",
	}

	n := NormalizeSettings(s)

	got := strings.Join(n.ActivationAutoProcessBlacklist, ",")
	if got != "chrome.exe,firefox.exe,custom.exe" {
		t.Fatalf("expected normalized auto blacklist, got %s", got)
	}
}

func TestNormalizeSettings_DefaultsActivationAutoProcessBlacklistWhenMissing(t *testing.T) {
	s := domain.DefaultSettings()
	s.ActivationAutoProcessBlacklist = nil

	n := NormalizeSettings(s)

	got := strings.Join(n.ActivationAutoProcessBlacklist, ",")
	if got != "chrome.exe,msedge.exe,firefox.exe,opera.exe" {
		t.Fatalf("expected default auto blacklist, got %s", got)
	}
}

func TestNormalizeSettings_RespectsExplicitEmptyActivationAutoProcessBlacklist(t *testing.T) {
	s := domain.DefaultSettings()
	s.ActivationAutoProcessBlacklist = []string{}

	n := NormalizeSettings(s)

	if len(n.ActivationAutoProcessBlacklist) != 0 {
		t.Fatalf("expected explicit empty auto blacklist to be preserved, got %#v", n.ActivationAutoProcessBlacklist)
	}
}

func TestNormalizeSettings_DefaultsActivationTickIntervalWhenMissing(t *testing.T) {
	s := domain.DefaultSettings()
	s.ActivationTickIntervalMS = 0

	n := NormalizeSettings(s)

	if n.ActivationTickIntervalMS != domain.DefaultActivationTickIntervalMS() {
		t.Fatalf("expected default activation tick interval %d, got %d", domain.DefaultActivationTickIntervalMS(), n.ActivationTickIntervalMS)
	}
}

func TestLoad_DefaultsAdvancedToFalseWhenMissing(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("APPDATA", dir)

	configDir := filepath.Join(dir, appmeta.DirName)
	if err := os.MkdirAll(configDir, 0o755); err != nil {
		t.Fatalf("MkdirAll failed: %v", err)
	}

	configPath := filepath.Join(configDir, configFile)
	legacy := `
[application]
version = "1.0.0"
enabled = true
logging = false

[appearance]
language = "eng"
theme = "Dark"
`
	if err := os.WriteFile(configPath, []byte(legacy), 0o644); err != nil {
		t.Fatalf("WriteFile failed: %v", err)
	}

	settings, err := Load()
	if err != nil {
		t.Fatalf("Load failed: %v", err)
	}

	if settings.Advanced {
		t.Fatal("expected missing application.advanced to default to false")
	}
}
