// +build windows

package leaseui

import (
	"context"
	"fmt"

	"github.com/lxn/walk"

	ui "github.com/lxn/walk/declarative"
)

// DisconnectedDialog is a lost connection dialog.
type DisconnectedDialog struct {
	ui             *ui.Dialog
	form           *walk.Dialog
	remainingLabel *walk.Label
	model          *ConnectionModel
}

// NewDisconnectedDialog returns a new connection dialog with the given view
// model.
func NewDisconnectedDialog(icon *Icon, model *ConnectionModel) (dlg *DisconnectedDialog, err error) {
	dlg = &DisconnectedDialog{
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
			ui.Label{Text: dlg.description(), Row: 0, Column: 0, ColumnSpan: 2},
			ui.Label{Text: dlg.remaining(), AssignTo: &dlg.remainingLabel, Row: 1, Column: 0, ColumnSpan: 2},
			ui.Label{Text: dlg.warning(), Row: 2, Column: 0, ColumnSpan: 2},
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
		dlg.form.SetTitle(dlg.title())
		dlg.remainingLabel.SetText(dlg.remaining())
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

// title returns the title for the dialog.
func (dlg *DisconnectedDialog) title() string {
	return fmt.Sprintf("%s until lease for %s expires", dlg.model.TimeRemaining().String(), dlg.model.ResourceName())
}

// description returns the description for the dialog.
func (dlg *DisconnectedDialog) description() string {
	//return fmt.Sprintf("%s could not be started because %d of %d license(s) are in use.", m.ResourceName(), consumed, m.response.Lease.Limit)
	return fmt.Sprintf("The lease for %s could not be renewed. This is probably due to a network or server failure.", dlg.model.ResourceName())
}

// remaining returns the remaining lease time text for the dialog.
func (dlg *DisconnectedDialog) remaining() string {
	return fmt.Sprintf("%s will forcibly be shut down in %s, when its lease expires.", dlg.model.Program, dlg.model.TimeRemaining().String())
}

// warning returns the warning text for the view.
func (dlg *DisconnectedDialog) warning() string {
	return fmt.Sprintf("Please save your work and close %s before then, or you may lose your work.", dlg.model.Program)
}
