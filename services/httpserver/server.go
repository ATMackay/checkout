package httpserver

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"sync/atomic"
	"time"

	srverrors "github.com/ATMackay/checkout/errors"
	"github.com/ATMackay/checkout/services"
	"github.com/ATMackay/checkout/services/httpserver/middleware"
	"github.com/julienschmidt/httprouter"
)

type HTTPServer struct {
	http.Server
	service services.Service

	started atomic.Bool
}

func New(port int, svc services.Service) *HTTPServer {
	return &HTTPServer{
		Server: http.Server{
			Addr:              fmt.Sprintf(":%d", port),
			ReadHeaderTimeout: 5 * time.Second,
		},
		service: svc,
	}
}

// Start boots the wrapped service's background work, then serves HTTP. It is the
// single lifecycle root for a node: the same call starts the domain service
// (e.g. the orders outbox relay) and the listener, so every service — orders
// today, the notifier next — gets identical wiring. A service start failure is
// returned and the listener is not started.
func (h *HTTPServer) Start(ctx context.Context) error {
	// Register handlers, wrapped with shared observability so every service on
	// this server is logged and metered uniformly.
	h.Handler = middleware.Observer(h.service.RegisterHandlers())
	// Boot the domain service's background processes before accepting traffic.
	if err := h.service.Start(ctx); err != nil {
		return err
	}
	// Start server in new goroutine
	go func() {
		h.started.Store(true)
		if err := h.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			slog.Warn("serviceTerminated", "error", err)
		}
	}()
	slog.Info("listening on port", "address", h.Port())
	return nil
}

// Stop gracefully shuts down the node: the HTTP listener first (so no new
// requests can enqueue work), then the domain service (drain background work).
func (h *HTTPServer) Stop() error {
	if !h.started.Load() {
		return nil
	}
	// Shutdown should not be called more than once
	h.started.Store(false)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := h.Shutdown(ctx); err != nil {
		return err
	}
	return h.service.Stop()
}

func (h *HTTPServer) Port() string {
	return h.Addr
}

// WriteJSON encodes payload as a JSON response with the given status code. It is
// exported for the handful of endpoints that must set a non-200 status on a
// success-shaped body (e.g. a health probe returning 503 with its report), which
// the Handle adapter deliberately cannot express.
func WriteJSON(w http.ResponseWriter, code int, payload any) error {
	var response []byte
	var err error
	if payload != nil {
		response, err = json.Marshal(payload)
		if err != nil {
			return err
		}
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	_, err = w.Write(response)
	return err
}

func respondWithError(w http.ResponseWriter, code int, err error) {
	if writeErr := WriteJSON(w, code, srverrors.JSONError{Error: err.Error()}); writeErr != nil {
		slog.Error("failed to write error response", "error", writeErr, "original_error", err)
	}
}

// APIHandler is a transport-decoupled handler: it reads the request and returns
// a payload or an error. It never touches the ResponseWriter or an HTTP status
// code — encoding and status selection are the adapter's job (handle), so the
// business logic stays reusable outside HTTP.
type APIHandler func(r *http.Request, p httprouter.Params) (any, error)

// handle adapts an apiHandler into an httprouter.Handle: encode the payload as
// 200 JSON, or map the error to a status and encode it. This and Health are the
// only places that write responses.
func Handle(h APIHandler) httprouter.Handle {
	return func(w http.ResponseWriter, r *http.Request, p httprouter.Params) {
		payload, err := h(r, p)
		if err != nil {
			respondWithError(w, statusFor(err), err)
			return
		}
		if err := WriteJSON(w, http.StatusOK, payload); err != nil {
			respondWithError(w, http.StatusInternalServerError, err)
		}
	}
}

// statusFor maps a domain error's category to an HTTP status. This is the ONLY
// place HTTP status codes are chosen; handlers return semantic errors and an
// unclassified error defaults to 500.
func statusFor(err error) int {
	switch {
	case errors.Is(err, srverrors.ErrInvalidInput):
		return http.StatusBadRequest
	case errors.Is(err, srverrors.ErrNotFound):
		return http.StatusNotFound
	default:
		return http.StatusInternalServerError
	}
}
