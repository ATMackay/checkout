package server

import (
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/ATMackay/checkout/database"
	"github.com/julienschmidt/httprouter"
)

type PriceResponse struct {
	Items             []*Item     `json:"items"`
	Promotions        *Promotions `json:"promotions,omitempty"`
	TotalGross        float64     `json:"total_gross"`
	TotalWithDiscount float64     `json:"total_with_discount"`
}

type Item struct {
	Name  string  `json:"name"`
	SKU   string  `json:"sku"`
	Price float64 `json:"price"`
}

func (h *HTTPServer) PriceItem() httprouter.Handle {
	return httprouter.Handle(func(w http.ResponseWriter, r *http.Request, p httprouter.Params) {

		ctx := r.Context()
		nameOrSku := p.ByName("key")
		var dbItem *database.InventoryItem
		var err error

		if isSKU(nameOrSku) {
			dbItem, err = h.db.GetItemBySKU(ctx, nameOrSku)
		} else {
			dbItem, err = h.db.GetItemByName(ctx, nameOrSku)
		}
		if err != nil {
			respondWithError(w, http.StatusBadRequest, fmt.Errorf("could not get item with key '%s' :%w", nameOrSku, err))
			return
		}
		if dbItem.InventoryQuantity < 1 {
			respondWithError(w, http.StatusNotFound, fmt.Errorf("item %s empty", dbItem.SKU))
			return
		}

		if err := respondWithJSON(w, http.StatusOK, &PriceResponse{
			Items:             []*Item{{Name: dbItem.Name, SKU: dbItem.SKU, Price: dbItem.Price}},
			TotalGross:        dbItem.Price,
			TotalWithDiscount: dbItem.Price,
		}); err != nil {
			respondWithError(w, http.StatusInternalServerError, err)
		}
	})
}

type PriceItemsRequest struct {
	SKUs []string `json:"skus"`
}

func (h *HTTPServer) PriceItems() httprouter.Handle {
	return httprouter.Handle(func(w http.ResponseWriter, r *http.Request, _ httprouter.Params) {

		ctx := r.Context()

		var pReq PriceItemsRequest

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

		resp := &PriceResponse{}
		var total float64
		for _, it := range dbItems {

			if it.InventoryQuantity < 1 {
				respondWithError(w, http.StatusNotFound, fmt.Errorf("item %s empty", it.SKU))
				return
			}
			// TODO refactor to check inventory of promoted..

			resp.Items = append(resp.Items, &Item{
				Name:  it.Name,
				SKU:   it.SKU,
				Price: it.Price,
			})
			total += it.Price
		}

		// TODO - apply promotions
		promotions := applyPromotions(resp.Items)

		resp.Promotions = promotions
		resp.TotalGross = total
		resp.TotalWithDiscount = total - promotions.Deduction

		if err := respondWithJSON(w, http.StatusOK, resp); err != nil {
			respondWithError(w, http.StatusInternalServerError, err)
		}
	})
}
