// +build windows

package leaseui

import (
	"context"

	"github.com/lxn/walk"
	"github.com/scjalliance/resourceful/guardian"

	ui "github.com/lxn/walk/declarative"
)

// DisconnectedDialog is a lost connection dialog.
type DisconnectedDialog struct {
	ui        *ui.Dialog
	form      *walk.Dialog
	remaining *walk.Label
	model     *DisconnectedModel
}

// NewDisconnectedDialog returns a new connection dialog with the given view
// model.
func NewDisconnectedDialog(model *DisconnectedModel) (dlg *DisconnectedDialog, err error) {
	dlg = &DisconnectedDialog{
		model: model,
	}

	size := ui.Size{Width: 500, Height: 200}

	dlg.ui = &ui.Dialog{
		Icon:     (*walk.Icon)(model.Icon()),
		Title:    model.Title(),
		MinSize:  size,
		MaxSize:  size,
		Layout:   ui.Grid{Columns: 2},
		AssignTo: &dlg.form,
		Children: []ui.Widget{
			ui.Label{Text: model.Description(), Row: 0, Column: 0, ColumnSpan: 2},
			ui.Label{Text: model.Remaining(), AssignTo: &dlg.remaining, Row: 1, Column: 0, ColumnSpan: 2},
			ui.Label{Text: model.Warning(), Row: 2, Column: 0, ColumnSpan: 2},
			ui.VSpacer{Row: 3, Column: 0, ColumnSpan: 2},
			ui.HSpacer{Row: 4, Column: 0},
			ui.PushButton{
				Row:    4,
				Column: 1,
				Text:   "Ignore",
				OnClicked: func() {
					dlg.form.Close(walk.DlgCmdAbort)
				},
			},
		},
	}

	model.RefreshEvent().Attach(func() {
		dlg.form.SetTitle(model.Title())
		dlg.remaining.SetText(model.Remaining())
	})

	err = dlg.ui.Create(nil)

	return
}

// Run will display the disconnected dialog.
//
// Run blocks until the dialog is closed. The dialog can be closed by the user
// or by cancelling the provided context.
//
// Run returns the result of the dialog. If the context was cancelled run will
// return walk.DlgCmdCancel.
func (dlg *DisconnectedDialog) Run(ctx context.Context) int {
	return runDialog(ctx, dlg.form)
}

// RunWithSync will display the connection dialog. As long as the dialog is
// running its view model will be synchronized with the responses received
// on the provided channel.
//
// RunWithSync blocks until the dialog is closed. The dialog will be closed when
// an active lease is acquired from the channel or the channel is closed. The
// dialog can also be closed by the user or by cancelling the provided context.
//
// RunWithSync returns the result of the dialog. If an active lease was acquired
// it will return Success.
func (dlg *DisconnectedDialog) RunWithSync(ctx context.Context, responses <-chan guardian.Acquisition) (result Result) {
	return runDialogWithSync(ctx, dlg.form, dlg.model, responses, ConnectionAcquired)
}

// Result returns the result returned by the dialog.
//
// Result should be called after the dialog has been closed.
func (dlg *DisconnectedDialog) Result() int {
	return dlg.form.Result()
}

// Cancelled returns true if the dialog was cancelled by the user.
//
// Cancelled should be called after the dialog has been closed.
func (dlg *DisconnectedDialog) Cancelled() bool {
	switch dlg.Result() {
	case walk.DlgCmdAbort, walk.DlgCmdNone:
		return true
	default:
		return false
	}
}
