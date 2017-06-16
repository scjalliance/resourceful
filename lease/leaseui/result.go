package leaseui

// Result is the outcome of a lease user interface.
type Result int

const (
	// Success is returned when the objective of the user interface was completed
	// successfully.
	Success Result = iota

	// UserClosed is returned when a user interface was closed by the user.
	UserClosed

	// UserCancelled is returned when an action is canclled by the user.
	UserCancelled

	// ContextCancelled is returned when a context is cancelled.
	ContextCancelled

	// ChannelClosed is returned when the update source for a user interface is
	// closed.
	ChannelClosed
)
