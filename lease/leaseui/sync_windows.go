// +build windows

package leaseui

import (
	"context"
	"time"

	"github.com/scjalliance/resourceful/guardian"
)

type SyncFunc func(response guardian.Acquisition) (success bool)

// Sync will synchronize the model with the responses received on the given
// channel until an active lease is acquired or the channel is closed.
//
// Sync returns true if an active lease was acquired.
func Sync(ctx context.Context, model Model, responses <-chan guardian.Acquisition, fn SyncFunc) (result Result) {
	sleepRound()

	ticker := time.NewTicker(time.Second)
	defer ticker.Stop()

	for {
		select {
		case response, ok := <-responses:
			if !ok {
				return ChannelClosed
			}

			model.Update(response)

			// TODO: Examine and report errors?

			if fn(response) {
				return Success
			}
		case <-ticker.C:
			model.Refresh()
		case <-ctx.Done():
			return ContextCancelled
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
