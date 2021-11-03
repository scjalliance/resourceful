//go:build windows
// +build windows

package leaseui

import (
	"context"
	"fmt"

	"github.com/lxn/walk"

	ui "github.com/lxn/walk/declarative"
)

// StartupDialog is a startup failure dialog.
type StartupDialog struct {
	ui         *ui.Dialog
	form       *walk.Dialog
	detailsBox *walk.TextEdit
	model      *ConnectionModel
}

// NewStartupDialog returns a new startup dialog with the given view model.
func NewStartupDialog(icon *Icon, model *ConnectionModel) (dlg *StartupDialog, err error) {
	dlg = &StartupDialog{
		model: model,
	}

	size := ui.Size{Width: 600, Height: 200}

	dlg.ui = &ui.Dialog{
		Icon:     (*walk.Icon)(icon),
		Title:    dlg.title(),
		MinSize:  size,
		MaxSize:  size,
		Layout:   ui.Grid{Columns: 2},
		AssignTo: &dlg.form,
		Children: []ui.Widget{
			ui.Label{Text: dlg.situation(), Row: 0, Column: 0, ColumnSpan: 2},
			ui.Label{Text: dlg.cause(), Row: 1, Column: 0, ColumnSpan: 2},
			ui.Label{Text: dlg.outcome(), Row: 2, Column: 0, ColumnSpan: 2},
			ui.Composite{
				Row:        3,
				Column:     0,
				ColumnSpan: 2,
				Layout:     ui.Grid{Columns: 2},
				Children: []ui.Widget{
					ui.Label{Text: dlg.detailsCaption(), Row: 0, Column: 0},
					ui.TextEdit{Text: dlg.details(), AssignTo: &dlg.detailsBox, ReadOnly: true, Row: 0, Column: 1, RowSpan: 2},
					ui.VSpacer{Row: 1, Column: 0},
				},
			},
			ui.VSpacer{Row: 5, Column: 0, ColumnSpan: 2},
			ui.HSpacer{Row: 6, Column: 0},
			ui.PushButton{
				Row:    6,
				Column: 1,
				Text:   "Close",
				OnClicked: func() {
					dlg.form.Close(walk.DlgCmdAbort)
				},
			},
		},
	}

	model.UpdateEvent().Attach(func() {
		dlg.detailsBox.SetText(dlg.details())
	})

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
func (dlg *StartupDialog) Run(ctx context.Context) int {
	return runDialog(ctx, dlg.form)
}

// title returns the title for the dialog.
func (dlg *StartupDialog) title() string {
	return fmt.Sprintf("Unable to launch %s", dlg.model.Program)
}

// situation returns the situation text for the dialog.
func (dlg *StartupDialog) situation() string {
	return fmt.Sprintf("%s could not be started because an active lease could not be acquired.", dlg.model.ResourceName())
}

// cause returns the cause text for the dialog.
func (dlg *StartupDialog) cause() string {
	return "This is probably due to a network or server failure."
}

// outcome returns the outcome text for the dialog.
func (dlg *StartupDialog) outcome() string {
	return "It will be started automatically once the server can be contacted and a lease has been acquired."
}

// detailsCaption returns the error details caption for the dialog.
func (dlg *StartupDialog) detailsCaption() string {
	return "Error Details:"
}

// details returns the error details for the dialog.
func (dlg *StartupDialog) details() string {
	if err := dlg.model.Error(); err != nil {
		return fmt.Sprintf("%v", dlg.model.Error())
	}
	return ""
}
