// +build windows

package leaseui

import (
	"context"

	"github.com/lxn/walk"
	"github.com/scjalliance/resourceful/guardian"

	ui "github.com/lxn/walk/declarative"
)

// ConnectionDialog is a lost connection dialog.
type ConnectionDialog struct {
	ui        *ui.Dialog
	form      *walk.Dialog
	remaining *walk.Label
	model     *ConnectionModel
}

// NewConnectionDialog returns a new connection dialog with the given view
// model.
func NewConnectionDialog(model *ConnectionModel) (dlg *ConnectionDialog, err error) {
	dlg = &ConnectionDialog{
		model: model,
	}

	dlg.ui = &ui.Dialog{
		Icon:     (*walk.Icon)(model.Icon()),
		Title:    model.Title(),
		MinSize:  ui.Size{Width: 300, Height: 200},
		Layout:   ui.Grid{},
		AssignTo: &dlg.form,
		Children: []ui.Widget{
			ui.Composite{
				Layout: ui.Grid{Columns: 1},
				Children: []ui.Widget{
					ui.Label{Text: model.Description()},
					ui.Label{Text: model.Remaining(), AssignTo: &dlg.remaining},
					ui.Label{Text: model.Warning()},
				},
			},
		},
	}

	model.RefreshEvent().Attach(func() {
		dlg.remaining.SetText(model.Remaining())
	})

	err = dlg.ui.Create(nil)

	return
}

// Run will display the connection dialog.
//
// Run blocks until the dialog is closed. The dialog can be closed by the user
// or by cancelling the provided context.
//
// Run returns the result of the dialog. If the context was cancelled run will
// return walk.DlgCmdCancel.
func (dlg *ConnectionDialog) Run(ctx context.Context) int {
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
func (dlg *ConnectionDialog) RunWithSync(ctx context.Context, responses <-chan guardian.Acquisition) (result Result) {
	return runDialogWithSync(ctx, dlg.form, dlg.model, responses, ConnectionAcquired)
}

// Result returns the result returned by the dialog.
//
// Result should be called after the dialog has been closed.
func (dlg *ConnectionDialog) Result() int {
	return dlg.form.Result()
}

// Cancelled returns true if the dialog was cancelled by the user.
//
// Cancelled should be called after the dialog has been closed.
func (dlg *ConnectionDialog) Cancelled() bool {
	switch dlg.Result() {
	case walk.DlgCmdAbort, walk.DlgCmdNone:
		return true
	default:
		return false
	}
}
