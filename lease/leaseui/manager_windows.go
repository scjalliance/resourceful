// +build windows

package leaseui

import (
	"context"
	"fmt"

	"github.com/lxn/walk"
)

func call(callback Callback, t Type, r int) {
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

	callback(t, result, nil)
}

func (m *Manager) none(ctx context.Context, callback Callback) error {
	m.mutex.Lock()
	m.model = nil
	m.mutex.Unlock()

	<-ctx.Done()

	if callback != nil {
		callback(None, Success, nil)
	}

	return nil
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

	call(callback, Queued, dlg.Run(ctx))

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

	call(callback, Connected, dlg.Run(ctx))

	return nil
}

func (m *Manager) disconnected(ctx context.Context, callback Callback) error {
	m.mutex.Lock()
	model := NewConnectionModel(m.cfg, m.lease)
	m.model = model
	m.mutex.Unlock()

	defer model.Close()

	// Create the disconnected dialog
	dlg, err := NewDisconnectedDialog(m.cfg.Icon, model)
	if err != nil {
		return fmt.Errorf("unable to create connected user interface: %v", err)
	}

	call(callback, Disconnected, dlg.Run(ctx))

	return nil
}
