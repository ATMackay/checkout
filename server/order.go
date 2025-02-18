package server

import (
	"fmt"
	"net/http"

	"github.com/julienschmidt/httprouter"
)

type OrdersResponse []*Order

type Order struct {
	Reference string   `json:"reference"`
	SKUs      []string `json:"skus"`
	Cost      float64  `json:"cost"`
}

func (h *HTTPServer) Orders() httprouter.Handle {
	return httprouter.Handle(func(w http.ResponseWriter, r *http.Request, _ httprouter.Params) {

		os, err := h.db.GetOrders(r.Context())
		if err != nil {
			respondWithError(w, http.StatusInternalServerError, fmt.Errorf("could not get orders from db: %w", err))
			return
		}
		var ors = make(OrdersResponse, len(os))
		for i, o := range os {
			skus, err := o.GetSKUList()
			if err != nil {
				respondWithError(w, http.StatusInternalServerError, fmt.Errorf("could not get sku list: %w", err))
				return
			}
			ors[i] = &Order{Reference: o.Reference, SKUs: skus, Cost: o.Price}
		}
		if err := respondWithJSON(w, http.StatusOK, ors); err != nil {
			respondWithError(w, http.StatusInternalServerError, err)
		}
	})

}
