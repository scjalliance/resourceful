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
