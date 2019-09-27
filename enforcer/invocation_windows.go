// +build windows

package enforcer

import (
	"context"
	"fmt"
	"os/exec"
	"strings"
	"sync"
	"syscall"
	"time"

	"github.com/gentlemanautomaton/winproc"
	"github.com/gentlemanautomaton/winsession/wtsapi"
	"github.com/scjalliance/resourceful/guardian"
	"github.com/scjalliance/resourceful/lease"
	"github.com/scjalliance/resourceful/policy"
)

type absorptionRequest struct {
	process *Process
	result  chan bool
}

// Invocation respresents the intent of a user to launch and operate a program
// that requires a lease according to the current policy set.
type Invocation struct {
	instance    lease.Instance
	name        string
	path        string
	commandLine string
	user        winproc.User
	sessionID   uint32
	session     *Session
	logger      Logger

	mutex      sync.Mutex
	stop       context.CancelFunc
	stopped    <-chan struct{}
	absorption chan absorptionRequest
	pols       policy.Set

	stateMutex sync.Mutex
	state      lease.State
}

// NewInvocation returns a new invocation for the given process data.
func NewInvocation(client *guardian.Client, instance lease.Instance, process *Process, session *Session, logger Logger) *Invocation {
	data := process.Data()
	ctx, cancel := context.WithCancel(context.Background())
	absorption := make(chan absorptionRequest, 1)
	stopped := make(chan struct{})

	inv := &Invocation{
		instance:    instance,
		name:        data.Name,
		path:        data.Path,
		commandLine: data.CommandLine,
		user:        data.User,
		sessionID:   data.SessionID,
		session:     session,
		logger:      logger,
		stop:        cancel,
		stopped:     stopped,
		absorption:  absorption,
	}

	go inv.manage(ctx, client, process, absorption, stopped)

	return inv
}

// Absorb asks the invocation to absorb the process. It returns true if it succeeds.
func (inv *Invocation) Absorb(process *Process) bool {
	inv.mutex.Lock()
	defer inv.mutex.Unlock()

	// Don't absorb processes if we've been closed
	if inv.absorption == nil {
		return false
	}

	// Send the absorption request
	req := absorptionRequest{
		process: process,
		result:  make(chan bool, 1),
	}
	inv.absorption <- req

	// Wait for a response
	return <-req.result
}

// Stop causes mp to stop managing its process without killing it.
func (inv *Invocation) Stop() {
	inv.mutex.Lock()
	defer inv.mutex.Unlock()

	if inv.stop != nil {
		// Tell the management goroutine to stop and then wait for it
		inv.stop()
		<-inv.stopped
		inv.stop = nil
		inv.stopped = nil
	}

	if inv.absorption != nil {
		// Drain and close the absorption channel
		select {
		case <-inv.absorption:
		default:
		}
		close(inv.absorption)
		inv.absorption = nil
	}
}

// Done returns true if the invocation has ceased management.
func (inv *Invocation) Done() bool {
	inv.mutex.Lock()
	defer inv.mutex.Unlock()

	return inv.done()
}

func (inv *Invocation) done() bool {
	if inv.stopped == nil {
		return true
	}

	select {
	case <-inv.stopped:
		return true
	default:
		return false
	}
}

// UpdatePolicies updates the set of policies used by the invocation.
func (inv *Invocation) UpdatePolicies(pols policy.Set) {
	inv.mutex.Lock()
	defer inv.mutex.Unlock()
	inv.pols = pols
}

func (inv *Invocation) manage(ctx context.Context, client *guardian.Client, process *Process, absorption <-chan absorptionRequest, stopped chan<- struct{}) {
	// Things we need to handle here:
	// * We're instructed by the service to stop managing the process
	// * The process exits of its own volition (this must be distinguished from the manager killing it)
	// * The process needs to be killed due to a lease violation
	// * The process needs to be respawned due to a lease acquisition
	// * There are policy changes relating to this process (this might be handled by the service instead, which can stop this manager and start a new one)
	// * The wait call for the process fails
	// * The terminate call for the process fails

	defer close(stopped)

	retry := time.Second * 5
	maintainer := guardian.NewLeaseMaintainer(client, inv.instance, Properties(process.Data(), inv.instance.Host), retry)
	maintainer.Start()
	states := maintainer.Listen(1)

	defer func() {
		go maintainer.Close()
		for range states {
			// Drain the states
		}
	}()

	inv.log("Managing process %d", process.Data().ID)

	for {
		// Keep the program running as long as we hold a lease
		ok := inv.maintain(ctx, absorption, states, process)
		if !ok {
			// The process exited on its own or ctx was cancelled
			return
		}

		// Wait for lease acquisition
		pid, ok := inv.waitForAquisition(ctx, absorption, states)
		if !ok {
			// The lease maintainer closed, ctx was cancelled or the user
			// cancelled the invocation via the UI
			return
		}

		// Wait for process absorption
		process, ok = inv.waitForAbsorption(ctx, absorption, pid)
		if !ok {
			return
		}
	}
}

