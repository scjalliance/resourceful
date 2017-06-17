package logprov

import (
	"fmt"
	"log"

	"github.com/scjalliance/resourceful/lease"
)

// Provider provides boltdb-backed lease management.
type Provider struct {
	source lease.Provider
	log    *log.Logger
}

// New returns a new transaction logging provider.
func New(source lease.Provider, logger *log.Logger) *Provider {
	return &Provider{
		source: source,
		log:    logger,
	}
}

// Close releases any resources consumed by the provider and it source.
func (p *Provider) Close() error {
	return p.source.Close()
}

// ProviderName returns the name of the provider.
func (p *Provider) ProviderName() string {
	return fmt.Sprintf("%s (with logged transactions)", p.source.ProviderName())
}

// LeaseResources returns all of the resources with lease data.
func (p *Provider) LeaseResources() (resources []string, err error) {
	return p.source.LeaseResources()
}

// LeaseView returns the current revision and lease set for the resource.
func (p *Provider) LeaseView(resource string) (revision uint64, leases lease.Set, err error) {
	return p.source.LeaseView(resource)
}

// LeaseCommit will attempt to apply the operations described in the lease
// transaction.
func (p *Provider) LeaseCommit(tx *lease.Tx) error {
	err := p.source.LeaseCommit(tx)
	if err == nil {
		for _, op := range tx.Ops() {
			if op.Type == lease.Update && op.UpdateType() == lease.Renew {
				// Don't record renewals
				continue
			}
			for _, effect := range op.Effects() {
				p.log.Println(effect)
			}
		}
	}
	return err
}
