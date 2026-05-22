package config

import (
	"afk-hero/internal/appmeta"
	"afk-hero/internal/domain"
	"fmt"
	"os"
	"path/filepath"

	"github.com/pelletier/go-toml/v2"
)

const configFile = "config.toml"

// FileConfig represents the on-disk TOML config structure.
type FileConfig struct {
	Application ApplicationSection `toml:"application"`
	Appearance  AppearanceSection  `toml:"appearance"`
	Pattern     PatternSection     `toml:"pattern"`
	Distance    RangeSection       `toml:"distance"`
	Interval    RangeSection       `toml:"interval"`
	Speed       RangeSection       `toml:"speed"`
	Inactivity  RangeSection       `toml:"inactivity"`
	Activation  ActivationSection  `toml:"activation"`
}

type ApplicationSection struct {
	Enabled   bool  `toml:"enabled"`
	Autostart bool  `toml:"autostart"`
	Advanced  *bool `toml:"advanced"`
	Logging   bool  `toml:"logging"`
}

type AppearanceSection struct {
	Language string `toml:"language"`
	Theme    string `toml:"theme"`
}

type PatternSection struct {
	Shape     string `toml:"shape"`
	Direction string `toml:"direction"`
}

type RangeSection struct {
	Mode   string `toml:"mode"`
	Value  int    `toml:"value"`
	MinVal int    `toml:"min_val"`
	MaxVal int    `toml:"max_val"`
}

type ActivationSection struct {
	Enabled              bool     `toml:"enabled"`
	Mode                 string   `toml:"mode"`
	Timeout              int      `toml:"timeout"`
	TargetWindowOnly     bool     `toml:"target_window_only"`
	TickIntervalMS       int      `toml:"tick_interval_ms"`
	WindowTitle          string   `toml:"window_title"`
	WindowClass          string   `toml:"window_class"`
	ProcessName          string   `toml:"process_name"`
	AutoProcessBlacklist []string `toml:"blacklist_auto"`
}

// ConfigDir returns the directory where config is stored.
func ConfigDir() (string, error) {
	userConfig, err := os.UserConfigDir()
	if err != nil {
		return "", fmt.Errorf("get user config dir: %w", err)
	}
	return filepath.Join(userConfig, appmeta.DirName), nil
}

// ConfigPath returns the full path to the config file.
func ConfigPath() (string, error) {
	dir, err := ConfigDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, configFile), nil
}

// Load reads the config file and returns settings.
// Returns default settings if the file doesn't exist.
// Returns normalized settings if the file is partially invalid.
func Load() (domain.Settings, error) {
	path, err := ConfigPath()
	if err != nil {
		return domain.DefaultSettings(), fmt.Errorf("config path: %w", err)
	}

	data, err := os.ReadFile(path)
	if os.IsNotExist(err) {
		return domain.DefaultSettings(), nil
	}
	if err != nil {
		return domain.DefaultSettings(), fmt.Errorf("read config: %w", err)
	}

	var fc FileConfig
	if err := toml.Unmarshal(data, &fc); err != nil {
		// Config is corrupted, use defaults
		return domain.DefaultSettings(), fmt.Errorf("parse config: %w", err)
	}

	settings := fileConfigToSettings(fc)
	settings = NormalizeSettings(settings)
	return settings, nil
}

// Save writes the settings to the config file.
func Save(s domain.Settings) error {
	path, err := ConfigPath()
	if err != nil {
		return fmt.Errorf("config path: %w", err)
	}

	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return fmt.Errorf("create config dir: %w", err)
	}

	fc := settingsToFileConfig(s)
	data, err := toml.Marshal(fc)
	if err != nil {
		return fmt.Errorf("marshal config: %w", err)
	}

	if err := writeFileAtomically(path, data, 0o644); err != nil {
		return fmt.Errorf("write config: %w", err)
	}

	return nil
}

// Delete removes the config file.
func Delete() error {
	path, err := ConfigPath()
	if err != nil {
		return fmt.Errorf("config path: %w", err)
	}
	err = os.Remove(path)
	if os.IsNotExist(err) {
		return nil
	}
	return err
}

