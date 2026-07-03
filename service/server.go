// @title Checkout Service API
// @version 1.0
// @description API for managing inventory and orders
// @host localhost:8000
// @BasePath /
package service

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"sync/atomic"
	"time"

	"github.com/ATMackay/checkout/database"
	"github.com/ATMackay/checkout/promotions"
)

// Service executes business logic, handles requests, and provides access to the connected Database.
type Service struct {
	server           *http.Server
	db               database.Database
	promotionsEngine *promotions.PromotionsEngine

	authPassword string

	started atomic.Bool
}

// NewService returns a Service with httprouter Router
// handling requests.
func NewService(port int, db database.Database, authPasswd string) *Service {
	srv := &Service{
		server: &http.Server{
			Addr:              fmt.Sprintf(":%d", port),
			ReadHeaderTimeout: 5 * time.Second,
		},
		db: db,
		promotionsEngine: promotions.NewPromotionsEngine(
			promotions.NewMacBookProPromotion(db),
			&promotions.GoogleTVPromotion{},
			&promotions.AlexaSpeakerPromotion{}, // Add more deals/promotions to the engine
		),
		authPassword: authPasswd,
	}

	srv.registerHandlers()

	return srv
}

func (h *Service) registerHandlers() {

	handler := makeServiceAPI(h).routes()

	h.server.Handler = handler
}

func (h *Service) Addr() string {
	return h.server.Addr
}

// Start spawns the service which will listen on the TCP address srv.Addr
// for incoming requests.
func (h *Service) Start() {
	go func() {
		h.started.Store(true)
		if err := h.server.ListenAndServe(); err != nil {
			slog.Warn("serviceTerminated", "error", err)
		}
	}()
	slog.Info("listening on port", "address", h.Addr())
}

// Stop gracefully shuts down the HTTP service.
func (h *Service) Stop() error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	return h.server.Shutdown(ctx)
}
