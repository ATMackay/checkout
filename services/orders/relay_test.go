//go:build !integration

package orders

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/ATMackay/checkout/database"
	dbmock "github.com/ATMackay/checkout/database/mock"
	"github.com/ATMackay/checkout/event"
	msgmock "github.com/ATMackay/checkout/messaging/mock"
	"github.com/ATMackay/checkout/model"
	"go.uber.org/goleak"
	"go.uber.org/mock/gomock"
)

func TestMain(m *testing.M) {
	goleak.VerifyTestMain(m)
}

// testItem builds an outbox row carrying an encoded event, the way the purchase
// handler does.
func testItem(t *testing.T, id int64, key string) *model.OutboxItem {
	t.Helper()
	item, err := newOutboxItem(event.New(event.TopicOrderCreated, key, map[string]string{"ref": key}))
	if err != nil {
		t.Fatalf("build outbox item: %v", err)
	}
	item.ID = id
	return item
}

// unpublishedBatch is the query the relay issues each scan.
func unpublishedBatch() *database.OutboxQuery {
	return &database.OutboxQuery{OnlyUnpublished: true, Limit: defaultBatchSize}
}

// drain is exercised directly (rather than via the ticker goroutine) so the mock
// expectations are exact: one scan of two rows publishes and marks both.
func TestOutboxRelayer_DrainPublishesAndMarks(t *testing.T) {
	ctrl := gomock.NewController(t)
	store := dbmock.NewMockOutboxStore(ctrl)
	pub := msgmock.NewMockPublisher(ctrl)

	item1 := testItem(t, 1, "ref-1")
	item2 := testItem(t, 2, "ref-2")

	store.EXPECT().GetOutboxItems(gomock.Any(), unpublishedBatch()).
		Return([]*model.OutboxItem{item1, item2}, nil)

	var published []*event.Event
	pub.EXPECT().Publish(gomock.Any(), gomock.Any()).
		DoAndReturn(func(_ context.Context, ev *event.Event) error {
			published = append(published, ev)
			return nil
		}).Times(2)

	store.EXPECT().SetPublishedAt(gomock.Any(), int64(1), gomock.Any()).Return(nil)
	store.EXPECT().SetPublishedAt(gomock.Any(), int64(2), gomock.Any()).Return(nil)

	NewOutboxRelayer(store, pub).drain(context.Background())

	if len(published) != 2 {
		t.Fatalf("published %d events, want 2", len(published))
	}
	// Routing metadata survives the store round-trip.
	if published[0].Topic != event.TopicOrderCreated || published[0].Key != "ref-1" {
		t.Errorf("event 0 = {topic:%s key:%s}, want {orders.created ref-1}", published[0].Topic, published[0].Key)
	}
}

// A publish failure must leave the row unpublished: no SetPublishedAt is
// expected, so gomock fails the test if the relay marks it anyway.
func TestOutboxRelayer_DrainPublishFailureLeavesRowUnpublished(t *testing.T) {
	ctrl := gomock.NewController(t)
	store := dbmock.NewMockOutboxStore(ctrl)
	pub := msgmock.NewMockPublisher(ctrl)

	store.EXPECT().GetOutboxItems(gomock.Any(), unpublishedBatch()).
		Return([]*model.OutboxItem{testItem(t, 1, "ref-1")}, nil)
	pub.EXPECT().Publish(gomock.Any(), gomock.Any()).Return(errors.New("broker down"))

	NewOutboxRelayer(store, pub).drain(context.Background())
}

// A scan error ends the cycle without publishing.
func TestOutboxRelayer_DrainScanError(t *testing.T) {
	ctrl := gomock.NewController(t)
	store := dbmock.NewMockOutboxStore(ctrl)
	pub := msgmock.NewMockPublisher(ctrl)

	store.EXPECT().GetOutboxItems(gomock.Any(), unpublishedBatch()).
		Return(nil, errors.New("db down"))

	NewOutboxRelayer(store, pub).drain(context.Background())
}

// Stop is idempotent and must not panic on a second call. A long poll interval
// keeps the ticker from firing, so the store is never touched.
func TestOutboxRelayer_StopIsIdempotent(t *testing.T) {
	ctrl := gomock.NewController(t)
	store := dbmock.NewMockOutboxStore(ctrl)
	pub := msgmock.NewMockPublisher(ctrl)

	pub.EXPECT().Ping(gomock.Any()).Return(nil)
	// Stop closes the publisher on each call; the second Stop is a no-op for the
	// quit channel but still calls Close.
	pub.EXPECT().Close().Return(nil).Times(2)

	r := NewOutboxRelayer(store, pub, WithPollInterval(time.Hour))
	if err := r.Start(context.Background()); err != nil {
		t.Fatalf("start: %v", err)
	}
	if err := r.Stop(); err != nil {
		t.Fatalf("first stop: %v", err)
	}
	if err := r.Stop(); err != nil {
		t.Fatalf("second stop: %v", err)
	}
}

// A failing broker ping aborts Start before any goroutine spawns.
func TestOutboxRelayer_StartFailsWhenBrokerUnreachable(t *testing.T) {
	ctrl := gomock.NewController(t)
	store := dbmock.NewMockOutboxStore(ctrl)
	pub := msgmock.NewMockPublisher(ctrl)

	pub.EXPECT().Ping(gomock.Any()).Return(errors.New("unreachable"))

	r := NewOutboxRelayer(store, pub, WithPollInterval(time.Hour))
	if err := r.Start(context.Background()); err == nil {
		t.Fatal("expected Start to fail when broker Ping fails")
	}
}
