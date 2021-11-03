//go:build windows
// +build windows

package enforcerui

import (
	"fmt"
	"sync"

	"github.com/lxn/walk"
)

// UI is responsible for running the enforcement user interface for the
// current user.
type UI struct {
	icon    *walk.Icon
	name    string
	version string

	mutex sync.Mutex
	state State
	tray  *Tray
}

// New returns creates and starts a new UI instance.
//
// It is the caller's responsiblity to call Close when finished with the UI.
func New(icon *walk.Icon, name, version string) (*UI, error) {
	tray := NewTray(icon, name, version)
	if err := tray.Start(); err != nil {
		return nil, err
	}

	return &UI{
		icon:    icon,
		name:    name,
		version: version,
		tray:    tray,
	}, nil
}

// Close stops the user interface and releases any resources consumed by it.
func (ui *UI) Close() error {
	ui.mutex.Lock()
	defer ui.mutex.Unlock()

	if ui.tray == nil {
		return nil
	}

	err := ui.tray.Stop()
	ui.tray = nil
	return err
}

// Handle instructs the user interface to take action on the given message.
//
// If the UI is not running the message will be dropped. If the UI is
// overloaded this call can block until the UI makes room in its queue.
func (ui *UI) Handle(msg Message) {
	ui.mutex.Lock()
	defer ui.mutex.Unlock()

	if ui.tray == nil {
		return
	}

	//fmt.Printf("Message Received: %#v\n", msg)
	switch msg.Type {
	case TypePolicyUpdate:
		ui.state.Policies = msg.Policies.New
		ui.tray.Update(ui.state)
	case TypeLeaseUpdate:
		ui.state.Leases = msg.Leases.New
		ui.tray.Update(ui.state)
	case TypeProcessTermination:
		ui.tray.Notify(Notice{
			Title:   "No licenses available",
			Message: fmt.Sprintf("The %s process has been terminated because no licenses are available.", msg.ProcTerm.Name),
		})
	}
}
