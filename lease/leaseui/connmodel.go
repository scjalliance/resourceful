// +build !windows

package leaseui

import (
	"github.com/scjalliance/resourceful/guardian"
	"github.com/scjalliance/resourceful/lease"
)

// ConnectionModel holds lease information while waiting for a lost
// connection to be recovered.
type ConnectionModel struct {
	current lease.Lease // The last successful lease acquisition
	err     error       // The error from the last attempted acquisition
}

// NewConnectionModel returns a connection model.
func NewConnectionModel(current lease.Lease, err error) *ConnectionModel {
	return &ConnectionModel{
		current: current,
		err:     err,
	}
}

// Update will update the model for the given response.
func (m *ConnectionModel) Update(response guardian.Acquisition) {
	if response.Err == nil {
		m.current = response.Lease
	}
	m.err = response.Err
}

// Refresh is a no-op.
func (m *ConnectionModel) Refresh() {}

// Lease returns the current content of the model.
func (m *ConnectionModel) Lease() lease.Lease {
	return m.current
}

// Error returns the error from the last lease response.
func (m *ConnectionModel) Error() error {
	return m.err
}
