package domain

// State represents the runtime state of the application.
type State string

const (
	StateDisabled             State = "Disabled"
	StateWaitingForInactivity State = "WaitingForInactivity"
	StateWaitingForInterval   State = "WaitingForInterval"
	StateAnimating            State = "Animating"
	StateError                State = "Error"
)

// AllStates returns all recognized states.
func AllStates() []State {
	return []State{
		StateDisabled,
		StateWaitingForInactivity,
		StateWaitingForInterval,
		StateAnimating,
		StateError,
	}
}

// Event represents a state machine event.
type Event string

const (
	EventEnable             Event = "Enable"
	EventDisable            Event = "Disable"
	EventInactivityReached  Event = "InactivityReached"
	EventIntervalElapsed    Event = "IntervalElapsed"
	EventAnimationCompleted Event = "AnimationCompleted"
	EventUserInput          Event = "UserInput"
	EventError              Event = "Error"
	EventRecover            Event = "Recover"
)

// RuntimeState is the typed payload emitted on the "state:changed" Wails
// event and returned by App.GetState. Keeping it a concrete struct (instead
// of a bare map[string]any) gives the TS layer a real generated type and
// catches field renames at build time.
type RuntimeState struct {
	State     State   `json:"state"`
	Enabled   bool    `json:"enabled"`
	Progress  float64 `json:"progress"`
	StatusKey string  `json:"statusKey"`
}
