//go:build !integration

package notifier

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/ATMackay/checkout/database"
	dbmock "github.com/ATMackay/checkout/database/mock"
	"github.com/ATMackay/checkout/event"
	msgmock "github.com/ATMackay/checkout/messaging/mock"
	"github.com/ATMackay/checkout/model"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
)

type recordingSink struct{ got []*model.Notification }

func (r *recordingSink) Write(_ context.Context, n *model.Notification) error {
	r.got = append(r.got, n)
	return nil
}

func orderEvent(ref string) *event.Event {
	return event.New("orders.created", ref, &model.Order{Reference: ref, CustomerID: "c-" + ref})
}

// dispatch renders the event to the sink and marks the outbox row delivered.
func Test_Dispatch(t *testing.T) {
	ctrl := gomock.NewController(t)
	store := dbmock.NewMockDatabase(ctrl)
	sink := &recordingSink{}
	s := NewService(nil, store, msgmock.NewMockConsumer(ctrl), sink)

	ev := orderEvent("ref-1")
	store.EXPECT().SetDeliveredByEventID(gomock.Any(), ev.ID, gomock.Any()).Return(nil)

	s.dispatch(context.Background(), ev)

	require.Len(t, sink.got, 1)
	require.Equal(t, "ref-1", sink.got[0].Reference)
	require.Equal(t, "c-ref-1", sink.got[0].CustomerID)
	require.True(t, sink.got[0].Delivered)
}

// The /v1/notifications endpoint maps outbox rows to notifications and passes
// the undelivered filter through to the store query.
func Test_NotificationsEndpoint(t *testing.T) {
	ctrl := gomock.NewController(t)
	store := dbmock.NewMockDatabase(ctrl)

	mkItem := func(ref string, delivered bool) *model.OutboxItem {
		ev := orderEvent(ref)
		data, err := ev.Encode()
		require.NoError(t, err)
		it := &model.OutboxItem{EventID: ev.ID, Topic: ev.Topic, PartitionKey: ev.Key, Data: data}
		if delivered {
			now := time.Now()
			it.DeliveredAt = &now
		}
		return it
	}

	router := NewService(nil, store, msgmock.NewMockConsumer(ctrl), terminalSink{}).RegisterHandlers()

	get := func(t *testing.T, path string) []*model.Notification {
		t.Helper()
		rr := httptest.NewRecorder()
		router.ServeHTTP(rr, httptest.NewRequest(http.MethodGet, path, nil))
		require.Equal(t, http.StatusOK, rr.Code)
		var got []*model.Notification
		require.NoError(t, json.Unmarshal(rr.Body.Bytes(), &got))
		return got
	}

	t.Run("all", func(t *testing.T) {
		store.EXPECT().GetOutboxItems(gomock.Any(), &database.OutboxQuery{OnlyUndelivered: false}).
			Return([]*model.OutboxItem{mkItem("a", true), mkItem("b", false)}, nil)
		require.Len(t, get(t, NotificationsEndPnt), 2)
	})

	t.Run("undelivered only", func(t *testing.T) {
		store.EXPECT().GetOutboxItems(gomock.Any(), &database.OutboxQuery{OnlyUndelivered: true}).
			Return([]*model.OutboxItem{mkItem("b", false)}, nil)
		got := get(t, NotificationsEndPnt+"?undelivered=true")
		require.Len(t, got, 1)
		require.False(t, got[0].Delivered)
	})
}
