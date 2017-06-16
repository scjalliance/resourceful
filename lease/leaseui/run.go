// +build !windows

package leaseui

import (
	"context"

	"github.com/scjalliance/resourceful/guardian"
)

// WaitForActive will create and manage a queued lease user interface. It will
// return when an active lease is acquired or the user has closed the interface.
func WaitForActive(ctx context.Context, icon *Icon, program, consumer string, current guardian.Acquisition, responses <-chan guardian.Acquisition) (result Result, final guardian.Acquisition, err error) {
	model := NewLeaseModel(current)
	result = Sync(ctx, model, responses, ActiveLeaseAcquired)
	final = model.Response()
	return
}
