package orders

import (
	"net/http"

	"github.com/ATMackay/checkout/constants"
	"github.com/ATMackay/checkout/httpserver"
	api "github.com/ATMackay/checkout/httpserver/api"
	"github.com/ATMackay/checkout/httpserver/middleware"
	"github.com/julienschmidt/httprouter"
)

const ServiceName = "orders service"

// TopicOrderCreated carries an event per completed purchase order.
const TopicOrderCreated = "orders.created"

var (
	ItemsEndPnt        = "/v1/inventory/items"
	ItemPriceEndPnt    = "/v1/inventory/item/price"
	ItemsPriceEndPnt   = "/v1/inventory/items/price"
	ItemPurchaseEndPnt = "/v1/inventory/items/purchase"
	KeyParam           = "/:key"

	OrdersEndPnt = "/v1/orders"
)

func (h *Service) RegisterHandlers() *httprouter.Router {
	return api.AddEndpoints([]api.EndPoint{
		// Liveness/Readiness probing — mechanism shared via httpserver; this
		// service supplies only its own checks.
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
				httpserver.Check{Name: "broker", Probe: h.relay.Ping},
			),
		},
		//
		// Checkout Application HTTP API
		//
		{
			Path:       ItemsEndPnt, // Add new items to the inventory item table
			MethodType: http.MethodGet,
			Handler:    h.ListItems(),
		},
		{
			Path:       ItemPriceEndPnt + KeyParam, // Price for single item
			MethodType: http.MethodGet,
			Handler:    h.ItemPrice(),
		},
		{
			Path:       ItemPriceEndPnt, // Alternative total price for items batch
			MethodType: http.MethodPost,
			Handler:    h.ItemsPrice(),
		},
		// Authenticated requests
		{
			Path:       ItemPurchaseEndPnt, // Execute purchase order (records the buyer)
			MethodType: http.MethodPost,
			Handler:    middleware.Auth(h.authn)(h.PurchaseItems()),
		},
		{
			Path:       OrdersEndPnt, // List the authenticated customer's orders
			MethodType: http.MethodGet,
			Handler:    middleware.Auth(h.authn)(h.Orders()),
		},
		{
			Path:       ItemsEndPnt, // Add items to the inventory item table
			MethodType: http.MethodPost,
			Handler:    middleware.Auth(h.authn)(h.AddItems()),
		},
	}).Routes()
}
