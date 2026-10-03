//go:build !integration

package pricing

import (
	"testing"

	"github.com/ATMackay/checkout/errors"
	"github.com/ATMackay/checkout/model"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/require"
)

var (
	tv    = &model.Item{Name: "Google TV", SKU: "120P90", Price: decimal.RequireFromString("49.99"), InventoryQuantity: 10}
	mac   = &model.Item{Name: "MacBook Pro", SKU: "43N23P", Price: decimal.RequireFromString("5399.99"), InventoryQuantity: 5}
	alexa = &model.Item{Name: "Alexa Speaker", SKU: "A304SD", Price: decimal.RequireFromString("109.50"), InventoryQuantity: 10}
	pi    = &model.Item{Name: "Raspberry Pi B", SKU: "234234", Price: decimal.RequireFromString("30.00"), InventoryQuantity: 2}
)

func storePricer() *Pricer {
	return NewPricer(
		FreeItem("MacBook Pro", "Raspberry Pi B"),
		BuyXPayY("Google TV", 3, 2),
		BulkDiscount("Alexa Speaker", 3, decimal.NewFromInt(10)),
	)
}

func repeat(sku string, n int) []string {
	skus := make([]string, n)
	for i := range skus {
		skus[i] = sku
	}
	return skus
}

func TestQuote(t *testing.T) {
	inv := NewInventory(tv, mac, alexa, pi)

	tests := []struct {
		name      string
		skus      []string
		inv       Inventory
		wantTotal string
		wantFree  []string
		wantErr   error
	}{
		{name: "empty basket", skus: nil, wantErr: errors.ErrInvalidInput},
		{name: "basket too large", skus: repeat(tv.SKU, MaxBasketSize+1), wantErr: errors.ErrInvalidInput},
		{name: "malformed sku", skus: []string{"not-a-sku"}, wantErr: errors.ErrInvalidInput},
		{name: "unknown sku", skus: []string{"ZZZ999"}, wantErr: errors.ErrNotFound},
		{name: "more units than in stock", skus: repeat(pi.SKU, 3), wantErr: errors.ErrNotFound},
		{name: "every unit is priced", skus: repeat(tv.SKU, 2), wantTotal: "99.98"},
		{name: "three tvs for the price of two", skus: repeat(tv.SKU, 3), wantTotal: "99.98"},
		{name: "five tvs: one free", skus: repeat(tv.SKU, 5), wantTotal: "199.96"},
		{name: "six tvs: two free", skus: repeat(tv.SKU, 6), wantTotal: "199.96"},
		{name: "three alexas get no discount", skus: repeat(alexa.SKU, 3), wantTotal: "328.50"},
		{name: "four alexas are ten percent off", skus: repeat(alexa.SKU, 4), wantTotal: "394.20"},
		{name: "a free pi per macbook", skus: repeat(mac.SKU, 2), wantTotal: "10799.98", wantFree: repeat(pi.SKU, 2)},
		{name: "free pis capped by stock", skus: repeat(mac.SKU, 3), wantTotal: "16199.97", wantFree: repeat(pi.SKU, 2)},
		{name: "a bought pi leaves less to give away", skus: []string{mac.SKU, mac.SKU, pi.SKU}, wantTotal: "10829.98", wantFree: []string{pi.SKU}},
		{name: "no free pi when none is stocked", skus: []string{mac.SKU}, inv: NewInventory(mac), wantTotal: "5399.99"},
		{name: "promotions combine", skus: append(repeat(tv.SKU, 3), repeat(alexa.SKU, 4)...), wantTotal: "494.18"},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			in := inv
			if tc.inv != nil {
				in = tc.inv
			}
			q, err := storePricer().Quote(tc.skus, in)
			if tc.wantErr != nil {
				require.ErrorIs(t, err, tc.wantErr)
				return
			}
			require.NoError(t, err)
			require.Equal(t, tc.wantTotal, q.Total().StringFixed(2))
			var free []string
			for _, l := range q.Free {
				free = append(free, repeat(l.Item.SKU, l.Quantity)...)
			}
			require.Equal(t, tc.wantFree, free)
		})
	}
}

