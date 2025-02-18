package database

import (
	"context"
	"fmt"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
	"gorm.io/gorm/logger"
)

//go:generate mockgen -source database.go -destination ../mock/database_mock.go -package mock database
type Database interface {
	HealthChecker
	InventoryStore
	OrderStore
}

type HealthChecker interface {
	Ping(ctx context.Context) error
}

type InventoryStore interface {
	AddItems(ctx context.Context, items []*InventoryItem) error
	GetItemByName(ctx context.Context, name string) (*InventoryItem, error)
	GetItemBySKU(ctx context.Context, sku string) (*InventoryItem, error)
	GetItemsBySKU(ctx context.Context, sku []string) ([]*InventoryItem, error)
}

type OrderStore interface {
	AddOrder(ctx context.Context, o *Order) error
	GetOrders(ctx context.Context) ([]*Order, error)
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
	if err := db.AutoMigrate(&InventoryItem{}); err != nil {
		return nil, fmt.Errorf("failed to auto migrate gormDeposit: %w", err)
	}
	if err := db.AutoMigrate(&Order{}); err != nil {
		return nil, fmt.Errorf("failed to auto migrate gormDeposit: %w", err)
	}
	return &GormDB{db}, nil
}

func deleteStorage(db *gorm.DB) error {
	if err := db.Migrator().DropTable(&InventoryItem{}); err != nil {
		return fmt.Errorf("failed to drop table gormDeposit: %w", err)
	}
	if err := db.Migrator().DropTable(&Order{}); err != nil {
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

func (g *GormDB) GetItemByName(ctx context.Context, name string) (*InventoryItem, error) {
	return g.getItemByKey(ctx, "name", name)
}

func (g *GormDB) GetItemBySKU(ctx context.Context, sku string) (*InventoryItem, error) {
	return g.getItemByKey(ctx, "sku", sku)
}

func (g *GormDB) getItemByKey(ctx context.Context, key, name string) (*InventoryItem, error) {

	var it *InventoryItem

	if err := g.db.WithContext(ctx).Where(key, name).Find(it).Error; err != nil {
		return nil, err
	}

	return it, nil
}

func (g *GormDB) GetItemsBySKU(ctx context.Context, skus []string) ([]*InventoryItem, error) {
	var it []*InventoryItem

	if err := g.db.WithContext(ctx).Where("sku", skus).Scan(it).Error; err != nil {
		return nil, err
	}

	return it, nil
}

func (g *GormDB) AddItems(ctx context.Context, items []*InventoryItem) error {
	if err := g.db.WithContext(ctx).Clauses(clause.OnConflict{
		UpdateAll: true,
	}).Create(items).Error; err != nil {
		return err
	}
	return nil
}

func (g *GormDB) AddOrder(ctx context.Context, o *Order) error {
	return g.db.WithContext(ctx).Create(o).Error
}

func (g *GormDB) GetOrders(ctx context.Context) ([]*Order, error) {

	var os []*Order

	if err := g.db.WithContext(ctx).Order("id DESC").Scan(os).Error; err != nil {
		return nil, err
	}

	return os, nil
}
