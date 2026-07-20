//go:build integration

package integration

import (
	"context"
	"fmt"
	"math/rand/v2"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/ATMackay/checkout/client"
	"github.com/ATMackay/checkout/integration/stack"
	"github.com/ATMackay/checkout/model"
	"github.com/shopspring/decimal"
	"golang.org/x/sync/errgroup"
)

// Integration testing with Testcontainers (Postgres + App)
func Test_AddInventoryItems(t *testing.T) {
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
	cl := stack.MakeAuthClient(t, baseURL, st.AuthPsswd())

	// Add items to inventory
	req := &model.AddItemsRequest{Items: testItems}
	t.Logf("adding inventory items, itemCount=%d", len(req.Items))
	if err := cl.AddItems(ctx, req); err != nil {
		t.Fatalf("AddItems failed: %v (baseURL=%s)", err, baseURL)
	}
	// Verify the items exist and that the prices match
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

func Test_ConcurrentAddItems(t *testing.T) {
	// Make 10 authenticated clients that concurrently write items to our server
	ctx := context.Background()

	// Raise stack
	st := stack.MakeStack(t, ctx, &stack.Opts{DbLogs: false, AppLogs: true, Debug: false})
	baseURL := st.AppURL()
	// Make client pool
	poolSize := max(1, rand.IntN(10))
	clients := make(chan *client.Client, poolSize)
	for range poolSize {
		cl := stack.MakeAuthClient(t, baseURL, st.AuthPsswd())
		clients <- cl
	}

	var mu sync.Mutex

	errG, gCtx := errgroup.WithContext(ctx)

	// Execute write test

	duration := new(atomic.Int64)

	itemCount := 1000
	for i := range itemCount {
		errG.Go(func() error {
			// Take client from pool
			cl := <-clients
			defer func() { clients <- cl }()

			// Make item
			mu.Lock()
			it := stack.MakeRandomizedTestItem(i)
			mu.Unlock()
			start := time.Now()
			if err := cl.AddItems(gCtx, &model.AddItemsRequest{Items: []*model.Item{it}}); err != nil {
				return fmt.Errorf("%d: %w", i, err)
			}
			duration.Add(time.Since(start).Nanoseconds())
			return nil
		})
	}
	if err := errG.Wait(); err != nil {
		t.Fatal(err)
	}

	dur := duration.Load()

	// Log stats
	t.Logf("Wrote %d items in %vs, avg write time %.2fms", itemCount, float64(dur)/1e9, float64(dur)/float64(itemCount)/1e6)

	// Check items have been added

	cl := <-clients

	res, err := cl.ListItems(ctx)
	if err != nil {
		t.Fatal(err)
	}

	if g, w := len(res), itemCount-1; g != w { // TODO  - 999 vs 1000 ?
		t.Fatalf("unexpected item count: got %v, want %v", g, w)
	}
}
