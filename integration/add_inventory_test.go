//go:build integration

package testing

import (
	"context"
	"testing"

	"github.com/ATMackay/checkout/client"
	"github.com/ATMackay/checkout/model"
)

var testItems = []*model.Item{
	{Name: "Google TV", SKU: "120P90", Price: 49.99, InventoryQuantity: 10},
	{Name: "MacBook Pro", SKU: "43N23P", Price: 5399.99, InventoryQuantity: 5},
	{Name: "Alexa Speaker ", SKU: "A304SD", Price: 109.50, InventoryQuantity: 10},
	{Name: "Raspberry Pi B", SKU: "234234", Price: 30.0, InventoryQuantity: 2},
}

func Test_AddTestInventoryItems(t *testing.T) {
	//
	// Requires background checkout server process
	//
	// $ make build
	// $ ./build/checkout run --sqlite data/db --log-level debug --password 1234

	cl := client.New("http://0.0.0.0:8000")
	cl.AddAuthorizationHeader("1234")

	ctx := context.Background()

	reqIts := &model.AddItemsRequest{}
	for _, it := range testItems {
		reqIts.Items = append(reqIts.Items, it)
	}

	if err := cl.AddItems(ctx, reqIts); err != nil {
		t.Fatal(err)
	}
}
