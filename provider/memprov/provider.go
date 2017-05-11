package memprov

import (
	"errors"
	"fmt"
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
	err = errors.New("Lease listing has not been written yet")
	return
}

// Acquire will attempt to create or renew a lease for the given resource and
// consumer.
func (p *Provider) Acquire(resource, consumer string, env environment.Environment, limit uint, duration time.Duration) (result lease.Lease, err error) {
	now := time.Now()

	p.mutex.Lock()
	defer p.mutex.Unlock()

	p.cull()

	allocation := p.allocations[resource]
	if allocation >= limit {
		err = fmt.Errorf("resource limit of %d has already been met", limit)
		return
	}

	index := -1
	if len(p.leases) > 0 {
		index = p.leases.Index(resource, consumer)
	}

	result.Resource = resource
	result.Consumer = consumer
	result.Environment = env
	result.Started = now
	result.Renewed = now
	result.Duration = duration

	if index == -1 {
		p.leases = append(p.leases, result)
	} else {
		p.leases[index] = result
	}

	return
}

// Update will update the environment associated with a lease. It will not
// renew the lease.
func (p *Provider) Update(resource, consumer string, env environment.Environment) (result lease.Lease, err error) {
	err = errors.New("Lease updating has not been written yet")
	return
}

// Release will remove the lease for the given resource and consumer.
func (p *Provider) Release(resource, consumer string) (err error) {
	err = errors.New("Lease releasing has not been written yet")
	return
}

// cull will remove all expired leases from the provider. The caller is
// expected to hold a write lock for the duration of the call.
func (p *Provider) cull() {

}
