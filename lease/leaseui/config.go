package leaseui

import "github.com/scjalliance/resourceful/lease"

// Config holds common configuration for a lease user interface.
type Config struct {
	Icon     *Icon
	Program  string
	Instance lease.Instance // Only used to identify our own lease in a list
}
