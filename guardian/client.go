package guardian

import (
	"fmt"
	"time"

	"github.com/scjalliance/resourceful/guardian/transport"
	"github.com/scjalliance/resourceful/lease"
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

// Policies will attempt to remove the lease for the given resource and consumer.
func (c *Client) Policies() (response transport.PoliciesResponse, err error) {
	response, err = c.endpoint.Policies()
	if err != nil {
		if c.SelectEndpoint() == nil {
			response, err = c.endpoint.Policies()
		}
	}
	return
}

// Acquire will attempt to acquire a lease for the given resource and consumer.
func (c *Client) Acquire(subject lease.Subject, props lease.Properties) (response transport.AcquireResponse, err error) {
	response, err = c.endpoint.Acquire(subject, props)
	if err != nil {
		if c.SelectEndpoint() == nil {
			response, err = c.endpoint.Acquire(subject, props)
		}
	}
	return
}

// Release will attempt to remove the lease for the given resource and consumer.
func (c *Client) Release(subject lease.Subject) (response transport.ReleaseResponse, err error) {
	response, err = c.endpoint.Release(subject)
	if err != nil {
		if c.SelectEndpoint() == nil {
			response, err = c.endpoint.Release(subject)
		}
	}
	return
}
