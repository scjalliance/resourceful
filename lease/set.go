package lease

import "time"

// Set is a set of leases.
type Set []Lease

// TODO: Create a Resource() method that filters by resource, then remove
//       the resource argument from HostUser and Instance.

// Resource returns the set of leases matching the requested resource.
func (s Set) Resource(resource string) (matched Set) {
	for i := range s {
		if s[i].MatchResource(resource) {
			matched = append(matched, Clone(s[i]))
		}
	}
	return
}

// HostUser returns the set of leases matching the requested resource, host
// and user.
func (s Set) HostUser(resource, host, user string) (matched Set) {
	for i := range s {
		if s[i].MatchHostUser(resource, host, user) {
			matched = append(matched, Clone(s[i]))
		}
	}
	return
}

// User returns the set of leases for the given user.
func (s Set) User(user string) (matched Set) {
	for i := range s {
		if s[i].Instance.User == user {
			matched = append(matched, Clone(s[i]))
		}
	}
	return
}

// Instance returns the first lease that matches the requested resource and
// instance.
func (s Set) Instance(resource string, instance Instance) (ls Lease, found bool) {
	for i := range s {
		if s[i].MatchInstance(resource, instance) {
			return Clone(s[i]), true
		}
	}
	return
}

// Index returns the index of the first lease within s that matches the given
// parameters, or -1 if no such lease is present in s.
func (s Set) Index(resource string, instance Instance) (index int) {
	for i := range s {
		if s[i].MatchInstance(resource, instance) {
			return i
		}
	}
	return -1
}

// Property returns a slice of property values from the leases. Keys are
// are supplied in preferential order, and the first key in each lease that
// exists is returned as the value for that key.
func (s Set) Property(keys ...string) (values []string) {
	for _, l := range s {
		for _, key := range keys {
			if value, ok := l.Properties[key]; ok {
				values = append(values, value)
				break
			}
		}
	}
	return
}

// Status returns the subset of leases with the requested status.
func (s Set) Status(status Status) (matched Set) {
	for i := range s {
		if s[i].MatchStatus(status) {
			matched = append(matched, Clone(s[i]))
		}
	}
	return
}

// Stats returns the resource consumption statistics for each resource
// counting strategy.
//
// The set must be sorted prior to calling this function.
func (s Set) Stats() (stats Stats) {
	consumers := make(map[string]struct{}, len(s)) // Consumers that have already been seen

	for _, ls := range s {
		// The instance strategy is a simple tally of each kind of lease.
		stats.Instance.Add(ls.Instance.User, ls.Status)

		// The consumer strategy is more complicated; it requires that we only count
		// each consumer once, despite how many instances the consumer may have.
		//
		// Leases are processed in sorted order, which means the active lease will
		// be processed first. Consumers with both active and released leases will
		// only count as active.
		c := ls.HostUser()
		if _, seen := consumers[c]; !seen {
			consumers[c] = struct{}{}
			stats.Consumer.Add(ls.Instance.User, ls.Status)
		}
	}
	return
}

// ExpirationTime returns the earliest time at which a member of the set will
// expire.
func (s Set) ExpirationTime() (expiration time.Time) {
	if len(s) == 0 {
		return
	}
	expiration = s[0].ExpirationTime()
	for i := 1; i < len(s); i++ {
		if e := s[i].ExpirationTime(); e.Before(expiration) {
			expiration = e
		}
	}
	return
}

// DecayTime returns the earliest time at which a member of the set will
// decay.
func (s Set) DecayTime() (decay time.Time) {
	if len(s) == 0 {
		return
	}
	decay = s[0].DecayTime()
	for i := 1; i < len(s); i++ {
		if d := s[i].DecayTime(); d.Before(decay) {
			decay = d
		}
	}
	return
}

// DecayDuration returns the interval between the given time and the earliest
// time at which a member of the set will decay.
//
// If the decay time is before the given time a zero duration will be returned.
func (s Set) DecayDuration(at time.Time) (duration time.Duration) {
	dt := s.DecayTime()
	if dt.After(at) {
		duration = dt.Sub(at)
	}
	return
}

// Len is the number of leases in the collection.
func (s Set) Len() int {
	return len(s)
}

// Less reports whether the lease with index i should sort before the lease
// with index j.
//
// Leases of greater permanence come before leases of lesser permanence.
func (s Set) Less(i, j int) bool {
	o1 := s[i].Status.Order()
	o2 := s[j].Status.Order()
	if o1 < o2 {
		return true
	}
	if o1 > o2 {
		return false
	}

	switch s[i].Status {
	case Released:
		r1 := s[i].Released
		r2 := s[j].Released
		e1 := r1.Add(s[i].Decay)
		e2 := r2.Add(s[j].Decay)
		// Expiration: Latest first
		if e1.After(e2) {
			return true
		}
		if e1.Before(e2) {
			return false
		}
		// Release: Oldest first
		if r1.Before(r2) {
			return true
		}
		if r1.After(r2) {
			return false
		}
		fallthrough
	case Active, Queued:
		s1 := s[i].Started
		s2 := s[j].Started
		if s1.Before(s2) {
			return true
		}
		if s1.After(s2) {
			return false
		}
	}
	return false
}

// Swap swaps the leases with indices i and j.
func (s Set) Swap(i, j int) {
	s[i], s[j] = s[j], s[i]
}
