// +build windows

package leaseui

import (
	"context"
	"fmt"
	"sync"

	"github.com/lxn/walk"
	"github.com/scjalliance/resourceful/guardian"
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

func runDialogWithSync(ctx context.Context, form *walk.Dialog, model Model, responses <-chan guardian.Acquisition, fn SyncFunc) (result Result) {
	// Keep the dialog in sync with lease responses
	ctx, cancel := context.WithCancel(ctx)

	var wg sync.WaitGroup
	wg.Add(1)

	go func() {
		result = Sync(ctx, model, responses, fn) // Block until acquisition or shutdown
		cancel()                                 // Close the dialog
		wg.Done()                                // Indicate that the sync has exited
	}()

	r := runDialog(ctx, form)

	cancel()  // Tell the sync to stop
	wg.Wait() // Wait until the sync stops

	switch r {
	case walk.DlgCmdAbort:
		result = UserCancelled
	case walk.DlgCmdNone:
		result = UserClosed
	}

	return
}

// WaitForActive will create and manage a queued lease user interface. It will
// return when an active lease is acquired or the user has indicated that they
// would like to cancel.
func WaitForActive(ctx context.Context, icon *Icon, program, consumer string, current guardian.Acquisition, responses <-chan guardian.Acquisition) (result Result, final guardian.Acquisition, err error) {
	// Create a view model that will be consumed by the queued lease dialog.
	// Prime it with the most recent response that was received.
	model := NewLeaseModel(icon, program, consumer, current)

	// Create the queued lease dialog.
	dlg, err := NewLeaseDialog(model)
	if err != nil {
		err = fmt.Errorf("unable to create lease status user interface: %v", err)
		return
	}

	// Run the dialog while syncing the view model with responses that are
	// coming in on responses.
	result = dlg.RunWithSync(ctx, responses)

	// Return the last response that was fed into the model
	final = model.Response()
	if final.Err != nil {
		err = final.Err
	}

	return
}
