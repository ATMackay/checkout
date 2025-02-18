package server

import (
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/ATMackay/checkout/database"
	"github.com/julienschmidt/httprouter"
)

// TODO

type PurchaseItemsRequest struct {
	SKUs []string `json:"skus"`
}

type PurchaseItemsResponse struct {
	OrderReference string  `json:"order_reference"`
	Cost           float64 `json:"cost"`
}

func (h *HTTPServer) PurchaseItems() httprouter.Handle {
	return httprouter.Handle(func(w http.ResponseWriter, r *http.Request, _ httprouter.Params) {

		ctx := r.Context()

		var pReq PurchaseItemsRequest

		if err := json.NewDecoder(r.Body).Decode(&pReq); err != nil {
			respondWithError(w, http.StatusBadRequest, err)
			return
		}

		// validate request params
		for _, sku := range pReq.SKUs {
			if !isSKU(sku) {
				respondWithError(w, http.StatusBadRequest, fmt.Errorf("invalid sku input '%s'", sku))
				return
			}
		}

		dbItems, err := h.db.GetItemsBySKU(ctx, pReq.SKUs)
		if err != nil {
			respondWithError(w, http.StatusBadRequest, fmt.Errorf("could not get items: %w", err))
			return
		}

		skus := []string{}
		items := []*Item{}
		var total float64
		for _, it := range dbItems {

			if it.InventoryQuantity < 1 {
				respondWithError(w, http.StatusNotFound, fmt.Errorf("item %s empty", it.SKU))
				return
			}

			items = append(items, &Item{
				Name:  it.Name,
				SKU:   it.SKU,
				Price: it.Price,
			})
			skus = append(skus, it.SKU)
		}

		// TODO - apply promotions
		promotions := applyPromotions(items)

		price := total - promotions.Deduction

		// Execute order
		order := &database.Order{Price: price, Reference: database.GenerateReference()}
		if err := order.SetSKUList(skus); err != nil {
			respondWithError(w, http.StatusInternalServerError, err)
			return
		}
		if err := h.db.AddOrder(ctx, order); err != nil {
			respondWithError(w, http.StatusInternalServerError, err)
			return
		}

		// Deduct
		if err := respondWithJSON(w, http.StatusOK, &PurchaseItemsResponse{OrderReference: order.Reference, Cost: price}); err != nil {
			respondWithError(w, http.StatusInternalServerError, err)
		}
	})
}
