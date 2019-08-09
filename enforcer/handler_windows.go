// +build windows

package enforcer

import (
	"fmt"
	"time"

	"golang.org/x/sys/windows/svc"
)

// Commands that we accept.
const (
	cmdsAccepted = svc.AcceptStop | svc.AcceptShutdown | svc.AcceptPauseAndContinue
)

// Handler is a windows service handler.
type Handler struct {
	Name    string
	Service *Service
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

	// Start the service
	if err := h.Service.Start(); err != nil {
		h.Service.logError(fmt.Sprintf("Failed to start service: %v.", err))
		return
	}

	// Stop the service when we exit
	defer h.Service.Stop()

	// Switch to the running state
	changes <- svc.Status{State: svc.Running, Accepts: cmdsAccepted}

	for {
		req := <-requests
		switch req.Cmd {
		case svc.Interrogate:
			changes <- req.CurrentStatus
			// Testing deadlock from https://code.google.com/p/winsvc/issues/detail?id=4
			time.Sleep(100 * time.Millisecond)
			changes <- req.CurrentStatus
		case svc.Stop, svc.Shutdown:
			h.Service.logInfo("Service shutting down.")
			return
		case svc.Pause:
			h.Service.logInfo("Service paused.")
			changes <- svc.Status{State: svc.Paused, Accepts: cmdsAccepted}
			h.Service.Stop()
		case svc.Continue:
			h.Service.logInfo("Service unpaused.")
			changes <- svc.Status{State: svc.Running, Accepts: cmdsAccepted}
			h.Service.Start()
		}
	}
}
