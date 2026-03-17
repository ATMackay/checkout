package model

import (
	"encoding/json"
	"fmt"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
)

type Order struct {
	ID        int             `json:"id,omitempty" gorm:"primaryKey;type:integer"`
	Reference string          `json:"reference" gorm:"column:reference;type:string;uniqueIndex"` // Unique random reference
	SKUList   string          `json:"sku_list" gorm:"column:sku_list;type:text"`
	Price     decimal.Decimal `json:"price" gorm:"column:price;type:numeric(12,2)"`
}

func (o *Order) TableName() string {
	return "orders"
}

type Orders []Order

// GetSKUList returns the SKU list as a slice of strings
func (o *Order) GetSKUList() ([]string, error) {
	var skuList []string
	if err := json.Unmarshal([]byte(o.SKUList), &skuList); err != nil {
		return nil, fmt.Errorf("failed to unmarshal SKU list: %w", err)
	}
	return skuList, nil
}

// SetSKUList sets the SKU list from a slice of strings
func (o *Order) SetSKUList(skuList []string) error {
	skuListJSON, err := json.Marshal(skuList)
	if err != nil {
		return fmt.Errorf("failed to marshal SKU list: %w", err)
	}
	o.SKUList = string(skuListJSON)
	return nil
}

// GenerateReference generates a random reference using UUID
func GenerateReference() string {
	return uuid.New().String()
}
