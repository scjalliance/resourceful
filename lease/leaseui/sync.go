package leaseui

import (
	"context"
	"time"

	"github.com/scjalliance/resourceful/guardian"
)

// Sync will synchronize the model with the responses received on the given
// channel until the evaluation function reports success, the channel is
// closed or the context is cancelled.
//
// Sync returns Success, ChannelClosed or ContextCancelled.
func Sync(ctx context.Context, model Model, responses <-chan guardian.Acquisition, fn EvalFunc) (result Result) {
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
