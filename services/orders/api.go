package orders

import (
	"net/http"

	"github.com/ATMackay/checkout/services/httpserver/middleware"
	"github.com/julienschmidt/httprouter"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

const ServiceName = "orders service"

// TopicOrderCreated carries an event per completed purchase order.
const TopicOrderCreated = "orders.created"

var (
	StatusEndPnt = "/status"
	HealthEndPnt = "/health"

	ItemsEndPnt        = "/v1/inventory/items"
	ItemPriceEndPnt    = "/v1/inventory/item/price"
	ItemsPriceEndPnt   = "/v1/inventory/items/price"
	ItemPurchaseEndPnt = "/v1/inventory/items/purchase"
	KeyParam           = "/:key"

	OrdersEndPnt = "/v1/orders"
)

// API is a collection of endpoints.
type API struct {
	endpoints []endPoint
}

// makeServerAPI returns a service http API with endpoints
func makeServiceAPI(h *Service) *API {
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
		// Authenticated requests
		{
			path:       ItemPurchaseEndPnt, // Execute purchase order (records the buyer)
			methodType: http.MethodPost,
			handler:    middleware.Auth(h.authn)(h.PurchaseItems()),
		},
		{
			path:       OrdersEndPnt, // List the authenticated customer's orders
			methodType: http.MethodGet,
			handler:    middleware.Auth(h.authn)(h.Orders()),
		},
		{
			path:       ItemsEndPnt, // Add new items to the inventory item table
			methodType: http.MethodPost,
			handler:    middleware.Auth(h.authn)(h.AddItems()),
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
	r.endpoints = append(r.endpoints, endpoints...)
	return r
}

// routes configures a new httprouter.Router.
func (a *API) routes() *httprouter.Router {

	router := httprouter.New()

	// Observability is applied by the httpserver around the whole router, so
	// handlers register unwrapped here.
	for _, e := range a.endpoints {
		router.Handle(e.methodType, e.path, e.handler)
	}

	// Add metrics server
	router.Handler(http.MethodGet, "/metrics", promhttp.Handler())

	return router
}
