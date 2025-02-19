package server

import (
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/ATMackay/checkout/model"
	"github.com/julienschmidt/httprouter"
)

// AddItems godoc
// @Summary Add new or updated items to the inventory table
// @Description Add new or updated items
// @Tags inventory
// @Accept json
// @Produce json
// @Param skus body AddItemsRequest true "List of Items"
// @Success 200
// @Failure 400 {object} JSONError
// @Failure 404 {object} JSONError
// @Failure 503 {object} JSONError
// @Router /v0/inventory/items [post]
func (h *Server) AddItems() httprouter.Handle {
	return httprouter.Handle(func(w http.ResponseWriter, r *http.Request, p httprouter.Params) {

		var iReq model.AddItemsRequest

		if err := json.NewDecoder(r.Body).Decode(&iReq); err != nil {
			respondWithError(w, http.StatusBadRequest, err)
			return
		}

		if len(iReq.Items) < 1 {
			respondWithError(w, http.StatusBadRequest, fmt.Errorf("no items provided"))
			return
		}

		// validate
		for i, it := range iReq.Items {
			if err := it.Validate(); err != nil {
				respondWithError(w, http.StatusBadRequest, fmt.Errorf("item at index %d was invalid: %w", i, err))
				return
			}

		}

		its, err := h.db.UpsertItems(r.Context(), iReq.Items)
		if err != nil {
			respondWithError(w, http.StatusInternalServerError, err)
			return
		}

		if err := respondWithJSON(w, http.StatusOK, its); err != nil {
			respondWithError(w, http.StatusInternalServerError, err)
		}
	})
}

// ItemPrice godoc
// @Summary Get price for a single item
// @Description Get price information for a single item by SKU or name
// @Tags inventory
// @Produce json
// @Param key path string true "Item SKU or Name"
// @Success 200 {object} PriceResponse
// @Failure 400 {object} JSONError
// @Failure 404 {object} JSONError
// @Router /v0/inventory/item/price/{key} [get]
func (h *Server) ItemPrice() httprouter.Handle {
	return httprouter.Handle(func(w http.ResponseWriter, r *http.Request, p httprouter.Params) {

		ctx := r.Context()
		nameOrSku := p.ByName("key")
		var dbItem *model.Item
		var err error

		if model.IsSKU(nameOrSku) {
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

		if err := respondWithJSON(w, http.StatusOK, &model.PriceResponse{
			Items:             []*model.Item{{Name: dbItem.Name, SKU: dbItem.SKU, Price: dbItem.Price}},
			TotalGross:        dbItem.Price,
			TotalWithDiscount: dbItem.Price,
		}); err != nil {
			respondWithError(w, http.StatusInternalServerError, err)
		}
	})
}

// ItemsPrice godoc
// @Summary Get prices for multiple items
// @Description Get total price for a batch of items by SKUs
// @Tags inventory
// @Accept json
// @Produce json
// @Param skus body PriceItemsRequest true "List of SKUs"
// @Success 200 {object} PriceResponse
// @Failure 400 {object} JSONError
// @Failure 404 {object} JSONError
// @Failure 503 {object} JSONError
// @Router /v0/inventory/items/price [post]
func (h *Server) ItemsPrice() httprouter.Handle {
	return httprouter.Handle(func(w http.ResponseWriter, r *http.Request, _ httprouter.Params) {

		ctx := r.Context()

		var pReq model.ItemsPriceRequest

		if err := json.NewDecoder(r.Body).Decode(&pReq); err != nil {
			respondWithError(w, http.StatusBadRequest, err)
			return
		}

		// validate request params
		for _, sku := range pReq.SKUs {
			if !model.IsSKU(sku) {
				respondWithError(w, http.StatusBadRequest, fmt.Errorf("invalid sku input '%s'", sku))
				return
			}
		}

		dbItems, err := h.db.GetItemsBySKU(ctx, pReq.SKUs)
		if err != nil {
			respondWithError(w, http.StatusBadRequest, fmt.Errorf("could not get items: %w", err))
			return
		}

		resp := &model.PriceResponse{}
		var total float64
		for _, it := range dbItems {
			if it.InventoryQuantity < 1 {
				respondWithError(w, http.StatusNotFound, fmt.Errorf("item %s empty", it.SKU))
				return
			}
			resp.Items = append(resp.Items, it)
			total += it.Price
		}

		promotions, err := h.promotionsEngine.ApplyPromotions(resp.Items)
		if err != nil {
			respondWithError(w, http.StatusInternalServerError, fmt.Errorf("could not apply promotion/deals: %w", err))
			return
		}

		resp.Promotions = promotions
		resp.TotalGross = total
		resp.TotalWithDiscount = total - promotions.Deduction

		if err := respondWithJSON(w, http.StatusOK, resp); err != nil {
			respondWithError(w, http.StatusInternalServerError, err)
		}
	})
}
