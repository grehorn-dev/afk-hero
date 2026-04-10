package config

import (
	"afk-hero/internal/domain"
	"strings"
)

const (
	MinDistance        = 1
	DefaultMaxDistance = 2000 // Fallback; real max depends on screen dimensions

	MinInterval = 1
	MaxInterval = 3600

	MinSpeed = 1
	MaxSpeed = 60

	MinInactivity = 1
	MaxInactivity = 3600

	MinActivationTimeout = 1
	MaxActivationTimeout = 3600

	MinActivationTickIntervalMS = 100
	MaxActivationTickIntervalMS = 60000
)

// NormalizeSettings clamps and fixes all settings values to valid ranges.
func NormalizeSettings(s domain.Settings) domain.Settings {
	// Validate shape
	if !domain.ValidShape(s.Shape) {
		s.Shape = domain.ShapeRandom
	}

	// Validate direction
	if !domain.ValidDirection(s.Direction) {
		s.Direction = domain.DirectionRandom
	}

	// Normalize ranges
	s.Distance = s.Distance.Normalize(MinDistance, DefaultMaxDistance)
	s.Interval = s.Interval.Normalize(MinInterval, MaxInterval)
	s.Speed = s.Speed.Normalize(MinSpeed, MaxSpeed)
	s.Inactivity = s.Inactivity.Normalize(MinInactivity, MaxInactivity)

	// Normalize activation timeout
	if s.ActivationTimeout < MinActivationTimeout {
		s.ActivationTimeout = MinActivationTimeout
	}
	if s.ActivationTimeout > MaxActivationTimeout {
		s.ActivationTimeout = MaxActivationTimeout
	}
	if s.ActivationTickIntervalMS == 0 {
		s.ActivationTickIntervalMS = domain.DefaultActivationTickIntervalMS()
	}
	if s.ActivationTickIntervalMS < MinActivationTickIntervalMS {
		s.ActivationTickIntervalMS = MinActivationTickIntervalMS
	}
	if s.ActivationTickIntervalMS > MaxActivationTickIntervalMS {
		s.ActivationTickIntervalMS = MaxActivationTickIntervalMS
	}

	// Validate activation mode
	if s.ActivationMode != domain.ActivationModeAuto && s.ActivationMode != domain.ActivationModeManual {
		s.ActivationMode = domain.ActivationModeAuto
	}
	s.ActivationAutoProcessBlacklist = normalizeProcessBlacklist(s.ActivationAutoProcessBlacklist)
	if s.ActivationMode == domain.ActivationModeManual {
		if domain.NormalizeProcessName(s.TargetProcessName) == "" || strings.TrimSpace(s.TargetWindowClass) == "" {
			s.ActivationMode = domain.ActivationModeAuto
			s.TargetWindowTitle = ""
			s.TargetWindowClass = ""
			s.TargetProcessName = ""
		}
	}

	// Validate language (basic check: non-empty, 3-letter code)
	if len(s.Language) != 3 {
		s.Language = "eng"
	}

	// Validate theme
	if !domain.ValidTheme(s.Theme) {
		s.Theme = domain.ThemeDark
	}

	return s
}

func normalizeProcessBlacklist(processes []string) []string {
	if processes == nil {
		return domain.DefaultActivationAutoProcessBlacklist()
	}

	normalized := make([]string, 0, len(processes))
	seen := make(map[string]struct{}, len(processes))
	for _, process := range processes {
		name := domain.NormalizeProcessName(process)
		if name == "" {
			continue
		}
		if _, exists := seen[name]; exists {
			continue
		}

		seen[name] = struct{}{}
		normalized = append(normalized, name)
	}

	return normalized
}
