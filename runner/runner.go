package runner

import (
	"context"
	"fmt"
	"log"
	"os/exec"
	"strings"
	"sync"
	"time"

	"github.com/scjalliance/resourceful/environment"
	"github.com/scjalliance/resourceful/guardian"
	"github.com/scjalliance/resourceful/lease"
	"github.com/scjalliance/resourceful/lease/leaseui"
)

// Runner is a command runner that acquires a lease before running the command.
// It is capable of showing a queued lease dialog to the user if a lease cannot
// be acquired immediately.
type Runner struct {
	config    Config
	consumer  string
	instance  string
	env       environment.Environment
	retry     time.Duration
	icon      *leaseui.Icon
	client    *guardian.Client
	running   bool
	online    bool
	warned    bool
	current   lease.Lease
	dismissal time.Time
	ui        *leaseui.Manager
}

// New creates a new runner for the given program and arguments.
//
// If the given set of guardian server addresses is empty the servers will be
// detected via service discovery.
func New(config Config) (*Runner, error) {
	r := &Runner{
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
	r.consumer, r.instance, r.env, err = DetectEnvironment()
	if err != nil {
		return fmt.Errorf("runner: unable to detect environment: %v", err)
	}

	if len(r.config.Servers) == 0 {
		r.client, err = guardian.NewClient("resourceful")
	} else {
		r.client, err = guardian.NewClientWithServers(r.config.Servers)
	}
	if err != nil {
		return fmt.Errorf("runner: unable to create resourceful guardian client: %v", err)
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
		Consumer: r.consumer,
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
	maintainer := r.client.Maintain(ctx, r.config.Program, r.consumer, r.instance, r.env, r.retry)
	defer shutdown() // Make sure the lease maintainer is shut down if there's an error

	ch := maintainer.Listen(1)

	for {
		response, ok := <-ch
		if !ok {
			return nil
		}

		/*
			err = r.handleDowngrade()
			if err != nil {
				return
			}
		*/

		if response.Err == nil {
			r.setOnline(true)
			r.current = response.Lease
		} else {
			r.setOnline(false)
		}

		r.ui.Update(r.current, response)

		if response.Err != nil {
			err = r.handleError(ctx, response, shutdown)
		} else {
			switch response.Lease.Status {
			case lease.Active:
				err = r.handleActive(ctx, shutdown)
			case lease.Queued:
				err = r.handleQueued(ctx, shutdown)
			default:
				err = fmt.Errorf("unexpected lease status: \"%s\"", response.Lease.Status)
			}
		}

		if err != nil {
			return fmt.Errorf("runner: %v", err)
		}
	}
}

// handleError processes lease retrieval errors.
func (r *Runner) handleError(ctx context.Context, response guardian.Acquisition, shutdown context.CancelFunc) error {
	if !r.running {
		return response.Err
	}

	log.Printf("Lease renewal failed: %v", response.Err)

	now := time.Now()
	if r.current.Expired(now) {
		log.Printf("Lease has expired. Shutting down %s", r.config.Program)
		shutdown()
		return nil
	}

	expiration := r.current.ExpirationTime()
	remaining := expiration.Sub(now)

	log.Printf("Lease time remaining: %s", remaining.String())

	if shouldWarn(r.current, now, r.dismissal) {
		r.ui.Change(leaseui.Disconnected, nil)
	}

	return nil
}

/*
// handleDowngrade will return an error if a lease was downgraded unexpectedly.
func (r *Runner) handleDowngrade() error {
	if !r.running {
		return nil
	}

	if r.current.Status == lease.Active {
		return nil
	}

	// Check whether our lease got downgraded
	if r.warned {
		return fmt.Errorf("Lease was downgraded after the server came back online. Shutting down %s", r.program)
	}

	return fmt.Errorf("Lease was downgraded unexpectedly. Shutting down %s", r.program)
}
*/

// handleRestored processes restoration after a connection to the server has
// been restored.
/*
func (r *Runner) handleRestored(ctx context.Context, ch <-chan guardian.Acquisition, shutdown context.CancelFunc) error {
	if !r.running {
		return nil
	}

	// Check whether our lease got downgraded
	if r.current.Status != lease.Active {
		log.Printf("Lease was downgraded after the server came back online. Shutting down %s", r.program)
		shutdown()
		return nil
	}

	// Tell the user that the connection has been restored.
	var result leaseui.Result
	var err error
	result, r.current, err = leaseui.Connected(ctx, r.icon, r.program, r.consumer, r.current, err, ch)

	// Check whether our lease got downgraded
	if r.current.Status != lease.Active {
		log.Printf("Lease was downgraded unexpectedly. Shutting down %s", r.program)
		shutdown()
		return nil
	}

	// If the dialog returned success it means that the server went offline
	// again. If that's the case then we shoul continue with our error loop.
	// If the user closed the dialog or the system is shutting down then we
	// exit.
	if result != leaseui.Success {
		return nil
	}
}
*/

// handleActive processes active lease acquisitions.
func (r *Runner) handleActive(ctx context.Context, completion context.CancelFunc) (err error) {
	r.ui.Change(leaseui.None, nil)

	if r.running {
		if !r.online {
			log.Printf("Lease recovered")
		} else {
			log.Printf("Lease maintained")
		}
		return
	}

	log.Printf("Lease acquired")

	return r.execute(ctx, completion)
}

// handleQueued processes queued lease acquisitions.
func (r *Runner) handleQueued(ctx context.Context, shutdown context.CancelFunc) (err error) {
	if r.running {
		// TODO: When a lease is downgraded to released or queued status
		//       show the lease UI again?
		log.Printf("Lease downgraded")
		shutdown()
		return nil
	}

	log.Printf("Lease queued")

	r.ui.Change(leaseui.Queued, func(result leaseui.Result, err error) {
		switch result {
		case leaseui.UserCancelled, leaseui.UserClosed:
			log.Printf("User stopped waiting for an active lease")
			shutdown()
		}
	})

	/*
		result, response, err := leaseui.Queued(ctx, r.icon, r.program, r.consumer, response, ch)
		if err != nil || result != leaseui.Success {
			// The user intentionally stopped waiting or something went wrong
			shutdown()
			return
		}

		r.current = response.Lease

		return r.execute(ctx, shutdown)
	*/
	return nil
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

func (r *Runner) setOnline(online bool) {
	if r.online == online {
		return
	}
	if online {
		log.Printf("Connection online")
	} else {
		log.Printf("Connection offline")
	}
	r.online = online
}

// shouldWarn returns true if the runner should display a connection dialog.
func shouldWarn(ls lease.Lease, at, last time.Time) bool {
	expiration := ls.ExpirationTime()
	if expiration.Before(at) {
		return true
	}

	remaining := expiration.Sub(at)
	if remaining < time.Minute*1 {
		log.Printf("Less than 1 minute remains")
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
