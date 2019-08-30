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
	instance   lease.Instance
	maintainer *guardian.LeaseMaintainer
	logger     Logger
	stop       context.CancelFunc

	mutex   sync.Mutex
	stopped <-chan struct{}
	pols    policy.Set
}

// Manage performs lease management for the given process.
func Manage(client *guardian.Client, proc Process, instance lease.Instance, passive bool, logger Logger) (*ManagedProcess, error) {
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

	maintainer := guardian.NewLeaseMaintainer(client, instance, Properties(proc, instance.Host), retry)
	ctx, cancel := context.WithCancel(context.Background())
	stopped := make(chan struct{})

	mp := &ManagedProcess{
		proc:       proc,
		instance:   instance,
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

	prefix := fmt.Sprintf("PROCESS: %s (%s)", mp.proc.Name, mp.instance.ID)

	var termPending bool

	mp.maintainer.Start()
	ch := mp.maintainer.Listen(1)
	for {
		select {
		case err := <-exited:
			switch err {
			case nil:
				mp.log("%s: Exited", prefix)
			case context.Canceled, context.DeadlineExceeded:
				mp.log("%s: Ceasing management", prefix)
			default:
				mp.log("%s: Observation failed: %v", prefix, err)
				// FIXME: Continue holding a lease but release the reference?
			}
			go mp.maintainer.Close()
			for range ch {
				// Drain the states
			}
			return
		case state, ok := <-ch:
			if !ok {
				mp.log("%s: Lease maintainer closed", prefix)
				return
			}
			switch {
			case state.Acquired:
				switch state.Lease.Status {
				case lease.Active:
					termPending = false
					mp.log("%s: Leased (%s, %s)", prefix, state.Lease.Resource, state.Lease.Duration)
				case lease.Queued:
					mp.log("%s: Queued (%s)", prefix, state.Lease.Resource)
				}
			case state.LeaseNotRequired:
				mp.log("%s: Lease Not Required", prefix)
			}
			if (!state.Acquired && !state.LeaseNotRequired) || state.Lease.Status != lease.Active {
				uptime := time.Now().Sub(mp.proc.Times.Creation)
				if uptime < time.Second*5 {
					// Insta-kill
				}
				if !termPending {
					mp.log("%s: Terminating", prefix)
				}
				if err := ref.Terminate(5877); err != nil {
					if !termPending {
						mp.log("%s: Termination failed: %v", prefix, err)
						termPending = true
					}
				} else {
					mp.log("%s: Terminated", prefix)
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
