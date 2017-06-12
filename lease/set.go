package lease

import "time"

// Set is a set of leases.
type Set []Lease

// Consumer returns the set of leases matching the requested resource and
// consumer.
func (s Set) Consumer(resource, consumer string) (matched Set) {
	for i := range s {
		if s[i].MatchConsumer(resource, consumer) {
			matched = append(matched, Clone(s[i]))
		}
	}
	return
}

// Instance returns the first lease that matches the requested resource,
// consumer and instance.
func (s Set) Instance(resource, consumer, instance string) (ls Lease, found bool) {
	for i := range s {
		if s[i].MatchInstance(resource, consumer, instance) {
			return Clone(s[i]), true
		}
	}
	return
}

// Index returns the index of the first lease within s that matches the given
// parameters, or -1 if no such lease is present in s.
func (s Set) Index(resource, consumer, instance string) (index int) {
	for i := range s {
		if s[i].MatchInstance(resource, consumer, instance) {
			return i
		}
	}
	return -1
}

// Environment returns a slice of environment values from the leases. Keys are
// are supplied in preferential order, and the first key in each lease that
// exists is returned as the value for that key.
func (s Set) Environment(keys ...string) (values []string) {
	for _, l := range s {
		for _, key := range keys {
			if value, ok := l.Environment[key]; ok {
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

// Stats returns the number of leases with each status.
func (s Set) Stats() (active, released, pending uint) {
	for _, ls := range s {
		switch ls.Status {
		case Active:
			active++
		case Released:
			released++
		case Queued:
			pending++
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
