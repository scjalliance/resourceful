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
		return s.Instance.Consumed
	case strategy.Consumer:
		return s.Consumer.Consumed
	default:
		panic("unknown strategy")
	}
}

var emptyUserMap = map[string]uint{}

// Users returns a map of users and the number of resources consumed
// by each according to the provided resource counting strategy.
func (s *Stats) Users(strat strategy.Strategy) map[string]uint {
	switch strat {
	case strategy.Instance:
		if s.Instance.Users == nil {
			return emptyUserMap
		}
		return s.Instance.Users
	case strategy.Consumer:
		if s.Consumer.Users == nil {
			return emptyUserMap
		}
		return s.Consumer.Users
	default:
		panic("unknown strategy")
	}
}

// Tally is a set of resource statistics for a particular resource counting
// strategy.
type Tally struct {
	Active   uint
	Released uint
	Queued   uint
	Consumed uint
	Users    map[string]uint
}

// Add will increase the tally for the specified status.
func (t *Tally) Add(user string, status Status) {
	switch status {
	case Active:
		t.Active++
		t.Consumed++
		if t.Users == nil {
			t.Users = make(map[string]uint)
		}
		t.Users[user]++
	case Released:
		t.Released++
		t.Consumed++
	case Queued:
		t.Queued++
	}
}
