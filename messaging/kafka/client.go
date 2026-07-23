// Package kafka implements the messaging interfaces against a Kafka broker.
package kafka

import (
	"context"
	"errors"
	"fmt"
	"log/slog"

	"github.com/ATMackay/checkout/event"
	"github.com/ATMackay/checkout/messaging"
	"github.com/twmb/franz-go/pkg/kgo"
)

// ErrNotConsumer is returned by the consumer methods on a Client that was not
// built with WithConsumerGroup.
var ErrNotConsumer = errors.New("client is not configured to consume")

// Client implements messaging.Publisher, and additionally messaging.Consumer
// when constructed with WithConsumerGroup.
type Client struct {
	addresses []string
	group     string
	client    *kgo.Client
}

// Option configures a Client.
type Option func(*options)

type options struct {
	group  string
	topics []string
}

// WithConsumerGroup makes the Client a member of the named consumer group,
// subscribed to topics.
//
// Partitions are shared among the members of a group, so scaling a group beyond
// the partition count leaves the extra members idle. Independent groups each
// receive the full stream, which is how one topic fans out to several services.
//
// Auto-committing is disabled: offsets advance only when Commit is called, so
// an event is redelivered if the process dies before it is processed.
func WithConsumerGroup(group string, topics ...string) Option {
	return func(o *options) {
		o.group = group
		o.topics = topics
	}
}

// NewClient connects to the brokers at brokerAddresses.
//
// Producing is idempotent by default in franz-go: the broker discards a
// duplicate caused by a retried produce request. Note this only spans a single
// producer session — it is not protection against the application publishing
// the same event again after a restart, which is what event.Event.ID is for.
func NewClient(brokerAddresses []string, opts ...Option) (*Client, error) {
	if len(brokerAddresses) == 0 {
		return nil, errors.New("no broker addresses supplied")
	}

	var o options
	for _, opt := range opts {
		opt(&o)
	}

	kopts := []kgo.Opt{kgo.SeedBrokers(brokerAddresses...)}
	if o.group != "" {
		kopts = append(kopts,
			kgo.ConsumerGroup(o.group),
			kgo.ConsumeTopics(o.topics...),
			// Start from the beginning of the log when the group has no
			// committed offset, so a new consumer sees existing history.
			kgo.ConsumeResetOffset(kgo.NewOffset().AtStart()),
			kgo.DisableAutoCommit(),
		)
	} else {
		// Producer: let the broker create the topic on first publish so a fresh
		// deployment needs no manual topic-create step.
		kopts = append(kopts, kgo.AllowAutoTopicCreation())
	}

	kcl, err := kgo.NewClient(kopts...)
	if err != nil {
		return nil, fmt.Errorf("error creating Kafka client: %w", err)
	}
	return &Client{
		addresses: brokerAddresses,
		group:     o.group,
		client:    kcl,
	}, nil
}

//
// Publisher
//

var _ messaging.Publisher = (*Client)(nil)

// Publish writes the event synchronously and returns once the brokers have
// acknowledged it.
//
// The event key becomes the record key, which is what determines the partition
// and therefore the ordering guarantee: events sharing a key land on the same
// partition and are consumed in publish order.
func (c *Client) Publish(ctx context.Context, ev *event.Event) error {
	value, err := ev.Encode()
	if err != nil {
		return err
	}
	res := c.client.ProduceSync(ctx, &kgo.Record{
		Topic: ev.Topic,
		Key:   []byte(ev.Key),
		Value: value,
	})
	return res.FirstErr()
}

func (c *Client) Close() error {
	// Close client
	slog.Debug("closing kafka client")
	c.client.Close()
	return nil
}

func (c *Client) Ping(ctx context.Context) error {
	return c.client.Ping(ctx)
}

//
// Consumer
//

var _ messaging.Consumer = (*Client)(nil)

// Poll blocks until at least one event is available, ctx is cancelled, or the
// Client is closed.
func (c *Client) Poll(ctx context.Context) ([]*event.Event, error) {
	if c.group == "" {
		return nil, ErrNotConsumer
	}

	fetches := c.client.PollFetches(ctx)
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	// Fetch errors are per-partition, and a batch may carry records even when
	// one partition failed. Surface the errors but keep whatever arrived.
	var errs []error
	for _, fe := range fetches.Errors() {
		errs = append(errs, fmt.Errorf("fetch %s[%d]: %w", fe.Topic, fe.Partition, fe.Err))
	}

	var events []*event.Event
	var decodeErrs []error
	fetches.EachRecord(func(r *kgo.Record) {
		ev, err := event.Decode(r.Topic, string(r.Key), r.Value)
		if err != nil {
			decodeErrs = append(decodeErrs, fmt.Errorf("record %s[%d]@%d: %w", r.Topic, r.Partition, r.Offset, err))
			return
		}
		events = append(events, ev)
	})
	errs = append(errs, decodeErrs...)

	if len(events) == 0 && len(errs) > 0 {
		return nil, errors.Join(errs...)
	}
	return events, nil
}

// Commit acknowledges every event returned by Poll so far.
func (c *Client) Commit(ctx context.Context) error {
	if c.group == "" {
		return ErrNotConsumer
	}
	return c.client.CommitUncommittedOffsets(ctx)
}
