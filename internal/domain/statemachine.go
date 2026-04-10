package domain

import (
	"fmt"
	"sync"
)

// Transition defines a valid state machine transition.
type Transition struct {
	From  State
	Event Event
	To    State
}

// transitions defines all valid state transitions.
var transitions = []Transition{
	// From Disabled
	{StateDisabled, EventEnable, StateWaitingForInactivity},

	// From WaitingForInactivity
	{StateWaitingForInactivity, EventDisable, StateDisabled},
	{StateWaitingForInactivity, EventInactivityReached, StateAnimating},
	{StateWaitingForInactivity, EventError, StateError},

	// From WaitingForInterval
	{StateWaitingForInterval, EventDisable, StateDisabled},
	{StateWaitingForInterval, EventIntervalElapsed, StateAnimating},
	{StateWaitingForInterval, EventUserInput, StateWaitingForInactivity},
	{StateWaitingForInterval, EventError, StateError},

	// From Animating
	{StateAnimating, EventDisable, StateDisabled},
	{StateAnimating, EventAnimationCompleted, StateWaitingForInterval},
	{StateAnimating, EventUserInput, StateWaitingForInactivity},
	{StateAnimating, EventError, StateError},

	// From Error
	{StateError, EventRecover, StateWaitingForInactivity},
	{StateError, EventDisable, StateDisabled},
}

// transitionMap builds a lookup map for fast transition validation.
var transitionMap map[State]map[Event]State

func init() {
	transitionMap = make(map[State]map[Event]State)
	for _, t := range transitions {
		if _, ok := transitionMap[t.From]; !ok {
			transitionMap[t.From] = make(map[Event]State)
		}
		transitionMap[t.From][t.Event] = t.To
	}
}

// StateChangeCallback is invoked when the state machine transitions.
type StateChangeCallback func(from, to State, event Event)

// StateMachine manages application runtime state transitions.
type StateMachine struct {
	mu       sync.RWMutex
	current  State
	onChange StateChangeCallback
}

// NewStateMachine creates a state machine starting in the given state.
func NewStateMachine(initial State, onChange StateChangeCallback) *StateMachine {
	return &StateMachine{
		current:  initial,
		onChange: onChange,
	}
}

// Current returns the current state.
func (sm *StateMachine) Current() State {
	sm.mu.RLock()
	defer sm.mu.RUnlock()
	return sm.current
}

// Send attempts a state transition for the given event.
// Returns the new state or an error if the transition is invalid.
//
// The onChange callback, if any, is invoked AFTER releasing the state
// machine mutex. This guarantees that callbacks which re-enter the
// state machine (directly or indirectly through Store/Engine callbacks)
// cannot deadlock.
func (sm *StateMachine) Send(event Event) (State, error) {
	sm.mu.Lock()

	evts, ok := transitionMap[sm.current]
	if !ok {
		current := sm.current
		sm.mu.Unlock()
		return current, fmt.Errorf("no transitions from state %s: %w",
			current, ErrInvalidTransition)
	}

	next, ok := evts[event]
	if !ok {
		current := sm.current
		sm.mu.Unlock()
		return current, fmt.Errorf("event %s not valid in state %s: %w",
			event, current, ErrInvalidTransition)
	}

	prev := sm.current
	sm.current = next
	onChange := sm.onChange
	sm.mu.Unlock()

	if onChange != nil {
		onChange(prev, next, event)
	}

	return next, nil
}

// CanSend checks whether the event is valid in the current state without transitioning.
func (sm *StateMachine) CanSend(event Event) bool {
	sm.mu.RLock()
	defer sm.mu.RUnlock()
	evts, ok := transitionMap[sm.current]
	if !ok {
		return false
	}
	_, ok = evts[event]
	return ok
}

// ValidTransitions returns the valid transitions from the current state.
func ValidTransitions() []Transition {
	return transitions
}

// ErrInvalidTransition is the sentinel for invalid transitions.
var ErrInvalidTransition = fmt.Errorf("invalid state transition")
