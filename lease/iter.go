package lease

// Processor is a function that processes a lease iterator.
type Processor func(iter *Iter)

// Iter is a lease iterator that allows changes to be recorded in a transaction.
type Iter struct {
	action Action
	Lease
}

// Update will update the lease and record the update in the transaction.
func (iter *Iter) Update() {
	iter.action = Update
}

// Delete will delete the lease and record the deletion in the transaction.
func (iter *Iter) Delete() {
	iter.action = Delete
}
