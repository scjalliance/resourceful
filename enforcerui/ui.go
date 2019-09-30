// +build windows

package enforcerui

import (
	"context"

	"github.com/lxn/walk"
)

// UI is responsible for running the enforcement user interface for the
// current user.
type UI struct {
	icon    *walk.Icon
	name    string
	version string
	msgs    chan Message
}

// New returns a new UI instance.
func New(icon *walk.Icon, name, version string) *UI {
	return &UI{
		icon:    icon,
		name:    name,
		version: version,
		msgs:    make(chan Message, 128),
	}
}

// Run executes the user interface until ctx is cancelled.
func (ui *UI) Run(ctx context.Context) error {
	t := NewTray(ui.icon, ui.name, ui.version)
	if err := t.Start(); err != nil {
		return err
	}
	defer t.Stop()

	for {
		select {
		case msg := <-ui.msgs:
			//fmt.Printf("Message Received: %#v\n", msg)
			switch msg.Type {
			case TypePolicyChange:
				t.Handle(msg)
				//additions, deletions := msg.PolicyChange.Old.Diff
			case TypeProcessTermination:
				t.Handle(msg)
			}
		case <-ctx.Done():
			return ctx.Err()
		}
	}
}

// Handle instructs the user interface to take action on the given message.
//
// If the UI is not running or is overloaded this call can block.
func (ui *UI) Handle(msg Message) {
	ui.msgs <- msg
}
