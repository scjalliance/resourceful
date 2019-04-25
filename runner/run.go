package runner

import (
	"context"

	"github.com/scjalliance/resourceful/guardian"
	"github.com/scjalliance/resourceful/lease/leaseui"
)

// Run will attempt to run a program with arguments specified by the given
// configuration.
//
// The program will only be run when and while an active lease for the program
// has been acquired from a guardian server. If an active lease cannot be
// acquired immediately a queued lease dialog will be displayed to the user.
func Run(ctx context.Context, client *guardian.Client, config Config) (err error) {
	if config.Icon == nil {
		config.Icon = leaseui.DefaultIcon()
	}

	runner, err := New(client, config)
	if err != nil {
		return err
	}

	return runner.Run(ctx)
}
