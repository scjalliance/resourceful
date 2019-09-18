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

// Tray runs the enforcer's tray.
type Tray struct {
	icon *walk.Icon

	mutex   sync.RWMutex
	stop    context.CancelFunc
	stopped <-chan struct{}
	msgs    chan Message
}

// NewTray returns a new system tray instance.
func NewTray(icon *walk.Icon) *Tray {
	return &Tray{
		icon: icon,
		msgs: make(chan Message, 128),
	}
}

// Start causes the tray to begin operation.
func (t *Tray) Start() error {
	t.mutex.Lock()
	defer t.mutex.Unlock()

	if t.stop != nil {
		return errors.New("the tray instance is already running")
	}

	// Create the tray objects and start the tray
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

	// Start a manager that updates the tray in response to external messages
	go t.manage(ctx, data.Window, data.Tray, data.Completion, stopped)

	t.stop = stop
	t.stopped = stopped

	return nil
}

// Stop instructs the tray to cease operation and waits for it to close.
func (t *Tray) Stop() error {
	t.mutex.RLock()
	stop, stopped := t.stop, t.stopped
	t.mutex.RUnlock()

	if t.stop == nil {
		return errors.New("the tray instance is not running")
	}

	stop()

	fmt.Printf("waiting for close\n")
	<-stopped

	return nil
}

// Handle updates the tray state to reflect the given message.
func (t *Tray) Handle(msg Message) {
	t.msgs <- msg
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

	fmt.Printf("starting message pump\n")
	result := mw.Run()
	fmt.Printf("stopped message pump\n")

	t.mutex.Lock()
	t.stop = nil
	t.stopped = nil
	t.mutex.Unlock()

	completion <- result
}

func (t *Tray) manage(ctx context.Context, window *walk.MainWindow, ni *walk.NotifyIcon, completion <-chan int, stopped chan<- struct{}) {
	defer close(stopped)
	for {
		select {
		case <-completion:
			return
		case msg := <-t.msgs:
			switch msg.Type {
			case "process.terminated":
				title := fmt.Sprintf("%s Terminated", msg.ProcTerm.Name)
				text := "A proces has been terminated because there aren't enough licenses available."
				window.Synchronize(func() {
					ni.ShowWarning(title, text)
				})
			case "policy.change":
				var summary string
				{
					count := len(msg.PolicyChange.New)
					switch count {
					case 1:
						summary = "Enforcing 1 policy"
					default:
						summary = fmt.Sprintf("Enforcing %d policies", count)
					}
				}
				window.Synchronize(func() {
					// Update tool tip
					fmt.Println(summary)
					ni.SetToolTip(summary)

					// Update menu
					actions := ni.ContextMenu().Actions()
					actions.Clear()
					for i, pol := range msg.PolicyChange.New {
						action := walk.NewAction()
						desc := fmt.Sprintf("%d: %s", i, pol.Resource)
						if pol.Limit != policy.DefaultLimit {
							desc = fmt.Sprintf("%s: %d", desc, pol.Limit)
						}
						action.SetText(desc)
						actions.Add(action)
					}
				})
			}
		case <-ctx.Done():
			// Here we use the synchronize function to ensure that our call to Close
			// pushes the WM_CLOSE message onto the message queue of the correct
			// thread. If we call Close() directly it could fail silently and
			// deadlock.
			fmt.Printf("calling close on mainwindow\n")
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
