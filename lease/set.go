package lease

// Set is a set of leases.
type Set []Lease

// Match returns the subset of leases which match the given parameters.
func (s Set) Match(resource, consumer, instance string) (matches Set) {
	for p := range s {
		if s[p].Match(resource, consumer, instance) {
			matches = append(matches, s[p])
		}
	}
	return
}

// Index returns the index of the first lease within s that matches the given
// parameters, or -1 if no such lease is present in s.
func (s Set) Index(resource, consumer, instance string) (index int) {
	for p := range s {
		if s[p].Match(resource, consumer, instance) {
			return p
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
