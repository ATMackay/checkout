package model

import "time"

// Notification is a rendered order event ready to be delivered to a client.
type Notification struct {
	EventID    string    `json:"event_id"`
	Reference  string    `json:"reference"`
	CustomerID string    `json:"customer_id"`
	OccurredAt time.Time `json:"occurred_at"`
	Delivered  bool      `json:"delivered"`
}
