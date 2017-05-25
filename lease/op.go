package lease

// Action is a type of lease operation
type Action uint32

// Lease action types
const (
	None Action = iota
	Create
	Update
	Delete
)

// Op is a lease operation describing a create, update or delete action
type Op struct {
	Type     Action
	Previous Lease
	Lease    Lease
}
