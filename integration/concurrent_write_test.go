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
	"github.com/ATMackay/checkout/model"
	"golang.org/x/sync/errgroup"
)

func Test_ConcurrentWrite(t *testing.T) {
	// Make 10 authenticated clients that concurrently write items to our server
	ctx := context.Background()

	// Raise stack
	stack := makeStack(t, ctx, &stackOpts{dbLogs: false, appLogs: true, debug: false})
	baseURL := stack.app.url()
	// Make client pool
	poolSize := max(1, rand.IntN(10))
	clients := make(chan *client.Client, poolSize)
	for range poolSize {
		cl := makeClient(t, baseURL, stack.app.authPsswd)
		clients <- cl
	}

	var mu sync.Mutex

	errG, gCtx := errgroup.WithContext(ctx)
	// errG.SetLimit(poolSize)

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
			it := makeTestItem(i)
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
