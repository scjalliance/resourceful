// +build windows

package enforcer

import (
	"github.com/gentlemanautomaton/winproc"
	"github.com/scjalliance/resourceful/lease"
)

// Properties returns the lease properties for p.
func Properties(p winproc.Process, host string) lease.Properties {
	return lease.Properties{
		"program.name":     p.Name,
		"program.path":     p.Path,
		"process.id":       p.ID.String(),
		"process.creation": p.Times.Creation.String(),
		"host.name":        host,
		"user.id":          p.User.SID,
		"user.username":    p.User.Account,
		"user.name":        p.User.String(),
	}
}
