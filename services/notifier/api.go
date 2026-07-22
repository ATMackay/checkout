package notifier

import (
	"net/http"

	"github.com/ATMackay/checkout/constants"
	"github.com/ATMackay/checkout/httpserver"
	api "github.com/ATMackay/checkout/httpserver/api"
	"github.com/julienschmidt/httprouter"
)

var (
	StatusEndPnt = "/status"
	HealthEndPnt = "/health"
)

// RegisterHandlers exposes only liveness/readiness — the notifier has no domain
// REST API; its work happens in the consume loop (see Start). The probe
// mechanism is shared via httpserver; the notifier supplies its own checks.
func (h *Service) RegisterHandlers() *httprouter.Router {
	return api.AddEndpoints([]api.EndPoint{
		{
			Path:       StatusEndPnt,
			MethodType: http.MethodGet,
			Handler:    httpserver.StatusHandler(ServiceName, constants.Version),
		},
		{
			Path:       HealthEndPnt,
			MethodType: http.MethodGet,
			Handler: httpserver.HealthHandler(ServiceName, constants.Version,
				httpserver.Check{Name: "database", Probe: h.store.Ping},
				httpserver.Check{Name: "consumer", Probe: h.consumer.Ping},
			),
		},
	}).Routes()
}
