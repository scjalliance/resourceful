// +build windows

package enforcer

import (
	"github.com/gentlemanautomaton/winproc"
	"github.com/scjalliance/resourceful/lease"
)

// Subject returns the lease subject for p.
func Subject(host string, p winproc.Process) lease.Subject {
	return lease.Subject{
		Resource: p.Name,
		Consumer: host + " " + p.User.String(),
		Instance: p.UniqueID().String(),
	}
}
