package notifier

import (
	"net/http"

	"github.com/ATMackay/checkout/constants"
	"github.com/ATMackay/checkout/httpserver"
	api "github.com/ATMackay/checkout/httpserver/api"
	"github.com/ATMackay/checkout/httpserver/middleware"
	"github.com/julienschmidt/httprouter"
)

var (
	NotificationsEndPnt = "/v1/notifications"
)

// RegisterHandlers exposes liveness/readiness and the notifications view. The
// notifier has no write API; its work happens in the consume loop (see Start).
// The probe mechanism is shared via httpserver; the notifier supplies its checks.
func (h *Service) RegisterHandlers() *httprouter.Router {
	return api.AddEndpoints([]api.EndPoint{
		{
			Path:       httpserver.StatusEndPnt,
			MethodType: http.MethodGet,
			Handler:    httpserver.StatusHandler(ServiceName, constants.Version),
		},
		{
			Path:       httpserver.HealthEndPnt,
			MethodType: http.MethodGet,
			Handler: httpserver.HealthHandler(ServiceName, constants.Version,
				httpserver.Check{Name: "database", Probe: h.store.Ping},
				httpserver.Check{Name: "consumer", Probe: h.consumer.Ping},
			),
		},
		{
			Path:       NotificationsEndPnt,
			MethodType: http.MethodGet,
			Handler:    middleware.Auth(h.authn)(h.Notifications()),
		},
	}).Routes()
}
