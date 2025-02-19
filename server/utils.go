package server

import (
	"github.com/ATMackay/checkout/database"
	"github.com/ATMackay/checkout/model"
)

func itemJsonToGORM(it *model.Item) *database.InventoryItem {
	return &database.InventoryItem{
		Name:              it.Name,
		SKU:               it.SKU,
		Price:             it.Price,
		InventoryQuantity: it.InventoryQuantity,
	}
}

func itemGormToJSON(dbIt *database.InventoryItem) *model.Item {
	return &model.Item{
		Name:              dbIt.Name,
		SKU:               dbIt.SKU,
		Price:             dbIt.Price,
		InventoryQuantity: dbIt.InventoryQuantity,
	}
}

func itemsGormToJSON(dbIts []*database.InventoryItem) []*model.Item {
	its := make([]*model.Item, len(dbIts))
	for i, dbIt := range dbIts {
		its[i] = &model.Item{
			Name:              dbIt.Name,
			SKU:               dbIt.SKU,
			Price:             dbIt.Price,
			InventoryQuantity: dbIt.InventoryQuantity,
		}
	}
	return its
}
