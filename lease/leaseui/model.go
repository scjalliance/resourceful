package leaseui

import (
	"github.com/scjalliance/resourceful/lease"
)

// Model defines the functions common to all view models.
type Model interface {
	Update(state lease.State)
	Refresh()
	Close()
}
