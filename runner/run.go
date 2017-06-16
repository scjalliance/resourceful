package runner

import (
	"context"

	"github.com/scjalliance/resourceful/lease/leaseui"
)

// Run will attempt to run the specified program with the given args once
// an active lease for the program has been acquired. If an active lease cannot
// be acquired immediately a queued lease dialog will be displayed to the user.
func Run(ctx context.Context, program string, args []string, servers []string) (err error) {
	return RunWithIcon(ctx, program, args, servers, leaseui.DefaultIcon())
}

// RunWithIcon will attempt to run the specified program with the given args
// once an active lease for the program has been acquired. If an active lease
// cannot be acquired immediately a queued lease dialog will be displayed to
// the user.
//
// The provided icon will be used for the queued lease dialog.
func RunWithIcon(ctx context.Context, program string, args []string, servers []string, icon *leaseui.Icon) (err error) {
	runner, err := New(program, args, servers)
	if err != nil {
		return err
	}

	runner.SetIcon(icon)

	return runner.Run(ctx)
}
