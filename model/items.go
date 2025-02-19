package model

import (
	"fmt"
	"regexp"
)

type Item struct {
	ID                int     `json:"id,omitempty" gorm:"primaryKey;type:integer"`
	SKU               string  `json:"sku" gorm:"column:sku;type:string;unique"`
	Name              string  `json:"name" gorm:"column:name;type:string;unique"`
	Price             float64 `json:"price" gorm:"column:price;type:double"`
	InventoryQuantity int     `json:"inventory_quantity" gorm:"column:inventory_quantity;type:integer"`
}

func (i *Item) TableName() string {
	return "inventory"
}

type AddItemsRequest struct {
	Items []*Item `json:"items"`
}

func (i *Item) Validate() error {
	if i.Name == "" {
		return fmt.Errorf("item name must be a non-empty string")
	}
	if !IsSKU(i.SKU) {
		return fmt.Errorf("item SKU must be a valid SKU of length 6")
	}
	if i.Price < 0 {
		return fmt.Errorf("invalid price less than 0")
	}
	if i.InventoryQuantity < 1 {
		return fmt.Errorf("invalid inventory_quantity less than 1")
	}
	return nil
}

// IsSKU checks if the input string is an SKU
func IsSKU(input string) bool {
	// Define a regex pattern for SKUs (alphanumeric, no spaces, 6 characters)
	skuPattern := `^[a-zA-Z0-9]{6,6}$`
	matched, err := regexp.MatchString(skuPattern, input)
	if err != nil {
		return false
	}
	return matched
}

type PurchaseItemsRequest struct {
	SKUs []string `json:"skus"`
}

type PurchaseItemsResponse struct {
	OrderReference string  `json:"order_reference"`
	Cost           float64 `json:"cost"`
}
