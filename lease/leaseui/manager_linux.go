// +build !windows

package leaseui

import "context"

func (m *Manager) none(ctx context.Context, callback Callback) error {
	<-ctx.Done()
	callback(None, Success, nil)
	return nil
}

func (m *Manager) startup(ctx context.Context, callback Callback) error {
	<-ctx.Done()
	callback(Startup, Success, nil)
	return nil
}

func (m *Manager) queued(ctx context.Context, callback Callback) error {
	<-ctx.Done()
	callback(Queued, Success, nil)
	return nil
}

func (m *Manager) connected(ctx context.Context, callback Callback) error {
	<-ctx.Done()
	callback(Connected, Success, nil)
	return nil
}

func (m *Manager) disconnected(ctx context.Context, callback Callback) error {
	<-ctx.Done()
	callback(Disconnected, Success, nil)
	return nil
}
