package server

import (
	"context"
	"fmt"
	"net/http"
	"sync/atomic"
	"time"

	"github.com/ATMackay/checkout/database"

	"github.com/sirupsen/logrus"
)

type HTTPServer struct {
	server *http.Server
	log    logrus.FieldLogger
	db     database.Database

	authPassword string

	started atomic.Bool
}

// NewHTTPServer returns a HTTP server with httprouter Router
// handling requests.
func NewHTTPServer(port int, l logrus.FieldLogger, db database.Database, authPasswd string) *HTTPServer {

	srv := &HTTPServer{
		server: &http.Server{
			Addr:              fmt.Sprintf(":%d", port),
			ReadHeaderTimeout: 5 * time.Second,
		},
		db:           db,
		log:          l,
		authPassword: authPasswd,
		started:      atomic.Bool{},
	}

	srv = srv.RegisterHandlers()

	return srv
}

func (h *HTTPServer) RegisterHandlers() *HTTPServer {

	handler := h.MakeServerAPI(h.db).routes()

	h.server.Handler = handler
	return h
}

func (h *HTTPServer) Addr() string {
	return h.server.Addr
}

// Start spawns the server which will listen on the TCP address srv.Addr
// for incoming requests.
func (h *HTTPServer) Start() {
	go func() {
		h.started.Store(true)
		if err := h.server.ListenAndServe(); err != nil {
			h.log.WithFields(logrus.Fields{"error": err}).Warn("serverTerminated")
		}
	}()
}

func (h *HTTPServer) Stop() error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	return h.server.Shutdown(ctx)
}

// MakeServerAPI returns a server http API with endpoints
func (h *HTTPServer) MakeServerAPI(db database.Database) *API {
	return addEndpoints([]endPoint{
		// Liveness/Readiness probing
		{
			path:       "/status",
			methodType: http.MethodGet,
			handler:    Status(),
		},
		{
			path:       "/health",
			methodType: http.MethodGet,
			handler:    h.Health(),
		},
		//
		// Checkout Application HTTP API
		//
		{
			path:       "/v0/inventory/item/price/:key", // Price for single item
			methodType: http.MethodGet,
			handler:    h.PriceItem(),
		},
		{
			path:       "/v0/inventory/items/price", // Alternative total price for items batch
			methodType: http.MethodPost,
			handler:    h.PriceItems(),
		},
		{
			path:       "/v0/inventory/items/purchase", // Execute purchase order - TODO
			methodType: http.MethodPost,
			handler:    h.PurchaseItems(),
		},
		// Authenticated requests - TODO
		/*
			{
				path:       "/v0/orders", // Add new items to the inventory item table  - TODO
				methodType: http.MethodGet,
				handler:    h.authMiddleware(h.Orders()),
			},
			{
				path:       "/v0/inventory/items", // Add new items to the inventory item table  - TODO
				methodType: http.MethodPost,
				handler:    h.authMiddleware(h.Addtems()),
			},
		*/
	},
	)
}
