// +build !windows

package leaseui

import (
	"github.com/scjalliance/resourceful/guardian"
	"github.com/scjalliance/resourceful/lease"
)

// DisconnectedModel holds lease information while waiting for a lost
// connection to be recovered.
type DisconnectedModel struct {
	current lease.Lease // The last successful lease acquisition
	err     error       // The error from the last attempted acquisition
}

// NewDisconnectedModel returns a connection model.
func NewDisconnectedModel(current lease.Lease, err error) *DisconnectedModel {
	return &DisconnectedModel{
		current: current,
		err:     err,
	}
}

// Update will update the model for the given response.
func (m *DisconnectedModel) Update(response guardian.Acquisition) {
	if response.Err == nil {
		m.current = response.Lease
	}
	m.err = response.Err
}

// Refresh is a no-op.
func (m *DisconnectedModel) Refresh() {}

// Lease returns the current content of the model.
func (m *DisconnectedModel) Lease() lease.Lease {
	return m.current
}

// Error returns the error from the last lease response.
func (m *DisconnectedModel) Error() error {
	return m.err
}
