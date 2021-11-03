//go:build windows
// +build windows

package leaseui

import (
	"sync"
	"time"

	"github.com/lxn/walk"
	"github.com/scjalliance/resourceful/lease"
	"github.com/scjalliance/resourceful/policy"
	"github.com/scjalliance/resourceful/strategy"
)

// QueuedModel is a view model for the queued lease dialog.
//
// QueuedModel is not threadsafe. Its operation should be managed by a single
// goroutine, such as the Sync function.
type QueuedModel struct {
	Config

	mutex sync.RWMutex
	walk.TableModelBase
	walk.SorterBase
	sortColumn int
	sortOrder  walk.SortOrder
	state      lease.State
	closed     bool
}

// NewQueuedModel returns a queued lease dialog view model.
func NewQueuedModel(config Config, state lease.State) *QueuedModel {
	m := &QueuedModel{
		Config: config,
		state:  state,
	}
	m.PublishRowsReset()
	return m
}

// Consumed returns the current number of resources that have been consumed.
func (m *QueuedModel) Consumed() uint {
	m.mutex.RLock()
	defer m.mutex.RUnlock()

	strat := m.state.Lease.Strategy
	if !strategy.Valid(strat) || strat == strategy.Empty {
		strat = policy.DefaultStrategy
	}
	stats := m.state.Leases.Stats()
	return stats.Consumed(strat)
}

// ResourceName returns the user-friendly name of the resource.
func (m *QueuedModel) ResourceName() string {
	m.mutex.RLock()
	defer m.mutex.RUnlock()

	name := m.state.Lease.ResourceName()
	if name != "" {
		return name
	}

	return m.Program
}

// Update will replace the current model's lease response with the one provided.
func (m *QueuedModel) Update(state lease.State) {
	m.mutex.Lock()
	m.state = state
	m.mutex.Unlock()

	m.mutex.RLock()
	defer m.mutex.RUnlock()

	if m.closed {
		return
	}

	// TODO: Intelligently compare new data to old, and update invidivual rows
	m.PublishRowsReset()
}

// Refresh will update all of the rows in the lease dialog.
func (m *QueuedModel) Refresh() {
	m.mutex.RLock()
	defer m.mutex.RUnlock()

	if m.closed {
		return
	}

	for r := 0; r < m.RowCount(); r++ {
		m.PublishRowChanged(r)
	}
}

// Close will prevent the model from broadcasting updates.
func (m *QueuedModel) Close() {
	m.mutex.Lock()
	m.mutex.Unlock()
	m.closed = true
}

// RowCount returns the number of rows in the model.
func (m *QueuedModel) RowCount() int {
	return len(m.state.Leases)
}

// Value returns the value for the cell at the given row and column.
func (m *QueuedModel) Value(row, col int) interface{} {
	ls := m.state.Leases[row]

	switch col {
	case 0:
		return ls.Status
	case 1:
		return ls.Properties["user.name"]
	case 2:
		return ls.Properties["host.name"]
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
		if ls.Instance.Host == m.Instance.Host && ls.Instance.User == m.Instance.User {
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
