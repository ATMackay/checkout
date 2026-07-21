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
	"github.com/julienschmidt/httprouter"
)

// Service executes business logic, handles requests, and provides access to the connected Database.
type Service struct {
	// Service attributed must be non-empty
	db               database.Database
	promotionsEngine *promotions.PromotionsEngine
	relay            Relayer
	// Basic Auth
	authPassword string
}

// NewService constructs the orders domain service. The listening port is not
// its concern — the httpserver that wraps it owns that.
func NewService(db database.Database, authPasswd string, relayer Relayer) *Service {
	srv := &Service{
		db: db,
		promotionsEngine: promotions.NewPromotionsEngine(
			promotions.NewMacBookProPromotion(db),
			&promotions.GoogleTVPromotion{},
			&promotions.AlexaSpeakerPromotion{}, // Add more deals/promotions to the engine
		),
		relay:        relayer,    // Noop or Kafka
		authPassword: authPasswd, // TODO - Strengthen Auth system
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
