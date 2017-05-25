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

/*
// Leases will return the current set of leases for the requested resource.
//
// If the provided resource is empty all leases will be returned.
func (p *Provider) Leases(resource string) (leases lease.Set, err error) {
	p.mutex.Lock()
	defer p.mutex.Unlock()
	p.refresh(resource)
	p.leases[]
	leases = make(lease.Set, len(p.leases))
	copy(leases, p.leases)
	return
}

// Acquire will attempt to create or renew a lease for the given resource and
// consumer.
func (p *Provider) Acquire(resource, consumer, instance string, env environment.Environment, limit uint, duration, decay time.Duration) (result lease.Lease, allocation uint, accepted bool, err error) {
	now := time.Now()

	p.mutex.Lock()
	defer p.mutex.Unlock()

	p.refresh()

	// Check to see whether this is a renewal or an existing allocation
	index := -1
	if len(p.leases) > 0 {
		index = p.leases.Index(resource, consumer, instance)
	}

	// Determine whether this is a new allocation

	// Allocate a map if this is the first lease
	if p.allocations == nil {
		p.allocations = make(map[string]uint)
	}

	// Record the lease
	result.Resource = resource
	result.Consumer = consumer
	result.Instance = instance
	result.Environment = env
	result.Status = lease.Active
	result.Renewed = now
	result.Duration = duration
	result.Decay = decay

	if index == -1 {
		// This is a new lease
		result.Started = now

		// If this is a new allocation, check whether we've already exceeded the limit
		allocation = p.allocations[resource]
		if allocation < limit {
			allocation++
			p.allocations[resource] = allocation
		}

		p.leases = append(p.leases, result)
	} else {
		// This is a renewal of a lease that may be active, expired or pending.
		result.Started = p.leases[index].Started
		result.Status = p.leases[index].Status
		p.leases[index] = result
	}

	accepted = result.Status == lease.Active

	return
}

// Update will update the environment associated with a lease. It will not
// renew the lease.
func (p *Provider) Update(resource, consumer, instance string, env environment.Environment) (result lease.Lease, err error) {
	err = errors.New("Lease updating has not been written yet")

	p.mutex.Lock()
	defer p.mutex.Unlock()

	p.refresh()

	index := p.leases.Index(resource, consumer, instance)
	if index == -1 {
		// TODO: Return error?
		return
	}

	p.leases[index].Environment = env

	return
}

// Release will remove the lease for the given resource, consumer and instance.
func (p *Provider) Release(resource, consumer, instance string) (err error) {
	p.mutex.Lock()
	defer p.mutex.Unlock()

	p.refresh()

	// Look for the lease, which might not exist after the cull
	index := -1
	if len(p.leases) > 0 {
		index = p.leases.Index(resource, consumer, instance)
	}

	// Exit if there's no lease to remove
	if index == -1 {
		return
	}

	// Remove the lease
	p.remove(index)

	return
}

// refresh will update leases statuses and remove all decayed leases from the
// provider. The caller is expected to hold a write lock for the duration of
// the call.
func (p *Provider) refresh() {
	// Remove decayed leases, update expired leases and promote pending leases
	//
	// It's safe to do this in one pass because the leases are sorted with
	// active and released coming before pending.
	for i := 0; i < len(p.leases); i++ {
		l := &p.leases[i]
		switch l.Status {
		case lease.Active, lease.Released:
			if l.Decayed() {
				p.remove(i)
				i--
			} else if l.Expired() {
				l.Status = lease.Released
			}
		case lease.Queued:
			allocation = p.allocations[l.Resource]
			if allocation < l.Limit {

			}
		}
	}

	// Pass 2: Promote pending leases to active
	for i := 0; i < len(p.leases); {
		l := &p.leases[i]
		if l.Status != lease.Queued {
			continue
		}
	}
}

// remove will remove the lease at the given index. If the index is invalid
// remove will panic.
//
// The caller is expected to hold a write lock for the duration of the call.
func (p *Provider) remove(index int) {
	// Determine the resource
	resource := p.leases[index].Resource

	// Perform some sanity checks
	if p.allocations == nil {
		panic("allocation map is nil when it shouldn't be")
	}
	allocation := p.allocations[resource]
	if allocation <= 0 {
		panic("allocation dropped below zero")
	}

	// Remove the lease
	p.leases = append(p.leases[:index], p.leases[index+1:]...)
	p.allocations[resource] = allocation - 1
}
*/
