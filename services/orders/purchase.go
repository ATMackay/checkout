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
// @Security XAuthPassword
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

		q, err := h.quote(ctx, pReq.SKUs)
		if err != nil {
			return nil, err
		}

		// Create order
		order := &model.Order{
			Price:      q.Total(),
			Reference:  model.GenerateReference(),
			CustomerID: customerID,
		}
		if err := order.SetSKUList(q.SKUs()); err != nil {
			return nil, err
		}

		// Execute purchase in a transaction to ensure atomicity
		err = h.store.Transaction(ctx, func(tx database.Database) error {
			if _, err := tx.UpsertItems(ctx, stockUpdates(q)); err != nil {
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

		return &model.PurchaseItemsResponse{OrderReference: order.Reference, Cost: q.Total().InexactFloat64()}, nil
	})
}
