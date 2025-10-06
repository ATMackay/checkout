package database

import (
	"context"
	"fmt"

	"github.com/ATMackay/checkout/model"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
	"gorm.io/gorm/logger"
)

//go:generate mockgen -source database.go -destination ./mock/database_mock.go -package mock database
type Database interface {
	HealthChecker
	InventoryStore
	OrderStore
}

type HealthChecker interface {
	Ping(ctx context.Context) error
}

type InventoryStore interface {
	UpsertItems(ctx context.Context, items []*model.Item) ([]*model.Item, error)
	ListItems(ctx context.Context) ([]*model.Item, error) // TODO - add pagination support
	GetItemByName(ctx context.Context, name string) (*model.Item, error)
	GetItemBySKU(ctx context.Context, sku string) (*model.Item, error)
	GetItemsBySKU(ctx context.Context, sku []string) ([]*model.Item, error)
}

type OrderStore interface {
	AddOrder(ctx context.Context, o *model.Order) error
	GetOrders(ctx context.Context) ([]*model.Order, error)
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
		return nil, fmt.Errorf("failed to auto migrate gormDeposit: %w", err)
	}
	if err := db.AutoMigrate(&model.Order{}); err != nil {
		return nil, fmt.Errorf("failed to auto migrate gormDeposit: %w", err)
	}
	return &GormDB{db}, nil
}

func deleteStorage(db *gorm.DB) error {
	if err := db.Migrator().DropTable(&model.Item{}); err != nil {
		return fmt.Errorf("failed to drop table gormDeposit: %w", err)
	}
	if err := db.Migrator().DropTable(&model.Order{}); err != nil {
		return fmt.Errorf("failed to drop table gormDeposit: %w", err)
	}
	return nil
}

func (g *GormDB) Ping(ctx context.Context) error {
	// Get the underlying *sql.DB
	sqlDB, err := g.db.DB()
	if err != nil {
		return fmt.Errorf("failed to get underlying sql.DB: %w", err)
	}
	return sqlDB.PingContext(ctx)
}

func (g *GormDB) GetItemByName(ctx context.Context, name string) (*model.Item, error) {
	return g.getItemByKey(ctx, "name", name)
}

func (g *GormDB) GetItemBySKU(ctx context.Context, sku string) (*model.Item, error) {
	return g.getItemByKey(ctx, "sku", sku)
}

func (g *GormDB) getItemByKey(ctx context.Context, key, name string) (*model.Item, error) {

	var it *model.Item

	if err := g.db.WithContext(ctx).Where(key, name).Find(&it).Error; err != nil {
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

	db := g.db.WithContext(ctx).Debug()

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

func (g *GormDB) AddOrder(ctx context.Context, o *model.Order) error {
	return g.db.WithContext(ctx).Create(o).Error
}

func (g *GormDB) GetOrders(ctx context.Context) ([]*model.Order, error) {

	var os []*model.Order

	if err := g.db.WithContext(ctx).Order("id DESC").Find(&os).Error; err != nil {
		return nil, err
	}

	return os, nil
}
