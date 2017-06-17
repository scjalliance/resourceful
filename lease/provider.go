package lease

// Provider is a lease management interface. It provides access to transactions
// for specific resources.
type Provider interface {
	// ProviderName returns the name of the provider.
	ProviderName() string

	// LeaseResources returns all of the resources with lease data.
	LeaseResources() (resources []string, err error)

	// LeaseView returns the current revision and lease set for the resource.
	LeaseView(resource string) (revision uint64, leases Set, err error)

	// LeaseCommit will attempt to commit the lease transaction.
	LeaseCommit(tx *Tx) (err error)

	// Close releases any resources consumed by the provider.
	Close() error
}
