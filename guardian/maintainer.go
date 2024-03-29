package guardian

import (
	"context"
	"sync"
	"time"

	"github.com/scjalliance/resourceful/guardian/transport"
	"github.com/scjalliance/resourceful/lease"
)

// Acquisition is the result of an attempted lease acquisition. It is returned
// to listeners of LeaseMaintainer.
type Acquisition struct {
	transport.AcquireResponse
	Err error
}

// LeaseMaintainer performs lease maintenance for a particular subject.
// Once started, it acquires and maintains a lease as long as it is running
// and can communicate with a guardian. When stopped it releases whatever
// lease it might hold.
type LeaseMaintainer struct {
	client   *Client
	instance lease.Instance
	retry    time.Duration // 0 == no retry
	// maxRetries?

	opMutex  sync.RWMutex
	shutdown chan bool // receives true to release before shutdown
	stopped  chan struct{}

	stateMutex sync.RWMutex
	state      lease.State
	props      lease.Properties
	listeners  []chan lease.State
}

// NewLeaseMaintainer returns a maintainer that is capable of maintaining a
// lease for the given subject. The result of each acquisition or observation
// will be returned to all listeners registered through calls to Listen().
//
// If retry is a non-zero duration the maintainer will attempt to acquire a
// lease on an interval of retry.
//
// It is the caller's responsibility to close the lease maintainer when
// finished with it.
func NewLeaseMaintainer(client *Client, instance lease.Instance, props lease.Properties, retry time.Duration) *LeaseMaintainer {
	return &LeaseMaintainer{
		client:   client,
		instance: instance,
		props:    props,
		retry:    retry,
	}
}

// Acquire causes the maintainer to acquire and maintain a lease.
func (lm *LeaseMaintainer) Acquire() error {
	lm.opMutex.Lock()
	defer lm.opMutex.Unlock()

	if lm.shutdown != nil {
		return ErrStarted
	}

	lm.shutdown = make(chan bool)
	lm.stopped = make(chan struct{})

	go lm.run(lm.shutdown, lm.stopped)

	return nil
}

// Release causes the maintainer to stop maintenance of any lease it might
// hold and to release it.
//
// Release does not close the channels of any registered listeners. To close
// all registered listeners call Close after calling Release.
func (lm *LeaseMaintainer) Release() error {
	lm.opMutex.Lock()
	defer lm.opMutex.Unlock()

	return lm.stop(true) // Release and then stop
}

// Stop causes the maintainer to stop maintenance of any lease it might hold
// without releasing it.
//
// Stop does not close the channels of any registered listeners. To close
// all registered listeners call Close after calling Stop.
func (lm *LeaseMaintainer) Stop() error {
	lm.opMutex.Lock()
	defer lm.opMutex.Unlock()

	return lm.stop(false) // Stop without releasing
}

// stop tells the current run() goroutine to stop and then waits for it
// to finish. The caller must hold a lock on the opMutex for the duration
// of the call.
func (lm *LeaseMaintainer) stop(release bool) error {
	if lm.shutdown == nil {
		return ErrClosed
	}

	lm.shutdown <- release
	close(lm.shutdown)
	<-lm.stopped // closed when lm.run exits

	lm.shutdown = nil
	lm.stopped = nil
	lm.state.Online = false

	return nil
}

// Close causes the maintainer to stop maintenance of any lease it might hold
// and close all listener channels.
func (lm *LeaseMaintainer) Close() error {
	lm.opMutex.Lock()
	defer lm.opMutex.Unlock()

	lm.stop(false)

	lm.stateMutex.Lock()
	defer lm.stateMutex.Unlock()

	for _, listener := range lm.listeners {
		close(listener)
	}

	lm.listeners = nil

	return nil
}

// Update instructs the lease maintainer to update the properties of the
// lease.
func (lm *LeaseMaintainer) Update(props lease.Properties) {
	lm.stateMutex.Lock()
	defer lm.stateMutex.Unlock()
	lm.props = props

	// TODO: Issue an acquisition?
}

// State returns the current lease state.
func (lm *LeaseMaintainer) State() lease.State {
	lm.stateMutex.RLock()
	defer lm.stateMutex.RUnlock()

	return lm.state
}

// Listen returns a channel on which lease states will be broadcast.
//
// If the lease maintainer has already received the result of an acquisition,
// the most recent state will be returned on the channel immediately.
//
// It is important that the caller drains states from the channel in a
// timely manner. Failure to do so will cause the lease maintainer to block
// until the channel's buffer is no longer full.
//
// The provided buffer size will be used for the returned channel.
//
// When the lease maintainer is closed the returned channel will also be closed.
// If the lease maintainer has not yet been started a closed channel will
// be returned.
func (lm *LeaseMaintainer) Listen(bufferSize int) (ch <-chan lease.State) {
	listener := make(chan lease.State, bufferSize)

	lm.opMutex.RLock()
	defer lm.opMutex.RUnlock()

	if lm.shutdown == nil {
		close(listener)
		return listener
	}

	lm.stateMutex.Lock()
	lm.listeners = append(lm.listeners, listener)
	if lm.state.IsZero() {
		lm.stateMutex.Unlock()
	} else {
		go func() {
			defer lm.stateMutex.Unlock()
			listener <- lm.state
		}()
	}

	return listener
}

