package domain

import (
	"fmt"
	"path/filepath"
	"strings"
)

// Shape represents a mouse movement pattern shape.
type Shape string

const (
	ShapeRandom    Shape = "Random"
	ShapePentagon  Shape = "Pentagon"
	ShapeRectangle Shape = "Rectangle"
	ShapeTriangle  Shape = "Triangle"
	ShapeSegment   Shape = "Segment"
	ShapeOnePixel  Shape = "OnePixel"
)

// RandomShapes returns shapes eligible for random selection (excludes Segment and OnePixel).
func RandomShapes() []Shape {
	return []Shape{ShapePentagon, ShapeRectangle, ShapeTriangle}
}

// AllShapes returns all available shapes.
func AllShapes() []Shape {
	return []Shape{ShapeRandom, ShapePentagon, ShapeRectangle, ShapeTriangle, ShapeSegment, ShapeOnePixel}
}

// ValidShape checks if the shape is a recognized value.
func ValidShape(s Shape) bool {
	for _, v := range AllShapes() {
		if v == s {
			return true
		}
	}
	return false
}

// Direction represents the direction of mouse movement relative to cursor.
type Direction string

const (
	DirectionRandom Direction = "Random"
	DirectionCenter Direction = "Center"
	DirectionUp     Direction = "Up"
	DirectionDown   Direction = "Down"
	DirectionLeft   Direction = "Left"
	DirectionRight  Direction = "Right"
)

// RandomDirections returns directions eligible for random selection.
func RandomDirections() []Direction {
	return []Direction{DirectionCenter, DirectionUp, DirectionDown, DirectionLeft, DirectionRight}
}

// AllDirections returns all available directions.
func AllDirections() []Direction {
	return []Direction{DirectionRandom, DirectionCenter, DirectionUp, DirectionDown, DirectionLeft, DirectionRight}
}

// ValidDirection checks if the direction is a recognized value.
func ValidDirection(d Direction) bool {
	for _, v := range AllDirections() {
		if v == d {
			return true
		}
	}
	return false
}

// RangeMode determines whether a numeric value is fixed or random.
type RangeMode string

const (
	RangeModeFixed  RangeMode = "Fixed"
	RangeModeRandom RangeMode = "Random"
)

// NumericRange holds a configurable numeric value with fixed or random mode.
type NumericRange struct {
	Mode   RangeMode `toml:"mode" json:"mode"`
	Value  int       `toml:"value" json:"value"`
	MinVal int       `toml:"min_val" json:"minVal"`
	MaxVal int       `toml:"max_val" json:"maxVal"`
}

// Validate checks that the range values are within the given bounds.
func (r NumericRange) Validate(minBound, maxBound int) error {
	switch r.Mode {
	case RangeModeFixed:
		if r.Value < minBound || r.Value > maxBound {
			return fmt.Errorf("value %d out of range [%d, %d]", r.Value, minBound, maxBound)
		}
	case RangeModeRandom:
		if r.MinVal < minBound || r.MinVal > maxBound {
			return fmt.Errorf("min value %d out of range [%d, %d]", r.MinVal, minBound, maxBound)
		}
		if r.MaxVal < minBound || r.MaxVal > maxBound {
			return fmt.Errorf("max value %d out of range [%d, %d]", r.MaxVal, minBound, maxBound)
		}
		if r.MinVal > r.MaxVal {
			return fmt.Errorf("min value %d exceeds max value %d", r.MinVal, r.MaxVal)
		}
	default:
		return fmt.Errorf("unknown range mode: %s", r.Mode)
	}
	return nil
}

// Normalize clamps the range to the given bounds and fixes ordering.
func (r NumericRange) Normalize(minBound, maxBound int) NumericRange {
	out := r
	if out.Mode != RangeModeFixed && out.Mode != RangeModeRandom {
		out.Mode = RangeModeFixed
	}
	out.Value = clamp(out.Value, minBound, maxBound)
	out.MinVal = clamp(out.MinVal, minBound, maxBound)
	out.MaxVal = clamp(out.MaxVal, minBound, maxBound)
	if out.MinVal > out.MaxVal {
		out.MinVal, out.MaxVal = out.MaxVal, out.MinVal
	}
	return out
}

