// +build windows

package leaseui

import (
	"context"

	"github.com/lxn/walk"
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
