package guardian

import (
	"context"
	"errors"
	"sync"
	"time"

	"github.com/scjalliance/resourceful/guardian/transport"
	"github.com/scjalliance/resourceful/lease"
)

var (
	errSelectionInterval = errors.New("the selection interval has not been reached")
	errResolverInterval  = errors.New("the resolver interval has not been reached")
)

// Client coordinates resource leasing with a resourceful guardian server.
type Client struct {
	resolver Resolver

	mutex     sync.RWMutex
	endpoints EndpointSet
	endpoint  Endpoint

	selection sync.Mutex
	resolved  time.Time
	selected  time.Time
}

// NewClient creates a new guardian client that retrieves endpoints from
// resolver.
func NewClient(resolver Resolver) *Client {
	return &Client{
		resolver: resolver,
	}
}

// Resolve causes the client to query its resolver for an updated set of
// endpoints. It looks for a healthy endpoint and selects the first one that
// it finds for use in future queries. It returns an error if it fails to
// select a healthy endpoint.
func (c *Client) Resolve(ctx context.Context) error {
	if err := ctx.Err(); err != nil {
		return err
	}

	c.selection.Lock()
	defer c.selection.Unlock()

	endpoints, err := c.resolver.Resolve(ctx)
	if err != nil {
		return err
	}

	endpoint, err := endpoints.Select(ctx)
	if err != nil {
		return err
	}

	now := time.Now()
	c.resolved = now
	c.selected = now

	c.mutex.Lock()
	defer c.mutex.Unlock()

	c.endpoints = endpoints
	c.endpoint = endpoint
	return nil
}

// failover is called when an API call fails. It looks for a healthy endpoint
// and selects the first one that it finds for use in future queries. If no
// healthy endpoints can be found in the current set, it attempts to resolve
// a new endpoint set and select a healthy endpoint from the new set.
func (c *Client) failover(ctx context.Context, essential bool) (Endpoint, error) {
	if err := ctx.Err(); err != nil {
		return "", err
	}

	c.selection.Lock()
	defer c.selection.Unlock()

	now := time.Now()

	if !essential && !c.selected.IsZero() && now.Sub(c.selected) < 2*time.Second {
		// Don't select more than once every 2 seconds
		return "", errSelectionInterval
	}

	c.mutex.RLock()
	endpoints := c.endpoints
	c.mutex.RUnlock()

	// Step 1: Attempt to select a healthy endpoint from the current set
	endpoint, err := endpoints.Select(ctx)
	if err != nil {
		// Step 2: Ask the resolver for an updated set of endpoints
		if !essential && !c.resolved.IsZero() && now.Sub(c.resolved) < 10*time.Second {
			// Don't resolve more than once every 10 seconds
			return "", errResolverInterval
		}

		endpoints, err = c.resolver.Resolve(ctx)
		if err != nil {
			return "", err
		}

		// Step 3: Attempt to select a healthy endpoint from the updated set
		endpoint, err = endpoints.Select(ctx)
		if err != nil {
			return "", err
		}

		c.resolved = time.Now()
	}

	c.mutex.Lock()
	defer c.mutex.Unlock()

	c.endpoint = endpoint
	c.endpoints = endpoints

	return endpoint, nil
}

// Policies will attempt to remove the lease for the given resource and consumer.
func (c *Client) Policies(ctx context.Context) (response transport.PoliciesResponse, err error) {
	c.mutex.RLock()
	endpoint := c.endpoint
	c.mutex.RUnlock()

	response, err = endpoint.Policies(ctx)
	if err != nil {
		if isContextErr(err) {
			return response, err
		}
		failover, err2 := c.failover(ctx, false)
		if err2 != nil {
			return response, err
		}
		return failover.Policies(ctx)
	}

	return response, nil
}

// Acquire will attempt to acquire a lease for subject based on the property
// set.
func (c *Client) Acquire(ctx context.Context, subject lease.Subject, props lease.Properties) (response transport.AcquireResponse, err error) {
	c.mutex.RLock()
	endpoint := c.endpoint
	c.mutex.RUnlock()

	response, err = endpoint.Acquire(ctx, subject, props)
	if err != nil {
		if isContextErr(err) {
			return response, err
		}
		failover, err2 := c.failover(ctx, false)
		if err2 != nil {
			return response, err
		}
		return failover.Acquire(ctx, subject, props)
	}

	return response, nil
}

// Release will attempt to remove the lease for the given resource and consumer.
func (c *Client) Release(ctx context.Context, subject lease.Subject) (response transport.ReleaseResponse, err error) {
	c.mutex.RLock()
	endpoint := c.endpoint
	c.mutex.RUnlock()

	response, err = endpoint.Release(ctx, subject)
	if err != nil {
		if isContextErr(err) {
			return response, err
		}
		failover, err2 := c.failover(ctx, true)
		if err2 != nil {
			return response, err
		}
		return failover.Release(ctx, subject)
	}

	return response, nil
}

func isContextErr(err error) bool {
	switch err {
	case context.DeadlineExceeded, context.Canceled:
		return true
	default:
		return false
	}
}
