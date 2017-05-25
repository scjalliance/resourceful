// +build windows

package main

import (
	"fmt"
	"time"

	"github.com/lxn/walk"
	. "github.com/lxn/walk/declarative"
	"github.com/scjalliance/resourceful/guardian/transport"
	"github.com/scjalliance/resourceful/lease"
)

func msgBox(title, msg string) {
	walk.MsgBox(nil, title, msg, walk.MsgBoxIconInformation)
}

func leasePendingDlg(program string, response transport.AcquireResponse) {
	icon, err := walk.NewIconFromResourceId(5)
	if err != nil {
		icon = walk.IconInformation()
	}

	resName := response.Resource
	if rn, ok := response.Environment["resource.name"]; ok {
		resName = rn
	}

	active, released, _ := response.Leases.Stats()

	title := fmt.Sprintf("Unable to launch %s", program)
	label1 := fmt.Sprintf("%s could not be started because %d of %d license(s) are in use.", resName, active+released, response.Lease.Limit)
	label2 := fmt.Sprintf("Here's a list of everyone that's using or waiting for a license right now:")

	var dlg *walk.Dialog

	// Create a lease view model that announces a refresh every second
	m := &leaseModel{items: response.Leases}
	done := make(chan struct{})
	defer close(done)
	go func() {
		timeTicker := time.NewTicker(time.Second)
		refreshTicker := time.NewTicker(time.Second * 10)
		defer timeTicker.Stop()
		defer refreshTicker.Stop()
		for {
			select {
			case <-done:
				return
			case <-timeTicker.C:
				for r := 0; r < m.RowCount(); r++ {
					m.PublishRowChanged(r)
				}
			case <-refreshTicker.C:
				// TODO: Continue trying to acquire a lease?
			}
		}
	}()

	Dialog{
		Icon:     icon,
		Title:    title,
		MinSize:  Size{Width: 600, Height: 400},
		Layout:   Grid{},
		AssignTo: &dlg,
		Children: []Widget{
			Label{Text: label1, Row: 0, Column: 0, ColumnSpan: 2},
			VSpacer{Size: 1, Row: 1, Column: 0, ColumnSpan: 2},
			Label{Text: label2, Row: 2, Column: 0, ColumnSpan: 2},
			TableView{
				Row:        3,
				Column:     0,
				ColumnSpan: 2,
				Columns: []TableViewColumn{
					TableViewColumn{Name: "Status", Width: 50},
					TableViewColumn{Title: "User", Width: 200},
					TableViewColumn{Name: "Computer", Width: 150},
					TableViewColumn{Name: "Time", Width: 50},
					TableViewColumn{Name: "Earliest Availability", Width: 110},
				},
				Model: m,
			},
			HSpacer{Row: 4, Column: 0},
			PushButton{
				Row:    4,
				Column: 1,
				Text:   "OK",
				OnClicked: func() {
					dlg.Accept()
				},
			},
		},
	}.Run(nil)
}

type leaseModel struct {
	walk.TableModelBase
	walk.SorterBase
	sortColumn int
	sortOrder  walk.SortOrder
	items      lease.Set
}

func newLeaseModel(leases lease.Set) *leaseModel {
	m := new(leaseModel)
	m.items = leases
	m.PublishRowsReset()
	return m
}

func (lm *leaseModel) RowCount() int {
	return len(lm.items)
}

func (lm *leaseModel) Value(row, col int) interface{} {
	ls := lm.items[row]

	switch col {
	case 0:
		return ls.Status
	case 1:
		return ls.Environment["user.name"]
	case 2:
		return ls.Environment["host.name"]
	case 3:
		started := ls.Started.Round(time.Second)
		now := time.Now().Round(time.Second)
		return now.Sub(started).String()
	case 4:
		if ls.Decay == 0 {
			return ""
		}
		switch ls.Status {
		case lease.Active:
			return ls.Decay.String()
		case lease.Released:
			available := ls.Released.Add(ls.Decay)
			return available.Sub(time.Now()).String()
		default:
			return ""
		}
	}
	return nil
}

/*
func (lm leaseModel) Checked(row int) bool {
	return false
}

func (lm leaseModel) SetChecked(row int, checked bool) error {
	return nil
}

func (lm leaseModel) ResetRows() {
}

func (lm leaseModel) Sort(col int, order walk.SortOrder) error {
	return nil
}
*/
