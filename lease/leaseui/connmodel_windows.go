// +build windows

package leaseui

import (
	"fmt"
	"time"

	"github.com/lxn/walk"
	"github.com/scjalliance/resourceful/guardian"
	"github.com/scjalliance/resourceful/lease"
)

// ConnectionModel is a view model for the connection dialog.
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

// Title returns the title for the view.
func (m *ConnectionModel) Title() string {
	return fmt.Sprintf("License for %s has been lost", m.program)
}

// Description returns the description for the view.
func (m *ConnectionModel) Description() string {
	//return fmt.Sprintf("%s could not be started because %d of %d license(s) are in use.", m.ResourceName(), consumed, m.response.Lease.Limit)
	return fmt.Sprintf("The lease for %s cannot be renewed. This is probably due to a network or server failure.", m.program)
}

// Remaining returns the remaining lease time text for the view.
func (m *ConnectionModel) Remaining() string {
	now := time.Now().Round(time.Second)
	remaining := m.current.ExpirationTime().Round(time.Second).Sub(now)
	return fmt.Sprintf("%s will forcibly be shut down in %s, when its lease expires.", m.program, remaining.String())
}

// Warning returns the warning text for the view.
func (m *ConnectionModel) Warning() string {
	return fmt.Sprintf("Please save your work and close %s before then, or you may lose your work.", m.program)
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
