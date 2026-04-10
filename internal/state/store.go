package state

import (
	"afk-hero/internal/domain"
	"sync"
	"time"
)

// progressEmitInterval is the minimum wall-clock interval between two
// consecutive change-callback invocations from SetProgress when the status
// key stays the same. 16 ms ≈ 60 Hz — enough for smooth UI, but cuts the
// ~200 calls/sec from unit-step animation down to ~60 while leaving the
// 4 Hz idle/interval polling completely unaffected.
const progressEmitInterval = 16 * time.Millisecond

// ChangeCallback is invoked when the applied state changes.
type ChangeCallback func(settings domain.Settings, state domain.State, progress float64, statusKey string)

// Store holds the applied runtime state of the application.
// It is the single source of truth for current settings and runtime state.
type Store struct {
	mu               sync.RWMutex
	settings         domain.Settings
	state            domain.State
	progress         float64 // 0.0 to 1.0
	statusKey        string  // i18n key for current status
	lastProgressEmit time.Time
	onChange         ChangeCallback
}

// NewStore creates a new state store with initial settings.
func NewStore(initial domain.Settings, onChange ChangeCallback) *Store {
	state := domain.StateDisabled
	if initial.Enabled {
		state = domain.StateWaitingForInactivity
	}
	return &Store{
		settings:  initial,
		state:     state,
		statusKey: "status." + string(state),
		onChange:  onChange,
	}
}

// Settings returns the current applied settings.
func (s *Store) Settings() domain.Settings {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.settings
}

// UpdateSettings applies new settings atomically.
func (s *Store) UpdateSettings(settings domain.Settings) {
	s.mu.Lock()
	s.settings = settings
	cb := s.onChange
	state := s.state
	progress := s.progress
	statusKey := s.statusKey
	s.mu.Unlock()

	if cb != nil {
		cb(settings, state, progress, statusKey)
	}
}

// State returns the current runtime state.
func (s *Store) State() domain.State {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.state
}

// SetState updates the runtime state.
func (s *Store) SetState(state domain.State) {
	s.mu.Lock()
	s.state = state
	s.statusKey = "status." + string(state)
	settings := s.settings
	cb := s.onChange
	progress := s.progress
	statusKey := s.statusKey
	s.mu.Unlock()

	if cb != nil {
		cb(settings, state, progress, statusKey)
	}
}

// Progress returns the current progress (0.0 - 1.0) and status i18n key.
func (s *Store) Progress() (float64, string) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.progress, s.statusKey
}

// SetProgress updates the progress and status key. The change callback is
// always invoked on status-key transitions (state changes) but is rate-limited
// to progressEmitInterval within the same status key. This throttles
// high-frequency animation ticks (~200/sec) to ~60 Hz while keeping the
// slower idle/interval polling (4 Hz) fully responsive.
func (s *Store) SetProgress(progress float64, statusKey string) {
	s.mu.Lock()
	previousStatusKey := s.statusKey

	s.progress = progress
	if statusKey != "" {
		s.statusKey = statusKey
	}

	currentStatusKey := s.statusKey
	now := time.Now()
	shouldEmit := currentStatusKey != previousStatusKey || now.Sub(s.lastProgressEmit) >= progressEmitInterval
	if !shouldEmit {
		s.mu.Unlock()
		return
	}

	s.lastProgressEmit = now
	settings := s.settings
	state := s.state
	cb := s.onChange
	s.mu.Unlock()

	if cb != nil {
		cb(settings, state, progress, currentStatusKey)
	}
}

// Enabled returns the current enabled state.
func (s *Store) Enabled() bool {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.settings.Enabled
}

// SetEnabled updates the enabled setting.
func (s *Store) SetEnabled(enabled bool) {
	s.mu.Lock()
	s.settings.Enabled = enabled
	settings := s.settings
	state := s.state
	cb := s.onChange
	progress := s.progress
	statusKey := s.statusKey
	s.mu.Unlock()

	if cb != nil {
		cb(settings, state, progress, statusKey)
	}
}
