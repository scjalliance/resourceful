package enforcer

import "fmt"

// Event is an event that can be logged.
type Event interface {
	ID() uint32
	IsDebug() bool
	String() string
}

// Event IDs.
const (
	ServiceEventID = 100
	SessionEventID = 200
	ProcessEventID = 300
)

// ServiceEvent is an event originating from session management.
type ServiceEvent struct {
	Msg   string
	Debug bool
}

// ID returns the ID of the event.
func (e ServiceEvent) ID() uint32 {
	return ServiceEventID
}

// IsDebug returns true if the event is intended for development and
// debugging.
func (e ServiceEvent) IsDebug() bool {
	return e.Debug
}

// String returns a string representation of the event.
func (e ServiceEvent) String() string {
	return fmt.Sprintf("[SERVICE] %s", e.Msg)
}

// ProcessEvent is an event originating from process management.
type ProcessEvent struct {
	ProcessName string
	InstanceID  string
	Msg         string
	Debug       bool
}

// ID returns the ID of the event.
func (e ProcessEvent) ID() uint32 {
	return ProcessEventID
}

// IsDebug returns true if the event is intended for development and
// debugging.
func (e ProcessEvent) IsDebug() bool {
	return e.Debug
}

// String returns a string representation of the event.
func (e ProcessEvent) String() string {
	return fmt.Sprintf("[PROCESS] %s (%s): %s", e.ProcessName, e.InstanceID, e.Msg)
}

// SessionEvent is an event originating from session management.
type SessionEvent struct {
	SessionID     uint32
	WindowStation string
	SessionUser   string
	Msg           string
	Debug         bool
}

// ID returns the ID of the event.
func (e SessionEvent) ID() uint32 {
	return SessionEventID
}

// IsDebug returns true if the event is intended for development and
// debugging.
func (e SessionEvent) IsDebug() bool {
	return e.Debug
}

// String returns a string representation of the event.
func (e SessionEvent) String() string {
	if e.SessionUser == "" {
		return fmt.Sprintf("[SESSION] %s (session %d): %s", e.WindowStation, e.SessionID, e.Msg)
	}
	return fmt.Sprintf("[SESSION] %s (session %d, %s): %s", e.WindowStation, e.SessionID, e.SessionUser, e.Msg)
}
