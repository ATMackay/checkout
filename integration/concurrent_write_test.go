//go:build integration

package integration

//func Test_ConcurrentWrite(t *testing.T) {
//	ctx := context.Background()
//	stack := makeStack(t, ctx, &stackOpts{dbLogs: false, buildFromDockerfile: true, appLogs: true})
//
//	// Create simple client pool for concurrent writes
//	N := 10
//	clientChan := make(chan *client.Client, N)
//	for range N {
//		cl, err := client.New(stack.app.url())
//		if err != nil {
//			t.Fatal(err)
//		}
//		cl.AddAuthorizationHeader(stack.app.authPsswd)
//		clientChan <- cl
//	}
//
//	// Spawn go routines to write new items to the database concurrently
//	writeCount := 1000
//	errG, gCtx := errgroup.WithContext(ctx)
//	errG.SetLimit(N)
//
//	randSrc := rand.New(rand.NewPCG(uint64(time.Now().UnixNano()), uint64(time.Now().UnixNano())))
//
//	duration := new(atomic.Int64)
//
//	var mu sync.Mutex
//	for i := range writeCount {
//		i := i
//		errG.Go(func() error {
//			cl := <-clientChan
//			defer func() {
//				clientChan <- cl
//			}()
//
//			// generate random item
//			mu.Lock()
//			name := fmt.Sprintf("Item-%04d", i)
//			sku := randomSKU(randSrc)
//			price := decimal.NewFromFloat(randSrc.Float64()*100 + 1) // 1–100
//			qty := randSrc.IntN(50) + 1                              // 1–50
//			mu.Unlock()
//
//			start := time.Now()
//			if err := cl.AddItems(gCtx, &model.AddItemsRequest{Items: []*model.Item{{Name: name, SKU: sku, Price: price, InventoryQuantity: qty}}}); err != nil {
//				return fmt.Errorf("entry %d: %v", i, err)
//			}
//			duration.Add(time.Since(start).Nanoseconds())
//
//			return nil
//		})
//	}
//	if err := errG.Wait(); err != nil {
//		t.Fatal(err)
//	}
//	t.Logf("Total execution time for %v writes: %v", writeCount, time.Duration(duration.Load()))
//	t.Logf("Average write duration: %v ms", decimal.NewFromInt(duration.Load()).Div(decimal.NewFromInt(int64(writeCount*1000*1000))))
//}
