package orders

import (
	"context"
	"log/slog"
	"sync"
	"time"

	"github.com/ATMackay/checkout/database"
	"github.com/ATMackay/checkout/event"
	"github.com/ATMackay/checkout/messaging"
	"github.com/ATMackay/checkout/model"
)

//go:generate mockgen -destination mock/relay.go -package mock github.com/ATMackay/checkout/services/orders Relayer

// Relayer drains the transactional outbox to the message broker. It is a
// long-lived background process: Start spawns its goroutine, Stop terminates it.
type Relayer interface {
	Start(ctx context.Context) error
	Stop() error
	Ping(ctx context.Context) error
}

const (
	// defaultPollInterval is how often the relay scans the outbox when idle.
	defaultPollInterval = time.Second
	// defaultBatchSize caps how many rows are claimed per scan.
	defaultBatchSize = 100
)

// OutboxRelayer polls the outbox for unpublished rows and publishes them to the
// broker, marking each published on success. It closes the loop opened by the
// purchase handler writing an outbox row inside the order transaction:
// durability lives in the table, so an unpublished row simply waits for the next
// scan (or the next process start) rather than being lost.
type OutboxRelayer struct {
	outboxStore  database.OutboxStore
	publisher    messaging.Publisher
	pollInterval time.Duration
	batchSize    int

	quit     chan struct{}
	wg       sync.WaitGroup
	stopOnce sync.Once
}

// Option configures an OutboxRelayer.
type Option func(*OutboxRelayer)

// WithPollInterval overrides how often the relay scans the outbox when idle.
func WithPollInterval(d time.Duration) Option {
	return func(o *OutboxRelayer) { o.pollInterval = d }
}

// WithBatchSize overrides how many rows are claimed per scan.
func WithBatchSize(n int) Option {
	return func(o *OutboxRelayer) { o.batchSize = n }
}

// NewOutboxRelayer builds a relayer over the given store and publisher.
func NewOutboxRelayer(store database.OutboxStore, publisher messaging.Publisher, opts ...Option) *OutboxRelayer {
	o := &OutboxRelayer{
		outboxStore:  store,
		publisher:    publisher,
		pollInterval: defaultPollInterval,
		batchSize:    defaultBatchSize,
		quit:         make(chan struct{}),
	}
	for _, opt := range opts {
		opt(o)
	}
	return o
}

// Start launches the poll loop. ctx bounds a fail-fast connectivity check only;
// the loop itself is torn down via Stop, not ctx, so no context is retained.
func (o *OutboxRelayer) Start(ctx context.Context) error {
	if err := o.publisher.Ping(ctx); err != nil {
		return err
	}
	o.wg.Add(1)
	go o.run()
	slog.Info("outbox relayer started", "poll_interval", o.pollInterval, "batch_size", o.batchSize)
	return nil
}

// run scans the outbox on every tick until quit is closed. It owns no request
// context, so publishes use context.Background() (an honest "no upstream
// deadline" for a background worker).
func (o *OutboxRelayer) run() {
	defer o.wg.Done()

	ticker := time.NewTicker(o.pollInterval)
	defer ticker.Stop()

	var counter int64
	for {
		select {
		case <-o.quit:
			return
		case <-ticker.C:
			slog.Debug("checking outbox", "tick", counter)
			o.drain(context.Background())
		}
	}
}

// drain publishes unpublished rows until a scan returns less than a full batch
// (caught up) or an error. A scan error ends this cycle; the next tick retries.
func (o *OutboxRelayer) drain(ctx context.Context) {
	for {
		items, err := o.outboxStore.GetOutboxItems(ctx, &database.OutboxQuery{
			OnlyUnpublished: true,
			Limit:           o.batchSize,
		})
		if err != nil {
			slog.Error("outbox scan failed", "error", err)
			return
		}
		if len(items) < 1 {
			return
		}
		slog.Info("publishing events", "item_count", len(items))
		for _, item := range items {
			o.publish(ctx, item)
		}
		// A short batch means the outbox is drained; wait for the next tick.
		if len(items) < o.batchSize {
			return
		}
	}
}

// publish sends one row and marks it published. A publish failure leaves the row
// unpublished for the next scan. A mark failure after a good publish is logged
// and tolerated: the row republishes next scan and the consumer deduplicates on
// event_id, so at-least-once holds.
func (o *OutboxRelayer) publish(ctx context.Context, item *model.OutboxItem) {
	ev, err := event.Decode(item.Topic, item.PartitionKey, item.Data)
	if err != nil {
		slog.Error("outbox item decode failed", "id", item.ID, "event_id", item.EventID, "error", err)
		return
	}
	if err := o.publisher.Publish(ctx, ev); err != nil {
		slog.Error("outbox item publish failed", "id", item.ID, "event_id", item.EventID, "error", err)
		return
	}
	if err := o.outboxStore.SetPublishedAt(ctx, item.ID, time.Now().UTC()); err != nil {
		slog.Error("outbox item mark-published failed", "id", item.ID, "event_id", item.EventID, "error", err)
	}
	slog.Debug("published event", "event_id", ev.ID, "payload_size", len(item.Data))
}

// Ping reports broker reachability, for the service health probe.
func (o *OutboxRelayer) Ping(ctx context.Context) error {
	return o.publisher.Ping(ctx)
}

// Stop terminates the poll loop and closes the publisher. It is idempotent and
// safe to call from multiple goroutines: close broadcasts to run(), wg.Wait
// blocks until the loop has returned. It never sends on quit, so it cannot panic
// on a second call.
func (o *OutboxRelayer) Stop() error {
	o.stopOnce.Do(func() { close(o.quit) })
	o.wg.Wait()
	slog.Debug("stopped relayer")
	return o.publisher.Close()
}
