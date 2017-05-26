package leaseutil

import (
	"time"

	"github.com/scjalliance/resourceful/lease"
)

// Refresh will update lease statuses and remove all decayed leases through the
// transaction.
func Refresh(tx *lease.Tx, at time.Time) {
	var allocation uint

	tx.Process(func(iter *lease.Iter) {
		switch iter.Status {
		case lease.Active:
			if iter.Decayed(at) {
				iter.Delete()
				return
			}

			if iter.Expired(at) {
				iter.Status = lease.Released
				iter.Update()
			}

			allocation++
		case lease.Released:
			if iter.Decayed(at) {
				iter.Delete()
				return
			}

			allocation++
		case lease.Queued:
			if iter.Expired(at) {
				iter.Delete()
				return
			}

			if allocation < iter.Limit {
				iter.Status = lease.Active
				iter.Update()
				allocation++
			}
		}
	})
}
