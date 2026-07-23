// Package worker provides the goroutine-lifecycle boilerplate shared by every
// background process in the app (the orders outbox relay, the notifier consume
// loop). It is deliberately tiny: the value is having the error-prone
// concurrency plumbing — a single goroutine, context cancellation, WaitGroup
// join, idempotent stop — written and tested once.
package worker

import (
	"context"
	"sync"
)

// Runner supervises one background goroutine. Embed it in a service and drive it
// with Start/Stop; the loop function receives a context that is cancelled when
// Stop is called, and Stop blocks until the goroutine has returned.
type Runner struct {
	cancel context.CancelFunc
	wg     sync.WaitGroup
	once   sync.Once
}

// Start launches loop in a new goroutine. loop must return when its context is
// cancelled (Stop cancels it). Call Start once.
func (r *Runner) Start(loop func(context.Context)) {
	ctx, cancel := context.WithCancel(context.Background())
	r.cancel = cancel
	r.wg.Go(func() {
		loop(ctx)
	})
}

// Stop cancels the loop's context and waits for the goroutine to exit. It is
// idempotent and safe to call from multiple goroutines; calling it before Start
// (or twice) is a no-op that cannot panic.
func (r *Runner) Stop() {
	r.once.Do(func() {
		if r.cancel != nil {
			r.cancel()
		}
	})
	r.wg.Wait()
}
