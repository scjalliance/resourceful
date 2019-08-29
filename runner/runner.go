package runner

import (
	"context"
	"fmt"
	"log"
	"os/exec"
	"strings"
	"sync"
	"time"

	"github.com/scjalliance/resourceful/guardian"
	"github.com/scjalliance/resourceful/lease"
	"github.com/scjalliance/resourceful/lease/leaseui"
)

// Runner is a command runner that acquires a lease before running the command.
// It is capable of showing a queued lease dialog to the user if a lease cannot
// be acquired immediately.
type Runner struct {
	config     Config
	instance   lease.Instance
	properties lease.Properties
	retry      time.Duration
	icon       *leaseui.Icon
	client     *guardian.Client
	running    bool
	failed     bool // Have we lost connection?
	warned     bool // Has the user been warned about a connection loss?
	restored   bool // Are we waiting for the user to acknowledge that the connection was restored?
	ui         *leaseui.Manager
	dismissal  time.Time

	mutex sync.Mutex // Held during processing
}

// New creates a new runner for the given program and arguments.
//
// If the given set of guardian server addresses is empty the servers will be
// detected via service discovery.
func New(client *guardian.Client, config Config) (*Runner, error) {
	r := &Runner{
		client: client,
		config: config,
		retry:  time.Second * 5,
		icon:   leaseui.DefaultIcon(),
	}
	if err := r.init(); err != nil {
		return nil, err
	}
	return r, nil
}

func (r *Runner) init() (err error) {
	r.instance, r.properties, err = DetectEnvironment(r.config)
	if err != nil {
		return fmt.Errorf("runner: unable to detect environment: %v", err)
	}
	return
}

// Run will attempt to acquire an active lease and run the command.
func (r *Runner) Run(ctx context.Context) (err error) {
	ctx, shutdown := context.WithCancel(ctx)
	var wg sync.WaitGroup
	wg.Add(2)
	var done = func() {
		wg.Done()
		shutdown()
	}

	r.ui = leaseui.New(leaseui.Config{
		Icon:     r.config.Icon,
		Program:  r.config.Program,
		Instance: r.instance, // Only used to identify our own lease in a list
	})

	var runErr error
	go func() {
		runErr = r.run(ctx)
		done()
	}()

	var uiErr error
	uiErr = r.ui.Run(ctx, shutdown) // Must run on the main thread
	done()

	wg.Wait()

	if runErr != nil {
		return runErr
	}
	return uiErr
}

func (r *Runner) run(ctx context.Context) (err error) {
	ctx, shutdown := context.WithCancel(ctx)
	maintainer := guardian.NewLeaseMaintainer(r.client, r.instance, r.properties, r.retry)
	defer shutdown() // Make sure the lease maintainer is shut down if there's an error

	go func() {
		<-ctx.Done()
		maintainer.Close()
	}()

	if err := maintainer.Start(); err != nil {
		return err
	}

	ch := maintainer.Listen(1)

	var last lease.State
	for {
		state, ok := <-ch
		if !ok {
			return nil
		}

		if last.Online != state.Online {
			if state.Online {
				log.Printf("Connection online")
			} else {
				log.Printf("Connection offline")
			}
		}
		last = state

		r.mutex.Lock()
		r.ui.Update(state)

		if state.Err != nil {
			err = r.handleError(ctx, state, shutdown)
		} else if state.LeaseNotRequired {
			err = r.handleLeaseNotRequired(ctx, shutdown)
		} else {
			switch state.Lease.Status {
			case lease.Queued:
				err = r.handleQueued(ctx, shutdown)
			case lease.Active:
				err = r.handleActive(ctx, shutdown)
			default:
				err = fmt.Errorf("unexpected lease status: \"%s\"", state.Lease.Status)
			}
		}

		r.mutex.Unlock()

		if err != nil {
			return fmt.Errorf("runner: %v", err)
		}
	}
}

// handleError processes lease retrieval errors.
func (r *Runner) handleError(ctx context.Context, state lease.State, shutdown context.CancelFunc) error {
	r.failed = true

	if !r.running {
		log.Printf("Lease acquisition failed: %v", state.Err)
		r.ui.Change(leaseui.Startup, r.startupCallback(shutdown))
		return nil
	}

	log.Printf("Lease renewal failed: %v", state.Err)

	now := time.Now()
	if state.Lease.Expired(now) {
		log.Printf("Lease has expired. Shutting down %s", r.config.Program)
		shutdown()
		return nil
	}

	expiration := state.Lease.ExpirationTime()
	remaining := expiration.Sub(now)

	log.Printf("Lease time remaining: %s", remaining.String())

	if shouldWarn(state.Lease, now, r.dismissal) {
		log.Printf("Warning the user")
		r.warned = true
		r.ui.Change(leaseui.Disconnected, r.disconnectedCallback())
	}

	return nil
}

