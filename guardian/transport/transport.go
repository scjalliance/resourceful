// Package transport defines JSON-compatible guardian messages.
package transport

import (
	"time"

	"github.com/scjalliance/resourceful/environment"
	"github.com/scjalliance/resourceful/lease"
)

// HealthResponse reports the health of a guardian server.
type HealthResponse struct {
	OK bool `json:"ok"`
}

// Request represents a request from a resourceful client.
type Request struct {
	Resource    string                  `json:"resource"`
	Consumer    string                  `json:"consumer"`
	Environment environment.Environment `json:"environment,omitempty"`
}

// AcquireResponse reports the result of a resource acquisition attempt.
type AcquireResponse struct {
	Request
	Accepted bool          `json:"accepted"`
	Message  string        `json:"message,omitempty"`
	Duration time.Duration `json:"duration"` // FIXME: JSON duration codec
	Leases   []lease.Lease `json:"lease"`
}

// UpdateResponse represents a lease environment update request.
type UpdateResponse struct {
	Request
	Success bool   `json:"success"`
	Message string `json:"message,omitempty"`
}
