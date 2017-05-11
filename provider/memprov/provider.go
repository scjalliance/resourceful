package memprov

import (
	"errors"
	"sync"
	"time"

	"github.com/scjalliance/resourceful/environment"
	"github.com/scjalliance/resourceful/lease"
)

// Provider provides memory-based lease management.
type Provider struct {
	mutex       sync.RWMutex
	leases      lease.Set       // the current set of leases
	allocations map[string]uint // maps resources to allocation counts
}

// New returns a new memory provider.
func New() *Provider {
	return &Provider{}
}

// Leases will return the current set of leases for the requested resource.
//
// If the provided resource is empty all leases will be returned.
func (p *Provider) Leases(resource string) (leases lease.Set, err error) {
	p.mutex.Lock()
	defer p.mutex.Unlock()
	p.cull()
	leases = make(lease.Set, len(p.leases))
	copy(leases, p.leases)
	return
}

// Acquire will attempt to create or renew a lease for the given resource and
// consumer.
func (p *Provider) Acquire(resource, consumer string, env environment.Environment, limit uint, duration time.Duration) (result lease.Lease, allocation uint, accepted bool, err error) {
	now := time.Now()

	p.mutex.Lock()
	defer p.mutex.Unlock()

	p.cull()

	// Check to see whether this is a renewal or an existing allocation
	index := -1
	if len(p.leases) > 0 {
		index = p.leases.Index(resource, consumer)
	}

	// If this is a new allocation, check whether we've already exceeded the limit
	allocation = p.allocations[resource]
	if index == -1 && allocation >= limit {
		return
	}

	// Allocate a map if this is the first lease
	if p.allocations == nil {
		p.allocations = make(map[string]uint)
	}

	// Record the lease
	result.Resource = resource
	result.Consumer = consumer
	result.Environment = env
	result.Renewed = now
	result.Duration = duration

	if index == -1 {
		// This is a new lease
		result.Started = now
		allocation++
		p.leases = append(p.leases, result)
	} else {
		// This is a renewal
		result.Started = p.leases[index].Started
		p.leases[index] = result
	}

	p.allocations[resource] = allocation
	accepted = true

	return
}

// Update will update the environment associated with a lease. It will not
// renew the lease.
func (p *Provider) Update(resource, consumer string, env environment.Environment) (result lease.Lease, err error) {
	err = errors.New("Lease updating has not been written yet")

	p.mutex.Lock()
	defer p.mutex.Unlock()

	p.cull()

	index := p.leases.Index(resource, consumer)
	if index == -1 {
		// TODO: Return error?
		return
	}

	p.leases[index].Environment = env

	return
}

// Release will remove the lease for the given resource and consumer.
func (p *Provider) Release(resource, consumer string) (err error) {
	p.mutex.Lock()
	defer p.mutex.Unlock()

	p.cull()

	// Look for the lease, which might not exist after the cull
	index := -1
	if len(p.leases) > 0 {
		index = p.leases.Index(resource, consumer)
	}

	// Exit if there's no lease to remove
	if index == -1 {
		return
	}

	// Remove the lease
	p.remove(index)

	return
}

// cull will remove all expired leases from the provider. The caller is
// expected to hold a write lock for the duration of the call.
func (p *Provider) cull() {
	for i := 0; i < len(p.leases); {
		if p.leases[i].Expired() {
			p.remove(i)
		} else {
			i++
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
