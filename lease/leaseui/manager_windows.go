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

func (m *Manager) startup(ctx context.Context, callback Callback) error {
	model := m.connectionModel()
	defer model.Close()

	dlg, err := NewStartupDialog(m.cfg.Icon, model)
	if err != nil {
		return fmt.Errorf("unable to create startup user interface: %v", err)
	}

	call(callback, Startup, dlg.Run(ctx))

	return nil
}

func (m *Manager) queued(ctx context.Context, callback Callback) error {
	model := m.queuedModel()
	defer model.Close()

	dlg, err := NewQueuedDialog(m.cfg.Icon, model)
	if err != nil {
		return fmt.Errorf("unable to create queued user interface: %v", err)
	}

	call(callback, Queued, dlg.Run(ctx))

	return nil
}

func (m *Manager) connected(ctx context.Context, callback Callback) error {
	model := m.connectionModel()
	defer model.Close()

	dlg, err := NewConnectedDialog(m.cfg.Icon, model)
	if err != nil {
		return fmt.Errorf("unable to create connected user interface: %v", err)
	}

	call(callback, Connected, dlg.Run(ctx))

	return nil
}

func (m *Manager) disconnected(ctx context.Context, callback Callback) error {
	model := m.connectionModel()
	defer model.Close()

	dlg, err := NewDisconnectedDialog(m.cfg.Icon, model)
	if err != nil {
		return fmt.Errorf("unable to create connected user interface: %v", err)
	}

	call(callback, Disconnected, dlg.Run(ctx))

	return nil
}

func (m *Manager) connectionModel() (model *ConnectionModel) {
	m.mutex.Lock()
	defer m.mutex.Unlock()

	model = NewConnectionModel(m.cfg, m.lease)
	model.Update(m.lease, m.acquisition)
	m.model = model

	return
}

func (m *Manager) queuedModel() (model *QueuedModel) {
	m.mutex.Lock()
	defer m.mutex.Unlock()

	model = NewQueuedModel(m.cfg, m.acquisition)
	model.Update(m.lease, m.acquisition)
	m.model = model

	return
}
