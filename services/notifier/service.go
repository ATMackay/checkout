package notifier

import (
	"context"
	"log/slog"

	"github.com/ATMackay/checkout/database"
	"github.com/ATMackay/checkout/messaging"
	"github.com/ATMackay/checkout/services/auth"
	"github.com/ATMackay/checkout/services/worker"
)

const ServiceName = "notifier service"

// ConsumerGroup is the Kafka consumer group the notifier joins. All notifier
// replicas share it, so the broker splits partitions among them.
const ConsumerGroup = "notifier"

// store is the notifier's view of the database: only the outbox (to mark events
// delivered) and a health probe. This narrow interface is where splitting the
// stores actually pays off — unlike orders, the notifier touches almost none of
// the DB, and the single GormDB satisfies this face of it.
type store interface {
	database.OutboxStore
	database.HealthChecker
}

// Service consumes order events and (eventually) dispatches notifications. It is
// a background consumer with health/status HTTP endpoints and no domain REST
// API — the mirror image of the orders relay: relay drains outbox → broker;
// notifier consumes broker → processes → marks delivered.
type Service struct {
	authn    auth.Authenticator
	store    store
	consumer messaging.Consumer
	runner   worker.Runner
}

// NewService constructs the notifier. The listening port is the httpserver's
// concern, not the service's.
func NewService(authn auth.Authenticator,
	store store,
	consumer messaging.Consumer,
) *Service {
	return &Service{authn: authn, store: store, consumer: consumer}
}

// Start pings the broker, then launches the consume loop. Teardown is via
// context cancellation (worker.Runner): Poll blocks until events arrive, so only
// cancelling its context can interrupt it promptly.
func (h *Service) Start(startCtx context.Context) error {
	if err := h.consumer.Ping(startCtx); err != nil {
		return err
	}
	h.runner.Start(h.consume)
	slog.Info("notifier consumer started")
	return nil
}

// consume polls the broker until its context is cancelled by Stop.
func (h *Service) consume(ctx context.Context) {
	for ctx.Err() == nil {
		events, err := h.consumer.Poll(ctx)
		if err != nil {
			if ctx.Err() != nil {
				return // cancelled during Stop
			}
			slog.Error("notifier poll failed", "error", err)
			continue
		}
		for _, ev := range events {
			// TODO — the notifier's domain logic lives here: dedup on ev.ID,
			// dispatch the notification, then mark the outbox row delivered via
			// h.store.SetDeliveredAt. For now we only observe the stream.
			slog.Debug("notifier received event", "id", ev.ID, "topic", ev.Topic, "key", ev.Key)
		}
		// Commit only after processing — at-least-once: a crash before commit
		// redelivers, which the (future) dedup step absorbs.
		if err := h.consumer.Commit(ctx); err != nil {
			slog.Error("notifier commit failed", "error", err)
		}
	}
}

// Stop cancels the consume loop and waits for it to exit. Idempotent.
func (h *Service) Stop() error {
	h.runner.Stop()
	return nil
}
