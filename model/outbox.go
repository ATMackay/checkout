package model

import "time"

// OutboxItem is one row of the transactional outbox. It is written in the same
// database transaction as the business change it describes, then drained to the
// message broker by a separate relay. The relay routes on Topic and
// PartitionKey and ships Data verbatim as the record value.
type OutboxItem struct {
	// ID is assigned by the database sequence, not the application. It gives
	// the relay a stable, monotonic handle for a specific row. It is a physical
	// ordering handle only — never the broker key or the dedup key; EventID is
	// the business identity.
	ID int64 `json:"id,omitempty" gorm:"primaryKey;autoIncrement"`

	// EventID is the business identity consumers deduplicate on. Unique so a
	// retried producer transaction cannot enqueue the same event twice.
	EventID string `json:"event_id" gorm:"column:event_id;uniqueIndex"`

	// Topic and PartitionKey are the broker routing metadata. They are columns
	// rather than fields inside Data so the relay can route without decoding the
	// payload.
	Topic        string `json:"topic" gorm:"column:topic"`
	PartitionKey string `json:"partition_key" gorm:"column:partition_key"`

	// Data is the encoded event value, shipped to the broker as-is.
	Data []byte `json:"data" gorm:"column:data"`

	// OccurredAt is when the event happened; CreatedAt is when the row was
	// enqueued. The relay does not read these, but they support retention and
	// debugging.
	OccurredAt time.Time `json:"occurred_at" gorm:"column:occurred_at"`
	CreatedAt  time.Time `json:"created_at" gorm:"column:created_at;autoCreateTime"`

	// PublishedAt and DeliveredAt are nil until the respective milestone is
	// reached. They MUST be pointers: a zero time.Time is a real timestamp, not
	// "not yet", so a non-pointer field could never express an unpublished row
	// and the relay's "WHERE published_at IS NULL" scan would match nothing.
	PublishedAt *time.Time `json:"published_at,omitempty" gorm:"column:published_at;index"`
	DeliveredAt *time.Time `json:"delivered_at,omitempty" gorm:"column:delivered_at"`
}

func (o *OutboxItem) TableName() string {
	return "outbox"
}

type OutboxItems []OutboxItem
