package server

import (
	"fmt"
	"net/http"

	"github.com/ATMackay/checkout/constants"
	"github.com/ATMackay/checkout/model"
	"github.com/julienschmidt/httprouter"
)

// Status godoc
// @Summary Get service status
// @Description Returns the status of the service
// @Tags status
// @Produce json
// @Success 200 {object} StatusResponse
// @Failure 500 {object} JSONError
// @Router /status [get]
func Status() httprouter.Handle {
	return httprouter.Handle(func(w http.ResponseWriter, r *http.Request, _ httprouter.Params) {
		// Fixed response never errors
		_ = respondWithJSON(w, http.StatusOK, &model.StatusResponse{Message: "OK", Version: constants.Version, Service: constants.ServiceName})
	})
}

// Health godoc
// @Summary Get service health
// @Description Checks the health of the service and its dependencies.
// @Tags health
// @Produce json
// @Success 200 {object} HealthResponse
// @Failure 503 {object} HealthResponse
// @Failure 500 {object} JSONError
// @Router /health [get]
func (h *Server) Health() httprouter.Handle {
	return httprouter.Handle(func(w http.ResponseWriter, r *http.Request, _ httprouter.Params) {
		health := &model.HealthResponse{
			Service: constants.ServiceName,
			Version: constants.Version,
		}
		var failures = []string{}
		var httpCode = http.StatusOK

		if err := h.db.Ping(r.Context()); err != nil {
			failures = append(failures, fmt.Sprintf("db Ping error: %v", err))
		}

		health.Failures = failures

		if len(health.Failures) > 0 {
			httpCode = http.StatusServiceUnavailable
		}

		if err := respondWithJSON(w, httpCode, health); err != nil {
			respondWithError(w, http.StatusInternalServerError, err)
		}
	})
}
