package leaseui

import (
	"context"

	"github.com/scjalliance/resourceful/guardian"
)

// Wait will monitor the responses received on the given channel until the
// evaluation function reports success, the channel is closed or the context
// is cancelled.
//
// Wait returns Success, ChannelClosed or ContextCancelled.
func Wait(ctx context.Context, responses <-chan guardian.Acquisition, fn EvalFunc) (result Result) {
	for {
		select {
		case response, ok := <-responses:
			if !ok {
				return ChannelClosed
			}

			if fn(response) {
				return Success
			}
		case <-ctx.Done():
			return ContextCancelled
		}
	}
}
