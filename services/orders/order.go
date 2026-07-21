package orders

import (
	"fmt"
	"net/http"

	"github.com/ATMackay/checkout/errors"
	"github.com/ATMackay/checkout/services/auth"
	"github.com/ATMackay/checkout/services/httpserver"
	"github.com/julienschmidt/httprouter"
)

// Orders godoc
// @Summary Get list of purchase orders
// @Description List all purchase orders
// @Tags inventory
// @Produce json
// @Success 200 {array}  model.Order
// @Failure 400 {object} errors.JSONError
// @Failure 401 {object} errors.JSONError
// @Failure 404 {object} errors.JSONError
// @Failure 500 {object} errors.JSONError
// @Router /v1/orders [get]
func (h *Service) Orders() httprouter.Handle {
	return httpserver.Handle(func(r *http.Request, _ httprouter.Params) (any, error) {
		ctx := r.Context()
		// Inspect UserID
		userID, ok := auth.UserID(ctx)
		if !ok {
			return nil, fmt.Errorf("%w", errors.ErrInvalidInput)
		}
		os, err := h.db.GetOrders(ctx, userID)
		if err != nil {
			return nil, fmt.Errorf("could not get orders from db: %w", err)
		}
		return os, nil
	})
}
