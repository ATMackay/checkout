//go:build integration

package integration

import (
	"context"
	"encoding/json"
	"net/http"
	"testing"
	"time"

	"github.com/ATMackay/checkout/integration/stack"
	"github.com/ATMackay/checkout/model"
	"github.com/shopspring/decimal"
)

type notificationView struct {
	Reference string `json:"reference"`
	Delivered bool   `json:"delivered"`
}

// getNotifications reads the notifier's /v1/notifications endpoint.
func getNotifications(t *testing.T, ctx context.Context, baseURL string, undeliveredOnly bool) []notificationView {
	t.Helper()
	url := baseURL + "/v1/notifications"
	if undeliveredOnly {
		url += "?undelivered=true"
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		t.Fatal(err)
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("get notifications: %v", err)
	}
	defer resp.Body.Close()
	var out []notificationView
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		t.Fatalf("decode notifications: %v", err)
	}
	return out
}

// Test_Notification runs the full event path: an order purchased on the orders
// service is published to kafka, consumed by the notifier, and surfaced as a
// delivered notification.
func Test_Notification(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 600*time.Second)
	defer cancel()

	st := stack.MakeStack(t, ctx, &stack.Opts{DbLogs: false, AppLogs: true, Debug: true, EnableEvents: true})

	cl := stack.MakeAuthClient(t, st.AppURL(), stack.TestAuthPassword)

	item := &model.Item{Name: "Google TV", SKU: "120P90", Price: decimal.NewFromFloat(49.99), InventoryQuantity: 5}
	if err := cl.AddItems(ctx, &model.AddItemsRequest{Items: []*model.Item{item}}); err != nil {
		t.Fatalf("AddItems: %v", err)
	}

	resp, err := cl.PurchaseItems(ctx, &model.PurchaseItemsRequest{SKUs: []string{item.SKU}})
	if err != nil {
		t.Fatalf("PurchaseItems: %v", err)
	}
	t.Logf("purchased order %s", resp.OrderReference)

	// The relay publishes and the notifier consumes asynchronously; poll until
	// the order shows up delivered.
	notifierURL := st.NotifierURL()
	deadline := time.Now().Add(60 * time.Second)
	for {
		delivered := false
		for _, n := range getNotifications(t, ctx, notifierURL, false) {
			if n.Reference == resp.OrderReference && n.Delivered {
				delivered = true
			}
		}
		if delivered {
			break
		}
		if time.Now().After(deadline) {
			t.Fatalf("order %s not delivered within timeout", resp.OrderReference)
		}
		time.Sleep(time.Second)
	}

	// Once delivered, the undelivered-only view must not contain it.
	for _, n := range getNotifications(t, ctx, notifierURL, true) {
		if n.Reference == resp.OrderReference {
			t.Fatalf("order %s still listed as undelivered", resp.OrderReference)
		}
	}
}
