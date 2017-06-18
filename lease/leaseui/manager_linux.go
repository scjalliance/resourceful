// +build !windows

package leaseui

import "context"

func (m *Manager) queued(ctx context.Context, callback Callback) error {
	<-ctx.Done()
	callback(Success, nil)
	return nil
}

func (m *Manager) connected(ctx context.Context, callback Callback) error {
	<-ctx.Done()
	callback(Success, nil)
	return nil
}

func (m *Manager) disconnected(ctx context.Context, callback Callback) error {
	<-ctx.Done()
	callback(Success, nil)
	return nil
}
