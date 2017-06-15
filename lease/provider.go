package lease

// Provider is a lease management interface. It provides access to transactions
// for specific resources.
type Provider interface {
	// ProviderName returns the name of the provider.
	ProviderName() string

	// LeaseView returns the current revision and lease set for the resource.
	LeaseView(resource string) (revision uint64, leases Set, err error)

	// LeaseCommit will attempt to apply the operations described in the lease
	// transaction.
	LeaseCommit(tx *Tx) (err error)

	// Close releases any resources consumed by the provider.
	Close() error
}