// handleQueued processes queued lease acquisitions.
func (r *Runner) handleQueued(ctx context.Context, shutdown context.CancelFunc) (err error) {
	if r.running {
		// TODO: When a lease is downgraded to released or queued status
		//       show the lease UI again?
		if r.failed {
			r.failed = false
			log.Printf("Lease was downgraded after the server came back online. Shutting down %s", r.config.Program)
		} else {
			log.Printf("Lease was downgraded unexpectedly. Shutting down %s", r.config.Program)
		}
		shutdown()
		return nil
	}

	log.Printf("Lease queued")

	r.ui.Change(leaseui.Queued, r.queuedCallback(shutdown))

	return nil
}

// handleActive processes active lease acquisitions.
func (r *Runner) handleActive(ctx context.Context, completion context.CancelFunc) (err error) {
	switch {
	case r.warned:
		r.restored = true
		r.ui.Change(leaseui.Connected, r.connectedCallback())
	case r.restored:
		// Waiting for the user to acknowlege that the server has been restored
	default:
		r.ui.Change(leaseui.None, nil)
	}

	if r.running {
		if r.failed {
			log.Printf("Lease recovered")
		} else {
			log.Printf("Lease maintained")
		}
	} else {
		log.Printf("Lease acquired")
	}

	r.warned = false
	r.failed = false

	if !r.running {
		return r.execute(ctx, completion)
	}

	return
}

// handleLeaseNotRequired processes lease-not-required acquisitions.
func (r *Runner) handleLeaseNotRequired(ctx context.Context, completion context.CancelFunc) (err error) {
	switch {
	case r.warned:
		r.restored = true
		r.ui.Change(leaseui.Connected, r.connectedCallback())
	case r.restored:
		// Waiting for the user to acknowlege that the server has been restored
	default:
		r.ui.Change(leaseui.None, nil)
	}

	log.Printf("Lease not required")

	r.warned = false
	r.failed = false

	if !r.running {
		return r.execute(ctx, completion)
	}

	return
}

func (r *Runner) startupCallback(shutdown context.CancelFunc) leaseui.Callback {
	return func(t leaseui.Type, result leaseui.Result, err error) {
		r.mutex.Lock()
		defer r.mutex.Unlock()

		switch result {
		case leaseui.UserCancelled, leaseui.UserClosed:
			log.Printf("User stopped waiting for an active connection")
			shutdown()
		}
	}
}

func (r *Runner) queuedCallback(shutdown context.CancelFunc) leaseui.Callback {
	return func(t leaseui.Type, result leaseui.Result, err error) {
		r.mutex.Lock()
		defer r.mutex.Unlock()

		switch result {
		case leaseui.UserCancelled, leaseui.UserClosed:
			log.Printf("User stopped waiting for an active lease")
			shutdown()
		}
	}
}

func (r *Runner) disconnectedCallback() leaseui.Callback {
	return func(t leaseui.Type, result leaseui.Result, err error) {
		r.mutex.Lock()
		defer r.mutex.Unlock()

		switch result {
		case leaseui.UserCancelled, leaseui.UserClosed:
			log.Printf("User dismissed the warning")
			r.dismissal = time.Now()
		}

		r.ui.CompareAndChange(leaseui.Disconnected, leaseui.None, nil)
	}
}

func (r *Runner) connectedCallback() leaseui.Callback {
	return func(t leaseui.Type, result leaseui.Result, err error) {
		r.mutex.Lock()
		defer r.mutex.Unlock()

		r.restored = false
		r.ui.CompareAndChange(leaseui.Connected, leaseui.None, nil)
	}
}

// execute will start the command in its own child process. It ensures that
// the completion function will be called when the command finishes.
//
// The execute function itself does not block.
//
// If execute cannot start the command, err will be non-nil and the completion
// function will not be called.
func (r *Runner) execute(ctx context.Context, completion context.CancelFunc) (err error) {
	if r.running {
		panic("Command is already running")
	}

	log.Printf("Executing %s %s", r.config.Program, strings.Join(r.config.Args, " "))

	cmd := exec.CommandContext(ctx, r.config.Program, r.config.Args...)
	err = cmd.Start()
	if err != nil {
		return
	}

	r.running = true

	go func() {
		defer completion()
		cmd.Wait()
	}()

	return nil
}

// shouldWarn returns true if the runner should display a connection dialog.
func shouldWarn(ls lease.Lease, at, last time.Time) bool {
	expiration := ls.ExpirationTime()
	if expiration.Before(at) {
		return true
	}

	remaining := expiration.Sub(at)
	if remaining < time.Minute*1 {
		// Always nag the user if doom is approaching
		return true
	}

	var (
		oneThird   = ls.Duration / 3
		oneQuarter = ls.Duration / 4
	)

	if !last.IsZero() {
		if at.Before(last) {
			// Something weird happened, like a wall clock shift
			return true
		}
		sinceLast := at.Sub(last)
		if sinceLast < oneQuarter {
			// Warn once every quarter duration
			return false
		}
	}

	if remaining < oneThird {
		// Warn when we're two thirds through the current lease
		return true
	}

	refresh := ls.EffectiveRefresh()
	missed := ls.Renewed.Add(refresh)

	if at.After(missed.Add(oneQuarter)) {
		// Warn when a quarter duration has passed since the last renewal
		return true
	}

	return false
}
