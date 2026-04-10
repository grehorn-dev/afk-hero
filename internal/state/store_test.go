package state

import (
	"afk-hero/internal/domain"
	"testing"
)

func TestStore_InitialState(t *testing.T) {
	settings := domain.DefaultSettings()
	store := NewStore(settings, nil)

	if store.State() != domain.StateWaitingForInactivity {
		t.Errorf("expected WaitingForInactivity when enabled, got %s", store.State())
	}

	settings.Enabled = false
	store2 := NewStore(settings, nil)
	if store2.State() != domain.StateDisabled {
		t.Errorf("expected Disabled when not enabled, got %s", store2.State())
	}
}

func TestStore_UpdateSettings(t *testing.T) {
	settings := domain.DefaultSettings()
	var callbackCalled bool
	store := NewStore(settings, func(s domain.Settings, st domain.State, progress float64, statusKey string) {
		callbackCalled = true
	})

	newSettings := settings
	newSettings.Language = "rus"
	store.UpdateSettings(newSettings)

	if store.Settings().Language != "rus" {
		t.Error("settings not updated")
	}
	if !callbackCalled {
		t.Error("callback not invoked on settings update")
	}
}

func TestStore_SetEnabled(t *testing.T) {
	settings := domain.DefaultSettings()
	store := NewStore(settings, nil)

	store.SetEnabled(false)
	if store.Enabled() {
		t.Error("expected enabled to be false")
	}

	store.SetEnabled(true)
	if !store.Enabled() {
		t.Error("expected enabled to be true")
	}
}

func TestStore_Progress(t *testing.T) {
	store := NewStore(domain.DefaultSettings(), nil)

	store.SetProgress(0.5, "status.Animating")
	progress, key := store.Progress()
	if progress != 0.5 {
		t.Errorf("expected progress 0.5, got %f", progress)
	}
	if key != "status.Animating" {
		t.Errorf("expected status key status.Animating, got %s", key)
	}
}

func TestStore_SetState(t *testing.T) {
	store := NewStore(domain.DefaultSettings(), nil)

	store.SetState(domain.StateAnimating)
	if store.State() != domain.StateAnimating {
		t.Errorf("expected Animating, got %s", store.State())
	}
}

func TestStore_ConcurrentAccess(t *testing.T) {
	store := NewStore(domain.DefaultSettings(), nil)
	done := make(chan struct{})

	go func() {
		for i := 0; i < 1000; i++ {
			store.SetEnabled(i%2 == 0)
		}
		done <- struct{}{}
	}()

	go func() {
		for i := 0; i < 1000; i++ {
			_ = store.Settings()
			_ = store.State()
			_ = store.Enabled()
		}
		done <- struct{}{}
	}()

	<-done
	<-done
}
