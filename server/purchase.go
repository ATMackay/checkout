package server

import (
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/ATMackay/checkout/model"
	"github.com/julienschmidt/httprouter"
)

// PurchaseItems godoc
// @Summary Execute a purchase for the supplied item list.
// @Description Create a purchase order for the supplied item list.
// @Tags inventory
// @Accept json
// @Produce json
// @Param skus body PurchaseItemsRequest true "List of SKUs"
// @Success 200 {object} PurchaseItemsResponse
// @Failure 400 {object} JSONError
// @Failure 404 {object} JSONError
// @Failure 503 {object} JSONError
// @Router /v0/inventory/items/purchase [post]
func (h *HTTPServer) PurchaseItems() httprouter.Handle {
	return httprouter.Handle(func(w http.ResponseWriter, r *http.Request, _ httprouter.Params) {

		ctx := r.Context()

		var pReq model.PurchaseItemsRequest

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

		// Fetch items from DB
		dbItems, err := h.db.GetItemsBySKU(ctx, pReq.SKUs)
		if err != nil {
			respondWithError(w, http.StatusBadRequest, fmt.Errorf("could not get items: %w", err))
			return
		}

		dbItemMap := make(map[string]*model.Item)

		items := []*model.Item{}
		var total float64
		for _, dbIt := range dbItems {
			dbItemMap[dbIt.SKU] = dbIt
			items = append(items, dbIt)
		}

		itemCount := make(map[string]int)
		for _, sku := range pReq.SKUs {
			itemCount[sku]++
			it := dbItemMap[sku]
			if it.InventoryQuantity < itemCount[it.SKU] {
				respondWithError(w, http.StatusNotFound, fmt.Errorf("item %s empty", it.SKU))
				return
			}
			total += it.Price
			// deduct inventory
			it.InventoryQuantity--
		}

		skus := pReq.SKUs

		promotions := applyPromotions(items)

		for _, it := range promotions.AddedItems {
			sku := it.SKU
			itemCount[sku]++
			dbIt, err := h.db.GetItemBySKU(ctx, sku)
			if err != nil {
				respondWithError(w, http.StatusBadRequest, fmt.Errorf("could not get item: %w", err))
				return
			}
			if dbIt.InventoryQuantity < itemCount[sku] {
				// Skip if we cannot add
				continue
			}
			dbIt.InventoryQuantity--
			items = append(items, dbIt)
			skus = append(pReq.SKUs, sku)
		}

		price := total - promotions.Deduction

		// Create order
		order := &model.Order{Price: price, Reference: model.GenerateReference()}
		if err := order.SetSKUList(skus); err != nil {
			respondWithError(w, http.StatusInternalServerError, err)
			return
		}
		if err := h.db.AddOrder(ctx, order); err != nil {
			respondWithError(w, http.StatusInternalServerError, err)
			return
		}

		// Save updated dbItems with new inventory totals
		if _, err := h.db.UpsertItems(ctx, items); err != nil {
			respondWithError(w, http.StatusInternalServerError, err)
			return
		}

		if err := respondWithJSON(w, http.StatusOK, &model.PurchaseItemsResponse{OrderReference: order.Reference, Cost: price}); err != nil {
			respondWithError(w, http.StatusInternalServerError, err)
		}
	})
}
