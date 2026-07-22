package orders

import (
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/ATMackay/checkout/database"
	"github.com/ATMackay/checkout/errors"
	"github.com/ATMackay/checkout/event"
	"github.com/ATMackay/checkout/httpserver"
	"github.com/ATMackay/checkout/model"
	"github.com/ATMackay/checkout/services/auth"
	"github.com/julienschmidt/httprouter"
	"github.com/shopspring/decimal"
)

// PurchaseItems godoc
// @Summary Execute a purchase for the supplied item list.
// @Description Create a purchase order for the supplied item list.
// @Tags inventory
// @Accept json
// @Produce json
// @Param   request  body    model.PurchaseItemsRequest  true  "List of SKUs"
// @Success 200 {object} model.PurchaseItemsResponse
// @Failure 400 {object} errors.JSONError
// @Failure 404 {object} errors.JSONError
// @Failure 503 {object} errors.JSONError
// @Router /v1/inventory/items/purchase [post]
func (h *Service) PurchaseItems() httprouter.Handle {
	return httpserver.Handle(func(r *http.Request, _ httprouter.Params) (any, error) {
		ctx := r.Context()

		// Inspect UserID/CustomerID
		customerID, ok := auth.UserID(ctx)
		if !ok {
			return nil, fmt.Errorf("%w", errors.ErrInvalidInput)
		}

		var pReq model.PurchaseItemsRequest
		if err := json.NewDecoder(r.Body).Decode(&pReq); err != nil {
			return nil, fmt.Errorf("%w: %v", errors.ErrInvalidInput, err)
		}

		// validate request params
		for _, sku := range pReq.SKUs {
			if !model.IsSKU(sku) {
				return nil, fmt.Errorf("%w: invalid sku input '%s'", errors.ErrInvalidInput, sku)
			}
		}

		// Fetch items from DB
		dbItems, err := h.store.GetItemsBySKU(ctx, pReq.SKUs)
		if err != nil {
			return nil, fmt.Errorf("could not get items: %w", err)
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
				return nil, fmt.Errorf("%w: item %s empty", errors.ErrNotFound, it.SKU)
			}
			total = total.Add(it.Price)
			// deduct inventory
			it.InventoryQuantity--
		}

		skus := pReq.SKUs

		promotions, err := h.promotionsEngine.ApplyPromotions(ctx, items)
		if err != nil {
			return nil, fmt.Errorf("could not apply promotion/deals: %w", err)
		}

		for _, it := range promotions.AddedItems {
			sku := it.SKU
			itemCount[sku]++
			dbIt, err := h.store.GetItemBySKU(ctx, sku)
			if err != nil {
				return nil, fmt.Errorf("could not get item: %w", err)
			}
			if dbIt.InventoryQuantity < itemCount[sku] {
				// Skip if we cannot add
				continue
			}
			// Note: DB tx can fail if concurrent requests push InventoryQuantity below zero
			dbIt.InventoryQuantity--

			items = append(items, dbIt)
			skus = append(pReq.SKUs, sku)
		}

		price := total.Sub(decimal.NewFromFloat(promotions.Deduction))

		// Create order
		order := &model.Order{
			Price:      price,
			Reference:  model.GenerateReference(),
			CustomerID: customerID,
		}
		if err := order.SetSKUList(skus); err != nil {
			return nil, err
		}

		// Execute purchase in a transaction to ensure atomicity
		err = h.store.Transaction(ctx, func(tx database.Database) error {
			// Save updated dbItems with new inventory totals
			if _, err := tx.UpsertItems(ctx, items); err != nil {
				return fmt.Errorf("failed to update inventory: %w", err)
			}
			// Create order
			if err := tx.AddOrder(ctx, order); err != nil {
				return fmt.Errorf("failed to create order: %w", err)
			}
			// Enqueue the event in the SAME transaction as the order. The relay
			// publishes it to the broker asynchronously; writing it here (rather
			// than publishing inline) is what makes the order and its event
			// atomic — they commit together or not at all.
			outboxItem, err := newOutboxItem(event.New(
				event.TopicOrderCreated,
				order.Reference,
				order, // re-use order model for event propagation
			))
			if err != nil {
				return fmt.Errorf("failed to build outbox item: %w", err)
			}
			if err := tx.AddOutboxItems(ctx, []*model.OutboxItem{outboxItem}); err != nil {
				return fmt.Errorf("failed to enqueue event: %w", err)
			}
			return nil
		})
		if err != nil {
			return nil, err
		}

		return &model.PurchaseItemsResponse{OrderReference: order.Reference, Cost: price.InexactFloat64()}, nil
	})
}
