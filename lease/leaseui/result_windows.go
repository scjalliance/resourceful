// +build windows

package leaseui

// Result is a leaseui result. It explains why a lease user interface ended.
type Result int

const (
	// Success is returned when the objective of the dialog was completed
	// successfully.
	Success Result = iota

	// UserClosed is returned when a user closes a window.
	UserClosed

	// UserCancelled is returned when a user presses the cancel button.
	UserCancelled

	// ContextCancelled is returned when a window context is cancelled.
	ContextCancelled

	// ChannelClosed is returned when the update source for a window is closed.
	ChannelClosed
)
