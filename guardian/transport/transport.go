// Package transport defines JSON-compatible guardian messages.
package transport

import (
	"github.com/scjalliance/resourceful/environment"
	"github.com/scjalliance/resourceful/lease"
)

// Request represents a request from a resourceful client.
type Request struct {
	Resource    string                  `json:"resource,omitempty"`
	Consumer    string                  `json:"consumer,omitempty"`
	Instance    string                  `json:"instance,omitempty"`
	Environment environment.Environment `json:"environment,omitempty"`
}

// HealthResponse reports the health of a guardian server.
type HealthResponse struct {
	OK bool `json:"ok"`
}

// LeasesResponse reports the current sef of leases for a resource.
type LeasesResponse struct {
	Request
	Leases lease.Set `json:"leases"`
}

// AcquireResponse reports the result of a resource acquisition attempt.
type AcquireResponse struct {
	Request
	Lease   lease.Lease `json:"lease,omitempty"`
	Leases  lease.Set   `json:"leases"`
	Message string      `json:"message,omitempty"`
}

// ReleaseResponse reports the result of a resource release attempt.
type ReleaseResponse struct {
	Request
	Success bool   `json:"success"`
	Message string `json:"message,omitempty"`
}

// UpdateResponse represents a lease environment update request.
type UpdateResponse struct {
	Request
	Success bool   `json:"success"`
	Message string `json:"message,omitempty"`
}
