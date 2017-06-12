// +build windows

package leaseui

import (
	"context"
	"fmt"

	"github.com/scjalliance/resourceful/guardian"
	"github.com/scjalliance/resourceful/lease"
)

// WaitForActive will create and manage a queued lease user interface. It will
// return when an active lease is acquired or the user has indicated that they
// would like to cancel.
func WaitForActive(ctx context.Context, icon *Icon, program, consumer string, response guardian.Acquisition, responses <-chan guardian.Acquisition) (acquired bool, err error) {
	// Create a view model that will be consumed by the queued lease dialog.
	// Prime it with the most recent response that was received.
	model := NewModel(icon, program, consumer, response)

	// Create the queued lease dialog.
	dlg, err := New(model)
	if err != nil {
		err = fmt.Errorf("unable to create lease status user interface: %v", err)
		return
	}

	// Run the dialog while syncing the view model with responses that are
	// coming in on ch.
	dlg.RunWithSync(ctx, responses)
	if dlg.Cancelled() {
		return
	}

	// Return the last response that was fed into the model
	response = model.Response()
	if response.Err != nil {
		err = response.Err
	}

	acquired = response.Lease.Status == lease.Active
	return
}
