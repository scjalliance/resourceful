package runner

import (
	"os/exec"
	"os/user"
	"path/filepath"

	"github.com/scjalliance/resourceful/lease"
)

// Properties queries the execution context to determine the
// consumer, instance and environment for a resourceful run.
//
// The returned instance will be a random string.
func Properties(c Config, host string, u *user.User) lease.Properties {
	program := c.Program
	path := c.Program

	if abs, err := exec.LookPath(program); err == nil {
		path = abs
		program = filepath.Base(path)
	}

	return lease.Properties{
		"program.name":  program,
		"program.path":  path,
		"host.name":     host,
		"user.id":       u.Uid,
		"user.username": u.Username,
		"user.name":     u.Name,
	}
}
