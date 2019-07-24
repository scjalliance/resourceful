// +build windows

package leaseui

import (
	"time"

	"github.com/lxn/walk"
	"github.com/scjalliance/resourceful/lease"
)

// ConnectionModel is a view model for the disconnected dialog.
//
// ConnectionModel is not threadsafe. Its operation should be managed by a
// single goroutine, such as the Sync function.
type ConnectionModel struct {
	Config
	state lease.State

	updatePublisher  walk.EventPublisher
	refreshPublisher walk.EventPublisher
}

// NewConnectionModel returns a connection dialog view model.
func NewConnectionModel(config Config, state lease.State) *ConnectionModel {
	m := &ConnectionModel{
		Config: config,
		state:  state,
	}
	return m
}

// ResourceName returns the user-friendly name of the resource.
func (m *ConnectionModel) ResourceName() string {
	name := m.state.Lease.ResourceName()
	if name != "" {
		return name
	}

	return m.Program
}

// Lease returns the current content of the model.
func (m *ConnectionModel) Lease() lease.Lease {
	return m.state.Lease
}

// Error returns the last connection error.
func (m *ConnectionModel) Error() error {
	return m.state.Err
}

// Update will replace the current model's lease response with the one provided.
func (m *ConnectionModel) Update(state lease.State) {
	m.state = state
	m.updatePublisher.Publish()
}

// RefreshEvent returns the connection refresh event.
func (m *ConnectionModel) RefreshEvent() *walk.Event {
	return m.refreshPublisher.Event()
}

// UpdateEvent returns the connection update event.
func (m *ConnectionModel) UpdateEvent() *walk.Event {
	return m.updatePublisher.Event()
}

// Refresh will update the connection timeout information.
func (m *ConnectionModel) Refresh() {
	m.refreshPublisher.Publish()
}

// Close will prevent the model from broadcasting updates.
func (m *ConnectionModel) Close() {
}

// TimeRemaining returns the time remaining until the current lease expires,
// rounded to the nearest whole second.
func (m *ConnectionModel) TimeRemaining() (remaining time.Duration) {
	now := time.Now().Round(time.Second)
	expiration := m.state.Lease.ExpirationTime().Round(time.Second)
	if now.After(expiration) {
		return
	}
	return expiration.Sub(now)
}
