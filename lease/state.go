package lease

import (
	"time"
)

// State holds state information about a lease for a lease holder.
type State struct {
	Online   bool          // Do we have a live connection to the guardian server?
	Acquired bool          // Have we ever acquired a lease of any status?
	Lease    Lease         // The most recent lease received from the server
	Leases   Set           // All leases for a particular resource
	Retry    time.Duration // How often soon after a failure should an acquistion be attempted?
	Err      error         // The most recent acquisition error
}

// IsZero returns true if the state holds a zero value.
func (s *State) IsZero() bool {
	if s.Online || s.Acquired {
		return false
	}

	if s.Lease.Specified() {
		return false
	}

	if len(s.Leases) > 0 {
		return false
	}

	if s.Err != nil {
		return false
	}

	return true
}
