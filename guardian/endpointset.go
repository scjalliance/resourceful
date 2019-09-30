package guardian

import (
	"context"
	"fmt"
)

// EndpointSet is a set of endpoints.
type EndpointSet []Endpoint

// Select looks for a healthy endpoint within the set and returns the first
// one that it finds. It returns an error if it failed to contact one or
// the context is cancelled.
func (s EndpointSet) Select(ctx context.Context) (Endpoint, error) {
	if err := ctx.Err(); err != nil {
		return "", err
	}

	const rounds = 2

	var failed []error
	for round := 0; round < rounds; round++ {
		for _, endpoint := range s {
			health, err := endpoint.Health(ctx)
			if err != nil {
				failed = append(failed, err)
				continue
			}
			if !health.OK {
				continue
			}
			return endpoint, nil
		}
	}

	const task = "endpoint selection"

	f := len(failed)
	switch {
	case f == 1:
		return "", fmt.Errorf("%s failed: %v", task, failed[0])
	case f > 1:
		return "", fmt.Errorf("%s failed: %d attempts to connect to %d servers failed: %v", task, f, len(s), failed[0])
	default:
		return "", fmt.Errorf("%s failed: no servers available", task)
	}
}

// Resolve returns s. It allows a static endpoint set to full the Resolver
// interface.
func (s EndpointSet) Resolve(ctx context.Context) (EndpointSet, error) {
	return s, nil
}
