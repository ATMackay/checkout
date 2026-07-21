package orders

import (
	"fmt"
	"log/slog"
	"net/http"

	"github.com/ATMackay/checkout/constants"
	"github.com/ATMackay/checkout/model"
	"github.com/ATMackay/checkout/services/httpserver"
	"github.com/julienschmidt/httprouter"
)

// Status godoc
// @Summary Get service status
// @Description Returns the status of the service
// @Tags status
// @Produce json
// @Success 200 {object} model.StatusResponse
// @Failure 500 {object} errors.JSONError
// @Router /status [get]
func Status() httprouter.Handle {
	return httpserver.Handle(func(*http.Request, httprouter.Params) (any, error) {
		return &model.StatusResponse{Message: "OK", Version: constants.Version, Service: ServiceName}, nil
	})
}

// Health godoc
// @Summary Get service health
// @Description Checks the health of the service and its dependencies.
// @Tags health
// @Produce json
// @Success 200 {object} model.HealthResponse
// @Failure 503 {object} model.HealthResponse
// @Failure 500 {object} errors.JSONError
// @Router /health [get]
// Health is bespoke rather than routed through httpserver.Handle: it returns a
// full report body with a 503 status when unhealthy, which the Handle adapter
// (payload OR error) cannot express. It writes directly via httpserver.WriteJSON.
func (h *Service) Health() httprouter.Handle {
	return func(w http.ResponseWriter, r *http.Request, _ httprouter.Params) {
		health := &model.HealthResponse{
			Service: ServiceName,
			Version: constants.Version,
		}
		failures := []string{}
		// Check database connection
		if err := h.db.Ping(r.Context()); err != nil {
			failures = append(failures, fmt.Sprintf("db Ping error: %v", err))
		}
		// Ping event relay backend
		if err := h.relay.Ping(r.Context()); err != nil {
			failures = append(failures, fmt.Sprintf("event bus Ping error: %v", err))
		}
		health.Failures = failures

		code := http.StatusOK
		if len(failures) > 0 {
			code = http.StatusServiceUnavailable
		}
		if err := httpserver.WriteJSON(w, code, health); err != nil {
			slog.Error("failed to write health response", "error", err)
		}
	}
}