func (inv *Invocation) maintain(ctx context.Context, absorption <-chan absorptionRequest, states <-chan lease.State, process *Process) (ok bool) {
	defer process.Close()

	exited := make(chan error)
	go func(exited chan error) {
		defer close(exited)
		exited <- process.Wait(ctx)
	}(exited)

	var termPending bool
	for {
		select {
		case req := <-absorption:
			req.result <- false
		case err := <-exited:
			switch err {
			case nil:
				if termPending {
					// Indicate to the main loop that we want to wait for a lease
					return true
				}
				inv.log("Exited")
			case context.Canceled, context.DeadlineExceeded:
				inv.log("Ceasing management")
			default:
				inv.log("Observation failed: %v", err)
				// FIXME: Continue holding a lease but release the reference?
			}
			return false
		case state, ok := <-states:
			if !ok {
				inv.log("Lease maintainer closed")
				return false
			}

			now := time.Now()
			inv.recordState(state, now)

			var terminate bool

			switch {
			case state.Acquired:
				if state.Lease.Status != lease.Active || state.Lease.Expired(now) {
					terminate = true
				}

			case state.LeaseNotRequired:
				return false
			case !state.Online:
				terminate = true
			}

			if terminate {
				if termPending {
					break // Already sent a termination command
				}
				uptime := time.Now().Sub(process.Data().Times.Creation)
				if uptime < time.Second*5 {
					// Insta-kill
				} else {
					// Warn and then kill
				}
				id := process.Data().ID
				inv.log("Terminating process %d", id)
				if err := process.Terminate(); err != nil {
					inv.log("Termination of process %d failed: %v", id, err)
				} else {
					termPending = true
					inv.log("Terminated process %d", id)
				}
			}
		}
	}
}

func (inv *Invocation) waitForAquisition(ctx context.Context, absorption <-chan absorptionRequest, states <-chan lease.State) (pid PID, ok bool) {
	for {
		select {
		case <-ctx.Done():
			inv.log("Ceasing management")
			return 0, false
		case req := <-absorption:
			req.result <- false
		case state, ok := <-states:
			if !ok {
				inv.log("Lease maintainer closed")
				return 0, false
			}
			now := time.Now()
			inv.recordState(state, now)
			var start bool
			switch {
			case state.Acquired:
				if !state.Lease.Expired(now) && state.Lease.Status == lease.Active {
					start = true
				}
			case state.LeaseNotRequired:
				start = true
			}
			if start {
				return inv.spawn()
			}
		}
	}
}

func (inv *Invocation) recordState(state lease.State, now time.Time) {
	inv.stateMutex.Lock()
	old := inv.state
	inv.state = state
	inv.stateMutex.Unlock()

	if state.Online != old.Online {
		if state.Online {
			inv.log("Online")
		} else {
			inv.log("Offline")
		}
	}

	switch {
	case state.Acquired:
		if state.Lease.Expired(now) {
			inv.log("Lease Expired")
			return
		}
	case state.LeaseNotRequired:
		inv.log("Lease Not Required")
		return
	case !state.Online:
		inv.log("Lease Acquisition Failed")
		return
	}

	remaining := state.Lease.ExpirationTime().Sub(now)
	diff := state.Lease.Duration - remaining
	if diff < time.Second {
		inv.log("%s (%s, %s)", strings.Title(string(state.Lease.Status)), state.Lease.Resource, state.Lease.Duration)
	} else {
		inv.log("%s (%s, %s / %s)", strings.Title(string(state.Lease.Status)), state.Lease.Resource, remaining.Round(time.Second), state.Lease.Duration)
	}

	/*
		if !state.Online {
			const warningInterval = 30 * time.Second
			if !warned || now.Sub(warnedTime) > warningInterval {
				// TODO: Send the user a warning that the connection to
				// the server has been lost.
				warned = true
				warnedTime = now
				inv.log("Warning: Lease Time Remaining: %s", remaining)
			}
		}
	*/
}

func (inv *Invocation) spawn() (pid PID, ok bool) {
	inv.log("Starting new process")

	// Acquire a user token for the session
	token, err := wtsapi.QueryUserToken(inv.sessionID)
	if err != nil {
		inv.log("Failed to acquire token: %v", err)
		return 0, false
	}
	defer token.Close()

	// Make sure the token is valid for the user we expect
	userName := inv.user.Account
	userDomain := inv.user.Domain
	if err := validateTokenForUser(token, userName, userDomain); err != nil {
		inv.log("Failed to validate token: %v", err)
		return 0, false
	}

	inv.debug("Aquired Token: %s\\%s", userDomain, userName)

	// TODO: Apply security attributes from the original process

	cmd := exec.Command(inv.path)
	cmd.SysProcAttr = &syscall.SysProcAttr{
		CmdLine: inv.commandLine,
		Token:   token,
	}
	if err := cmd.Start(); err != nil {
		inv.log("Failed to start: %v", err)
		return 0, false
	}
	inv.log("Started process %d", cmd.Process.Pid)
	return PID(cmd.Process.Pid), true
}

func (inv *Invocation) waitForAbsorption(ctx context.Context, absorption <-chan absorptionRequest, pid PID) (process *Process, ok bool) {
	// Wait up to 5 seconds for the process manager to notice the process
	// and hand it to us.
	//
	// We intentionally do not process lease state changes during this time.
	t := time.NewTimer(5 * time.Second)
	defer func() {
		if !t.Stop() {
			<-t.C
		}
	}()
	for {
		select {
		case <-ctx.Done():
			inv.log("Ceasing management")
			return nil, false
		case req := <-absorption:
			if req.process.Data().ID != pid {
				req.result <- false
				break
			}
			inv.log("Absorbing process %d", pid)
			req.result <- true
			return req.process, true
		case <-t.C:
			inv.log("Failed to absorb process %d", pid)
			return nil, false
		}
	}
}

func (inv *Invocation) log(format string, v ...interface{}) {
	if inv.logger == nil {
		return
	}
	inv.logger.Log(InvocationEvent{
		Instance:    inv.instance,
		ProcessName: inv.name,
		Msg:         fmt.Sprintf(format, v...),
	})
}

func (inv *Invocation) debug(format string, v ...interface{}) {
	if inv.logger == nil {
		return
	}
	inv.logger.Log(InvocationEvent{
		Instance:    inv.instance,
		ProcessName: inv.name,
		Msg:         fmt.Sprintf(format, v...),
		Debug:       true,
	})
}
