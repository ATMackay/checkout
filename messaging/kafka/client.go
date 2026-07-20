package kafka

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/ATMackay/checkout/messaging"
	"github.com/twmb/franz-go/pkg/kgo"
)

// Client implements the messaging.Producer and messaging.Consumer
// interface for Kafka.
type Client struct {
	address string
	client  *kgo.Client
}

func NewClient(addr string) (*Client, error) {
	kcl, err := kgo.NewClient(kgo.SeedBrokers(addr))
	if err != nil {
		return nil, fmt.Errorf("error creating Kafka client: %w", err)
	}
	return &Client{
		address: addr,
		client:  kcl,
	}, nil
}

//
// Publisher
//

var _ messaging.Publisher = (*Client)(nil)

func (c *Client) Publish(ctx context.Context, ev *messaging.Event) error {
	if ev == nil || ev.Topic == "" {
		return fmt.Errorf("malformed event")
	}
	// TODO Create key value record from event
	b, err := json.Marshal(ev.Data)
	if err != nil {
		return err
	}
	// TODO - move to async event production
	res := c.client.ProduceSync(ctx, &kgo.Record{
		Topic: ev.Topic,
		Key:   nil, /* TODO*/
		Value: b,
	})
	return res.FirstErr()
}

func (c *Client) Close() error {
	// Close client
	c.client.Close()
	return nil
}

func (c *Client) Ping(ctx context.Context) error {
	return c.client.Ping(ctx)
}
