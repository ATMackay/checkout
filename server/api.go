package server

import (
	"encoding/json"
	"net/http"

	"github.com/julienschmidt/httprouter"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

var (
	StatusEndPnt = "/status"
	HealthEndPnt = "/health"

	ItemsEndPnt        = "/v0/inventory/items"
	ItemPriceEndPnt    = "/v0/inventory/item/price"
	ItemsPriceEndPnt   = "/v0/inventory/items/price"
	ItemPurchaseEndPnt = "/v0/inventory/items/purchase"
	KeyParam           = "/:key"

	OrdersEndPnt = "/v0/orders"
)

// API is a collection of endpoints.
type API struct {
	endpoints []endPoint
}

// makeServerAPI returns a server http API with endpoints
func makeServerAPI(h *Server) *API {
	return addEndpoints([]endPoint{
		// Liveness/Readiness probing
		{
			path:       StatusEndPnt,
			methodType: http.MethodGet,
			handler:    Status(),
		},
		{
			path:       HealthEndPnt,
			methodType: http.MethodGet,
			handler:    h.Health(),
		},
		//
		// Checkout Application HTTP API
		//
		{
			path:       ItemsEndPnt, // Add new items to the inventory item table
			methodType: http.MethodGet,
			handler:    h.ListItems(),
		},
		{
			path:       ItemPriceEndPnt + KeyParam, // Price for single item
			methodType: http.MethodGet,
			handler:    h.ItemPrice(),
		},
		{
			path:       ItemPriceEndPnt, // Alternative total price for items batch
			methodType: http.MethodPost,
			handler:    h.ItemsPrice(),
		},
		{
			path:       ItemPurchaseEndPnt, // Execute purchase order
			methodType: http.MethodPost,
			handler:    h.PurchaseItems(),
		},
		// Authenticated requests
		{
			path:       OrdersEndPnt, // Add new items to the inventory item table
			methodType: http.MethodGet,
			handler:    h.authMiddleware(h.Orders()),
		},
		{
			path:       ItemsEndPnt, // Add new items to the inventory item table
			methodType: http.MethodPost,
			handler:    h.authMiddleware(h.AddItems()),
		},
	},
	)
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
		router.Handle(e.methodType, e.path, observerMiddleware(e.handler))
	}

	// Add metrics server
	router.Handler(http.MethodGet, "/metrics", promhttp.Handler())

	return router
}

func respondWithJSON(w http.ResponseWriter, code int, payload any) error {
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
	_ = respondWithJSON(w, code, JSONError{Error: err.Error()})
}

type JSONError struct {
	Error string `json:"error,omitempty"`
}
