package leaseui

import "github.com/scjalliance/resourceful/guardian"

// Model defines the functions common to all view models.
type Model interface {
	Icon() *Icon
	Title() string
	Update(response guardian.Acquisition)
	Refresh()
}
