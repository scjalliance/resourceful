// +build windows

package leaseui

import (
	"context"
	"fmt"

	"github.com/lxn/walk"
	"github.com/scjalliance/resourceful/guardian"
	"github.com/scjalliance/resourceful/lease"
)

func runDialog(ctx context.Context, form *walk.Dialog) int {
	select {
	case <-ctx.Done():
		return walk.DlgCmdCancel
	default:
	}

	closed := make(chan struct{})
	defer close(closed)

	go func() {
		select {
		case <-closed:
		case <-ctx.Done():
			// Here we use the synchronize function to ensure that our call to Close
			// pushes the WM_CLOSE message onto the message queue of the correct
			// thread. If we call Close() directly it could fail silently and
			// deadlock.
			form.Synchronize(func() {
				form.Close(walk.DlgCmdCancel)
			})
		}
	}()

	return form.Run()
}

func runDialogWithSync(ctx context.Context, form *walk.Dialog, model Model, responses <-chan guardian.Acquisition) int {
	// Keep the dialog in sync with lease responses
	ctx, cancel := context.WithCancel(ctx)
	go func() {
		Sync(model, responses) // Block until acquisition or shutdown
		cancel()
	}()

	return runDialog(ctx, form)
}

// WaitForActive will create and manage a queued lease user interface. It will
// return when an active lease is acquired or the user has indicated that they
// would like to cancel.
func WaitForActive(ctx context.Context, icon *Icon, program, consumer string, response guardian.Acquisition, responses <-chan guardian.Acquisition) (acquired bool, err error) {
	// Create a view model that will be consumed by the queued lease dialog.
	// Prime it with the most recent response that was received.
	model := NewLeaseModel(icon, program, consumer, response)

	// Create the queued lease dialog.
	dlg, err := NewLeaseDialog(model)
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
