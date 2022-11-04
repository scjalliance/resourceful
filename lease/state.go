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

/*
// Interval returns the lease acquisition interval for the given state and time.
func (s *State) Interval(at time.Time) time.Duration {
	const transportTime = time.Millisecond * 800 // Ballpark guess at how long it takes to acquire a lease

	// If we haven't received a valid lease yet use the retry interval
	if !s.Acquired {
		if s.Retry < MinimumRefresh {
			return MinimumRefresh
		}
		return s.Retry
	}

	// We have a lease
	interval := s.Lease.EffectiveRefresh()

	// If the server went offline after we retreived a valid lease, use the
	// effective refresh interval or our retry interval, whichever is
	// less.
	if !s.Online && s.Retry < interval {
		interval = s.Retry
	}

	switch s.Lease.Status {
	case Active:
		// If our lease is active make sure we try again before the current lease
		// expires
		exp := s.Lease.ExpirationTime()
		if exp.After(at) {
			remaining := exp.Sub(at)
			if transportTime < remaining {
				remaining = remaining - transportTime
			} else {
				remaining = 0
			}
			if remaining < interval {
				interval = remaining
			}
		} else {
			interval = 0
		}
	case Queued:
		// If our lease is queued, take into consideration when the next
		// lease decays
		decay := s.Leases.DecayDuration(at)
		if decay > 0 && decay < interval {
			interval = decay
		}
	}

	// Under no circumstances should we hammer the server faster than
	// the minimum refresh interval.
	if interval < MinimumRefresh {
		interval = MinimumRefresh
	}

	return interval
}
*/
