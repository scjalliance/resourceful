// +build windows

package leaseui

import (
	"time"

	"github.com/scjalliance/resourceful/guardian"
	"github.com/scjalliance/resourceful/lease"
)

// Sync will synchronize the model with the responses received on the given
// channel until an active lease is acquired or the channel is closed.
//
// Sync returns true if an active lease was acquired.
func Sync(model *Model, responses <-chan guardian.Acquisition) (acquired bool) {
	sleepRound()

	ticker := time.NewTicker(time.Second)
	defer ticker.Stop()

	for {
		select {
		case response, ok := <-responses:
			if !ok {
				return false
			}

			model.Update(response)

			// TODO: Examine and report errors?

			if response.Lease.Status == lease.Active {
				return true
			}
		case <-ticker.C:
			for r := 0; r < model.RowCount(); r++ {
				model.PublishRowChanged(r)
			}
		}
	}
}

// sleepRound will sleep until the current time is near a whole second.
func sleepRound() {
	now := time.Now()
	start := now.Round(time.Second)
	if start.Before(now) {
		// We rounded down
		start.Add(time.Second)
	}
	duration := start.Sub(now)
	time.Sleep(duration)
}
