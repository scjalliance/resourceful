package lease

import "time"

// MinimumRefresh is the minimum amount of time that should pass between
// lease refresh attempts. It is a hard limit intended to avoid hammering
// the server when policies are misconfigured.
const MinimumRefresh = time.Second

// Refresh defines the active and queued refresh rates for a lease.
type Refresh struct {
	Active time.Duration `json:"active,omitempty"` // Active lease refresh interval
	Queued time.Duration `json:"queued,omitempty"` // Queued lease refresh interval
}
