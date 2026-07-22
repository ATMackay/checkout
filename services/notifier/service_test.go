//go:build !integration

package notifier

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	dbmock "github.com/ATMackay/checkout/database/mock"
	"github.com/ATMackay/checkout/event"
	msgmock "github.com/ATMackay/checkout/messaging/mock"
	"github.com/stretchr/testify/assert"
	"go.uber.org/goleak"
	"go.uber.org/mock/gomock"
)

func TestMain(m *testing.M) {
	goleak.VerifyTestMain(m)
}

// Start launches the consume loop; Stop cancels it and waits. goleak (TestMain)
// proves the goroutine is gone — a Poll that blocks forever would be caught.
func TestService_StartStop(t *testing.T) {
	ctrl := gomock.NewController(t)
	store := dbmock.NewMockDatabase(ctrl)
	consumer := msgmock.NewMockConsumer(ctrl)

	consumer.EXPECT().Ping(gomock.Any()).Return(nil) // Start
	// Poll blocks until the loop's context is cancelled by Stop.
	consumer.EXPECT().Poll(gomock.Any()).DoAndReturn(func(ctx context.Context) ([]*event.Event, error) {
		<-ctx.Done()
		return nil, ctx.Err()
	}).AnyTimes()

	s := NewService(nil, store, consumer)
	if err := s.Start(context.Background()); err != nil {
		t.Fatalf("start: %v", err)
	}
	if err := s.Stop(); err != nil {
		t.Fatalf("stop: %v", err)
	}
	// Second Stop is a no-op and must not panic.
	if err := s.Stop(); err != nil {
		t.Fatalf("second stop: %v", err)
	}
}

// Start aborts before spawning the loop if the broker is unreachable.
func TestService_StartFailsWhenBrokerDown(t *testing.T) {
	ctrl := gomock.NewController(t)
	store := dbmock.NewMockDatabase(ctrl)
	consumer := msgmock.NewMockConsumer(ctrl)
	consumer.EXPECT().Ping(gomock.Any()).Return(assert.AnError)

	s := NewService(nil, store, consumer)
	if err := s.Start(context.Background()); err == nil {
		t.Fatal("expected Start to fail when broker Ping fails")
	}
}

// Health reports 503 when the consumer probe fails, even though the database is
// healthy — proving the notifier wires its own checks.
func TestService_HealthReflectsConsumer(t *testing.T) {
	ctrl := gomock.NewController(t)
	store := dbmock.NewMockDatabase(ctrl)
	consumer := msgmock.NewMockConsumer(ctrl)

	store.EXPECT().Ping(gomock.Any()).Return(nil)
	consumer.EXPECT().Ping(gomock.Any()).Return(assert.AnError)

	router := NewService(nil, store, consumer).RegisterHandlers()
	rr := httptest.NewRecorder()
	router.ServeHTTP(rr, httptest.NewRequest(http.MethodGet, HealthEndPnt, nil))

	assert.Equal(t, http.StatusServiceUnavailable, rr.Code)
}
