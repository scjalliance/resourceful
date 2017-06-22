package leaseutil

import (
	"time"

	"github.com/scjalliance/resourceful/lease"
)

// Refresh will update lease statuses and remove all decayed leases through the
// transaction.
//
// Refresh returns an accumulator that can be queried lease information.
func Refresh(tx *lease.Tx, at time.Time) *Accumulator {
	acc := NewAccumulator()

	replacements := make(lease.Set, 0, 5)

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
			}

			acc.Add(iter.Consumer, iter.Status)
		case lease.Released:
			if iter.Decayed(at) {
				iter.Delete()
				return
			}

			acc.Add(iter.Consumer, iter.Status)
		case lease.Queued:
			if iter.Expired(at) {
				iter.Delete()
				return
			}

			consumed := acc.Total(iter.Strategy)

			// If we're already over-allocated there's no way this lease can be
			// promoted to active
			if consumed > iter.Limit {
				return
			}

			// When possible, replace an existing lease for the same consumer
			// that has already been released and is decaying.
			if acc.Released(iter.Consumer) > 0 {
				// This requires two passes. In this pass we'll note the replacement
				// and delete the queued lease. In the second pass we'll update the
				// decaying lease.
				replacements = append(replacements, iter.Lease)
				iter.Delete()
				acc.StartReplacement(iter.Consumer)
				return
			}

			if CanActivate(iter.Strategy, acc.Active(iter.Consumer), consumed, iter.Limit) {
				iter.Status = lease.Active
				iter.Update()
				acc.Add(iter.Consumer, iter.Status)
			}
		}
	})

	if acc.ReplacementsRecorded() {
		var r int

		tx.ProcessReverse(func(iter *lease.Iter) {
			if iter.Status != lease.Released {
				return
			}
			if acc.Replacements(iter.Consumer) == 0 {
				return
			}
			acc.FinishReplacement(iter.Consumer)
			iter.Lease = replacements[r]
			iter.Status = lease.Active
			iter.Update()
			r++
		})
	}

	return acc
}
