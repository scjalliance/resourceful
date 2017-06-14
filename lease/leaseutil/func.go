package leaseutil

import "github.com/scjalliance/resourceful/strategy"

// CanActivate returns true if a lease can be made active under the specified
// resource counting strategy.
//
// Active is the number of active leases for the consumer requesting a lease.
// Consumed is the total number of consumed resources according to strategy.
// Limit is the resource allocation limit.
func CanActivate(strat strategy.Strategy, active, consumed, limit uint) bool {
	if limit == 0 {
		return false
	}
	if consumed > limit {
		return false
	}
	if strat == strategy.Consumer {
		if active > 0 {
			return true
		}
	}
	return consumed < limit
}
