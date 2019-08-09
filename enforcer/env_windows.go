// +build windows

package enforcer

import (
	"github.com/gentlemanautomaton/winproc"
	"github.com/scjalliance/resourceful/environment"
)

// Env returns the lease environment for p.
func Env(host string, p winproc.Process) environment.Environment {
	return environment.Environment{
		"host.name":     host,
		"user.uid":      p.User.SID,
		"user.username": p.User.Account,
		"user.name":     p.User.String(),
	}
}
