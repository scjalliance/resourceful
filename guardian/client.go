package guardian

import (
	"context"
	"fmt"
	"time"

	"github.com/scjalliance/resourceful/environment"
	"github.com/scjalliance/resourceful/guardian/transport"
)

// Client coordinates resource leasing with a resourceful guardian server.
type Client struct {
	endpoints []Endpoint
	endpoint  Endpoint
}

// NewClient creates a new guardian client that relies on the given endpoints.
func NewClient(endpoints ...Endpoint) (*Client, error) {
	c := &Client{
		endpoints: endpoints,
	}
	if err := c.SelectEndpoint(); err != nil {
		return nil, err
	}
	return c, nil
}

// SelectEndpoint looks for a healthy endpoint and selects the first one that
// it finds for use in future queries. It returns an error if it failed to
// contact one.
func (c *Client) SelectEndpoint() error {
	const rounds = 2

	var failed []error
	for round := 0; round < rounds; round++ {
		timeout := DefaultHealthTimeout * time.Duration(round+1)
		for _, endpoint := range c.endpoints {
			health, err := endpoint.HealthWithTimeout(timeout)
			if err != nil {
				failed = append(failed, err)
				continue
			}
			if !health.OK {
				continue
			}
			c.endpoint = endpoint
			return nil
		}
	}

	const task = "endpoint selection"

	f := len(failed)
	switch {
	case f == 1:
		return fmt.Errorf("%s failed: %v", task, failed[0])
	case f > 1:
		return fmt.Errorf("%s failed: %d attempts to connect to %d servers failed: %v", task, f, len(c.endpoints), failed[0])
	default:
		return fmt.Errorf("%s failed: no servers available", task)
	}
}

// Acquire will attempt to acquire a lease for the given resource and consumer.
func (c *Client) Acquire(resource, consumer, instance string, env environment.Environment) (response transport.AcquireResponse, err error) {
	response, err = c.endpoint.Acquire(resource, consumer, instance, env)
	if err != nil {
		if c.SelectEndpoint() == nil {
			response, err = c.endpoint.Acquire(resource, consumer, instance, env)
		}
	}
	return
}

// Maintain will attempt to acquire and automatically renew a lease until ctx
// is cancelled. When ctx is cancelled the lease will be released.
//
// The result of each acquisition or observation will be retuned via the
// lease manager to all listeners.
//
// If retry is a non-zero duration the maintainer will attempt to acquire a
// lease on an interval of retry.
func (c *Client) Maintain(ctx context.Context, resource, consumer, instance string, env environment.Environment, retry time.Duration) (lm *LeaseMaintainer) {
	return newLeaseMaintainer(ctx, c, resource, consumer, instance, env, retry)
}

// Release will attempt to remove the lease for the given resource and consumer.
func (c *Client) Release(resource, consumer, instance string) (response transport.ReleaseResponse, err error) {
	response, err = c.endpoint.Release(resource, consumer, instance)
	if err != nil {
		if c.SelectEndpoint() == nil {
			response, err = c.endpoint.Release(resource, consumer, instance)
		}
	}
	return
}
