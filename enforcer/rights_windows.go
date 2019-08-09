// +build windows

package enforcer

import (
	"github.com/gentlemanautomaton/winproc"
	"github.com/gentlemanautomaton/winproc/processaccess"
)

var (
	activeRights = []processaccess.Rights{
		processaccess.QueryLimitedInformation | processaccess.Synchronize | processaccess.Terminate,
	}
	passiveRights = []processaccess.Rights{
		processaccess.QueryLimitedInformation | processaccess.Synchronize,
		processaccess.QueryLimitedInformation,
	}
)

func openProcess(id winproc.ID, passive bool) (ref *winproc.Ref, err error) {
	// Open a reference to the process with the highest level of privilege
	// that we can get
	if !passive {
		for _, rights := range activeRights {
			ref, err = winproc.Open(id, rights)
			if err == nil {
				return
			}
		}
	}

	for _, rights := range passiveRights {
		ref, err = winproc.Open(id, rights)
		if err == nil {
			return
		}
	}

	return
}
