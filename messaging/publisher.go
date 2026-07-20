package messaging

import (
	"context"
	"io"
)

// Producer is an event producer
type Publisher interface {
	io.Closer
	Publish(ctx context.Context, event *Event) error
	Ping(ctx context.Context) error
}
