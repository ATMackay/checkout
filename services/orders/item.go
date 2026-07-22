package orders

import (
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/ATMackay/checkout/errors"
	"github.com/ATMackay/checkout/httpserver"
	"github.com/ATMackay/checkout/model"
	"github.com/julienschmidt/httprouter"
	"github.com/shopspring/decimal"
)

// ListItems godoc
// @Summary      Returns a list of items in the inventory table
// @Description  Show all listed inventory items
// @Tags         inventory
// @Produce      json
// @Success      200      {array}  model.Item
// @Failure      500      {object} errors.JSONError
// @Security     XAuthPassword
// @Router       /v1/inventory/items [get]
func (h *Service) ListItems() httprouter.Handle {
	return httpserver.Handle(func(r *http.Request, _ httprouter.Params) (any, error) {
		return h.store.ListItems(r.Context())
	})
}

// AddItems godoc
// @Summary      Add new or updated items to the inventory table
// @Description  Add new or updated items
// @Tags         inventory
// @Accept       json
// @Produce      json
// @Param        request  body   model.AddItemsRequest  true  "List of items"
// @Success      200      {array}  model.Item
// @Failure      400      {object} errors.JSONError
// @Failure      401      {object} errors.JSONError
// @Failure      404      {object} errors.JSONError
// @Failure      500      {object} errors.JSONError
// @Security     XAuthPassword
// @Router       /v1/inventory/items [post]
func (h *Service) AddItems() httprouter.Handle {
	return httpserver.Handle(func(r *http.Request, _ httprouter.Params) (any, error) {
		var iReq model.AddItemsRequest
		if err := json.NewDecoder(r.Body).Decode(&iReq); err != nil {
			return nil, fmt.Errorf("%w: %v", errors.ErrInvalidInput, err)
		}

		if len(iReq.Items) == 0 {
			return nil, fmt.Errorf("%w: no items provided", errors.ErrInvalidInput)
		}

		// validate
		for i, it := range iReq.Items {
			if err := it.Validate(); err != nil {
				return nil, fmt.Errorf("%w: item at index %d was invalid: %v", errors.ErrInvalidInput, i, err)
			}
		}

		return h.store.UpsertItems(r.Context(), iReq.Items)
	})
}

// ItemPrice godoc
// @Summary      Get price for a single item
// @Description  Get price information for a single item by SKU or name
// @Tags         inventory
// @Produce      json
// @Param        key   path      string                true  "Item SKU or Name"
// @Success      200   {object}  model.PriceResponse
// @Failure      400   {object}  errors.JSONError
// @Failure      404   {object}  errors.JSONError
// @Failure      500   {object}  errors.JSONError
// @Router       /v1/inventory/item/price/{key} [get]
func (h *Service) ItemPrice() httprouter.Handle {
	return httpserver.Handle(func(r *http.Request, p httprouter.Params) (any, error) {
		ctx := r.Context()
		nameOrSku := p.ByName("key")
		var dbItem *model.Item
		var err error

		if model.IsSKU(nameOrSku) {
			dbItem, err = h.store.GetItemBySKU(ctx, nameOrSku)
		} else {
			dbItem, err = h.store.GetItemByName(ctx, nameOrSku)
		}
		if err != nil {
			return nil, fmt.Errorf("could not get item with key '%s': %w", nameOrSku, err)
		}
		if dbItem.InventoryQuantity < 1 {
			return nil, fmt.Errorf("%w: item %s empty", errors.ErrNotFound, dbItem.SKU)
		}

		return &model.PriceResponse{
			Items:             []*model.Item{{Name: dbItem.Name, SKU: dbItem.SKU, Price: dbItem.Price}},
			TotalGross:        dbItem.Price.InexactFloat64(),
			TotalWithDiscount: dbItem.Price.InexactFloat64(),
		}, nil
	})
}

// ItemsPrice godoc
// @Summary      Get prices for multiple items
// @Description  Get total price for a batch of items by SKUs
// @Tags         inventory
// @Accept       json
// @Produce      json
// @Param        request  body     model.ItemsPriceRequest  true  "List of SKUs"
// @Success      200      {object} model.PriceResponse
// @Failure      400      {object} errors.JSONError
// @Failure      404      {object} errors.JSONError
// @Failure      500      {object} errors.JSONError
// @Router       /v1/inventory/items/price [post]
func (h *Service) ItemsPrice() httprouter.Handle {
	return httpserver.Handle(func(r *http.Request, _ httprouter.Params) (any, error) {
		ctx := r.Context()

		var pReq model.ItemsPriceRequest
		if err := json.NewDecoder(r.Body).Decode(&pReq); err != nil {
			return nil, fmt.Errorf("%w: %v", errors.ErrInvalidInput, err)
		}

		// validate request params
		for _, sku := range pReq.SKUs {
			if !model.IsSKU(sku) {
				return nil, fmt.Errorf("%w: invalid sku input '%s'", errors.ErrInvalidInput, sku)
			}
		}

		dbItems, err := h.store.GetItemsBySKU(ctx, pReq.SKUs)
		if err != nil {
			return nil, fmt.Errorf("could not get items: %w", err)
		}

		resp := &model.PriceResponse{}
		total := decimal.Zero
		for _, it := range dbItems {
			if it.InventoryQuantity < 1 {
				return nil, fmt.Errorf("%w: item %s empty", errors.ErrNotFound, it.SKU)
			}
			resp.Items = append(resp.Items, it)
			total = total.Add(it.Price)
		}

		promotions, err := h.promotionsEngine.ApplyPromotions(ctx, resp.Items)
		if err != nil {
			return nil, fmt.Errorf("could not apply promotion/deals: %w", err)
		}

		resp.Promotions = promotions
		resp.TotalGross = total.InexactFloat64()
		resp.TotalWithDiscount = total.InexactFloat64() - promotions.Deduction

		return resp, nil
	})
}
