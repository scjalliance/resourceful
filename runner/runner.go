package runner

import (
	"context"
	"errors"
	"fmt"
	"log"
	"os/exec"
	"strings"
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
	program   string
	args      []string
	consumer  string
	instance  string
	env       environment.Environment
	retry     time.Duration
	icon      *leaseui.Icon
	client    *guardian.Client
	running   bool
	online    bool
	current   lease.Lease
	dismissal time.Time
}

// New creates a new runner for the given program and arguments.
//
// If the given set of guardian server addresses is empty the servers will be
// detected via service discovery.
func New(program string, args []string, servers []string) (*Runner, error) {
	r := &Runner{
		program: program,
		args:    args,
		retry:   time.Second * 5,
		icon:    leaseui.DefaultIcon(),
	}
	if err := r.init(servers); err != nil {
		return nil, err
	}
	return r, nil
}

func (r *Runner) init(servers []string) (err error) {
	r.consumer, r.instance, r.env, err = DetectEnvironment()
	if err != nil {
		return fmt.Errorf("runner: unable to detect environment: %v", err)
	}

	if len(servers) == 0 {
		r.client, err = guardian.NewClient("resourceful")
	} else {
		r.client, err = guardian.NewClientWithServers(servers)
	}
	if err != nil {
		return fmt.Errorf("runner: unable to create resourceful guardian client: %v", err)
	}

	return
}

// SetIcon will change the icon used by the queued lease dialog.
func (r *Runner) SetIcon(icon *leaseui.Icon) {
	r.icon = icon
}

// Run will attempt to acquire an active lease and run the command.
func (r *Runner) Run(ctx context.Context) (err error) {
	ctx, shutdown := context.WithCancel(ctx)
	maintainer := r.client.Maintain(ctx, r.program, r.consumer, r.instance, r.env, r.retry)
	defer shutdown() // Make sure the lease maintainer is shut down if there's an error

	ch := maintainer.Listen(1)

	for {
		response, ok := <-ch
		if !ok {
			return nil
		}

		if response.Err != nil {
			r.setOnline(false)
			err = r.handleError(ctx, ch, response.Err, shutdown)
		} else {
			r.setOnline(true)
			r.current = response.Lease

			switch response.Lease.Status {
			case lease.Active:
				err = r.handleActive(ctx, shutdown)
			case lease.Queued:
				err = r.handleQueued(ctx, response, ch, shutdown)
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
func (r *Runner) handleError(ctx context.Context, ch <-chan guardian.Acquisition, err error, shutdown context.CancelFunc) error {
	if !r.running {
		return err
	}

	if r.current.Status != lease.Active {
		return errors.New("unexpected lease state when program is running")
	}

	log.Printf("Unable to renew active lease due to error: %v", err)

	now := time.Now()
	if r.current.Expired(now) {
		log.Printf("Lease has expired. Shutting down %s", r.program)
		shutdown()
		return nil
	}

	expiration := r.current.ExpirationTime()
	remaining := expiration.Sub(now)

	log.Printf("Lease time remaining: %s", remaining.String())

	if !shouldWarn(r.current, now, r.dismissal) {
		return nil
	}

	monCtx, monCancel := context.WithTimeout(ctx, remaining)
	defer monCancel()

	result, current, err := leaseui.Disconnected(monCtx, r.icon, r.program, r.consumer, r.current, err, ch)
	if err != nil {
		return err
	}

	r.current = current

	switch result {
	case leaseui.Success:
		// The server came back online
		r.setOnline(true)
	case leaseui.UserCancelled, leaseui.UserClosed:
		// The user intentionally stopped waiting
		r.dismissal = time.Now()
	case leaseui.ChannelClosed:
		// The lease maintainer is shutting down
	case leaseui.ContextCancelled:
		// Either the system is shutting down or the lease expired
		if r.current.Expired(now) {
			log.Printf("Lease has expired. Shutting down %s", r.program)
			shutdown()
		}
	}

	return nil
}

// handleActive processes active lease acquisitions.
func (r *Runner) handleActive(ctx context.Context, completion context.CancelFunc) (err error) {
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
func (r *Runner) handleQueued(ctx context.Context, response guardian.Acquisition, ch <-chan guardian.Acquisition, shutdown context.CancelFunc) (err error) {
	if r.running {
		// TODO: When a lease is downgraded to released or queued status
		//       show the lease UI again.
		log.Printf("Lease lost")
		return nil
	}

	log.Printf("Lease queued")

	result, response, err := leaseui.Queued(ctx, r.icon, r.program, r.consumer, response, ch)
	if err != nil || result != leaseui.Success {
		// The user intentionally stopped waiting or something went wrong
		shutdown()
		return
	}

	r.current = response.Lease

	return r.execute(ctx, shutdown)
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

	log.Printf("Executing %s %s", r.program, strings.Join(r.args, " "))

	cmd := exec.CommandContext(ctx, r.program, r.args...)
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
