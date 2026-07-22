package services

import (
	"net/http"

	"github.com/julienschmidt/httprouter"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

// API is a collection of endpoints.
type API struct {
	endpoints []EndPoint
	// includemetrics bool
}

// EndPoint represents an api element.
type EndPoint struct {
	Path       string
	Handler    httprouter.Handle
	MethodType string
}

func AddEndpoints(endpoints []EndPoint) *API {
	r := &API{}
	r.endpoints = append(r.endpoints, endpoints...)
	return r
}

// routes configures a new httprouter.Router.
func (a *API) Routes() *httprouter.Router {

	router := httprouter.New()

	// Observability is applied by the httpserver around the whole router, so
	// handlers register unwrapped here.
	for _, e := range a.endpoints {
		router.Handle(e.MethodType, e.Path, e.Handler)
	}

	// Add metrics server
	router.Handler(http.MethodGet, "/metrics", promhttp.Handler())

	return router
}
