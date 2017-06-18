// +build !windows

package leaseui

import "github.com/scjalliance/resourceful/guardian"

// QueuedModel holds lease information while waiting for an active lease to be
// acquired.
type QueuedModel struct {
	response guardian.Acquisition
}

// NewQueuedModel returns a lease model.
func NewQueuedModel(response guardian.Acquisition) *QueuedModel {
	return &QueuedModel{
		response: response,
	}
}

// Update will update the model for the given response.
func (m *QueuedModel) Update(response guardian.Acquisition) {
	m.response = response
}

// Refresh is a no-op.
func (m *QueuedModel) Refresh() {}

// Response returns the current content of the model.
func (m *QueuedModel) Response() guardian.Acquisition {
	return m.response
}
