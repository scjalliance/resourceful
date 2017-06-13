package lease

import "github.com/scjalliance/resourceful/strategy"

// Stats is a set of resource consumption statistics for each resource
// counting strategy.
type Stats struct {
	Instance Tally // The number of leases with each lease status
	Consumer Tally // The number of consumers with each lease status
}

// Active returns the number of active resources according to the provided
// resource counting strategy.
func (s *Stats) Active(strat strategy.Strategy) uint {
	switch strat {
	case strategy.Instance:
		return s.Instance.Active
	case strategy.Consumer:
		return s.Consumer.Active
	default:
		panic("unknown strategy")
	}
}

// Released returns the number of released resources according to the provided
// resource counting strategy.
func (s *Stats) Released(strat strategy.Strategy) uint {
	switch strat {
	case strategy.Instance:
		return s.Instance.Released
	case strategy.Consumer:
		return s.Consumer.Released
	default:
		panic("unknown strategy")
	}
}

// Queued returns the number of queued resources according to the provided
// resource counting strategy.
func (s *Stats) Queued(strat strategy.Strategy) uint {
	switch strat {
	case strategy.Instance:
		return s.Instance.Queued
	case strategy.Consumer:
		return s.Consumer.Queued
	default:
		panic("unknown strategy")
	}
}

// Consumed returns the number of consumed resources according to the provided
// resource counting strategy.
//
// Both both active and released resources contribute towards consumption.
func (s *Stats) Consumed(strat strategy.Strategy) uint {
	switch strat {
	case strategy.Instance:
		return s.Instance.Consumed()
	case strategy.Consumer:
		return s.Consumer.Consumed()
	default:
		panic("unknown strategy")
	}
}

// Tally is a set of consumption statistics for a particular resource
// counting strategy.
type Tally struct {
	Active   uint
	Released uint
	Queued   uint
}

// Add will increase the tally for the specified status.
func (t *Tally) Add(status Status) {
	switch status {
	case Active:
		t.Active++
	case Released:
		t.Released++
	case Queued:
		t.Queued++
	}
}

// Consumed returns the number of consumed resources.
//
// Both both active and released resources contribute towards consumption.
func (t *Tally) Consumed() uint {
	return t.Active + t.Released
}
