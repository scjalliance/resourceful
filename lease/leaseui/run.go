// +build !windows

package leaseui

import (
	"context"

	"github.com/scjalliance/resourceful/guardian"
	"github.com/scjalliance/resourceful/lease"
)

// WaitForActive will create and manage a queued lease user interface. It will
// return when an active lease is acquired or the user has indicated that they
// would like to cancel.
func WaitForActive(ctx context.Context, icon *Icon, program, consumer string, response guardian.Acquisition, responses <-chan guardian.Acquisition) (acquired bool, err error) {
	for {
		response, ok := <-responses
		if !ok {
			return false, nil
		}

		// TODO: Examine and report errors?

		if response.Lease.Status == lease.Active {
			return true, nil
		}
	}
}
