// @title Checkout Orders Service API
// @version 1.0
// @description API for managing inventory and orders
// @host localhost:8000
// @BasePath /
package orders

import (
	"context"

	"github.com/ATMackay/checkout/database"
	"github.com/ATMackay/checkout/promotions"
	"github.com/ATMackay/checkout/services/auth"
)

// Service executes business logic covering order and inventory,
// handles requests, and provides access to the connected Database.
type Service struct {
	// Service attributed must be non-empty
	store            store
	promotionsEngine *promotions.PromotionsEngine
	relay            Relayer
	// authn resolves credentials for the service's protected routes. Injected
	// like any other dependency; the service knows which routes need it.
	authn auth.Authenticator
}

// store is the orders service's view of the database. Orders genuinely uses
// nearly all of it — inventory, orders, outbox, and cross-store transactions —
// so this composite is close to database.Database; that is honest, not a smell.
// The narrow-interface payoff shows up in the notifier, which needs only the
// outbox. Declaring it here (consumer-site) still documents the surface and
// keeps orders decoupled from the concrete GormDB.
type store interface {
	database.OrderStore
	database.InventoryStore
	database.OutboxStore
	database.HealthChecker
	// Transaction runs fn atomically; the callback receives a database.Database
	// so it can touch every store inside one transaction (see PurchaseItems).
	Transaction(ctx context.Context, fn func(database.Database) error) error
}

// NewService constructs the orders domain service. The listening port is not
// its concern — the httpserver that wraps it owns that.
func NewService(db store,
	relayer Relayer,
	authn auth.Authenticator,
) *Service {
	srv := &Service{
		store: db,
		promotionsEngine: promotions.NewPromotionsEngine(
			promotions.NewMacBookProPromotion(db),
			&promotions.GoogleTVPromotion{},
			&promotions.AlexaSpeakerPromotion{}, // Add more deals/promotions to the engine
		),
		relay: relayer, // Noop or Kafka
		authn: authn,
	}

	return srv
}

// Start boots the service's background processes (the outbox relay).
func (h *Service) Start(ctx context.Context) error {
	// Spawn dependent processes
	return h.relay.Start(ctx)
}

// Stop tears down the background processes started by Start.
func (h *Service) Stop() error {
	return h.relay.Stop()
}
