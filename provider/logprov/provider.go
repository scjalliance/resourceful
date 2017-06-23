package logprov

import (
	"fmt"
	"log"
	"strings"
	"sync"
	"time"

	"github.com/scjalliance/resourceful/counter"
	"github.com/scjalliance/resourceful/lease"
)

// Provider provides boltdb-backed lease management.
type Provider struct {
	source   lease.Provider
	log      *log.Logger
	schedule []Schedule
	ctr      counter.Counter

	mutex sync.RWMutex // Locked while checkpointing
	last  uint64       // Value of ctr at the last checkpoint
}

// New returns a new transaction logging provider.
func New(source lease.Provider, logger *log.Logger, schedule ...Schedule) *Provider {
	p := &Provider{
		source:   source,
		log:      logger,
		schedule: schedule,
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

	return p.checkpoint()
}

// record records the transaction in the transaction log.
//
// record assumes that a read lock is held for the duration of the call.
func (p *Provider) record(tx *lease.Tx) {
	var ops uint64 // Total number of consumptive ops (really op effects)

	for _, op := range tx.Ops() {
		if op.Type == lease.Update && op.UpdateType() == lease.Renew {
			// Don't record renewals
			continue
		}
		for _, effect := range op.Effects() {
			if !effect.Consumptive() {
				// Only record effects that affect consumption
				continue
			}
			p.log.Printf("TX %s", effect.String())
			ops++
		}
	}

	p.add(ops)
}

// add adds the given number of operations to the ops counter.
//
// add assumes that a read lock is held for the duration of the call.
func (p *Provider) add(ops uint64) {
	if ops == 0 {
		return
	}

	var (
		current = p.ctr.Add(ops)   // How many ops have passed since the provider started
		count   = current - p.last // How many ops have passed since the last checkpoint
	)
	for i := range p.schedule {
		if count >= p.schedule[i].ops {
			// Run the checkpoint in a separate goroutine so we don't deadlock
			go p.runCheckpoint(current)
			return
		}
	}
}

// runCheckpoint will run a checkpoint if the last checkpoint occurred at the
// expected time.
func (p *Provider) runCheckpoint(at uint64) {
	p.mutex.Lock()
	defer p.mutex.Unlock()

	if p.last > at {
		// Another goroutine beat us to the checkpoint
		return
	}

	p.checkpoint()
}

// checkpoint writes checkpoint data to the transaction log.
//
// checkpoint assumes that a write lock is held for the duration of the call.
func (p *Provider) checkpoint() (err error) {
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

	p.last = p.ctr.Value()

	return
}
