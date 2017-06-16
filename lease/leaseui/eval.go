package leaseui

import (
	"github.com/scjalliance/resourceful/guardian"
	"github.com/scjalliance/resourceful/lease"
)

// EvalFunc evaluates response and reports success or failure.
type EvalFunc func(response guardian.Acquisition) (success bool)

// ActiveLeaseAcquired is an evaluation function that returns true when an
// active lease is acquired.
func ActiveLeaseAcquired(response guardian.Acquisition) (success bool) {
	if response.Err != nil {
		return false
	}
	return response.Lease.Status == lease.Active
}

// ConnectionAcquired is an evaluation function that returns true when a
// successful connection to the server is acquired.
func ConnectionAcquired(response guardian.Acquisition) (success bool) {
	return response.Err == nil
}
