package lease

import (
	"time"
)

// State holds state information about a lease for a lease holder.
type State struct {
	Online           bool          // Do we have a live connection to the guardian server?
	LeaseNotRequired bool          // Did the server tell us we don't need a lease?
	Acquired         bool          // Have we acquired a lease of any status?
	Lease            Lease         // The most recent lease received from the server
	Leases           Set           // All leases for our lease resource
	Retry            time.Duration // Retry interval when not holding a lease
	Err              error         // The most recent acquisition error
}

// IsZero returns true if the state holds a zero value.
func (s *State) IsZero() bool {
	if s.Online || s.Acquired || s.LeaseNotRequired {
		return false
	}

	if !s.Lease.Subject.Empty() {
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
