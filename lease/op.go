package lease

import (
	"fmt"
	"strings"
)

// Action is a type of lease operation
type Action uint32

// Lease action types
const (
	None Action = iota
	Create
	Update
	Delete
)

// UpdateType is a type of lease update.
type UpdateType uint32

// Lease update types
const (
	Renew UpdateType = iota
	Replace
	Exchange
	Transmute
)

// String returns a string representation of the update type.
func (t UpdateType) String() string {
	switch t {
	case Renew:
		return "renew"
	case Replace:
		return "replace"
	case Exchange:
		return "exchange"
	case Transmute:
		return "transmute"
	default:
		return "unknown"
	}
}

// Op is a lease operation describing a create, update or delete action
type Op struct {
	Type     Action
	Previous Lease
	Lease    Lease
}

// UpdateType returns the type of update for update operations.
func (op *Op) UpdateType() UpdateType {
	switch {
	case op.Lease.Resource != op.Previous.Resource:
		return Transmute
	case op.Lease.Consumer != op.Previous.Consumer:
		return Exchange
	case op.Lease.Instance != op.Previous.Instance:
		return Replace
	default:
		return Renew
	}
}

// Effect returns a string representation summarizing the effect of the
// operation.
func (op *Op) Effect() string {
	switch op.Type {
	case Create:
		return fmt.Sprintf("CREATE LEASE %s", op.Lease.Subject())
	case Update:
		return fmt.Sprintf("%s LEASE %s FROM %s", strings.ToUpper(op.UpdateType().String()), op.Lease.Subject(), op.Previous.Subject())
	case Delete:
		return fmt.Sprintf("DELETE LEASE %s", op.Lease.Subject())
	default:
		return fmt.Sprintf("UNKNOWN LEASE EFFECT %d %s", op.Type, op.Lease.Subject())
	}
}
