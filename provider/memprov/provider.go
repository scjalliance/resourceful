package memprov

import (
	"errors"
	"sort"
	"sync"

	"github.com/scjalliance/resourceful/lease"
)

// leasePage is an in-memory page of lease data for a single resource.
type leasePage struct {
	mutex    sync.RWMutex
	revision uint64
	leases   lease.Set
}

// Provider provides memory-based lease management.
type Provider struct {
	mutex      sync.RWMutex
	leasePages map[string]*leasePage // The lease set for each resource
}

// New returns a new memory provider.
func New() *Provider {
	return &Provider{
		leasePages: make(map[string]*leasePage),
	}
}

// Close releases any resources consumed by the provider.
func (p *Provider) Close() error {
	return nil
}

// ProviderName returns the name of the provider.
func (p *Provider) ProviderName() string {
	return "In-Memory"
}

// LeaseResources returns all of the resources with lease data.
func (p *Provider) LeaseResources() (resources []string, err error) {
	p.mutex.RLock()
	defer p.mutex.RUnlock()
	for resource := range p.leasePages {
		resources = append(resources, resource)
	}
	return
}

// LeaseView returns the current revision and lease set for the resource.
func (p *Provider) LeaseView(resource string) (revision uint64, leases lease.Set, err error) {
	page := p.leasePage(resource)
	page.mutex.RLock()
	defer page.mutex.RUnlock()

	revision = page.revision
	leases = make(lease.Set, len(page.leases))
	copy(leases, page.leases)
	return
}

// LeaseCommit will attempt to apply the operations described in the lease
// transaction.
func (p *Provider) LeaseCommit(tx *lease.Tx) error {
	ops := tx.Ops()
	if len(ops) == 0 {
		// Nothing to commit
		return nil
	}

	page := p.leasePage(tx.Resource())
	page.mutex.Lock()
	defer page.mutex.Unlock()
	if page.revision != tx.Revision() {
		return errors.New("Unable to commit lease transaction due to opportunistic lock conflict")
	}
	page.revision++
	for _, op := range ops {
		switch op.Type {
		case lease.Create:
			page.leases = append(page.leases, op.Lease)
		case lease.Update:
			i := page.leases.Index(op.Previous.Resource, op.Previous.Consumer, op.Previous.Instance)
			if i >= 0 {
				page.leases[i] = lease.Clone(op.Lease)
			}
		case lease.Delete:
			i := page.leases.Index(op.Previous.Resource, op.Previous.Consumer, op.Previous.Instance)
			if i >= 0 {
				page.leases = append(page.leases[:i], page.leases[i+1:]...)
			}
		}
	}

	sort.Sort(page.leases)

	return nil
}

func (p *Provider) leasePage(resource string) *leasePage {
	p.mutex.RLock()
	page, ok := p.leasePages[resource]
	p.mutex.RUnlock()
	if ok {
		return page
	}

	p.mutex.Lock()
	defer p.mutex.Unlock()
	page, ok = p.leasePages[resource]
	if ok {
		return page
	}
	page = new(leasePage)
	p.leasePages[resource] = page
	return page
}
