package messaging

import (
	"context"
	"io"

	"github.com/ATMackay/checkout/event"
)

//go:generate mockgen -destination mock/messaging.go -package mock github.com/ATMackay/checkout/messaging Publisher,Consumer

// Producer is an event producer
type Publisher interface {
	io.Closer
	Publish(ctx context.Context, event *event.Event) error
	Ping(ctx context.Context) error
}
