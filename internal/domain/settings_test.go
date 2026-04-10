package domain

import (
	"testing"
)

func TestValidShape(t *testing.T) {
	for _, s := range AllShapes() {
		if !ValidShape(s) {
			t.Errorf("AllShapes() contains invalid shape: %s", s)
		}
	}
	if ValidShape("Hexagon") {
		t.Error("expected Hexagon to be invalid")
	}
}

func TestValidDirection(t *testing.T) {
	for _, d := range AllDirections() {
		if !ValidDirection(d) {
			t.Errorf("AllDirections() contains invalid direction: %s", d)
		}
	}
	if ValidDirection("Diagonal") {
		t.Error("expected Diagonal to be invalid")
	}
}

func TestValidTheme(t *testing.T) {
	for _, theme := range AllThemes() {
		if !ValidTheme(theme) {
			t.Errorf("AllThemes() contains invalid theme: %s", theme)
		}
	}
	if ValidTheme("System") {
		t.Error("expected System to be invalid")
	}
}

func TestRandomShapes_ExcludeSegmentAndOnePixel(t *testing.T) {
	for _, s := range RandomShapes() {
		if s == ShapeSegment || s == ShapeOnePixel {
			t.Errorf("RandomShapes() should not contain %s", s)
		}
	}
}

func TestNumericRange_ValidateFixed(t *testing.T) {
	tests := []struct {
		name    string
		r       NumericRange
		min     int
		max     int
		wantErr bool
	}{
		{"valid fixed", NumericRange{Mode: RangeModeFixed, Value: 50}, 1, 100, false},
		{"below min", NumericRange{Mode: RangeModeFixed, Value: 0}, 1, 100, true},
		{"above max", NumericRange{Mode: RangeModeFixed, Value: 101}, 1, 100, true},
		{"at min", NumericRange{Mode: RangeModeFixed, Value: 1}, 1, 100, false},
		{"at max", NumericRange{Mode: RangeModeFixed, Value: 100}, 1, 100, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.r.Validate(tt.min, tt.max)
			if (err != nil) != tt.wantErr {
				t.Errorf("Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestNumericRange_ValidateRandom(t *testing.T) {
	tests := []struct {
		name    string
		r       NumericRange
		min     int
		max     int
		wantErr bool
	}{
		{"valid range", NumericRange{Mode: RangeModeRandom, MinVal: 10, MaxVal: 50}, 1, 100, false},
		{"min below bound", NumericRange{Mode: RangeModeRandom, MinVal: 0, MaxVal: 50}, 1, 100, true},
		{"max above bound", NumericRange{Mode: RangeModeRandom, MinVal: 10, MaxVal: 101}, 1, 100, true},
		{"min > max", NumericRange{Mode: RangeModeRandom, MinVal: 60, MaxVal: 50}, 1, 100, true},
		{"equal min max", NumericRange{Mode: RangeModeRandom, MinVal: 50, MaxVal: 50}, 1, 100, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.r.Validate(tt.min, tt.max)
			if (err != nil) != tt.wantErr {
				t.Errorf("Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestNumericRange_Normalize(t *testing.T) {
	r := NumericRange{Mode: RangeModeRandom, MinVal: 200, MaxVal: 50}
	n := r.Normalize(1, 100)
	if n.MinVal > n.MaxVal {
		t.Errorf("Normalize should fix ordering: got min=%d max=%d", n.MinVal, n.MaxVal)
	}
	if n.MaxVal > 100 {
		t.Errorf("Normalize should clamp max to 100, got %d", n.MaxVal)
	}

	r2 := NumericRange{Mode: "InvalidMode", Value: 50}
	n2 := r2.Normalize(1, 100)
	if n2.Mode != RangeModeFixed {
		t.Errorf("Normalize should default unknown mode to Fixed, got %s", n2.Mode)
	}
}

func TestDefaultSettings(t *testing.T) {
	s := DefaultSettings()
	if s.Shape != ShapeRandom {
		t.Errorf("expected default shape Random, got %s", s.Shape)
	}
	if s.Direction != DirectionRandom {
		t.Errorf("expected default direction Random, got %s", s.Direction)
	}
	if !s.Enabled {
		t.Error("expected default enabled to be true")
	}
	if s.Advanced {
		t.Error("expected default advanced to be false")
	}
	if s.Logging {
		t.Error("expected default logging to be false")
	}
	if s.Language != "eng" {
		t.Errorf("expected default language eng, got %s", s.Language)
	}
	if s.Theme != ThemeDark {
		t.Errorf("expected default theme Dark, got %s", s.Theme)
	}
}

func TestRect_Dimensions(t *testing.T) {
	r := Rect{Left: 10, Top: 20, Right: 110, Bottom: 120}
	if r.Width() != 100 {
		t.Errorf("expected width 100, got %d", r.Width())
	}
	if r.Height() != 100 {
		t.Errorf("expected height 100, got %d", r.Height())
	}
}

func TestRect_Contains(t *testing.T) {
	r := Rect{Left: 0, Top: 0, Right: 100, Bottom: 100}
	if !r.Contains(Point{50, 50}) {
		t.Error("expected (50,50) to be inside rect")
	}
	if r.Contains(Point{100, 100}) {
		t.Error("expected (100,100) to be outside rect (exclusive)")
	}
	if r.Contains(Point{-1, 50}) {
		t.Error("expected (-1,50) to be outside rect")
	}
}

func TestWindowFingerprint(t *testing.T) {
	fp := WindowFingerprint{ProcessName: "Game.EXE", WindowClass: "UnrealWindow"}
	if !fp.Valid() {
		t.Fatal("expected fingerprint to be valid")
	}

	if !fp.Matches(WindowInfo{ProcessName: "game.exe", ClassName: "unrealwindow"}) {
		t.Fatal("expected case-insensitive fingerprint match")
	}

	if fp.Matches(WindowInfo{ProcessName: "launcher.exe", ClassName: "UnrealWindow"}) {
		t.Fatal("did not expect mismatched process name to match")
	}
}

func TestSettingsEffectiveUsesDefaultsWhenAdvancedIsDisabled(t *testing.T) {
	settings := DefaultSettings()
	settings.Advanced = false
	settings.Logging = true
	settings.Shape = ShapeOnePixel
	settings.Direction = DirectionLeft
	settings.Distance = NumericRange{Mode: RangeModeFixed, Value: 777}
	settings.Speed = NumericRange{Mode: RangeModeFixed, Value: 42}
	settings.Interval = NumericRange{Mode: RangeModeFixed, Value: 99}

	effective := settings.Effective()
	defaults := DefaultSettings()

	if effective.Logging != defaults.Logging {
		t.Fatalf("expected default logging %v, got %v", defaults.Logging, effective.Logging)
	}
	if effective.Shape != defaults.Shape {
		t.Fatalf("expected default shape %s, got %s", defaults.Shape, effective.Shape)
	}
	if effective.Direction != defaults.Direction {
		t.Fatalf("expected default direction %s, got %s", defaults.Direction, effective.Direction)
	}
	if effective.Distance != defaults.Distance {
		t.Fatalf("expected default distance %#v, got %#v", defaults.Distance, effective.Distance)
	}
	if effective.Speed != defaults.Speed {
		t.Fatalf("expected default speed %#v, got %#v", defaults.Speed, effective.Speed)
	}
	if effective.Interval != settings.Interval {
		t.Fatalf("expected interval to stay user-defined, got %#v", effective.Interval)
	}
}
