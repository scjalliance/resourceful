// +build windows

package leaseui

import (
	"fmt"
	"time"

	"github.com/lxn/walk"
	"github.com/scjalliance/resourceful/guardian"
	"github.com/scjalliance/resourceful/lease"
)

// DisconnectedModel is a view model for the connection dialog.
//
// DisconnectedModel is not threadsafe. Its operation should be managed by a
// single goroutine, such as the Sync function.
type DisconnectedModel struct {
	icon     *Icon
	program  string
	consumer string
	current  lease.Lease // The last successful lease acquisition
	err      error       // The error from the last attempted acquisition

	refreshPublisher walk.EventPublisher
}

// NewDisconnectedModel returns a connection dialog view model.
func NewDisconnectedModel(icon *Icon, program, consumer string, current lease.Lease, leaseErr error) *DisconnectedModel {
	m := &DisconnectedModel{
		icon:     icon,
		program:  program,
		consumer: consumer,
		current:  current,
		err:      leaseErr,
	}
	return m
}

// Icon returns the icon for the view.
func (m *DisconnectedModel) Icon() *Icon {
	return m.icon
}

// Title returns the title for the view.
func (m *DisconnectedModel) Title() string {
	return fmt.Sprintf("%s until lease for %s expires", m.TimeRemaining().String(), m.program)
}

// Description returns the description for the view.
func (m *DisconnectedModel) Description() string {
	//return fmt.Sprintf("%s could not be started because %d of %d license(s) are in use.", m.ResourceName(), consumed, m.response.Lease.Limit)
	return fmt.Sprintf("The lease for %s cannot be renewed. This is probably due to a network or server failure.", m.program)
}

// Remaining returns the remaining lease time text for the view.
func (m *DisconnectedModel) Remaining() string {
	return fmt.Sprintf("%s will forcibly be shut down in %s, when its lease expires.", m.program, m.TimeRemaining().String())
}

// Warning returns the warning text for the view.
func (m *DisconnectedModel) Warning() string {
	return fmt.Sprintf("Please save your work and close %s before then, or you may lose your work.", m.program)
}

// Lease returns the current content of the model.
func (m *DisconnectedModel) Lease() lease.Lease {
	return m.current
}

// Error returns the error from the last lease response.
func (m *DisconnectedModel) Error() error {
	return m.err
}

// Update will replace the current model's lease response with the one provided.
func (m *DisconnectedModel) Update(response guardian.Acquisition) {
	if response.Err == nil {
		m.current = response.Lease
	}
	m.err = response.Err
}

// RefreshEvent returns the connection refresh event.
func (m *DisconnectedModel) RefreshEvent() *walk.Event {
	return m.refreshPublisher.Event()
}

// Refresh will update the connection timeout information.
func (m *DisconnectedModel) Refresh() {
	m.refreshPublisher.Publish()
}

// TimeRemaining returns the time remaining until the current lease expires,
// rounded to the nearest whole second.
func (m *DisconnectedModel) TimeRemaining() (remaining time.Duration) {
	now := time.Now().Round(time.Second)
	expiration := m.current.ExpirationTime().Round(time.Second)
	if now.After(expiration) {
		return
	}
	return expiration.Sub(now)
}
