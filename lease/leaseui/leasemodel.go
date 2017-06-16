// +build !windows

package leaseui

import "github.com/scjalliance/resourceful/guardian"

// LeaseModel holds lease information while waiting for an active lease to be
// acquired.
type LeaseModel struct {
	response guardian.Acquisition
}

// NewLeaseModel returns a lease model.
func NewLeaseModel(response guardian.Acquisition) *LeaseModel {
	return &LeaseModel{
		response: response,
	}
}

// Update will update the model for the given response.
func (m *LeaseModel) Update(response guardian.Acquisition) {
	m.response = response
}

// Refresh is a no-op.
func (m *LeaseModel) Refresh() {}

// Response returns the current content of the model.
func (m *LeaseModel) Response() guardian.Acquisition {
	return m.response
}
