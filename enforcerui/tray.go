//go:build windows
// +build windows

package enforcerui

import (
	"context"
	"errors"
	"fmt"
	"runtime"
	"sync"

	"github.com/lxn/walk"
	"github.com/scjalliance/resourceful/policy"
)

type trayData struct {
	Window     *walk.MainWindow
	Tray       *walk.NotifyIcon
	Completion <-chan int
	Err        error
}

// Tray is an enforcer system tray agent.
type Tray struct {
	icon    *walk.Icon
	name    string
	version string

	mutex   sync.RWMutex
	stop    context.CancelFunc
	stopped <-chan struct{}
	states  chan<- State
	notices chan<- Notice
}

// NewTray returns a new system tray instance.
func NewTray(icon *walk.Icon, name, version string) *Tray {
	return &Tray{
		icon:    icon,
		name:    name,
		version: version,
	}
}

// Start causes the tray to begin operation.
func (t *Tray) Start() error {
	t.mutex.Lock()
	defer t.mutex.Unlock()

	if t.stop != nil {
		return errors.New("the tray instance is already running")
	}

	// Create the tray objects and start the tray on a dedicated thread
	startup := make(chan trayData)
	go t.run(startup)

	// Collect information about the tray once it's been initialized
	data := <-startup
	if data.Err != nil {
		return data.Err
	}

	// Prepare a context to stop the tray on request
	ctx, stop := context.WithCancel(context.Background())
	stopped := make(chan struct{})

	// Prepare input channels
	states := make(chan State, 128)
	notices := make(chan Notice, 128)

	// Start a manager that updates the tray in response to input
	go t.manage(ctx, data.Window, data.Tray, data.Completion, stopped, states, notices)

	t.stop = stop
	t.stopped = stopped
	t.states = states
	t.notices = notices

	return nil
}

// Stop instructs the tray to cease operation and waits for it to close.
//
// The tray will stop automatically if its state channel is closed.
func (t *Tray) Stop() error {
	t.mutex.RLock()
	stop, stopped := t.stop, t.stopped
	t.mutex.RUnlock()

	if t.stop == nil {
		return errors.New("the tray instance is not running")
	}

	stop()

	//fmt.Printf("waiting for close\n")
	<-stopped

	return nil
}

// Update updates the state of the tray.
func (t *Tray) Update(state State) {
	t.mutex.RLock()
	defer t.mutex.RUnlock()
	if t.states != nil {
		select {
		case t.states <- state:
		default:
		}
	}
}

// Notify causes the tray to send a notification.
func (t *Tray) Notify(notice Notice) {
	t.mutex.RLock()
	defer t.mutex.RUnlock()
	if t.notices != nil {
		select {
		case t.notices <- notice:
		default:
		}
	}
}

func (t *Tray) run(startup chan<- trayData) {
	runtime.LockOSThread()
	defer runtime.UnlockOSThread()

	mw, ni, err := createSystemTray(t.icon)
	if err != nil {
		startup <- trayData{Err: err}
		close(startup)
	}
	defer ni.Dispose()
	defer mw.Dispose()

	completion := make(chan int)
	defer close(completion)

	startup <- trayData{
		Window:     mw,
		Tray:       ni,
		Completion: completion,
	}
	close(startup)

	result := mw.Run()

	t.mutex.Lock()
	t.stop = nil
	t.stopped = nil
	close(t.states)
	t.states = nil
	close(t.notices)
	t.notices = nil
	t.mutex.Unlock()

	completion <- result
}

func (t *Tray) manage(ctx context.Context, window *walk.MainWindow, ni *walk.NotifyIcon, completion <-chan int, stopped chan<- struct{}, states <-chan State, notices <-chan Notice) {
	defer close(stopped)
	for {
		select {
		case <-completion:
			return
		case state := <-states:
			summary := state.Summary()
			info := fmt.Sprintf("%s %s", t.name, t.version)
			window.Synchronize(func() {
				// Update tool tip
				//fmt.Println(summary)
				ni.SetToolTip(summary)

				// Update menu
				actions := ni.ContextMenu().Actions()
				actions.Clear()
				{
					action := walk.NewAction()
					action.SetText(info)
					action.SetEnabled(false)
					actions.Add(action)
				}
				actions.Add(walk.NewSeparatorAction())
				for _, pol := range state.Policies {
					action := walk.NewAction()
					desc := pol.Resource
					if name := pol.Properties["resource.name"]; name != "" {
						desc = name
					}
					if pol.Limit != policy.DefaultLimit {
						desc = fmt.Sprintf("%s: %d", desc, pol.Limit)
					}
					action.SetText(desc)
					actions.Add(action)
				}
			})
		case notice := <-notices:
			window.Synchronize(func() {
				ni.ShowCustom(
					notice.Title,
					notice.Message,
					t.icon,
				)
			})
		case <-ctx.Done():
			// Here we use the synchronize function to ensure that our call to Close
			// pushes the WM_CLOSE message onto the message queue of the correct
			// thread. If we call Close() directly it could fail silently and
			// deadlock.
			window.Synchronize(func() {
				window.Close()
			})
			<-completion
			return
		}
	}
}

func createSystemTray(icon *walk.Icon) (mw *walk.MainWindow, ni *walk.NotifyIcon, err error) {
	mw, err = walk.NewMainWindow()
	if err != nil {
		return nil, nil, fmt.Errorf("failed to create window: %v", err)
	}

	ni, err = createNotifyIcon(mw, icon)
	if err != nil {
		mw.Dispose()
		return nil, nil, fmt.Errorf("failed to create system tray: %v", err)
	}

	return mw, ni, nil
}

func createNotifyIcon(form walk.Form, icon *walk.Icon) (*walk.NotifyIcon, error) {
	ni, err := walk.NewNotifyIcon(form)
	if err != nil {
		return nil, fmt.Errorf("creation failed: %v", err)
	}

	if err := ni.SetIcon(icon); err != nil {
		ni.Dispose()
		return nil, fmt.Errorf("unable to set icon: %v", err)
	}

	if err := ni.SetVisible(true); err != nil {
		ni.Dispose()
		return nil, fmt.Errorf("unable to make the system tray visible: %v", err)
	}

	return ni, nil
}
