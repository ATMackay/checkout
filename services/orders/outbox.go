package orders

import (
	"github.com/ATMackay/checkout/event"
	"github.com/ATMackay/checkout/model"
)

// newOutboxItem maps an event onto an outbox row. It lives here, in the domain
// service, rather than in model or event so that neither of those packages has
// to know about the other: model stays free of an event import, and event stays
// transport- and storage-agnostic.
//
// Data holds the fully encoded event value, so the relay ships it to the broker
// verbatim (reconstructing the Event with event.Decode). Topic, PartitionKey,
// EventID and OccurredAt are lifted into columns so the relay can route and the
// row can be queried without decoding the payload.
func newOutboxItem(ev *event.Event) (*model.OutboxItem, error) {
	data, err := ev.Encode()
	if err != nil {
		return nil, err
	}
	return &model.OutboxItem{
		EventID:      ev.ID,
		Topic:        ev.Topic,
		PartitionKey: ev.Key,
		Data:         data,
		OccurredAt:   ev.OccurredAt,
	}, nil
}
