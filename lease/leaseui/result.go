package leaseui

// Result is the outcome of a lease user interface.
type Result int

const (
	// Failure is returned when a user interface failed in some unexpected way.
	Failure Result = iota

	// Success is returned when the objective of the user interface was completed
	// successfully.
	Success

	// UserClosed is returned when a user interface was closed by the user.
	UserClosed

	// UserCancelled is returned when an action is canclled by the user.
	UserCancelled

	// ContextCancelled is returned when a context is cancelled.
	ContextCancelled
)
