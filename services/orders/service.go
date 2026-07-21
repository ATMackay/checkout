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
	"github.com/julienschmidt/httprouter"
)

// Service executes business logic, handles requests, and provides access to the connected Database.
type Service struct {
	// Service attributed must be non-empty
	db               database.Database
	promotionsEngine *promotions.PromotionsEngine
	relay            Relayer
	// authn resolves credentials for the service's protected routes. Injected
	// like any other dependency; the service knows which routes need it.
	authn auth.Authenticator
}

// NewService constructs the orders domain service. The listening port is not
// its concern — the httpserver that wraps it owns that.
func NewService(db database.Database, relayer Relayer, authn auth.Authenticator) *Service {
	srv := &Service{
		db: db,
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

func (h *Service) RegisterHandlers() *httprouter.Router {
	return makeServiceAPI(h).routes()
}
