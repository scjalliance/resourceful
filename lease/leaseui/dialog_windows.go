// +build windows

package leaseui

import (
	"context"

	"github.com/lxn/walk"
	"github.com/scjalliance/resourceful/guardian"

	ui "github.com/lxn/walk/declarative"
)

// Dialog is a queued lease dialog.
type Dialog struct {
	ui    *ui.Dialog
	form  *walk.Dialog
	model *Model
}

// New returns a new queued lease dialog with the given view model.
func New(model *Model) (dlg *Dialog, err error) {
	dlg = &Dialog{
		model: model,
	}

	dlg.ui = &ui.Dialog{
		Icon:     (*walk.Icon)(model.Icon()),
		Title:    model.Title(),
		MinSize:  ui.Size{Width: 600, Height: 400},
		Layout:   ui.Grid{},
		AssignTo: &dlg.form,
		Children: []ui.Widget{
			ui.Label{Text: model.Description(), Row: 0, Column: 0, ColumnSpan: 2},
			ui.VSpacer{Size: 1, Row: 1, Column: 0, ColumnSpan: 2},
			ui.Label{Text: model.TableCaption(), Row: 2, Column: 0, ColumnSpan: 2},
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
func (dlg *Dialog) Run(ctx context.Context) int {
	select {
	case <-ctx.Done():
		return walk.DlgCmdCancel
	default:
	}

	closed := make(chan struct{})
	defer close(closed)

	go func() {
		select {
		case <-closed:
		case <-ctx.Done():
			// Here we use the synchronize function to ensure that our call to Close
			// pushes the WM_CLOSE message onto the message queue of the correct
			// thread. If we call Close() directly it could fail silently and
			// deadlock.
			dlg.form.Synchronize(func() {
				dlg.form.Close(walk.DlgCmdCancel)
			})
		}
	}()

	return dlg.form.Run()
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
func (dlg *Dialog) RunWithSync(ctx context.Context, responses <-chan guardian.Acquisition) int {
	// Keep the dialog in sync with lease responses
	ctx, cancel := context.WithCancel(ctx)
	go func() {
		Sync(dlg.model, responses) // Block until acquisition or shutdown
		cancel()
	}()

	return dlg.Run(ctx)
}

// Result returns the result returned by the dialog.
//
// Result should be called after the dialog has been closed.
func (dlg *Dialog) Result() int {
	return dlg.form.Result()
}

// Cancelled returns true if the dialog was cancelled by the user.
//
// Cancelled should be called after the dialog has been closed.
func (dlg *Dialog) Cancelled() bool {
	switch dlg.Result() {
	case walk.DlgCmdAbort, walk.DlgCmdNone:
		return true
	default:
		return false
	}
}
