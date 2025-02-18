package server

import (
	"encoding/json"
	"net/http"

	"github.com/julienschmidt/httprouter"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

type API struct {
	endpoints []endPoint
}

// endPoint represents an api element.
type endPoint struct {
	path       string
	handler    httprouter.Handle
	methodType string
}

func addEndpoints(endpoints []endPoint) *API {
	r := &API{}
	for _, e := range endpoints {
		r.addEndpoint(e)
	}
	return r
}

func (a *API) addEndpoint(e endPoint) {
	a.endpoints = append(a.endpoints, e)
}

// routes configures a new httprouter.Router.
func (a *API) routes() *httprouter.Router {

	router := httprouter.New()

	for _, e := range a.endpoints {
		router.Handle(e.methodType, e.path, logHTTPMiddleware(e.handler))
	}

	// Add metrics server
	router.Handler(http.MethodGet, "/metrics", promhttp.Handler())

	return router
}

func respondWithJSON(w http.ResponseWriter, code int, payload any) error {
	response, err := json.Marshal(payload)
	if err != nil {
		return err
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	_, err = w.Write(response)
	return err
}

func respondWithError(w http.ResponseWriter, code int, err error) {
	_ = respondWithJSON(w, code, JSONError{Error: err.Error()})
}

type JSONError struct {
	Error string `json:"error,omitempty"`
}
