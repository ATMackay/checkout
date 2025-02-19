package database

import (
	"encoding/json"
	"errors"

	"github.com/google/uuid"
)

type InventoryItem struct {
	ID                int     `gorm:"primaryKey;type:integer"`
	SKU               string  `gorm:"column:sku;type:string;unique"`
	Name              string  `gorm:"column:name;type:string;unique"`
	Price             float64 `gorm:"column:price;type:double"`
	InventoryQuantity int     `gorm:"column:inventory_quantity;type:integer"`
}

func (i *InventoryItem) TableName() string {
	return "inventory"
}

type Order struct {
	ID        int     `gorm:"primaryKey;type:integer"`
	Reference string  `gorm:"column:reference;type:string;uniqueIndex"` // Unique random reference
	SKUList   string  `gorm:"column:sku_list;type:text"`
	Price     float64 `gorm:"column:price;type:double"`
}

func (o *Order) TableName() string {
	return "orders"
}

// GetSKUList returns the SKU list as a slice of strings
func (o *Order) GetSKUList() ([]string, error) {
	var skuList []string
	if err := json.Unmarshal([]byte(o.SKUList), &skuList); err != nil {
		return nil, errors.New("failed to unmarshal SKU list")
	}
	return skuList, nil
}

// SetSKUList sets the SKU list from a slice of strings
func (o *Order) SetSKUList(skuList []string) error {
	skuListJSON, err := json.Marshal(skuList)
	if err != nil {
		return errors.New("failed to marshal SKU list")
	}
	o.SKUList = string(skuListJSON)
	return nil
}

// GenerateReference generates a random reference using UUID
func GenerateReference() string {
	return uuid.New().String()
}
