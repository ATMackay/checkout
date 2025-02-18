package server

import (
	"fmt"
	"net/http"

	"github.com/ATMackay/checkout/constants"
	"github.com/julienschmidt/httprouter"
)

// StatusResponse contains status response fields.
type StatusResponse struct {
	Message string `json:"message,omitempty"`
	Version string `json:"version,omitempty"`
	Service string `json:"service,omitempty"`
}

// Status implements the status request endpoint. Always returns OK.
func Status() httprouter.Handle {
	return httprouter.Handle(func(w http.ResponseWriter, r *http.Request, _ httprouter.Params) {
		if err := respondWithJSON(w, http.StatusOK, &StatusResponse{Message: "OK", Version: constants.Version, Service: constants.ServiceName}); err != nil {
			respondWithError(w, http.StatusInternalServerError, err)
		}
	})

}

// HealthResponse contains health probe response fields.
type HealthResponse struct {
	Version  string   `json:"version,omitempty"`
	Service  string   `json:"service,omitempty"`
	Failures []string `json:"failures"`
}

// Health pings database clients. It ensures that the
// drivers are connected ready to accept requests.
func (h *HTTPServer) Health() httprouter.Handle {
	return httprouter.Handle(func(w http.ResponseWriter, r *http.Request, _ httprouter.Params) {
		health := &HealthResponse{
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