func TestQuoteSKUsAndStock(t *testing.T) {
	q, err := storePricer().Quote([]string{mac.SKU, pi.SKU, mac.SKU}, NewInventory(mac, pi))
	require.NoError(t, err)

	require.Equal(t, []string{mac.SKU, mac.SKU, pi.SKU, pi.SKU}, q.SKUs())

	after := q.StockAfter()
	require.Len(t, after, 2)
	require.Equal(t, pi.SKU, after[0].SKU)
	require.Equal(t, 0, after[0].InventoryQuantity)
	require.Equal(t, mac.SKU, after[1].SKU)
	require.Equal(t, 3, after[1].InventoryQuantity)
}

func TestQuoteLeavesInventoryUntouched(t *testing.T) {
	inv := NewInventory(mac, pi)
	_, err := storePricer().Quote(repeat(mac.SKU, 2), inv)
	require.NoError(t, err)
	require.Equal(t, 5, inv[mac.SKU].InventoryQuantity)
	require.Equal(t, 2, inv[pi.SKU].InventoryQuantity)
	require.Equal(t, 5, mac.InventoryQuantity)
}

func TestRewardNames(t *testing.T) {
	p := NewPricer(FreeItem("MacBook Pro", "Reward"), FreeItem("Google TV", "Reward"), BuyXPayY("Google TV", 3, 2))
	require.Equal(t, []string{"Reward"}, p.RewardNames(NewInventory(mac, tv)))
	require.Empty(t, p.RewardNames(NewInventory(alexa)), "no trigger requested, nothing to look up")
}

func TestQuoteSplitsGrossAndDeduction(t *testing.T) {
	q, err := storePricer().Quote(append(repeat(tv.SKU, 3), repeat(alexa.SKU, 4)...), NewInventory(tv, alexa))
	require.NoError(t, err)
	require.Equal(t, "587.97", q.Gross.StringFixed(2))
	require.Equal(t, "93.79", q.Deduction.StringFixed(2))
}

func TestBulkDiscountRoundsToCents(t *testing.T) {
	odd := &model.Item{Name: "Cable", SKU: "CBL001", Price: decimal.RequireFromString("1.115"), InventoryQuantity: 10}
	q, err := NewPricer(BulkDiscount("Cable", 3, decimal.NewFromInt(10))).Quote(repeat(odd.SKU, 4), NewInventory(odd))
	require.NoError(t, err)
	require.Equal(t, "0.45", q.Deduction.String())
}

func TestDeductionNeverExceedsGross(t *testing.T) {
	p := NewPricer(BuyXPayY("Google TV", 1, 0), BulkDiscount("Google TV", 0, decimal.NewFromInt(50)))
	q, err := p.Quote(repeat(tv.SKU, 2), NewInventory(tv))
	require.NoError(t, err)
	require.True(t, q.Total().IsZero(), "got total %s", q.Total())
}

func TestRulesIgnoreInvalidArguments(t *testing.T) {
	p := NewPricer(
		BuyXPayY("Google TV", 3, -1),
		BuyXPayY("Google TV", 0, 0),
		BulkDiscount("Google TV", 0, decimal.NewFromInt(-10)),
		BulkDiscount("Google TV", 0, decimal.NewFromInt(150)),
	)
	q, err := p.Quote(repeat(tv.SKU, 3), NewInventory(tv))
	require.NoError(t, err)
	require.True(t, q.Deduction.IsZero(), "got deduction %s", q.Deduction)
}

func TestNewInventorySkipsMissingItems(t *testing.T) {
	inv := NewInventory(nil, &model.Item{}, tv)
	require.Len(t, inv, 1)
}
