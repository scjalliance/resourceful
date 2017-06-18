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
	icon     *Icon
	program  string
	consumer string
	current  lease.Lease // The last successful lease acquisition
	err      error       // The error from the last attempted acquisition

	refreshPublisher walk.EventPublisher
}

// NewConnectionModel returns a connection dialog view model.
func NewConnectionModel(icon *Icon, program, consumer string, current lease.Lease, leaseErr error) *ConnectionModel {
	m := &ConnectionModel{
		icon:     icon,
		program:  program,
		consumer: consumer,
		current:  current,
		err:      leaseErr,
	}
	return m
}

// Icon returns the icon for the view.
func (m *ConnectionModel) Icon() *Icon {
	return m.icon
}

// ResourceName returns the user-friendly name of the resource.
func (m *ConnectionModel) ResourceName() string {
	name := m.current.ResourceName()
	if name != "" {
		return name
	}

	return m.program
}

// Lease returns the current content of the model.
func (m *ConnectionModel) Lease() lease.Lease {
	return m.current
}

// Error returns the error from the last lease response.
func (m *ConnectionModel) Error() error {
	return m.err
}

// Update will replace the current model's lease response with the one provided.
func (m *ConnectionModel) Update(response guardian.Acquisition) {
	if response.Err == nil {
		m.current = response.Lease
	}
	m.err = response.Err
}

// RefreshEvent returns the connection refresh event.
func (m *ConnectionModel) RefreshEvent() *walk.Event {
	return m.refreshPublisher.Event()
}

// Refresh will update the connection timeout information.
func (m *ConnectionModel) Refresh() {
	m.refreshPublisher.Publish()
}

// TimeRemaining returns the time remaining until the current lease expires,
// rounded to the nearest whole second.
func (m *ConnectionModel) TimeRemaining() (remaining time.Duration) {
	now := time.Now().Round(time.Second)
	expiration := m.current.ExpirationTime().Round(time.Second)
	if now.After(expiration) {
		return
	}
	return expiration.Sub(now)
}
