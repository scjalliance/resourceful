// +build windows

package enforcer

import (
	"github.com/gentlemanautomaton/winproc"
	"github.com/scjalliance/resourceful/environment"
)

// Process holds information about a windows process.
type Process struct {
	Host     string
	Resource string
	Consumer string
	Instance string
	winproc.Process
	//Detected time.Time
}

func newProcess(hostname string, p winproc.Process) Process {
	return Process{
		Resource: p.Name,
		Consumer: hostname + " " + p.User.String(),
		Instance: p.UniqueID().String(),
		Host:     hostname,
		Process:  p,
	}
}

// Env returns the environment for the process.
func (p Process) Env() environment.Environment {
	env := make(environment.Environment)
	env["host.name"] = p.Host
	env["user.uid"] = p.User.SID
	env["user.username"] = p.User.String()
	env["user.name"] = p.User.Account
	return env
}
