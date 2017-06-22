package lease

import (
	"sort"
	"time"
)

// Tx is a lease transaction that describes a series of operations to be
// atomically applied to a lease set.
type Tx struct {
	resource string
	revision uint64
	leases   Set
	ops      []Op
}

// NewTx creates a new transaction for the given resource, revision and lease
// set.
func NewTx(resource string, revision uint64, leases Set) *Tx {
	return &Tx{
		resource: resource,
		revision: revision,
		leases:   leases,
	}
}

// Process iterates through each lease and applies the given lease processing
// function to it.
func (tx *Tx) Process(process Processor) {
	for i := 0; i < len(tx.leases); i++ {
		iter := Iter{Lease: Clone(tx.leases[i])}
		process(&iter)
		i += tx.apply(iter.action, iter.Lease, i, 1)
	}
	sort.Sort(tx.leases)
}

// ProcessReverse iterates through each lease in reverse order and applies the
// given lease processing function to it.
func (tx *Tx) ProcessReverse(process Processor) {
	for i := 0; i < len(tx.leases); i++ {
		iter := Iter{Lease: Clone(tx.leases[i])}
		process(&iter)
		i += tx.apply(iter.action, iter.Lease, i, 1)
	}
	sort.Sort(tx.leases)
}

func (tx *Tx) apply(action Action, ls Lease, i int, step int) (shift int) {
	switch action {
	case Update:
		tx.ops = append(tx.ops, Op{
			Type:     Update,
			Previous: tx.leases[i],
			Lease:    ls,
		})
		tx.leases[i] = ls
	case Delete:
		tx.ops = append(tx.ops, Op{
			Type:     Delete,
			Previous: tx.leases[i],
		})
		tx.leases = append(tx.leases[:i], tx.leases[i+1:]...)
		shift = -step
	}
	return
}

// Resource returns the resource the transaction will operate on.
func (tx *Tx) Resource() string {
	return tx.resource
}

// Revision returns the revision of the lease set that the transaction is
// based on.
func (tx *Tx) Revision() uint64 {
	return tx.revision
}

// Consumer returns the set of leases matching the requested consumer.
func (tx *Tx) Consumer(consumer string) (matched Set) {
	return tx.leases.Consumer(tx.resource, consumer)
}

// Instance returns the first lease that matches the given parameters.
func (tx *Tx) Instance(consumer, instance string) (ls Lease, found bool) {
	return tx.leases.Instance(tx.resource, consumer, instance)
}

// Leases returns the lease set that the transaction will produce.
func (tx *Tx) Leases() Set {
	return tx.leases
}

// Create will add the given lease to the set.
func (tx *Tx) Create(ls Lease) error {
	tx.leases = append(tx.leases, ls)
	tx.ops = append(tx.ops, Op{
		Type:  Create,
		Lease: ls,
	})
	sort.Sort(tx.leases)
	return nil
}

// Update will update the lease within the set.
func (tx *Tx) Update(consumer, instance string, ls Lease) error {
	tx.Process(func(iter *Iter) {
		if iter.MatchInstance(tx.resource, consumer, instance) {
			iter.Lease = Clone(ls)
			iter.Update()
		}
	})
	return nil
}

// Release will change the status of the lease to released.
func (tx *Tx) Release(consumer, instance string, at time.Time) error {
	tx.Process(func(iter *Iter) {
		if iter.MatchInstance(tx.resource, consumer, instance) {
			if iter.Status == Active {
				iter.Status = Released
				iter.Released = at
				iter.Update()
			} else {
				iter.Delete()
			}
		}
	})
	return nil
}

// Delete will remove the lease from the set.
func (tx *Tx) Delete(consumer, instance string) error {
	tx.Process(func(iter *Iter) {
		if iter.MatchInstance(tx.resource, consumer, instance) {
			iter.Delete()
		}
	})
	return nil
}

// Stats returns the number of leases with each status.
func (tx *Tx) Stats() Stats {
	return tx.leases.Stats()
}

// Ops returns the series of operations encoded in the transaction.
func (tx *Tx) Ops() []Op {
	return tx.ops
}

// Empty returns true if the transaction is empty
func (tx *Tx) Empty() bool {
	return len(tx.ops) == 0
}

// Effects returns a set of strings describing the effects of the transaction.
func (tx *Tx) Effects() (effects []Effect) {
	for _, op := range tx.ops {
		for _, e := range op.Effects() {
			effects = append(effects, e)
		}
	}
	return
}
