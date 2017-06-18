package leaseui

// Directive instructs the lease user interface manager to display the given
// type of user interface and to call the given callback when finished.
type Directive struct {
	Type     Type
	Callback Callback
}
