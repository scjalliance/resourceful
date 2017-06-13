package leaseutil

import (
	"time"

	"github.com/scjalliance/resourceful/lease"
	"github.com/scjalliance/resourceful/strategy"
)

// Refresh will update lease statuses and remove all decayed leases through the
// transaction.
func Refresh(tx *lease.Tx, at time.Time) {
	var instances uint // active + released

	active := make(map[string]int)
	released := make(map[string]int)
	replaced := make(map[string]int)

	tx.Process(func(iter *lease.Iter) {
		switch iter.Status {
		case lease.Active:
			if iter.Decayed(at) {
				iter.Delete()
				return
			}

			if iter.Expired(at) {
				iter.Status = lease.Released
				iter.Released = iter.Renewed.Add(iter.Duration)
				iter.Update()
				released[iter.Consumer]++
			} else {
				active[iter.Consumer]++
			}

			instances++
		case lease.Released:
			if iter.Decayed(at) {
				iter.Delete()
				return
			}

			released[iter.Consumer]++

			instances++
		case lease.Queued:
			if iter.Expired(at) {
				iter.Delete()
				return
			}

			var allocation uint
			switch iter.Strategy {
			default:
				allocation = instances
			case strategy.Consumer:
				allocation = uint(len(active) + len(released))
			}

			// When possible, replace an existing lease for the same consumer
			// that has already been released and is decaying.
			if released[iter.Consumer] > 0 && allocation <= iter.Limit {
				// This requires two passes. In this pass we'll update the queued
				// lease to make it active. We note the replacement here and then delete
				// the lease that was replaced in the second pass.
				released[iter.Consumer]--
				if released[iter.Consumer] == 0 {
					delete(released, iter.Consumer)
				}
				replaced[iter.Consumer]++
				iter.Status = lease.Active
				iter.Update()
				return
			}

			if allocation < iter.Limit {
				iter.Status = lease.Active
				iter.Update()
				active[iter.Consumer]++
				instances++
			}
		}
	})

	if len(replaced) > 0 {
		tx.ProcessReverse(func(iter *lease.Iter) {
			if iter.Status != lease.Released {
				return
			}
			if replaced[iter.Consumer] == 0 {
				return
			}
			replaced[iter.Consumer]--
			iter.Delete()
		})
	}
}
