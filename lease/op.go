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

// String returns a string representation of the action type.
func (t Action) String() string {
	switch t {
	case None:
		return "none"
	case Create:
		return "create"
	case Update:
		return "update"
	case Delete:
		return "delete"
	default:
		return "unknown"
	}
}

// UpdateType is a type of lease update.
type UpdateType uint32

// Lease update types
const (
	Renew UpdateType = iota
	Downgrade
	Upgrade
	Replace
	Exchange
	Transmute
)

// String returns a string representation of the update type.
func (t UpdateType) String() string {
	switch t {
	case Renew:
		return "renew"
	case Upgrade:
		return "upgrade"
	case Downgrade:
		return "downgrade"
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

// Effect is an effect of a transaction.
type Effect struct {
	Action Action
	Lease
}

// String returns a string representation of the effect.
func (e *Effect) String() string {
	return fmt.Sprintf("%s %s %s %s %s", e.Instance, e.Consumer, strings.ToUpper(e.Action.String()), strings.ToUpper(string(e.Status)), e.Resource)
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
	case op.Lease.Status != op.Previous.Status:
		// Active < Released < Queued
		if op.Lease.Status.Order() < op.Previous.Status.Order() {
			return Upgrade
		}
		return Downgrade
	default:
		return Renew
	}
}

// Consumptive returns true if the operation affects a consumptive lease.
func (op *Op) Consumptive() bool {
	switch op.Type {
	case Create:
		return op.Lease.Consumptive()
	case Update:
		return op.Lease.Consumptive() || op.Previous.Consumptive()
	case Delete:
		return op.Previous.Consumptive()
	default:
		return false
	}
}

// Effects returns a set of strings describing the effects of the operation.
func (op *Op) Effects() (effects []Effect) {
	switch op.Type {
	case Delete, Update:
		effects = append(effects, Effect{
			Lease:  op.Previous,
			Action: Delete,
		})
	}

	switch op.Type {
	case Create, Update:
		effects = append(effects, Effect{
			Lease:  op.Lease,
			Action: Create,
		})
	}

	return
}
