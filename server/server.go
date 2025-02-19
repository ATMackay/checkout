// @title Checkout Service API
// @version 1.0
// @description API for managing inventory and orders
// @host localhost:8000
// @BasePath /
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

// HTTPServer handles requests and access to the connected Database.
type HTTPServer struct {
	server *http.Server
	log    logrus.FieldLogger
	db     database.Database

	authPassword string

	started atomic.Bool
}

// NewHTTPServer returns a HTTPServer with httprouter Router
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

	srv = srv.registerHandlers()

	return srv
}

func (h *HTTPServer) registerHandlers() *HTTPServer {

	handler := MakeServerAPI(h).routes()

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
	h.log.Infof("listening on port %v", h.Addr())
}

// Stop gracefully shuts down the HTTP server.
func (h *HTTPServer) Stop() error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	return h.server.Shutdown(ctx)
}
