// +build windows

package leaseui

import (
	"time"

	"github.com/lxn/walk"
	"github.com/scjalliance/resourceful/guardian"
	"github.com/scjalliance/resourceful/lease"
)

// ConnectionModel is a view model for the disconnected dialog.
//
// ConnectionModel is not threadsafe. Its operation should be managed by a
// single goroutine, such as the Sync function.
type ConnectionModel struct {
	Config
	lease lease.Lease // The last successful lease acquisition

	refreshPublisher walk.EventPublisher
}

// NewConnectionModel returns a connection dialog view model.
func NewConnectionModel(config Config, ls lease.Lease) *ConnectionModel {
	m := &ConnectionModel{
		Config: config,
		lease:  ls,
	}
	return m
}

// ResourceName returns the user-friendly name of the resource.
func (m *ConnectionModel) ResourceName() string {
	name := m.lease.ResourceName()
	if name != "" {
		return name
	}

	return m.Program
}

// Lease returns the current content of the model.
func (m *ConnectionModel) Lease() lease.Lease {
	return m.lease
}

// Update will replace the current model's lease response with the one provided.
func (m *ConnectionModel) Update(ls lease.Lease, acquisition guardian.Acquisition) {
	m.lease = ls
}

// RefreshEvent returns the connection refresh event.
func (m *ConnectionModel) RefreshEvent() *walk.Event {
	return m.refreshPublisher.Event()
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
	expiration := m.lease.ExpirationTime().Round(time.Second)
	if now.After(expiration) {
		return
	}
	return expiration.Sub(now)
}
