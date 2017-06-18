// +build !windows

package leaseui

import (
	"context"

	"github.com/scjalliance/resourceful/guardian"
	"github.com/scjalliance/resourceful/lease"
)

// Queued will create and manage a queued lease user interface. It will
// return when an active lease is acquired or the user has closed the interface.
func Queued(ctx context.Context, icon *Icon, program, consumer string, current guardian.Acquisition, responses <-chan guardian.Acquisition) (result Result, final guardian.Acquisition, err error) {
	model := NewQueuedModel(current)
	result = Sync(ctx, model, responses, ActiveLeaseAcquired)
	final = model.Response()
	return
}

// Disconnected will create and manage a disconnected user interface.
// It will return when a connection to the server is re-established or the
// user has closed the interface.
func Disconnected(ctx context.Context, icon *Icon, program, consumer string, current lease.Lease, leaseErr error, responses <-chan guardian.Acquisition) (result Result, final lease.Lease, err error) {
	model := NewConnectionModel(current, leaseErr)
	result = Sync(ctx, model, responses, ConnectionAcquired)
	final = model.Lease() // Return the last good lease that was fed into the model
	return
}
