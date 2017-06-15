package runner

import (
	"context"
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
	program  string
	args     []string
	consumer string
	instance string
	env      environment.Environment
	retry    time.Duration
	icon     *leaseui.Icon
	client   *guardian.Client
	running  bool
	current  lease.Lease
}

// New creates a new runner for the given program and arguments.
func New(program string, args []string) (*Runner, error) {
	r := &Runner{
		program: program,
		args:    args,
		retry:   time.Second * 5,
		icon:    leaseui.DefaultIcon(),
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

	r.client, err = guardian.NewClient("resourceful")
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
func (r *Runner) Run() (err error) {
	ctx, shutdown := context.WithCancel(context.Background())
	maintainer := r.client.Maintain(ctx, r.program, r.consumer, r.instance, r.env, r.retry)
	defer shutdown() // Make sure the lease maintainer is shut down if there's an error

	ch := maintainer.Listen(1)

	for {
		response, ok := <-ch
		if !ok {
			return nil
		}

		if response.Err != nil {
			err = r.handleError(response.Err, shutdown)
		} else {
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
func (r *Runner) handleError(err error, shutdown context.CancelFunc) error {
	if !r.running {
		return err
	}

	if r.current.Status != lease.Active {
		panic("unexpected lease state when program is running")
	}

	log.Printf("Unable to renew active lease due to error: %v", err)

	now := time.Now()
	if r.current.Expired(now) {
		log.Printf("Lease has expired. Shutting down %s", r.program)
		shutdown()
	} else {
		remaining := r.current.ExpirationTime().Sub(now)
		log.Printf("Lease time remaining: %s", remaining.String())
	}

	return nil
}

// handleActive processes active lease acquisitions.
func (r *Runner) handleActive(ctx context.Context, completion context.CancelFunc) (err error) {
	if r.running {
		log.Printf("Lease maintained")
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

	acquired, err := leaseui.WaitForActive(ctx, r.icon, r.program, r.consumer, response, ch)
	if err != nil || !acquired {
		// The user intentionally stopped waiting or something went wrong
		shutdown()
		return
	}

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
