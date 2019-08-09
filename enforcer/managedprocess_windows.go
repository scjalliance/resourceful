// +build windows

package enforcer

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/gentlemanautomaton/winproc"

	"github.com/scjalliance/resourceful/guardian"
	"github.com/scjalliance/resourceful/lease"
	"github.com/scjalliance/resourceful/policy"
)

// ManagedProcess represents a process for which policies are being enforced.
type ManagedProcess struct {
	proc       Process
	maintainer *guardian.LeaseMaintainer
	logger     Logger
	stop       context.CancelFunc

	mutex   sync.Mutex
	stopped <-chan struct{}
	pols    policy.Set
}

// Manage performs lease management for the given process.
func Manage(client *guardian.Client, hostname string, proc Process, passive bool, logger Logger) (*ManagedProcess, error) {
	// Open a reference to the process with the highest level of privilege
	// that we can get
	ref, err := openProcess(proc.ID, passive)
	if err != nil {
		return nil, err
	}

	// Obtain a unique ID for the process.
	uid, err := ref.UniqueID()
	if err != nil {
		ref.Close()
		return nil, fmt.Errorf("unable to retrieve unique ID for process: %v", err)
	}

	// Verify that the unique ID matches our expectation
	if proc.UniqueID() != uid {
		// The process ID was recycled into a new process. Abort.
		ref.Close()
		return nil, fmt.Errorf("the process to be managed has terminated")
	}

	retry := time.Second * 5
	subject := Subject(hostname, proc)

	maintainer := guardian.NewLeaseMaintainer(client, subject.Resource, subject.Consumer, subject.Instance, Env(hostname, proc), retry)
	ctx, cancel := context.WithCancel(context.Background())
	stopped := make(chan struct{})

	mp := &ManagedProcess{
		proc:       proc,
		maintainer: maintainer,
		logger:     logger,
		stop:       cancel,
		stopped:    stopped,
	}

	go mp.manage(ctx, ref, stopped)

	return mp, nil
}

// Stop causes mp to stop managing its process without killing it.
func (mp *ManagedProcess) Stop() {
	mp.mutex.Lock()
	defer mp.mutex.Unlock()

	if mp.stop != nil {
		mp.stop()
		<-mp.stopped
	}

	mp.stop = nil
	mp.stopped = nil
}

// UpdatePolicies updates the set of policies used by the process manager.
func (mp *ManagedProcess) UpdatePolicies(pols policy.Set) {
	mp.mutex.Lock()
	defer mp.mutex.Unlock()
	mp.pols = pols
}

func (mp *ManagedProcess) manage(ctx context.Context, ref *winproc.Ref, stopped chan<- struct{}) {
	// Things we need to handle here:
	// * We're instructed by the service to stop managing the process
	// * The process exits of its own volition (this must be distinguished from the manager killing it)
	// * The process needs to be killed due to a lease violation
	// * The process needs to be respawned due to a lease acquisition
	// * There are policy changes relating to this process (this might be handled by the service instead, which can stop this manager and start a new one)
	// * The wait call for the process fails
	// * The terminate call for the process fails

	defer close(stopped)
	defer ref.Close()

	exited := make(chan error)
	go func() {
		defer close(exited)
		exited <- ref.Wait(ctx)
	}()

	var termPending bool

	mp.maintainer.Start()
	ch := mp.maintainer.Listen(1)
	for {
		select {
		case err := <-exited:
			switch err {
			case nil:
				mp.log("Process exited: %s", mp.proc.Name)
			case context.Canceled, context.DeadlineExceeded:
				mp.log("Ceasing management of process: %s", mp.proc.Name)
			default:
				mp.log("Process observation failed: %s: %v", mp.proc.Name, err)
				// FIXME: Continue holding a lease but release the reference?
			}
			go mp.maintainer.Close()
			for range ch {
				// Drain the states
			}
			return
		case state, ok := <-ch:
			if !ok {
				mp.log("Lease maintainer closed for %s", mp.proc.Name)
				return
			}
			if state.Acquired {
				switch state.Lease.Status {
				case lease.Active:
					termPending = false
					mp.log("Lease: %s", state.Lease.Subject())
				case lease.Queued:
					mp.log("Queued: %s", state.Lease.Subject())
				}
			}
			if !state.Acquired || state.Lease.Status != lease.Active {
				if !termPending {
					mp.log("Terminating %s", mp.proc.Name)
				}
				if err := ref.Terminate(5877); err != nil {
					if !termPending {
						mp.log("Failed to terminate %s: %v", mp.proc.Name, err)
						termPending = true
					}
				} else {
					mp.log("Terminated %s", mp.proc.Name)
				}
			}
		}
	}
}

// TODO: Accept an event ID or event interface?
func (mp *ManagedProcess) log(format string, v ...interface{}) {
	// TODO: Try casting s.logger to a different interface so that we can log event IDs?
	if mp.logger != nil {
		mp.logger.Printf(format, v...)
	}
}
