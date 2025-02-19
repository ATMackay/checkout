package server

import (
	"fmt"
	"net/http"

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
func (h *Server) Orders() httprouter.Handle {
	return httprouter.Handle(func(w http.ResponseWriter, r *http.Request, _ httprouter.Params) {

		os, err := h.db.GetOrders(r.Context())
		if err != nil {
			respondWithError(w, http.StatusInternalServerError, fmt.Errorf("could not get orders from db: %w", err))
			return
		}

		if err := respondWithJSON(w, http.StatusOK, &os); err != nil {
			respondWithError(w, http.StatusInternalServerError, err)
		}
	})

}
