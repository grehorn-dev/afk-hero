package domain

// WindowActivationState describes the runtime progress of the window activation monitor.
type WindowActivationState struct {
	TargetFound bool    `json:"targetFound"`
	Tracking    bool    `json:"tracking"`
	Progress    float64 `json:"progress"`
	ProcessName string  `json:"processName"`
	WindowTitle string  `json:"windowTitle"`
}