// Point represents a 2D screen coordinate.
type Point struct {
	X int `json:"x"`
	Y int `json:"y"`
}

// Rect represents a screen rectangle (work area).
type Rect struct {
	Left   int `json:"left"`
	Top    int `json:"top"`
	Right  int `json:"right"`
	Bottom int `json:"bottom"`
}

// Width returns the width of the rectangle.
func (r Rect) Width() int { return r.Right - r.Left }

// Height returns the height of the rectangle.
func (r Rect) Height() int { return r.Bottom - r.Top }

// Contains checks if a point is inside the rectangle.
func (r Rect) Contains(p Point) bool {
	return p.X >= r.Left && p.X < r.Right && p.Y >= r.Top && p.Y < r.Bottom
}

// WindowInfo holds information about an enumerated window.
type WindowInfo struct {
	ID            uintptr `json:"id"`
	Title         string  `json:"title"`
	ClassName     string  `json:"className"`
	ProcessName   string  `json:"processName"`
	AutoCandidate bool    `json:"autoCandidate"`
}

// Fingerprint returns the stable process/class matcher for the window.
func (w WindowInfo) Fingerprint() WindowFingerprint {
	return WindowFingerprint{
		ProcessName: w.ProcessName,
		WindowClass: w.ClassName,
	}
}

// WindowFingerprint identifies a window family across restarts.
type WindowFingerprint struct {
	ProcessName string
	WindowClass string
}

// Valid reports whether the fingerprint can be used to find a window.
func (f WindowFingerprint) Valid() bool {
	return strings.TrimSpace(f.ProcessName) != "" && strings.TrimSpace(f.WindowClass) != ""
}

// Matches reports whether the fingerprint matches the given window info.
func (f WindowFingerprint) Matches(w WindowInfo) bool {
	return strings.EqualFold(strings.TrimSpace(f.ProcessName), strings.TrimSpace(w.ProcessName)) &&
		strings.EqualFold(strings.TrimSpace(f.WindowClass), strings.TrimSpace(w.ClassName))
}

// WindowActivationStatus describes the current activation state of a window.
type WindowActivationStatus struct {
	Exists     bool
	Minimized  bool
	Foreground bool
}

// ActivationMode determines how the target window is selected.
type ActivationMode string

const (
	ActivationModeAuto   ActivationMode = "Auto"
	ActivationModeManual ActivationMode = "Manual"
)

// Theme determines the application appearance.
type Theme string

const (
	ThemeDark  Theme = "Dark"
	ThemeLight Theme = "Light"
)

var defaultActivationAutoProcessBlacklist = []string{
	"chrome.exe",
	"msedge.exe",
	"firefox.exe",
	"opera.exe",
}

const defaultActivationTickIntervalMS = 500

// AllThemes returns all available themes.
func AllThemes() []Theme {
	return []Theme{ThemeDark, ThemeLight}
}

// ValidTheme checks if the theme is a recognized value.
func ValidTheme(t Theme) bool {
	for _, v := range AllThemes() {
		if v == t {
			return true
		}
	}
	return false
}

// DefaultActivationAutoProcessBlacklist returns a copy of the built-in auto-activation blacklist.
func DefaultActivationAutoProcessBlacklist() []string {
	return append([]string(nil), defaultActivationAutoProcessBlacklist...)
}

// DefaultActivationTickIntervalMS returns the default activation monitor cadence.
func DefaultActivationTickIntervalMS() int {
	return defaultActivationTickIntervalMS
}

