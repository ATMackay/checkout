package database

import (
	"context"
	"time"

	"github.com/ATMackay/checkout/model"
)

type InventoryStore interface {
	UpsertItems(ctx context.Context, items []*model.Item) ([]*model.Item, error)
	ListItems(ctx context.Context) ([]*model.Item, error) // TODO - add pagination support
	GetItemByName(ctx context.Context, name string) (*model.Item, error)
	GetItemBySKU(ctx context.Context, sku string) (*model.Item, error)
	GetItemsBySKU(ctx context.Context, sku []string) ([]*model.Item, error)
}

type OrderStore interface {
	AddOrder(ctx context.Context, o *model.Order) error
	GetOrders(ctx context.Context, userID string) ([]*model.Order, error)
}

// OutboxStore persists and drains transactional outbox rows.
type OutboxStore interface {
	// AddOutboxItems enqueues items. Intended to run inside the same
	// transaction as the business write it accompanies.
	AddOutboxItems(ctx context.Context, items []*model.OutboxItem) error

	// GetOutboxItems reads items, optionally filtered to those not yet
	// published or delivered. Results are ordered by ID ascending (enqueue
	// order).
	GetOutboxItems(ctx context.Context, q *OutboxQuery) ([]*model.OutboxItem, error)

	// SetPublishedAt strictly marks one item published: it errors with
	// ErrOutboxItemNotFound if no row has that ID.
	SetPublishedAt(ctx context.Context, id int64, t time.Time) error

	// SetDeliveredAt strictly marks one item delivered, with the same
	// not-found semantics as SetPublishedAt.
	SetDeliveredAt(ctx context.Context, id int64, t time.Time) error
}

// OutboxQuery filters an outbox read. The zero value selects everything.
type OutboxQuery struct {
	// OnlyUnpublished restricts to rows not yet sent to the broker
	// (published_at IS NULL). This is the relay's claim filter.
	OnlyUnpublished bool
	// OnlyUndelivered restricts to rows not yet marked delivered
	// (delivered_at IS NULL).
	OnlyUndelivered bool
	// Limit caps the batch size; <= 0 means no limit.
	Limit int
}
