//go:build !integration

package orders

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/ATMackay/checkout/database"
	"github.com/ATMackay/checkout/messaging/noop"
	"github.com/ATMackay/checkout/model"
	"github.com/ATMackay/checkout/services/auth"
	"github.com/julienschmidt/httprouter"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/require"
)

const testPassword = "secret"

var (
	googleTV  = model.Item{Name: "Google TV", SKU: "120P90", Price: decimal.RequireFromString("49.99"), InventoryQuantity: 10}
	macBook   = model.Item{Name: "MacBook Pro", SKU: "43N23P", Price: decimal.RequireFromString("5399.99"), InventoryQuantity: 5}
	alexa     = model.Item{Name: "Alexa Speaker", SKU: "A304SD", Price: decimal.RequireFromString("109.50"), InventoryQuantity: 10}
	raspberry = model.Item{Name: "Raspberry Pi B", SKU: "234234", Price: decimal.RequireFromString("30.00"), InventoryQuantity: 2}
)

// pricingFixture is the orders router over an in-memory SQLite store seeded
// with the given items.
type pricingFixture struct {
	t      *testing.T
	db     *database.GormDB
	router *httprouter.Router
}

func newPricingFixture(t *testing.T, items ...model.Item) *pricingFixture {
	t.Helper()
	db, err := database.NewSQLiteDB(database.InMemoryDSN, false)
	require.NoError(t, err)
	t.Cleanup(func() { require.NoError(t, db.Close()) })
	seed := make([]*model.Item, len(items))
	for i := range items {
		it := items[i]
		seed[i] = &it
	}
	if len(seed) > 0 {
		_, err = db.UpsertItems(context.Background(), seed)
		require.NoError(t, err)
	}
	authn := auth.NewPasswordAuthenticator(map[string]string{testPassword: "customer-1"})
	svc := NewService(db, NewOutboxRelayer(db, &noop.Client{}), authn)
	return &pricingFixture{t: t, db: db, router: svc.RegisterHandlers()}
}

func (f *pricingFixture) do(method, path string, body any) *httptest.ResponseRecorder {
	f.t.Helper()
	var buf bytes.Buffer
	if body != nil {
		require.NoError(f.t, json.NewEncoder(&buf).Encode(body))
	}
	req := httptest.NewRequest(method, path, &buf)
	req.Header.Set(auth.XAuthHeaderKey, testPassword)
	rr := httptest.NewRecorder()
	f.router.ServeHTTP(rr, req)
	return rr
}

func (f *pricingFixture) price(skus ...string) (*model.PriceResponse, int) {
	f.t.Helper()
	rr := f.do(http.MethodPost, ItemPriceEndPnt, model.ItemsPriceRequest{SKUs: skus})
	if rr.Code != http.StatusOK {
		return nil, rr.Code
	}
	var resp model.PriceResponse
	require.NoError(f.t, json.Unmarshal(rr.Body.Bytes(), &resp))
	return &resp, rr.Code
}

func (f *pricingFixture) purchase(skus ...string) (*model.PurchaseItemsResponse, int) {
	f.t.Helper()
	rr := f.do(http.MethodPost, ItemPurchaseEndPnt, model.PurchaseItemsRequest{SKUs: skus})
	if rr.Code != http.StatusOK {
		return nil, rr.Code
	}
	var resp model.PurchaseItemsResponse
	require.NoError(f.t, json.Unmarshal(rr.Body.Bytes(), &resp))
	return &resp, rr.Code
}

func (f *pricingFixture) stock(sku string) int {
	f.t.Helper()
	it, err := f.db.GetItemBySKU(context.Background(), sku)
	require.NoError(f.t, err)
	require.NotNil(f.t, it)
	return it.InventoryQuantity
}

func (f *pricingFixture) lastOrderSKUs() []string {
	f.t.Helper()
	orders, err := f.db.GetOrders(context.Background(), "customer-1")
	require.NoError(f.t, err)
	require.NotEmpty(f.t, orders) // newest first
	skus, err := orders[0].GetSKUList()
	require.NoError(f.t, err)
	return skus
}

func money(s string) float64 { return decimal.RequireFromString(s).InexactFloat64() }

