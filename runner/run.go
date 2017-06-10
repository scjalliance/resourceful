package runner

import (
	"github.com/scjalliance/resourceful/lease/leaseui"
)

// Run will attempt to run the specified program with the given args once
// an active lease for the program has been acquired. If an active lease cannot
// be acquired immediately a queued lease dialog will be displayed to the user.
func Run(program string, args []string) (err error) {
	return RunWithIcon(program, args, leaseui.DefaultIcon())
}

// RunWithIcon will attempt to run the specified program with the given args
// once an active lease for the program has been acquired. If an active lease
// cannot be acquired immediately a queued lease dialog will be displayed to
// the user.
//
// The provided icon will be used for the queued lease dialog.
func RunWithIcon(program string, args []string, icon *leaseui.Icon) (err error) {
	runner, err := New(program, args)
	if err != nil {
		return err
	}

	runner.SetIcon(icon)

	return runner.Run()
}