// NormalizeProcessName trims, lowercases and strips directory parts from a process name.
func NormalizeProcessName(name string) string {
	trimmed := strings.TrimSpace(name)
	trimmed = strings.Trim(trimmed, `"'`)
	trimmed = strings.TrimSpace(trimmed)
	if trimmed == "" {
		return ""
	}

	normalized := filepath.Base(strings.ReplaceAll(trimmed, `\`, `/`))
	return strings.ToLower(strings.TrimSpace(normalized))
}

// Settings holds all user-configurable application settings.
type Settings struct {
	// Application
	Enabled  bool `toml:"enabled" json:"enabled"`
	Advanced bool `toml:"advanced" json:"advanced"`
	Logging  bool `toml:"logging" json:"logging"`

	// Appearance
	Language string `toml:"language" json:"language"`
	Theme    Theme  `toml:"theme" json:"theme"`

	// Pattern
	Shape     Shape     `toml:"shape" json:"shape"`
	Direction Direction `toml:"direction" json:"direction"`

	// Ranges
	Distance   NumericRange `toml:"distance" json:"distance"`
	Interval   NumericRange `toml:"interval" json:"interval"`
	Speed      NumericRange `toml:"speed" json:"speed"`
	Inactivity NumericRange `toml:"inactivity" json:"inactivity"`

	// Window activation
	ActivationEnabled              bool           `toml:"activation_enabled" json:"activationEnabled"`
	ActivationMode                 ActivationMode `toml:"activation_mode" json:"activationMode"`
	ActivationTimeout              int            `toml:"activation_timeout" json:"activationTimeout"`
	TargetWindowTitle              string         `toml:"target_window_title" json:"targetWindowTitle"`
	TargetWindowClass              string         `toml:"target_window_class" json:"targetWindowClass"`
	TargetProcessName              string         `toml:"target_process_name" json:"targetProcessName"`
	ActivationTickIntervalMS       int            `toml:"activation_tick_interval_ms" json:"-"`
	ActivationAutoProcessBlacklist []string       `toml:"blacklist_auto" json:"-"`
}

// ActivationFingerprint returns the persisted target matcher for manual activation.
func (s Settings) ActivationFingerprint() WindowFingerprint {
	return WindowFingerprint{
		ProcessName: s.TargetProcessName,
		WindowClass: s.TargetWindowClass,
	}
}

// Effective returns the runtime-effective settings after applying the advanced-mode toggle.
func (s Settings) Effective() Settings {
	if s.Advanced {
		return s
	}

	defaults := DefaultSettings()
	s.Logging = defaults.Logging
	s.Shape = defaults.Shape
	s.Direction = defaults.Direction
	s.Distance = defaults.Distance
	s.Speed = defaults.Speed

	return s
}

// DefaultSettings returns the default application settings.
func DefaultSettings() Settings {
	return Settings{
		Enabled:  true,
		Advanced: false,
		Logging:  false,
		Language: "eng",
		Theme:    ThemeDark,

		Shape:     ShapeRandom,
		Direction: DirectionRandom,

		Distance: NumericRange{
			Mode: RangeModeRandom, Value: 150, MinVal: 150, MaxVal: 300,
		},
		Interval: NumericRange{
			Mode: RangeModeRandom, Value: 25, MinVal: 25, MaxVal: 35,
		},
		Speed: NumericRange{
			Mode: RangeModeRandom, Value: 1, MinVal: 1, MaxVal: 2,
		},
		Inactivity: NumericRange{
			Mode: RangeModeFixed, Value: 60, MinVal: 60, MaxVal: 120,
		},

		ActivationEnabled:              false,
		ActivationMode:                 ActivationModeAuto,
		ActivationTimeout:              239,
		TargetWindowTitle:              "",
		TargetWindowClass:              "",
		TargetProcessName:              "",
		ActivationTickIntervalMS:       DefaultActivationTickIntervalMS(),
		ActivationAutoProcessBlacklist: DefaultActivationAutoProcessBlacklist(),
	}
}

func clamp(v, lo, hi int) int {
	if v < lo {
		return lo
	}
	if v > hi {
		return hi
	}
	return v
}
