package guardian

import (
	"context"
)

// A Resolver is a function that collects a set of endpoints.
type Resolver interface {
	Resolve(ctx context.Context) (EndpointSet, error)
}
