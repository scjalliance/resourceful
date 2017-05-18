// Package transport defines JSON-compatible guardian messages.
package transport

import (
	"github.com/scjalliance/resourceful/environment"
	"github.com/scjalliance/resourceful/lease"
)

// HealthResponse reports the health of a guardian server.
type HealthResponse struct {
	OK bool `json:"ok"`
}

// Request represents a request from a resourceful client.
type Request struct {
	Resource    string                  `json:"resource,omitempty"`
	Consumer    string                  `json:"consumer,omitempty"`
	Environment environment.Environment `json:"environment,omitempty"`
}

// AcquireResponse reports the result of a resource acquisition attempt.
type AcquireResponse struct {
	Request
	Accepted bool      `json:"accepted"`
	Message  string    `json:"message,omitempty"`
	Leases   lease.Set `json:"lease"`
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
