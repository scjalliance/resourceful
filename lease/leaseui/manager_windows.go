// +build windows

package leaseui

import (
	"context"
	"fmt"

	"github.com/lxn/walk"
)

func call(callback Callback, r int) {
	if callback == nil {
		return
	}

	var result Result

	switch r {
	default:
		result = Success
	case walk.DlgCmdAbort:
		result = UserCancelled
	case walk.DlgCmdNone:
		result = UserClosed
	case walk.DlgCmdCancel:
		result = ContextCancelled
	}

	go callback(result, nil)
}

func (m *Manager) queued(ctx context.Context, callback Callback) error {
	m.mutex.Lock()
	model := NewQueuedModel(m.cfg, m.acquisition)
	m.model = model
	m.mutex.Unlock()

	defer model.Close()

	dlg, err := NewQueuedDialog(m.cfg.Icon, model)
	if err != nil {
		return fmt.Errorf("unable to create queued user interface: %v", err)
	}

	call(callback, dlg.Run(ctx))

	return nil
}

func (m *Manager) connected(ctx context.Context, callback Callback) error {
	m.mutex.Lock()
	model := NewConnectionModel(m.cfg, m.lease)
	m.model = model
	m.mutex.Unlock()

	defer model.Close()

	// Create the connected dialog
	dlg, err := NewConnectedDialog(m.cfg.Icon, model)
	if err != nil {
		return fmt.Errorf("unable to create connected user interface: %v", err)
	}

	call(callback, dlg.Run(ctx))

	return nil
}

func (m *Manager) disconnected(ctx context.Context, callback Callback) error {
	<-ctx.Done()

	/*
		switch result {
		case leaseui.Success:
			// The server came back online
			r.setOnline(true)
		case leaseui.Failure:
			return err
		case leaseui.UserCancelled, leaseui.UserClosed:
			// The user intentionally stopped waiting
			r.dismissal = time.Now()
		case leaseui.ChannelClosed:
			// The lease maintainer is shutting down
		case leaseui.ContextCancelled:
			// Either the system is shutting down or the lease expired
			if r.current.Expired(now) {
				log.Printf("Lease has expired. Shutting down %s", r.program)
				shutdown()
			}
		}
	*/

	return nil
}
