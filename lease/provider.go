package lease

import (
	"time"

	"github.com/scjalliance/resourceful/environment"
)

// Provider is a lease management interface.
type Provider interface {
	Leases(resource string) (leases Set, err error) // An empty resource returns all leases
	Acquire(resource, consumer, instance string, env environment.Environment, limit uint, duration, decay time.Duration) (result Lease, allocation uint, accepted bool, err error)
	Update(resource, consumer, instance string, env environment.Environment) (result Lease, err error)
	Release(resource, consumer, instance string) (err error)
}
