//go:build integration

package integration

import (
	"context"
	"testing"
	"time"

	"github.com/ATMackay/checkout/model"
	"github.com/shopspring/decimal"
)

// Integration testing with Testcontainers (Postgres + App)
func Test_AddInventoryItems(t *testing.T) {
	var testItems = []*model.Item{
		{Name: "Google TV", SKU: "120P90", Price: decimal.NewFromFloat(49.99), InventoryQuantity: 10},
		{Name: "MacBook Pro", SKU: "43N23P", Price: decimal.NewFromFloat(5399.99), InventoryQuantity: 5},
		{Name: "Alexa Speaker ", SKU: "A304SD", Price: decimal.NewFromFloat(109.50), InventoryQuantity: 10},
		{Name: "Raspberry Pi B", SKU: "234234", Price: decimal.NewFromFloat(30.0), InventoryQuantity: 2},
	}

	ctx, cancelFn := context.WithTimeout(context.Background(), 600*time.Second) // Set hard limit of 5 minutes
	defer cancelFn()

	stack := makeStack(t, ctx, &stackOpts{dbLogs: true, buildFromDockerfile: true, appLogs: true}) // Modify logging options as required

	// 4) Create Client
	baseURL := stack.app.url()
	cl := makeClient(t, baseURL, testAuthPassword)

	// 5) Add items to inventory
	req := &model.AddItemsRequest{Items: testItems}
	t.Logf("adding inventory items, itemCount=%d", len(req.Items))
	if err := cl.AddItems(ctx, req); err != nil {
		t.Fatalf("AddItems failed: %v (baseURL=%s)", err, baseURL)
	}
	// 6) Verify the items exist and that the prices match
	for _, it := range testItems {
		t.Logf("getting price for item %v", it.Name)
		price, err := cl.GetItemPrice(ctx, it.SKU)
		if err != nil {
			t.Errorf("failed to fetch item price, name %s, err %v", it.Name, err)
		}
		// check price matches
		if g, w := price.TotalGross, it.Price.InexactFloat64(); g != w {
			t.Errorf("%s: item price mismatch, got %v want %v", it.Name, g, w)
		}
	}
}
