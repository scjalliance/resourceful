package policy

// Cache is a cache of policies.
type Cache interface {
	// Policies returns the set of policies from the cache.
	Policies() (Set, error)

	// SetPolicies writes the set of policies to the cache.
	SetPolicies(Set) error
}
