package server

import (
	"fmt"
	"net/http"

	"github.com/ATMackay/checkout/model"
	"github.com/julienschmidt/httprouter"
)

// Orders godoc
// @Summary Get list of purchase orders
// @Description List all purchase orders
// @Tags inventory
// @Produce json
// @Success 200 {object} Orders
// @Failure 400 {object} JSONError
// @Failure 404 {object} JSONError
// @Failure 503 {object} JSONError
// @Router /v0/orders [get]
func (h *HTTPServer) Orders() httprouter.Handle {
	return httprouter.Handle(func(w http.ResponseWriter, r *http.Request, _ httprouter.Params) {

		os, err := h.db.GetOrders(r.Context())
		if err != nil {
			respondWithError(w, http.StatusInternalServerError, fmt.Errorf("could not get orders from db: %w", err))
			return
		}
		var ors = make(model.Orders, len(os))
		for i, o := range os {
			skus, err := o.GetSKUList()
			if err != nil {
				respondWithError(w, http.StatusInternalServerError, fmt.Errorf("could not get sku list: %w", err))
				return
			}
			ors[i] = model.Order{Reference: o.Reference, SKUs: skus, Cost: o.Price}
		}
		if err := respondWithJSON(w, http.StatusOK, &ors); err != nil {
			respondWithError(w, http.StatusInternalServerError, err)
		}
	})

}
