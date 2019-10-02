// +build windows

package enforcer

import (
	"github.com/gentlemanautomaton/winproc"
	"github.com/scjalliance/resourceful/lease"
)

// Properties returns the lease properties for p.
func Properties(p winproc.Process, environment lease.Properties) lease.Properties {
	props := make(lease.Properties, len(environment)+7)
	for k, v := range environment {
		props[k] = v
	}
	props["program.name"] = p.Name
	props["program.path"] = p.Path
	props["process.id"] = p.ID.String()
	props["process.creation"] = p.Times.Creation.String()
	props["user.id"] = p.User.SID
	props["user.account"] = p.User.Account
	props["user.domain"] = p.User.Domain
	return props
}
