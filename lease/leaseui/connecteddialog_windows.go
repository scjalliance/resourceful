// +build windows

package leaseui

import (
	"context"

	"github.com/lxn/walk"

	ui "github.com/lxn/walk/declarative"
)

// ConnectedDialog is a restored connection dialog.
type ConnectedDialog struct {
	ui        *ui.Dialog
	form      *walk.Dialog
	remaining *walk.Label
	model     *ConnectionModel
}

// NewConnectedDialog returns a new connected dialog with the given view
// model.
func NewConnectedDialog(icon *Icon, model *ConnectionModel) (dlg *ConnectedDialog, err error) {
	dlg = &ConnectedDialog{
		model: model,
	}

	size := ui.Size{Width: 500, Height: 200}

	dlg.ui = &ui.Dialog{
		Icon:     (*walk.Icon)(icon),
		Title:    dlg.title(),
		MinSize:  size,
		MaxSize:  size,
		Layout:   ui.Grid{Columns: 2},
		AssignTo: &dlg.form,
		Children: []ui.Widget{
			ui.Label{Text: dlg.condition(), Row: 0, Column: 0, ColumnSpan: 2},
			ui.Label{Text: dlg.risk(), Row: 1, Column: 0, ColumnSpan: 2},
			ui.VSpacer{Row: 3, Column: 0, ColumnSpan: 2},
			ui.HSpacer{Row: 4, Column: 0},
			ui.PushButton{
				Row:    4,
				Column: 1,
				Text:   "OK",
				OnClicked: func() {
					dlg.form.Close(walk.DlgCmdAbort)
				},
			},
		},
	}

	err = dlg.ui.Create(nil)

	return
}

// Run will display the connected dialog.
//
// Run blocks until the dialog is closed. The dialog can be closed by the user
// or by cancelling the provided context.
//
// Run returns the result of the dialog. If the context was cancelled run will
// return walk.DlgCmdCancel.
func (dlg *ConnectedDialog) Run(ctx context.Context) int {
	return runDialog(ctx, dlg.form)
}

// title returns the title for the dialog.
func (dlg *ConnectedDialog) title() string {
	return "Connection Restored"
}

// description returns the condition text for the dialog.
func (dlg *ConnectedDialog) condition() string {
	return "The connection to the server has been restored."
}

// description returns the risk text for the dialog.
func (dlg *ConnectedDialog) risk() string {
	return "Your work is no longer at risk."
}
