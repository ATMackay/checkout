package noop

import (
	"context"

	"github.com/ATMackay/checkout/event"
	"github.com/ATMackay/checkout/messaging"
)

// Client Implements a no-op messaging client. Useful if a non-nil messaging client is required
type Client struct{}

//
// Publisher
//

var _ messaging.Publisher = (*Client)(nil)

func (c *Client) Publish(context.Context, *event.Event) error {
	return nil
}

func (c *Client) Close() error {
	return nil
}

func (c *Client) Ping(context.Context) error {
	return nil
}

var _ messaging.Consumer = (*Client)(nil)

func (c *Client) Poll(ctx context.Context) ([]*event.Event, error) {
	return nil, nil
}

func (c *Client) Commit(ctx context.Context) error {
	return nil
}
