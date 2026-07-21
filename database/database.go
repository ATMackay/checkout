package database

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/ATMackay/checkout/model"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
	"gorm.io/gorm/logger"
)

//go:generate mockgen -destination ./mock/database_mock.go -package mock github.com/ATMackay/checkout/database Database,HealthChecker,InventoryStore,OrderStore,OutboxStore
type Database interface {
	HealthChecker
	InventoryStore
	OrderStore
	OutboxStore
	Transaction(ctx context.Context, fn func(Database) error) error
}

type HealthChecker interface {
	Ping(ctx context.Context) error
}

var _ Database = (*GormDB)(nil)

type GormDB struct {
	db *gorm.DB
}

func NewGormDB(d gorm.Dialector, recreateSchema bool) (*GormDB, error) {
	db, err := gorm.Open(d, &gorm.Config{
		Logger:      logger.Discard,
		PrepareStmt: true,
	})
	if err != nil {
		return nil, err
	}
	if recreateSchema {
		if err := deleteStorage(db); err != nil {
			return nil, err
		}
	}

	return newStorage(db)
}

func newStorage(db *gorm.DB) (*GormDB, error) {
	if err := db.AutoMigrate(&model.Item{}); err != nil {
		return nil, fmt.Errorf("failed to auto migrate inventory table: %w", err)
	}
	if err := db.AutoMigrate(&model.Order{}); err != nil {
		return nil, fmt.Errorf("failed to auto migrate orders table: %w", err)
	}
	if err := db.AutoMigrate(&model.OutboxItem{}); err != nil {
		return nil, fmt.Errorf("failed to auto migrate outbox table: %w", err)
	}
	return &GormDB{db}, nil
}

func deleteStorage(db *gorm.DB) error {
	if err := db.Migrator().DropTable(&model.Item{}); err != nil {
		return fmt.Errorf("failed to drop table inventory: %w", err)
	}
	if err := db.Migrator().DropTable(&model.Order{}); err != nil {
		return fmt.Errorf("failed to drop table orders: %w", err)
	}
	if err := db.Migrator().DropTable(&model.OutboxItem{}); err != nil {
		return fmt.Errorf("failed to drop table outbox: %w", err)
	}
	return nil
}

// HealthChecker Implementation

func (g *GormDB) Ping(ctx context.Context) error {
	// Get the underlying *sql.DB
	sqlDB, err := g.db.DB()
	if err != nil {
		return fmt.Errorf("failed to get underlying sql.DB: %w", err)
	}
	return sqlDB.PingContext(ctx)
}

// InventoryStore Implementation

func (g *GormDB) GetItemByName(ctx context.Context, name string) (*model.Item, error) {
	return g.getItemByKey(ctx, "name", name)
}

func (g *GormDB) GetItemBySKU(ctx context.Context, sku string) (*model.Item, error) {
	return g.getItemByKey(ctx, "sku", sku)
}

func (g *GormDB) getItemByKey(ctx context.Context, key, name string) (*model.Item, error) {

	var it *model.Item

	if err := g.db.WithContext(ctx).Where(key+" = ?", name).Find(&it).Error; err != nil {
		return nil, err
	}

	return it, nil
}

type searchOpts struct {
	skus []string
}

func (g *GormDB) ListItems(ctx context.Context) ([]*model.Item, error) {
	return g.getItems(ctx, nil)
}

func (g *GormDB) GetItemsBySKU(ctx context.Context, skus []string) ([]*model.Item, error) {
	return g.getItems(ctx, &searchOpts{skus: skus})
}

func (g *GormDB) getItems(ctx context.Context, opts *searchOpts) ([]*model.Item, error) {
	var it []*model.Item

	db := g.db.WithContext(ctx)

	if opts != nil && len(opts.skus) > 0 {
		db = db.Where("sku IN ?", opts.skus)
	}

	if err := db.Find(&it).Error; err != nil {
		return nil, err
	}

	return it, nil
}

func (g *GormDB) UpsertItems(ctx context.Context, items []*model.Item) ([]*model.Item, error) {
	if err := g.db.WithContext(ctx).Clauses(clause.OnConflict{
		UpdateAll: true,
	}).Create(items).Error; err != nil {
		return nil, err
	}
	return items, nil
}

// OrderStore Implementation

func (g *GormDB) AddOrder(ctx context.Context, o *model.Order) error {
	return g.db.WithContext(ctx).Create(o).Error
}

func (g *GormDB) GetOrders(ctx context.Context, customerID string) ([]*model.Order, error) {

	var os []*model.Order

	if err := g.db.WithContext(ctx).Order("id DESC").Where("customer_id = ?", customerID).Find(&os).Error; err != nil {
		return nil, err
	}

	return os, nil
}

// OutboxStore Implementation

// ErrOutboxItemNotFound is returned when a strict update matches no row.
var ErrOutboxItemNotFound = errors.New("outbox item not found")

func (g *GormDB) AddOutboxItems(ctx context.Context, items []*model.OutboxItem) error {
	if len(items) == 0 {
		return nil
	}
	return g.db.WithContext(ctx).Create(items).Error
}

func (g *GormDB) GetOutboxItems(ctx context.Context, q *OutboxQuery) ([]*model.OutboxItem, error) {
	db := g.db.WithContext(ctx)
	if q != nil {
		if q.OnlyUnpublished {
			db = db.Where("published_at IS NULL")
		}
		if q.OnlyUndelivered {
			db = db.Where("delivered_at IS NULL")
		}
		if q.Limit > 0 {
			db = db.Limit(q.Limit)
		}
	}

	var items []*model.OutboxItem
	if err := db.Order("id ASC").Find(&items).Error; err != nil {
		return nil, err
	}
	return items, nil
}

func (g *GormDB) SetPublishedAt(ctx context.Context, id int64, t time.Time) error {
	return g.setOutboxTimestamp(ctx, id, "published_at", t)
}

func (g *GormDB) SetDeliveredAt(ctx context.Context, id int64, t time.Time) error {
	return g.setOutboxTimestamp(ctx, id, "delivered_at", t)
}

// setOutboxTimestamp writes t to a single item's timestamp column. column is an
// internal literal, never caller input, so it is not an injection surface. The
// update is strict: matching no row is an error, so a bad ID cannot pass
// silently.
func (g *GormDB) setOutboxTimestamp(ctx context.Context, id int64, column string, t time.Time) error {
	res := g.db.WithContext(ctx).
		Model(&model.OutboxItem{}).
		Where("id = ?", id).
		Update(column, t)
	if res.Error != nil {
		return fmt.Errorf("set %s for outbox item %d: %w", column, id, res.Error)
	}
	if res.RowsAffected != 1 {
		return fmt.Errorf("set %s for outbox item %d: %w", column, id, ErrOutboxItemNotFound)
	}
	return nil
}

func (g *GormDB) Transaction(ctx context.Context, fn func(Database) error) error {
	return g.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		return fn(&GormDB{db: tx})
	})
}
