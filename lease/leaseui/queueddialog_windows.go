// +build windows

package leaseui

import (
	"context"
	"fmt"

	"github.com/lxn/walk"
	"github.com/scjalliance/resourceful/guardian"

	ui "github.com/lxn/walk/declarative"
)

// QueuedDialog is a queued lease dialog.
type QueuedDialog struct {
	ui    *ui.Dialog
	form  *walk.Dialog
	model *QueuedModel
}

// NewQueuedDialog returns a new queued lease dialog with the given view model.
func NewQueuedDialog(model *QueuedModel) (dlg *QueuedDialog, err error) {
	dlg = &QueuedDialog{
		model: model,
	}

	dlg.ui = &ui.Dialog{
		Icon:     (*walk.Icon)(model.Icon()),
		Title:    dlg.title(),
		MinSize:  ui.Size{Width: 600, Height: 400},
		Layout:   ui.Grid{},
		AssignTo: &dlg.form,
		Children: []ui.Widget{
			ui.Label{Text: dlg.description(), Row: 0, Column: 0, ColumnSpan: 2},
			ui.VSpacer{Size: 1, Row: 1, Column: 0, ColumnSpan: 2},
			ui.Label{Text: dlg.tableCaption(), Row: 2, Column: 0, ColumnSpan: 2},
			ui.TableView{
				Row:        3,
				Column:     0,
				ColumnSpan: 2,
				Columns: []ui.TableViewColumn{
					ui.TableViewColumn{Name: "Status", Width: 60},
					ui.TableViewColumn{Title: "User", Width: 200},
					ui.TableViewColumn{Name: "Computer", Width: 140},
					ui.TableViewColumn{Name: "Time", Width: 50},
					ui.TableViewColumn{Name: "Earliest Availability", Width: 110},
				},
				Model: model,
			},
			ui.HSpacer{Row: 4, Column: 0},
			ui.PushButton{
				Row:    4,
				Column: 1,
				Text:   "Cancel",
				OnClicked: func() {
					dlg.form.Close(walk.DlgCmdAbort)
				},
			},
		},
	}

	err = dlg.ui.Create(nil)

	return
}

// Run will display the queued lease dialog.
//
// Run blocks until the dialog is closed. The dialog can be closed by the user
// or by cancelling the provided context.
//
// Run returns the result of the dialog. If the context was cancelled run will
// return walk.DlgCmdCancel.
func (dlg *QueuedDialog) Run(ctx context.Context) int {
	return runDialog(ctx, dlg.form)
}

// RunWithSync will display the queued lease dialog. As long as the dialog is
// running its view model will be synchronized with the responses received
// on the provided channel.
//
// RunWithSync blocks until the dialog is closed. The dialog will be closed when
// an active lease is acquired from the channel or the channel is closed. The
// dialog can also be closed by the user or by cancelling the provided context.
//
// RunWithSync returns the result of the dialog. If the context was cancelled,
// an active lease was acquired or the channel was closed it will return
// walk.DlgCmdCancel.
func (dlg *QueuedDialog) RunWithSync(ctx context.Context, responses <-chan guardian.Acquisition) (result Result) {
	return runDialogWithSync(ctx, dlg.form, dlg.model, responses, ActiveLeaseAcquired)
}

// title returns the title for the dialog.
func (dlg *QueuedDialog) title() string {
	return fmt.Sprintf("Unable to launch %s", dlg.model.program)
}

// description returns the description for the dialog.
func (dlg *QueuedDialog) description() string {
	return fmt.Sprintf("%s could not be started because %d of %d license(s) are in use.", dlg.model.ResourceName(), dlg.model.Consumed(), dlg.model.response.Lease.Limit)
}

// tableCaption returns the caption for the table.
func (dlg *QueuedDialog) tableCaption() string {
	return "Here's a list of everyone that's using or waiting for a license right now:"
}
