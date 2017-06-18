// +build windows

package leaseui

import (
	"time"

	"github.com/lxn/walk"
	"github.com/scjalliance/resourceful/guardian"
	"github.com/scjalliance/resourceful/lease"
	"github.com/scjalliance/resourceful/policy"
	"github.com/scjalliance/resourceful/strategy"
)

// QueuedModel is a view model for the queued lease dialog.
//
// QueuedModel is not threadsafe. Its operation should be managed by a single
// goroutine, such as the Sync function.
type QueuedModel struct {
	walk.TableModelBase
	walk.SorterBase
	sortColumn int
	sortOrder  walk.SortOrder
	icon       *Icon
	program    string
	consumer   string
	response   guardian.Acquisition
}

// NewQueuedModel returns a queued lease dialog view model.
func NewQueuedModel(icon *Icon, program, consumer string, response guardian.Acquisition) *QueuedModel {
	m := &QueuedModel{
		icon:     icon,
		program:  program,
		consumer: consumer,
		response: response,
	}
	m.PublishRowsReset()
	return m
}

// Icon returns the icon for the view.
func (m *QueuedModel) Icon() *Icon {
	return m.icon
}

// Consumed returns the current number of resources that have been consumed.
func (m *QueuedModel) Consumed() uint {
	strat := m.response.Lease.Strategy
	if !strategy.Valid(strat) || strat == strategy.Empty {
		strat = policy.DefaultStrategy
	}
	stats := m.response.Leases.Stats()
	return stats.Consumed(strat)
}

// ResourceName returns the user-friendly name of the resource.
func (m *QueuedModel) ResourceName() string {
	name := m.response.Lease.ResourceName()
	if name != "" {
		return name
	}

	return m.program
}

// Response returns the current content of the model.
func (m *QueuedModel) Response() guardian.Acquisition {
	return m.response
}

// Update will replace the current model's lease response with the one provided.
func (m *QueuedModel) Update(response guardian.Acquisition) {
	m.response = response
	// TODO: Intelligently compare new data to old, and update invidivual rows
	m.PublishRowsReset()
}

// Refresh will update all of the rows in the lease dialog.
func (m *QueuedModel) Refresh() {
	for r := 0; r < m.RowCount(); r++ {
		m.PublishRowChanged(r)
	}
}

// RowCount returns the number of rows in the model.
func (m *QueuedModel) RowCount() int {
	return len(m.response.Leases)
}

// Value returns the value for the cell at the given row and column.
func (m *QueuedModel) Value(row, col int) interface{} {
	ls := m.response.Leases[row]

	switch col {
	case 0:
		return ls.Status
	case 1:
		return ls.Environment["user.name"]
	case 2:
		return ls.Environment["host.name"]
	case 3:
		if ls.Status == lease.Released {
			return ""
		}
		started := ls.Started.Round(time.Second)
		now := time.Now().Round(time.Second)
		return now.Sub(started).String()
	case 4:
		if ls.Decay == 0 {
			return ""
		}
		if ls.Consumer == m.consumer {
			// There is no decay period for leases belonging to the same consumer.
			return ""
		}
		switch ls.Status {
		case lease.Active:
			return ls.Decay.String()
		case lease.Released:
			available := ls.DecayTime().Round(time.Second)
			now := time.Now().Round(time.Second)
			if now.Before(available) {
				return available.Sub(now).String()
			}
			return "now"
		default:
			return ""
		}
	}
	return nil
}
