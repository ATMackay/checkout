package client

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"testing"
	"time"

	"github.com/ATMackay/checkout/database"
	"github.com/ATMackay/checkout/model"
	"github.com/ATMackay/checkout/server"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/require"
)

func TestClient(t *testing.T) {

	// Use in-memory  to execute tests
	db, err := database.NewSQLiteDB(database.InMemoryDSN, false)
	if err != nil {
		t.Fatal(err)
	}

	lo := &logrus.Logger{
		Out:       io.Discard,
		Formatter: &logrus.TextFormatter{DisableTimestamp: true},
		Level:     logrus.InfoLevel,
	}
	lo.SetLevel(logrus.DebugLevel)
	s := server.NewServer(8001, lo, db, "1234")
	s.Start()

	time.Sleep(10 * time.Millisecond)
	baseUrl := fmt.Sprintf("http://0.0.0.0%v", s.Addr())

	cl := New(baseUrl)
	cl.AddAuthorizationHeader("1234")

	ctx := context.Background()

	t.Run("status", func(t *testing.T) {
		stat, err := cl.Status(ctx)
		require.NoError(t, err)
		require.NotNil(t, stat)
		t.Log(*stat)
	})

	t.Run("health", func(t *testing.T) {
		health, err := cl.Health(ctx)
		require.NoError(t, err)
		require.NotNil(t, health)
		t.Log(*health)
	})

	it1 := &model.Item{Name: "Google TV", SKU: "120P90", Price: 49.99, InventoryQuantity: 10}
	it2 := &model.Item{Name: "MacBook Pro", SKU: "43N23P", Price: 5399.99, InventoryQuantity: 5}

	t.Run("add-item", func(t *testing.T) {
		err := cl.AddItems(ctx, &model.AddItemsRequest{Items: []*model.Item{it1, it2}})
		require.NoError(t, err)
		t.Log(*it1)
		t.Log(*it2)
	})

	t.Run("get-item-price", func(t *testing.T) {
		resp, err := cl.GetItemPrice(ctx, it1.SKU)
		require.NoError(t, err)
		require.NotNil(t, resp)
		require.Len(t, resp.Items, 1)
		t.Log(*resp)
	})

	t.Run("get-items-price", func(t *testing.T) {
		resp, err := cl.GetItemsPrice(ctx, &model.ItemsPriceRequest{SKUs: []string{it1.SKU, it2.SKU}})
		require.NoError(t, err)
		require.NotNil(t, resp)
		require.Len(t, resp.Items, 2)
		t.Log(*resp)
	})

	t.Run("purchase-items", func(t *testing.T) {
		resp, err := cl.PurchaseItems(ctx, &model.PurchaseItemsRequest{SKUs: []string{it1.SKU, it2.SKU}})
		require.NoError(t, err)
		require.NotNil(t, resp)
		require.NotEqual(t, "", resp.OrderReference)
		require.Equal(t, it1.Price+it2.Price, resp.Cost)
		t.Log(*resp)
	})

	t.Run("get-orders", func(t *testing.T) {
		resp, err := cl.GetOrders(ctx)
		require.NoError(t, err)
		require.NotNil(t, resp)
		require.Len(t, *resp, 1)
		t.Log(*resp)
	})

	// errors
	t.Run("context-cancelled", func(t *testing.T) {
		ctxCancelled, cancelFunc := context.WithCancel(ctx)
		cancelFunc()
		if _, err := cl.Status(ctxCancelled); !errors.Is(err, ctxCancelled.Err()) {
			t.Fatalf("expected error %v, got %v", ctxCancelled.Err(), err)
		}
	})

	t.Run("method-not-allowed", func(t *testing.T) {
		// incorrect verb
		if err := cl.executeRequest(ctx, nil, http.MethodPut, server.HealthEndPnt, nil); !errors.Is(err, ErrMethodNotAllowed) {
			t.Fatalf("expected error got %v", err)
		}
	})
}
