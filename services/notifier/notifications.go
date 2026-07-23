package notifier

import (
	"net/http"

	"github.com/ATMackay/checkout/database"
	"github.com/ATMackay/checkout/event"
	"github.com/ATMackay/checkout/httpserver"
	"github.com/ATMackay/checkout/model"
	"github.com/julienschmidt/httprouter"
)

// Notifications returns the notifications derived from the outbox. With
// ?undelivered=true only rows the notifier has not yet delivered are returned.
func (h *Service) Notifications() httprouter.Handle {
	return httpserver.Handle(func(r *http.Request, _ httprouter.Params) (any, error) {
		undelivered := r.URL.Query().Get("undelivered") == "true"

		items, err := h.store.GetOutboxItems(r.Context(), &database.OutboxQuery{OnlyUndelivered: undelivered})
		if err != nil {
			return nil, err
		}

		notifications := make([]*model.Notification, 0, len(items))
		for _, item := range items {
			ev, err := event.Decode(item.Topic, item.PartitionKey, item.Data)
			if err != nil {
				return nil, err
			}
			n, err := notificationFromEvent(ev, item.DeliveredAt != nil)
			if err != nil {
				return nil, err
			}
			notifications = append(notifications, n)
		}
		return notifications, nil
	})
}
