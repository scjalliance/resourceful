package policy

// Provider is a source of policies.
type Provider interface {
	// ProviderName returns the name of the provider.
	ProviderName() string

	// Policies returns the set of policies.
	Policies() (Set, error)

	// Close releases any resources consumed by the provider.
	Close() error
}
