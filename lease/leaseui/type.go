package leaseui

// Type is a type of lease user interface.
type Type int

// Lease types
const (
	None Type = iota
	Queued
	Connected
	Disconnected
)

// String returns a string representation of the type.
func (t Type) String() string {
	switch t {
	case None:
		return "none"
	case Queued:
		return "queued"
	case Connected:
		return "connected"
	case Disconnected:
		return "disconnected"
	default:
		return "unknown"
	}
}
