package logprov

import (
	"fmt"
	"log"
	"strings"
	"sync"
	"time"

	"github.com/scjalliance/resourceful/lease"
)

// Provider provides boltdb-backed lease management.
type Provider struct {
	source lease.Provider
	log    *log.Logger
	mutex  sync.RWMutex // Only locked for checkpointing
}

// New returns a new transaction logging provider.
func New(source lease.Provider, logger *log.Logger) *Provider {
	p := &Provider{
		source: source,
		log:    logger,
	}
	p.Checkpoint()
	return p
}

// Close releases any resources consumed by the provider and its source.
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
	p.mutex.RLock()
	defer p.mutex.RUnlock()
	return p.source.LeaseView(resource)
}

// LeaseCommit will attempt to apply the operations described in the lease
// transaction.
func (p *Provider) LeaseCommit(tx *lease.Tx) error {
	p.mutex.RLock()
	defer p.mutex.RUnlock()

	err := p.source.LeaseCommit(tx)
	if err == nil {
		p.record(tx)
	}
	return err
}

// Checkpoint will write all of the lease data to the transaction log in a
// checkpoint block.
//
// In order for the checkpoint to obtain a consistent view of the lease data it
// must hold an exclusive lock while the chekcpoint is being performed. All
// other operations on the provider will block until the checkpoint has
// finished.
func (p *Provider) Checkpoint() (err error) {
	p.mutex.Lock()
	defer p.mutex.Unlock()

	at := time.Now().UnixNano()

	resources, err := p.source.LeaseResources()
	if err != nil {
		return
	}

	p.log.Printf("CP %v START", at)

	for _, resource := range resources {
		revision, leases, viewErr := p.source.LeaseView(resource)
		if viewErr != nil {
			p.log.Printf("CP %v RESOURCE %s ERR %v", at, resource, err)
		} else {
			p.log.Printf("CP %v RESOURCE %s REV %d", at, resource, revision)
			for _, ls := range leases {
				if ls.Consumptive() {
					p.log.Printf("CP %v LEASE %s %s", at, ls.Subject(), strings.ToUpper(string(ls.Status)))
				}
			}
		}
	}

	p.log.Printf("CP %v END", at)

	return
}

func (p *Provider) record(tx *lease.Tx) {
	for _, op := range tx.Ops() {
		if op.Type == lease.Update && op.UpdateType() == lease.Renew {
			// Don't record renewals
			continue
		}
		for _, effect := range op.Effects() {
			if !effect.Consumptive() {
				// Only records effects that affect consumption
				continue
			}
			p.log.Printf("TX %s", effect.String())
		}
	}
}
