package logprov

import (
	"fmt"
	"log"
	"strings"
	"sync"
	"time"

	"github.com/scjalliance/resourceful/counter"
	"github.com/scjalliance/resourceful/lease"
)

type timestamp struct {
	ops  uint64    // How many operations had been committed
	when time.Time // What time it was
}

// Provider provides boltdb-backed lease management.
type Provider struct {
	source   lease.Provider
	log      *log.Logger
	schedule []Schedule
	ops      counter.Counter

	mutex  sync.RWMutex  // Locked while checkpointing
	last   timestamp     // Values of the last checkpoint
	closed chan struct{} // Closed when the provider closes
}

// New returns a new transaction logging provider.
func New(source lease.Provider, logger *log.Logger, schedule ...Schedule) *Provider {
	p := &Provider{
		source:   source,
		log:      logger,
		schedule: schedule,
		closed:   make(chan struct{}),
	}
	if d, ok := p.durationSchedule(); ok {
		// Start a goroutine to handle duration scheduling
		go p.durationCheckpoint(d, p.closed)
	}
	return p
}

// Close releases any resources consumed by the provider and its source.
func (p *Provider) Close() error {
	p.mutex.Lock()
	defer p.mutex.Unlock()
	if p.closed != nil {
		close(p.closed)
		p.closed = nil
	}
	return p.source.Close()
}

// ProviderName returns the name of the provider.
func (p *Provider) ProviderName() string {
	return fmt.Sprintf("%s (with logged transactions)", p.source.ProviderName())
}

// LeaseResources returns all of the resources with lease data.
func (p *Provider) LeaseResources() (resources []string, err error) {
	return p.source.LeaseResources()
}

// LeaseView returns the current revision and lease set for the resource.
func (p *Provider) LeaseView(resource string) (revision uint64, leases lease.Set, err error) {
	p.mutex.RLock()
	defer p.mutex.RUnlock()
	return p.source.LeaseView(resource)
}

// LeaseCommit will attempt to apply the operations described in the lease
// transaction.
func (p *Provider) LeaseCommit(tx *lease.Tx) error {
	p.mutex.RLock()
	defer p.mutex.RUnlock()

	err := p.source.LeaseCommit(tx)
	if err == nil {
		p.record(tx)
	}
	return err
}

// Checkpoint will write all of the lease data to the transaction log in a
// checkpoint block.
//
// In order for the checkpoint to obtain a consistent view of the lease data it
// must hold an exclusive lock while the chekcpoint is being performed. All
// other operations on the provider will block until the checkpoint has
// finished.
func (p *Provider) Checkpoint() (err error) {
	p.mutex.Lock()
	defer p.mutex.Unlock()

	return p.checkpoint()
}

// record records the transaction in the transaction log.
//
// record assumes that a read lock is held for the duration of the call.
func (p *Provider) record(tx *lease.Tx) {
	var ops uint64 // Total number of consumptive ops (really op effects)

	for _, op := range tx.Ops() {
		if op.Type == lease.Update && op.UpdateType() == lease.Renew {
			// Don't record renewals
			continue
		}
		for _, effect := range op.Effects() {
			if !effect.Consumptive() {
				// Only record effects that affect consumption
				continue
			}
			p.log.Printf("TX %s", effect.String())
			ops++
		}
	}

	p.add(ops)
}

// add adds the given number of operations to the ops counter.
//
// add assumes that a read lock is held for the duration of the call.
func (p *Provider) add(ops uint64) {
	if ops == 0 {
		return
	}

	var (
		current = p.ops.Add(ops)       // How many ops have passed since the provider started
		count   = current - p.last.ops // How many ops have passed since the last checkpoint
	)

	if ops, ok := p.opsSchedule(); ok && count >= ops {
		// Run the checkpoint in a separate goroutine so we don't deadlock
		go p.opsCheckpoint(ops)
	}
}

// opsCheckpoint will start a checkpoint if the a sufficient number of ops
// have passed since the last checkpoint occurred.
func (p *Provider) opsCheckpoint(ops uint64) {
	p.mutex.Lock()
	defer p.mutex.Unlock()

	if p.ops.Value()-p.last.ops < ops {
		// Another goroutine beat us to the checkpoint
		return
	}

	p.checkpoint()
}

// durationCheckpoint will start a checkpoint whenever the amount of time since
// the last checkpoint exceeds the given interval.
//
// durationCheckpoint blocks until the given channel is closed.
func (p *Provider) durationCheckpoint(interval time.Duration, done chan struct{}) {
	for {
		p.mutex.Lock()

		select {
		case <-done:
			return // Closed
		default:
		}

		next := nextCheckpoint(time.Now(), p.last.when, interval)
		if next <= 0 {
			p.checkpoint()
			next = nextCheckpoint(time.Now(), p.last.when, interval)
		}

		p.mutex.Unlock()

		if next < MinimumDuration {
			next = MinimumDuration
		}

		t := time.NewTimer(next)
		select {
		case <-t.C:
		case <-done:
			if !t.Stop() {
				<-t.C
			}
			return
		}
	}
}

// checkpoint writes checkpoint data to the transaction log.
//
// checkpoint assumes that a write lock is held for the duration of the call.
func (p *Provider) checkpoint() (err error) {
	at := time.Now()
	nano := at.UnixNano()

	resources, err := p.source.LeaseResources()
	if err != nil {
		return
	}

	p.log.Printf("CP %v START", nano)

	for _, resource := range resources {
		_, leases, viewErr := p.source.LeaseView(resource)
		if viewErr != nil {
			p.log.Printf("CP %v RESOURCE %s ERR %v", nano, resource, err)
		} else {
			//p.log.Printf("CP %v RESOURCE %s REV %d", at, resource, revision)
			for _, ls := range leases {
				if ls.Consumptive() {
					p.log.Printf("CP %v LEASE %s %s", nano, ls.Subject(), strings.ToUpper(string(ls.Status)))
				}
			}
		}
	}

	p.log.Printf("CP %v END", nano)

	p.last.ops = p.ops.Value()
	p.last.when = at

	return
}

func (p *Provider) opsSchedule() (ops uint64, ok bool) {
	for _, s := range p.schedule {
		if s.t != opsSchedule {
			continue
		}
		if s.ops == 0 {
			continue
		}
		if !ok {
			ok = true
			ops = s.ops
		} else {
			if s.ops < ops {
				ops = s.ops
			}
		}
	}
	return
}

func (p *Provider) durationSchedule() (d time.Duration, ok bool) {
	for _, s := range p.schedule {
		if s.t != durationSchedule {
			continue
		}
		if s.duration < MinimumDuration {
			continue
		}
		if !ok {
			ok = true
			d = s.duration
		} else {
			if s.duration < d {
				d = s.duration
			}
		}
	}
	return
}

// nextCheckpoint returns the duration of time to wait for the next
// scheduled checkpoint.
func nextCheckpoint(now time.Time, last time.Time, interval time.Duration) (next time.Duration) {
	if last.IsZero() {
		// Checkpoint immediately if we haven't yet
		return
	}

	if now.Before(last) {
		// Something unexpected happened like a major wall clock shift (or bad code)
		return
	}

	elapsed := now.Sub(last)
	if elapsed >= interval {
		// Checkpoint now
		return
	}

	return interval - elapsed
}
