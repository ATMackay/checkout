package orders

import (
	"context"
	"fmt"
	"slices"

	"github.com/ATMackay/checkout/model"
	"github.com/ATMackay/checkout/pricing"
)

// quote prices skus against a fresh inventory snapshot that includes the items
// the basket's promotions may give away.
func (h *Service) quote(ctx context.Context, skus []string) (pricing.Quote, error) {
	if err := pricing.CheckSKUs(skus); err != nil {
		return pricing.Quote{}, err
	}
	items, err := h.store.GetItemsBySKU(ctx, slices.Compact(slices.Sorted(slices.Values(skus))))
	if err != nil {
		return pricing.Quote{}, fmt.Errorf("could not get items: %w", err)
	}
	for _, name := range h.pricer.RewardNames(pricing.NewInventory(items...)) {
		it, err := h.store.GetItemByName(ctx, name)
		if err != nil {
			return pricing.Quote{}, fmt.Errorf("could not get promotional item %q: %w", name, err)
		}
		items = append(items, it)
	}
	return h.pricer.Quote(skus, pricing.NewInventory(items...))
}

// priceResponse renders a quote in the price endpoint's wire format: one item
// per requested unit, money as float64.
func priceResponse(q pricing.Quote) *model.PriceResponse {
	return &model.PriceResponse{
		Items: units(q.Lines),
		Promotions: &model.Promotions{
			Deduction:  q.Deduction.InexactFloat64(),
			AddedItems: units(q.Free),
		},
		TotalGross:        q.Gross.InexactFloat64(),
		TotalWithDiscount: q.Total().InexactFloat64(),
	}
}

func units(lines []pricing.Line) []*model.Item {
	var items []*model.Item
	for _, l := range lines {
		for range l.Quantity {
			it := l.Item
			items = append(items, &it)
		}
	}
	return items
}

// stockUpdates is the inventory to write back once q is purchased.
func stockUpdates(q pricing.Quote) []*model.Item {
	after := q.StockAfter()
	items := make([]*model.Item, len(after))
	for i := range after {
		items[i] = &after[i]
	}
	return items
}
