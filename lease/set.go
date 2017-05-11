package lease

// Set is a set of leases.
type Set []Lease

// Match returns the subset of leases which match the given parameters.
func (s Set) Match(resource, consumer string) (matches Set) {
	for p := range s {
		if s[p].Match(resource, consumer) {
			matches = append(matches, s[p])
		}
	}
	return
}

// Index returns the index of the first lease within s that matches the given
// parameters, or -1 if no such lease is present in s.
func (s Set) Index(resource, consumer string) (index int) {
	for p := range s {
		if s[p].Match(resource, consumer) {
			return p
		}
	}
	return -1
}
