package leaseui

import (
	"github.com/scjalliance/resourceful/guardian"
	"github.com/scjalliance/resourceful/lease"
)

// Model defines the functions common to all view models.
type Model interface {
	Update(ls lease.Lease, acquisition guardian.Acquisition)
	Refresh()
	Close()
}
