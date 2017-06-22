package leaseutil

import (
	"fmt"

	"github.com/scjalliance/resourceful/lease"
	"github.com/scjalliance/resourceful/strategy"
)

// Accumulator tracks the number of leases with each status. It is used
// internally by the refresh function as it processes lease sets.
type Accumulator struct {
	total        uint            // The total number of active or released leases
	active       map[string]uint // The number of active leases for each consumer
	released     map[string]uint // The number of released leases for each consumer
	consumed     map[string]uint // The number of active or released leases for each consumer
	replacements map[string]uint // The number of outstanding lease replacements for each consumer
}

// NewAccumulator returns a new lease accumulator that tracks the number of
// leases with each status.
func NewAccumulator() *Accumulator {
	return &Accumulator{
		active:       make(map[string]uint),
		released:     make(map[string]uint),
		consumed:     make(map[string]uint),
		replacements: make(map[string]uint),
	}
}

// Add will record a lease with the given status and consumer within the
// accumulator.
func (a *Accumulator) Add(consumer string, status lease.Status) {
	switch status {
	case lease.Active:
		a.total++
		a.active[consumer]++
		a.consumed[consumer]++
	case lease.Released:
		a.total++
		a.released[consumer]++
		a.consumed[consumer]++
	}
}

// StartReplacement will record the start of a lease replacement for consumer.
//
// If there are no released leases available for replacement the function
// will panic.
func (a *Accumulator) StartReplacement(consumer string) {
	if a.released[consumer] == 0 {
		panic(fmt.Errorf("leaseutil: accumulator: cannot start replacement lease for \"%s\" because no leases are replaceable", consumer))
	}
	a.replacements[consumer]++
	a.active[consumer]++
	a.released[consumer]--
	if a.released[consumer] == 0 {
		delete(a.released, consumer)
	}
}

// FinishReplacement will record the completion of a lease replacement for
// consumer.
//
// If there are no lease replacements pending the function will panic.
func (a *Accumulator) FinishReplacement(consumer string) {
	if a.replacements[consumer] == 0 {
		panic(fmt.Errorf("leaseutil: accumulator: cannot finished replacement lease for \"%s\" because no replacement was started", consumer))
	}
	a.replacements[consumer]--
	if a.replacements[consumer] == 0 {
		delete(a.replacements, consumer)
	}
}

// Active returns the number of active leases for the consumer.
func (a *Accumulator) Active(consumer string) uint {
	return a.active[consumer]
}

// Released returns the number of released leases for the consumer.
func (a *Accumulator) Released(consumer string) uint {
	return a.released[consumer]
}

// Consumed returns the number of consumed resources for the consumer according
// to the resource counting strategy.
func (a *Accumulator) Consumed(consumer string, strat strategy.Strategy) uint {
	switch strat {
	default:
		if a.consumed[consumer] > 0 {
			return 1
		}
		return 0
	case strategy.Consumer:
		return a.consumed[consumer]
	}
}

// Replacements returns the number of outstanding lease replacements for
// consumer.
func (a *Accumulator) Replacements(consumer string) uint {
	return a.replacements[consumer]
}

// ReplacementsRecorded returns the true if one or more replacements were
// were started.
func (a *Accumulator) ReplacementsRecorded() bool {
	return len(a.replacements) > 0
}

// Total returns the total number of consumed resources according to the
// resource counting strategy.
func (a *Accumulator) Total(strat strategy.Strategy) uint {
	switch strat {
	default:
		return a.total
	case strategy.Consumer:
		return uint(len(a.consumed))
	}
}
