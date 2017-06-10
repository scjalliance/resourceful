package leaseui

import (
	"fmt"
	"time"

	"github.com/lxn/walk"
	"github.com/scjalliance/resourceful/guardian"
	"github.com/scjalliance/resourceful/lease"
)

// Model is a view model for the queued lease dialog.
//
// Model is not threadsafe. Its operation should be managed by a single
// goroutine, such as the Sync function.
type Model struct {
	walk.TableModelBase
	walk.SorterBase
	sortColumn int
	sortOrder  walk.SortOrder
	icon       *Icon
	program    string
	consumer   string
	response   guardian.Acquisition
}

// NewModel returns a view model for the given lease set.
func NewModel(icon *Icon, program, consumer string, response guardian.Acquisition) *Model {
	m := &Model{
		icon:     icon,
		program:  program,
		consumer: consumer,
		response: response,
	}
	m.PublishRowsReset()
	return m
}

// Icon returns the icon for the view.
func (m *Model) Icon() *Icon {
	return m.icon
}

// Title returns the title for the view.
func (m *Model) Title() string {
	return fmt.Sprintf("Unable to launch %s", m.program)
}

// Description returns the description for the view.
func (m *Model) Description() string {
	active, released, _ := m.response.Leases.Stats()
	return fmt.Sprintf("%s could not be started because %d of %d license(s) are in use.", m.ResourceName(), active+released, m.response.Lease.Limit)
}

// TableCaption returns the caption for the view's data.
func (m *Model) TableCaption() string {
	return "Here's a list of everyone that's using or waiting for a license right now:"
}

// ResourceName returns the user-friendly name of the resource for the view.
func (m *Model) ResourceName() string {
	name := m.response.Lease.Environment["resource.name"]
	if name != "" {
		return name
	}

	name = m.response.Lease.Environment["resource.id"]
	if name != "" {
		return name
	}

	name = m.response.Lease.Resource
	if name != "" {
		return name
	}

	return m.program
}

// Response returns the current content of the model.
func (m *Model) Response() guardian.Acquisition {
	return m.response
}

// Update will replace the current model's lease response with the one provided.
func (m *Model) Update(response guardian.Acquisition) {
	m.response = response
	// TODO: Intelligently compare new data to old, and update invidivual rows
	m.PublishRowsReset()
}

// RowCount returns the number of rows in the model.
func (m *Model) RowCount() int {
	return len(m.response.Leases)
}

// Value returns the value for the cell at the given row and column.
func (m *Model) Value(row, col int) interface{} {
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