func (lm *LeaseMaintainer) run(shutdown <-chan bool, stopped chan struct{}) {
	defer close(stopped)

	// Create a context from the shutdown channel so that we can interrupt
	// acquisitions when shutdown is called. We carefully intercept and pass
	// on the shutdown channel's value as we do so.
	var ctx context.Context
	{
		var cancel context.CancelFunc
		ctx, cancel = context.WithCancel(context.Background())
		upstream := shutdown
		downstream := make(chan bool, 1)
		shutdown = downstream
		go func() {
			defer close(downstream)
			release := <-upstream
			downstream <- release
			cancel()
		}()
	}

	// Give our operations 10 seconds to complete
	const timeout = 10 * time.Second

	timer := time.NewTimer(0)
	for {
		select {
		case release := <-shutdown:
			if !timer.Stop() {
				<-timer.C
			}

			if release {
				// Shutdown has already been called, so it's important that we
				// derive ctx from context.Background() here to avoid
				// premature cancellation.
				ctx, cancel := context.WithTimeout(context.Background(), timeout)
				lm.release(ctx)
				cancel()
			}

			return
		case <-timer.C:
			ctx, cancel := context.WithTimeout(ctx, timeout)
			state := lm.acquire(ctx)
			cancel()

			interval := lm.interval(state)
			timer.Reset(interval)
		}
	}
}

func (lm *LeaseMaintainer) acquire(ctx context.Context) lease.State {
	lm.stateMutex.Lock()
	defer lm.stateMutex.Unlock()

	var subject lease.Subject
	if lm.state.Acquired {
		// If we already have a lease, use its subject
		subject = lm.state.Lease.Subject
	} else {
		// If we don't already have a lease, leave the resource empty
		subject.Instance = lm.instance
	}

	response, err := lm.client.Acquire(ctx, subject, lm.props)

	switch err {
	case nil:
		lm.state.Online = true
		lm.state.LeaseNotRequired = false
		lm.state.Acquired = true
		lm.state.Lease = response.Lease
		lm.state.Leases = response.Leases
		lm.state.Err = nil
	case ErrLeaseNotRequired:
		lm.state.Online = true
		lm.state.LeaseNotRequired = true
		lm.state.Acquired = false
		lm.state.Lease = lease.Lease{}
		lm.state.Leases = nil
		lm.state.Err = nil
	default:
		lm.state.Online = false
		lm.state.LeaseNotRequired = false
		lm.state.Err = err
	}

	for _, listener := range lm.listeners {
		listener <- lm.state
	}
	return lm.state
}

func (lm *LeaseMaintainer) release(ctx context.Context) lease.State {
	lm.stateMutex.Lock()
	defer lm.stateMutex.Unlock()

	var err error
	if lm.state.Acquired {
		_, err = lm.client.Release(ctx, lm.state.Lease.Subject)
	}

	if err == nil {
		lm.state.Acquired = false
		lm.state.Online = false
	}

	lm.state.Err = err

	// TODO: Consider broadcasting the state change?
	/*
		for _, listener := range lm.listeners {
			listener <- lm.state
		}
	*/

	return lm.state
}

// interval computes the amount of time to wait until the next lease
// acquisition.
//
// acquired indicates whether we have acquired a lease.
// online indicates indicates whether the server is currently online.
// current is the most recent response without an error.
func (lm *LeaseMaintainer) interval(state lease.State) (interval time.Duration) {
	const transportTime = time.Millisecond * 800 // Ballpark guess at how long it takes to acquire a lease

	defer func() {
		if interval < lease.MinimumRefresh {
			interval = lease.MinimumRefresh
		}
	}()

	// If we haven't received a valid response yet use the retry interval
	if !state.Acquired {
		return lm.retry
	}

	// We have a lease
	interval = state.Lease.EffectiveRefresh()
	now := time.Now()

	// If the server went offline after we retreived a valid lease, use the
	// effective refresh interval or our retry interval, whichever is
	// less.
	if !state.Online && lm.retry < interval {
		interval = lm.retry
	}

	switch state.Lease.Status {
	case lease.Active:
		// If our lease is active make sure we try again before the current lease
		// expires
		exp := state.Lease.ExpirationTime()
		if exp.After(now) {
			remaining := exp.Sub(now)
			if transportTime < remaining {
				remaining = remaining - transportTime
			} else {
				remaining = 0
			}
			if remaining < interval {
				interval = remaining
			}
		} else {
			interval = 0
		}
	case lease.Queued:
		// If our lease is queued, take into consideration when the next
		// lease decays
		decay := state.Leases.DecayDuration(now)
		if decay > 0 && decay < interval {
			interval = decay
		}
	}

	// Under no circumstances should we hammer the server faster than
	// the minimum refresh interval.
	if interval < lease.MinimumRefresh {
		interval = lease.MinimumRefresh
	}

	return interval
}