func Test_ItemsPrice(t *testing.T) {
	t.Run("prices every requested unit", func(t *testing.T) {
		f := newPricingFixture(t, googleTV)
		resp, code := f.price(googleTV.SKU, googleTV.SKU)
		require.Equal(t, http.StatusOK, code)
		require.Equal(t, money("99.98"), resp.TotalGross)
		require.Len(t, resp.Items, 2)
	})
	t.Run("lists free items capped by stock", func(t *testing.T) {
		f := newPricingFixture(t, macBook, raspberry)
		resp, code := f.price(macBook.SKU, macBook.SKU, macBook.SKU)
		require.Equal(t, http.StatusOK, code)
		require.Len(t, resp.Promotions.AddedItems, 2)
		require.Equal(t, raspberry.SKU, resp.Promotions.AddedItems[0].SKU)
		require.Equal(t, money("16199.97"), resp.TotalWithDiscount)
	})
	t.Run("unknown sku is not found", func(t *testing.T) {
		f := newPricingFixture(t, googleTV)
		_, code := f.price(googleTV.SKU, "ZZZ999")
		require.Equal(t, http.StatusNotFound, code)
	})
	t.Run("matches what a purchase would charge", func(t *testing.T) {
		f := newPricingFixture(t, googleTV, alexa)
		basket := []string{googleTV.SKU, googleTV.SKU, googleTV.SKU, alexa.SKU}
		quote, code := f.price(basket...)
		require.Equal(t, http.StatusOK, code)
		bought, code := f.purchase(basket...)
		require.Equal(t, http.StatusOK, code)
		require.Equal(t, quote.TotalWithDiscount, bought.Cost)
	})
}

func Test_PurchaseItems(t *testing.T) {
	t.Run("unknown sku is not found", func(t *testing.T) {
		f := newPricingFixture(t, googleTV)
		_, code := f.purchase("ZZZ999")
		require.Equal(t, http.StatusNotFound, code)
	})
	t.Run("empty basket is rejected", func(t *testing.T) {
		f := newPricingFixture(t, googleTV)
		_, code := f.purchase()
		require.Equal(t, http.StatusBadRequest, code)
	})
	t.Run("more units than in stock is not found", func(t *testing.T) {
		f := newPricingFixture(t, raspberry)
		_, code := f.purchase(raspberry.SKU, raspberry.SKU, raspberry.SKU)
		require.Equal(t, http.StatusNotFound, code)
		require.Equal(t, 2, f.stock(raspberry.SKU))
	})
	t.Run("three google tvs cost the price of two", func(t *testing.T) {
		f := newPricingFixture(t, googleTV)
		resp, code := f.purchase(googleTV.SKU, googleTV.SKU, googleTV.SKU)
		require.Equal(t, http.StatusOK, code)
		require.Equal(t, money("99.98"), resp.Cost)
		require.Equal(t, 7, f.stock(googleTV.SKU))
	})
	t.Run("more than three alexa speakers are ten percent off", func(t *testing.T) {
		f := newPricingFixture(t, alexa)
		resp, code := f.purchase(alexa.SKU, alexa.SKU, alexa.SKU, alexa.SKU)
		require.Equal(t, http.StatusOK, code)
		require.Equal(t, money("394.20"), resp.Cost)
	})
	t.Run("every free raspberry pi is recorded on the order", func(t *testing.T) {
		f := newPricingFixture(t, macBook, raspberry)
		resp, code := f.purchase(macBook.SKU, macBook.SKU)
		require.Equal(t, http.StatusOK, code)
		require.Equal(t, money("10799.98"), resp.Cost)
		require.ElementsMatch(t, []string{macBook.SKU, macBook.SKU, raspberry.SKU, raspberry.SKU}, f.lastOrderSKUs())
		require.Equal(t, 0, f.stock(raspberry.SKU))
	})
	t.Run("a bought and a free raspberry pi both leave stock", func(t *testing.T) {
		f := newPricingFixture(t, macBook, raspberry)
		_, code := f.purchase(macBook.SKU, raspberry.SKU)
		require.Equal(t, http.StatusOK, code)
		require.Equal(t, 0, f.stock(raspberry.SKU))
	})
	t.Run("free items are capped by remaining stock", func(t *testing.T) {
		f := newPricingFixture(t, macBook, raspberry)
		_, code := f.purchase(macBook.SKU, macBook.SKU, macBook.SKU)
		require.Equal(t, http.StatusOK, code)
		require.Equal(t, 0, f.stock(raspberry.SKU))
		require.ElementsMatch(t, []string{macBook.SKU, macBook.SKU, macBook.SKU, raspberry.SKU, raspberry.SKU}, f.lastOrderSKUs())
	})
	t.Run("macbook sells without a free item when none is stocked", func(t *testing.T) {
		f := newPricingFixture(t, macBook)
		resp, code := f.purchase(macBook.SKU)
		require.Equal(t, http.StatusOK, code)
		require.Equal(t, money("5399.99"), resp.Cost)
	})
}
