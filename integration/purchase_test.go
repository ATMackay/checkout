//go:build integration

package integration

import (
	"context"
	"testing"
	"time"

	"github.com/ATMackay/checkout/integration/stack"
	"github.com/ATMackay/checkout/model"
	"github.com/shopspring/decimal"
)

func Test_PurchaseItems(t *testing.T) {
	var testItems = []*model.Item{
		{Name: "Google TV", SKU: "120P90", Price: decimal.NewFromFloat(49.99), InventoryQuantity: 10},
		{Name: "MacBook Pro", SKU: "43N23P", Price: decimal.NewFromFloat(5399.99), InventoryQuantity: 5},
		{Name: "Alexa Speaker ", SKU: "A304SD", Price: decimal.NewFromFloat(109.50), InventoryQuantity: 10},
		{Name: "Raspberry Pi B", SKU: "234234", Price: decimal.NewFromFloat(30.0), InventoryQuantity: 2},
	}

	ctx, cancelFn := context.WithTimeout(context.Background(), 600*time.Second) // Set hard limit of 10 minutes
	defer cancelFn()

	st := stack.MakeStack(t, ctx, &stack.Opts{DbLogs: true, AppLogs: true, Debug: true}) // Modify logging options as required
	// Create Client
	baseURL := st.AppURL()
	cl := stack.MakeAuthClient(t, baseURL, stack.TestAuthPassword)

	// Add items to inventory
	req := &model.AddItemsRequest{Items: testItems}
	t.Logf("adding inventory items, itemCount=%d", len(req.Items))
	if err := cl.AddItems(ctx, req); err != nil {
		t.Fatalf("AddItems failed: %v (baseURL=%s)", err, baseURL)
	}
	// Purchase items individually
	for _, it := range testItems {
		resp, err := cl.PurchaseItems(ctx, &model.PurchaseItemsRequest{SKUs: []string{it.SKU}})
		if err != nil {
			t.Errorf("unexpected error purchasing item '%v': %v", it.Name, err)
		}
		if resp.OrderReference == "" {
			t.Error("missing order reference for purchase")
		}
		if resp.Cost <= 0 {
			t.Errorf("invalid cost for purchase %v %v, got %v", resp.OrderReference, it.Name, resp.Cost)
		}
	}
}