func fileConfigToSettings(fc FileConfig) domain.Settings {
	advanced := domain.DefaultSettings().Advanced
	if fc.Application.Advanced != nil {
		advanced = *fc.Application.Advanced
	}

	return domain.Settings{
		Enabled:   fc.Application.Enabled,
		Autostart: fc.Application.Autostart,
		Advanced:  advanced,
		Logging:   fc.Application.Logging,
		Language:  fc.Appearance.Language,
		Theme:     domain.Theme(fc.Appearance.Theme),

		Shape:     domain.Shape(fc.Pattern.Shape),
		Direction: domain.Direction(fc.Pattern.Direction),

		Distance:   rangeToNumeric(fc.Distance),
		Interval:   rangeToNumeric(fc.Interval),
		Speed:      rangeToNumeric(fc.Speed),
		Inactivity: rangeToNumeric(fc.Inactivity),

		ActivationEnabled:              fc.Activation.Enabled,
		ActivationMode:                 domain.ActivationMode(fc.Activation.Mode),
		ActivationTimeout:              fc.Activation.Timeout,
		ActivationTargetWindowOnly:     fc.Activation.TargetWindowOnly,
		ActivationTickIntervalMS:       fc.Activation.TickIntervalMS,
		TargetWindowTitle:              fc.Activation.WindowTitle,
		TargetWindowClass:              fc.Activation.WindowClass,
		TargetProcessName:              fc.Activation.ProcessName,
		ActivationAutoProcessBlacklist: fc.Activation.AutoProcessBlacklist,
	}
}

func settingsToFileConfig(s domain.Settings) FileConfig {
	return FileConfig{
		Application: ApplicationSection{
			Enabled:   s.Enabled,
			Autostart: s.Autostart,
			Advanced:  boolPtr(s.Advanced),
			Logging:   s.Logging,
		},
		Appearance: AppearanceSection{
			Language: s.Language,
			Theme:    string(s.Theme),
		},
		Pattern: PatternSection{
			Shape:     string(s.Shape),
			Direction: string(s.Direction),
		},
		Distance:   numericToRange(s.Distance),
		Interval:   numericToRange(s.Interval),
		Speed:      numericToRange(s.Speed),
		Inactivity: numericToRange(s.Inactivity),
		Activation: ActivationSection{
			Enabled:              s.ActivationEnabled,
			Mode:                 string(s.ActivationMode),
			Timeout:              s.ActivationTimeout,
			TargetWindowOnly:     s.ActivationTargetWindowOnly,
			TickIntervalMS:       s.ActivationTickIntervalMS,
			WindowTitle:          s.TargetWindowTitle,
			WindowClass:          s.TargetWindowClass,
			ProcessName:          s.TargetProcessName,
			AutoProcessBlacklist: append([]string(nil), s.ActivationAutoProcessBlacklist...),
		},
	}
}

func boolPtr(v bool) *bool {
	return &v
}

func rangeToNumeric(rs RangeSection) domain.NumericRange {
	return domain.NumericRange{
		Mode:   domain.RangeMode(rs.Mode),
		Value:  rs.Value,
		MinVal: rs.MinVal,
		MaxVal: rs.MaxVal,
	}
}

func numericToRange(nr domain.NumericRange) RangeSection {
	return RangeSection{
		Mode:   string(nr.Mode),
		Value:  nr.Value,
		MinVal: nr.MinVal,
		MaxVal: nr.MaxVal,
	}
}

func writeFileAtomically(path string, data []byte, perm os.FileMode) error {
	dir := filepath.Dir(path)
	tempFile, err := os.CreateTemp(dir, configFile+".*.tmp")
	if err != nil {
		return fmt.Errorf("create temp file: %w", err)
	}

	tempPath := tempFile.Name()
	cleanupTemp := true
	defer func() {
		if cleanupTemp {
			_ = os.Remove(tempPath)
		}
	}()

	if err := tempFile.Chmod(perm); err != nil {
		_ = tempFile.Close()
		return fmt.Errorf("chmod temp file: %w", err)
	}

	if _, err := tempFile.Write(data); err != nil {
		_ = tempFile.Close()
		return fmt.Errorf("write temp file: %w", err)
	}

	if err := tempFile.Sync(); err != nil {
		_ = tempFile.Close()
		return fmt.Errorf("sync temp file: %w", err)
	}

	if err := tempFile.Close(); err != nil {
		return fmt.Errorf("close temp file: %w", err)
	}

	if err := replaceFile(tempPath, path); err != nil {
		return fmt.Errorf("replace config file: %w", err)
	}

	cleanupTemp = false
	return nil
}
