package server

import (
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/ATMackay/checkout/database"
	"github.com/ATMackay/checkout/model"
	"github.com/julienschmidt/httprouter"
	"github.com/shopspring/decimal"
)

// PurchaseItems godoc
// @Summary Execute a purchase for the supplied item list.
// @Description Create a purchase order for the supplied item list.
// @Tags inventory
// @Accept json
// @Produce json
// @Param   request  body    model.PurchaseItemsResponse  true  "List of SKUs"
// @Success 200 {object} model.PurchaseItemsResponse
// @Failure 400 {object} JSONError
// @Failure 404 {object} JSONError
// @Failure 503 {object} JSONError
// @Router /v0/inventory/items/purchase [post]
func (h *Server) PurchaseItems() httprouter.Handle {
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
		total := decimal.Zero
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
			total = total.Add(it.Price)
			// deduct inventory
			it.InventoryQuantity--
		}

		skus := pReq.SKUs

		promotions, err := h.promotionsEngine.ApplyPromotions(ctx, items)
		if err != nil {
			respondWithError(w, http.StatusInternalServerError, fmt.Errorf("could not apply promotion/deals: %w", err))
			return
		}

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

		price := total.Sub(decimal.NewFromFloat(promotions.Deduction))

		// Create order
		order := &model.Order{Price: price, Reference: model.GenerateReference()}
		if err := order.SetSKUList(skus); err != nil {
			respondWithError(w, http.StatusInternalServerError, err)
			return
		}

		// Execute purchase in a transaction to ensure atomicity
		err = h.db.Transaction(ctx, func(tx database.Database) error {
			// Save updated dbItems with new inventory totals
			if _, err := tx.UpsertItems(ctx, items); err != nil {
				return fmt.Errorf("failed to update inventory: %w", err)
			}
			// Create order
			if err := tx.AddOrder(ctx, order); err != nil {
				return fmt.Errorf("failed to create order: %w", err)
			}
			return nil
		})
		if err != nil {
			respondWithError(w, http.StatusInternalServerError, err)
			return
		}

		if err := respondWithJSON(w, http.StatusOK, &model.PurchaseItemsResponse{OrderReference: order.Reference, Cost: price.InexactFloat64()}); err != nil {
			respondWithError(w, http.StatusInternalServerError, err)
		}
	})
}
