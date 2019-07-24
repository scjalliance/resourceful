package guardian

import "errors"

var (
	// ErrEmptyEndpoint is returned when an action is taken on an empty
	// endpoint.
	ErrEmptyEndpoint = errors.New("empty endpoint")

	// ErrStarted is returned when a start action is taken on
	// a lease maintainer that's already been started.
	ErrStarted = errors.New("the lease maintainer has already been started")

	// ErrClosed is returned when a stop or close action is taken on
	// something that's already been stopped or closed.
	ErrClosed = errors.New("already closed")
)
