// +build windows

package enforcer

import (
	"github.com/gentlemanautomaton/winproc"
)

// PID is a process ID.
type PID = winproc.ID

// UniqueID is a unique process ID.
type UniqueID = winproc.UniqueID

// Ref is a process reference.
type Ref = winproc.Ref

// Process holds information about a windows process.
type Process = winproc.Process
