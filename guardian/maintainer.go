package guardian

import (
	"context"
	"sync"
	"time"

	"github.com/scjalliance/resourceful/environment"
	"github.com/scjalliance/resourceful/guardian/transport"
	"github.com/scjalliance/resourceful/lease"
)

// Acquisition is the result of an attempted lease acquisition. It is returned
// to listeners of LeaseMaintainer.
type Acquisition struct {
	transport.AcquireResponse
	Err error
}

// LeaseMaintainer will attempt to acquire and maintain a lease.
//
// A lease maintainer is created by calling Client.Maintain(), which accepts a
// context. When the context is cancelled the lease maintainer will be closed.
type LeaseMaintainer struct {
	client   *Client
	resource string
	consumer string
	instance string
	env      environment.Environment
	retry    time.Duration // 0 == no retry
	// maxRetries?

	mutex     sync.RWMutex
	current   Acquisition
	listeners []chan Acquisition
	ready     bool
	closed    bool
}

func newLeaseMaintainer(ctx context.Context, client *Client, resource, consumer, instance string, env environment.Environment, retry time.Duration) *LeaseMaintainer {
	lm := &LeaseMaintainer{
		client:   client,
		resource: resource,
		consumer: consumer,
		instance: instance,
		env:      env,
		retry:    retry,
	}
	go lm.run(ctx, client)
	return lm
}

// Listen returns a channel on which lease acquisitions will be broadcast.
//
// If the lease maintainer has already received the result of an acquisition,
// the most recent acquisition will be returned on the channel immediately.
//
// It is important that the caller drains acquisitions from the channel in a
// timely manner. Failure to do so will cause the lease maintainer to block
// until the channel's buffer is no longer full.
//
// The provided buffer size will be used for the returned channel.
//
// When the lease maintainer is closed the returned channel will also be closed.
// If the lease maintainer has already been closed a closed channel will
// be returned.
func (lm *LeaseMaintainer) Listen(bufferSize int) (ch <-chan Acquisition) {
	listener := make(chan Acquisition, bufferSize)

	lm.mutex.Lock()
	if lm.closed {
		lm.mutex.Unlock()
		close(listener)
		return listener
	}

	lm.listeners = append(lm.listeners, listener)

	if lm.ready {
		go func() {
			defer lm.mutex.Unlock()
			listener <- lm.current
		}()
	} else {
		lm.mutex.Unlock()
	}

	return listener
}

func (lm *LeaseMaintainer) run(ctx context.Context, client *Client) {
	defer lm.close()

	var (
		acquired bool
		current  transport.AcquireResponse
	)

	timer := time.NewTimer(0)
	for {
		select {
		case <-ctx.Done():
			if !timer.Stop() {
				<-timer.C
			}
			lm.release()
			return
		case <-timer.C:
			response, err := lm.acquire()
			var online bool
			if err == nil {
				acquired = true
				online = true
				current = response
			}
			interval := lm.interval(acquired, online, current)
			timer.Reset(interval)
		}
	}
}

func (lm *LeaseMaintainer) acquire() (response transport.AcquireResponse, err error) {
	var listeners []chan Acquisition // Local copy to avoid holding a lock during broadcast

	response, err = lm.client.Acquire(lm.resource, lm.consumer, lm.instance, lm.env)

	acquisition := Acquisition{
		AcquireResponse: response,
		Err:             err,
	}

	if err == nil {
		lm.mutex.Lock()
		listeners = make([]chan Acquisition, len(lm.listeners))
		copy(listeners, lm.listeners)
		lm.current = acquisition
		lm.ready = true
		lm.mutex.Unlock()
	} else {
		lm.mutex.RLock()
		listeners = make([]chan Acquisition, len(lm.listeners))
		copy(listeners, lm.listeners)
		lm.mutex.RUnlock()
	}

	for _, listener := range listeners {
		listener <- acquisition
	}
	return
}

func (lm *LeaseMaintainer) release() {
	lm.client.Release(lm.resource, lm.consumer, lm.instance)
	// TODO: Do something with the response?
}

func (lm *LeaseMaintainer) close() {
	lm.mutex.Lock()
	defer lm.mutex.Unlock()

	if lm.closed {
		// already closed
		return
	}

	for _, listener := range lm.listeners {
		close(listener)
	}

	lm.listeners = nil
	lm.closed = true
}

// interval computes the interval until the next lease acquisition.
//
// acquired indicates whether we have acquired a lease.
// online indicates indicates whether the server is currently online.
// current is the most recent response without an error.
func (lm *LeaseMaintainer) interval(acquired bool, online bool, current transport.AcquireResponse) (interval time.Duration) {
	defer func() {
		if interval < lease.MinimumRefresh {
			interval = lease.MinimumRefresh
		}
	}()

	// If we haven't received a valid response yet use the retry interval
	if !acquired {
		return lm.retry
	}

	// We have a lease
	interval = current.Lease.EffectiveRefresh()

	// If the server went offline after we retreived a valid lease, use the
	// effective refresh interval or our retry interval, whichever is
	// less.
	if !online {
		if lm.retry < interval {
			return lm.retry
		}
		return interval
	}

	// If our lease is queued, take into consideration when the next
	// lease decays
	if current.Lease.Status == lease.Queued {
		decay := current.Leases.DecayDuration(time.Now())
		if decay > 0 && decay < interval {
			interval = decay
		}
	}

	if interval < lease.MinimumRefresh {
		// Under no circumstances should we hammer the server faster than
		// the minimum refresh interval.
		interval = lease.MinimumRefresh
	}

	return interval
}
