package notifier

import (
	"context"
	"log/slog"
	"sync"

	"github.com/ATMackay/checkout/database"
	"github.com/ATMackay/checkout/messaging"
	"github.com/ATMackay/checkout/services/auth"
)

const ServiceName = "notifier service"

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

	quit     chan struct{}
	wg       sync.WaitGroup
	stopOnce sync.Once
}

// NewService constructs the notifier. The listening port is the httpserver's
// concern, not the service's.
func NewService(authn auth.Authenticator,
	store store,
	consumer messaging.Consumer,
) *Service {
	return &Service{authn: authn, store: store, consumer: consumer}
}

// Start launches the notifier consume loop
func (h *Service) Start(startCtx context.Context) error {
	slog.Info("notifier consumer started")
	// TODO
	return nil
}

// Stop cancels the consume loop and waits for it to exit. Idempotent and safe to
// call more than once.
func (h *Service) Stop() error {
	// TODO
	return nil
}
