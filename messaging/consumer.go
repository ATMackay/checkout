package messaging

import (
	"context"

	"github.com/ATMackay/checkout/event"
)

// Consumer reads events from a stream.
//
// Delivery is at-least-once: an event may be redelivered after a crash or a
// rebalance, so implementations of the read loop must deduplicate on
// event.Event.ID rather than assume each event arrives once.
type Consumer interface {
	// Poll blocks until at least one event is available, ctx is cancelled, or
	// the consumer is closed. A non-empty batch may be returned alongside a nil
	// error even when some partitions failed.
	Poll(ctx context.Context) ([]*event.Event, error)

	// Commit acknowledges every event returned by Poll so far. Call it only
	// after the events have been processed — committing first turns a crash
	// mid-processing into silent data loss.
	Commit(ctx context.Context) error

	// Ping reports broker reachability, for the consuming service's health probe.
	Ping(ctx context.Context) error
}
