package leaseui

import "github.com/scjalliance/resourceful/guardian"

// Model defines the functions common to all view models.
type Model interface {
	Update(response guardian.Acquisition)
	Refresh()
}
