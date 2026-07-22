package worker

import (
	"context"
	"sync"
	"testing"

	"go.uber.org/goleak"
)

func TestMain(m *testing.M) {
	goleak.VerifyTestMain(m)
}

// Start runs the loop; Stop cancels it and joins — goleak proves no leak.
func TestRunner_StartStop(t *testing.T) {
	var r Runner
	ran := make(chan struct{})
	var once sync.Once

	r.Start(func(ctx context.Context) {
		for ctx.Err() == nil {
			once.Do(func() { close(ran) })
		}
	})

	<-ran    // the loop has iterated at least once
	r.Stop() // cancels and joins
	r.Stop() // idempotent — must not panic or hang
}

// Stop before Start does nothing and does not panic.
func TestRunner_StopBeforeStart(t *testing.T) {
	var r Runner
	r.Stop()
}
