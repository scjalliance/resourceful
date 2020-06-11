// +build windows

package main

import (
	"fmt"
	"os"
	"time"

	"github.com/scjalliance/resourceful/enforcer"
	"golang.org/x/sys/windows/svc"
)

// Commands that we accept.
const (
	cmdsAccepted = svc.AcceptStop | svc.AcceptShutdown | svc.AcceptPauseAndContinue
)

// Handler is a windows service handler.
type Handler struct {
	Name    string
	Conf    EnforceConfig
	ConfErr error
	Logger  enforcer.Logger
}

// Run causes the service to run under the given name until it is instructed
// to stop by the operating system.
func (h Handler) Run() error {
	return svc.Run(h.Name, h)
}

// Execute performs service request processing for windows.
func (h Handler) Execute(args []string, requests <-chan svc.ChangeRequest, changes chan<- svc.Status) (ssec bool, errno uint32) {
	// In all circumstances, indicate that we're stopping when we exit
	defer func() {
		changes <- svc.Status{State: svc.StopPending}
	}()

	// Track progress
	var checkpoint uint32

	// Indicate to the system that we're starting up
	checkpoint = sendProgress(changes, checkpoint)

	h.log("OS Arguments: %#v", os.Args)

	// Check for errors in the arguments provided by the OS
	if h.ConfErr != nil {
		h.log("Invalid OS Arguments: %v", h.ConfErr)
		return false, 1
	}

	// Parse arguments provided by the service manager
	if len(args) > 0 {
		h.Name = args[0]
	}
	if len(args) > 1 {
		args = args[1:]
		h.log("Service Arguments: %#v", args)
		app := App()
		enforceCmd, enforceConf := EnforceCommand(app)
		command, err := app.Parse(args)
		if err != nil {
			h.log("Invalid Service Arguments: %v", err)
			return false, 1
		} else if command != enforceCmd.FullCommand() {
			h.log("Invalid Service Command: %s", command)
			return false, 1
		}
		if enforceConf.Server != "" {
			h.Conf.Server = enforceConf.Server
		}
		if enforceConf.Passive {
			h.Conf.Passive = true
		}
	}
	checkpoint = sendProgress(changes, checkpoint)

	// Determine how to invoke a UI process within a session
	executable, err := os.Executable()
	if err != nil {
		h.log("Failed to query executable: %v", err)
		return false, 1
	}
	uiCommand := enforcer.Command{
		Path: executable,
		Args: []string{"ui"},
	}
	checkpoint = sendProgress(changes, checkpoint)

	// Examine the environment
	environment, err := buildEnvironment()
	if err != nil {
		h.log("Failed to collect environment: %v\n", err)
		return false, 1
	}
	checkpoint = sendProgress(changes, checkpoint)

	// Prepare a guardian client
	client := newClient(h.Conf.Server)
	checkpoint = sendProgress(changes, checkpoint)

	// Prepare the service
	service := enforcer.New(client, time.Second, time.Minute, uiCommand, environment, nil, h.Conf.Passive, h.Logger)
	checkpoint = sendProgress(changes, checkpoint)

	// Start the service
	if err := service.Start(); err != nil {
		h.log("Failed to start service: %v.", err)
		return false, 1
	}

	// Stop the service when we exit
	defer service.Stop()

	// Switch to the running state
	changes <- svc.Status{State: svc.Running, Accepts: cmdsAccepted}

	// Main service loop
	for {
		req := <-requests
		switch req.Cmd {
		case svc.Interrogate:
			changes <- req.CurrentStatus
			// Testing deadlock from https://code.google.com/p/winsvc/issues/detail?id=4
			time.Sleep(100 * time.Millisecond)
			changes <- req.CurrentStatus
		case svc.Stop, svc.Shutdown:
			changes <- svc.Status{State: svc.StopPending}
			h.log("Service shutting down.")
			return false, 0
		case svc.Pause:
			changes <- svc.Status{State: svc.Paused, Accepts: cmdsAccepted}
			h.log("Service paused.")
			service.Stop()
		case svc.Continue:
			changes <- svc.Status{State: svc.Running, Accepts: cmdsAccepted}
			h.log("Service unpaused.")
			service.Start()
		}
	}
}

func (h *Handler) log(format string, v ...interface{}) {
	if h.Logger == nil {
		return
	}
	h.Logger.Log(enforcer.ServiceEvent{
		Msg: fmt.Sprintf(format, v...),
	})
}

func (h *Handler) debug(format string, v ...interface{}) {
	if h.Logger == nil {
		return
	}
	h.Logger.Log(enforcer.ServiceEvent{
		Msg:   fmt.Sprintf(format, v...),
		Debug: true,
	})
}

func sendProgress(changes chan<- svc.Status, checkpoint uint32) uint32 {
	changes <- svc.Status{State: svc.StartPending, CheckPoint: checkpoint}
	return checkpoint + 1
}
