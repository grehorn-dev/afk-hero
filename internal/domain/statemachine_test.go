package domain

import (
	"errors"
	"testing"
)

func mustSend(t *testing.T, sm *StateMachine, event Event) State {
	t.Helper()

	state, err := sm.Send(event)
	if err != nil {
		t.Fatalf("event %s: unexpected error: %v", event, err)
	}

	return state
}

func TestStateMachine_InitialState(t *testing.T) {
	sm := NewStateMachine(StateDisabled, nil)
	if sm.Current() != StateDisabled {
		t.Errorf("expected initial state Disabled, got %s", sm.Current())
	}
}

func TestStateMachine_EnableDisable(t *testing.T) {
	sm := NewStateMachine(StateDisabled, nil)

	state, err := sm.Send(EventEnable)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if state != StateWaitingForInactivity {
		t.Errorf("expected WaitingForInactivity, got %s", state)
	}

	state, err = sm.Send(EventDisable)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if state != StateDisabled {
		t.Errorf("expected Disabled, got %s", state)
	}
}

func TestStateMachine_FullCycleNoInterruption(t *testing.T) {
	sm := NewStateMachine(StateDisabled, nil)

	steps := []struct {
		event    Event
		expected State
	}{
		{EventEnable, StateWaitingForInactivity},
		{EventInactivityReached, StateAnimating},
		{EventAnimationCompleted, StateWaitingForInterval},
		{EventIntervalElapsed, StateAnimating},
	}

	for _, step := range steps {
		state, err := sm.Send(step.event)
		if err != nil {
			t.Fatalf("event %s: unexpected error: %v", step.event, err)
		}
		if state != step.expected {
			t.Errorf("event %s: expected %s, got %s", step.event, step.expected, state)
		}
	}
}

func TestStateMachine_UserInterruption(t *testing.T) {
	sm := NewStateMachine(StateDisabled, nil)

	mustSend(t, sm, EventEnable)
	mustSend(t, sm, EventInactivityReached)

	state, err := sm.Send(EventUserInput)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if state != StateWaitingForInactivity {
		t.Errorf("expected WaitingForInactivity, got %s", state)
	}
}

func TestStateMachine_IntervalUserInterruption(t *testing.T) {
	sm := NewStateMachine(StateDisabled, nil)

	mustSend(t, sm, EventEnable)
	mustSend(t, sm, EventInactivityReached)
	mustSend(t, sm, EventAnimationCompleted)

	state, err := sm.Send(EventUserInput)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if state != StateWaitingForInactivity {
		t.Errorf("expected WaitingForInactivity, got %s", state)
	}
}

func TestStateMachine_ErrorRecovery(t *testing.T) {
	sm := NewStateMachine(StateDisabled, nil)
	mustSend(t, sm, EventEnable)

	state, err := sm.Send(EventError)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if state != StateError {
		t.Errorf("expected Error, got %s", state)
	}

	state, err = sm.Send(EventRecover)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if state != StateWaitingForInactivity {
		t.Errorf("expected WaitingForInactivity, got %s", state)
	}
}

func TestStateMachine_InvalidTransition(t *testing.T) {
	sm := NewStateMachine(StateDisabled, nil)

	_, err := sm.Send(EventInactivityReached)
	if err == nil {
		t.Fatal("expected error for invalid transition")
	}
	if !errors.Is(err, ErrInvalidTransition) {
		t.Errorf("expected ErrInvalidTransition, got %v", err)
	}
}

func TestStateMachine_CallbackInvoked(t *testing.T) {
	var called bool
	var fromState, toState State
	var triggeredEvent Event

	cb := func(from, to State, event Event) {
		called = true
		fromState = from
		toState = to
		triggeredEvent = event
	}

	sm := NewStateMachine(StateDisabled, cb)
	mustSend(t, sm, EventEnable)

	if !called {
		t.Fatal("callback was not invoked")
	}
	if fromState != StateDisabled || toState != StateWaitingForInactivity || triggeredEvent != EventEnable {
		t.Errorf("callback got from=%s to=%s event=%s", fromState, toState, triggeredEvent)
	}
}

func TestStateMachine_CanSend(t *testing.T) {
	sm := NewStateMachine(StateDisabled, nil)

	if !sm.CanSend(EventEnable) {
		t.Error("expected CanSend(Enable) to be true in Disabled state")
	}
	if sm.CanSend(EventInactivityReached) {
		t.Error("expected CanSend(InactivityReached) to be false in Disabled state")
	}
}

func TestStateMachine_DisableFromAnyActiveState(t *testing.T) {
	activeStates := []struct {
		setup func(sm *StateMachine)
		state State
	}{
		{func(sm *StateMachine) { mustSend(t, sm, EventEnable) }, StateWaitingForInactivity},
		{func(sm *StateMachine) {
			mustSend(t, sm, EventEnable)
			mustSend(t, sm, EventInactivityReached)
		}, StateAnimating},
		{func(sm *StateMachine) {
			mustSend(t, sm, EventEnable)
			mustSend(t, sm, EventInactivityReached)
			mustSend(t, sm, EventAnimationCompleted)
		}, StateWaitingForInterval},
		{func(sm *StateMachine) {
			mustSend(t, sm, EventEnable)
			mustSend(t, sm, EventError)
		}, StateError},
	}

	for _, tc := range activeStates {
		sm := NewStateMachine(StateDisabled, nil)
		tc.setup(sm)
		if sm.Current() != tc.state {
			t.Errorf("setup didn't reach expected state %s, got %s", tc.state, sm.Current())
			continue
		}
		state, err := sm.Send(EventDisable)
		if err != nil {
			t.Errorf("from %s: Disable failed: %v", tc.state, err)
			continue
		}
		if state != StateDisabled {
			t.Errorf("from %s: expected Disabled, got %s", tc.state, state)
		}
	}
}
