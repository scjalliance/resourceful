package guardian

import "errors"

var (
	// ErrEmptyEndpoint is returned when an action is taken on an empty
	// endpoint.
	ErrEmptyEndpoint = errors.New("empty endpoint")
)
