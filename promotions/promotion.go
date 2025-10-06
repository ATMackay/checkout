package promotions

import (
	"context"

	"github.com/ATMackay/checkout/database"
	"github.com/ATMackay/checkout/model"
	"github.com/shopspring/decimal"
)

// Promotion defines the interface for a promotion strategy.
type Promotion interface {
	Apply(items []*model.Item) (*model.Promotions, error)
}

// MacBookProPromotion adds a free Raspberry Pi B for each MacBook Pro.
type MacBookProPromotion struct {
	db database.InventoryStore
}

func NewMacBookProPromotion(db database.Database) *MacBookProPromotion {
	return &MacBookProPromotion{db: db}
}

func (p *MacBookProPromotion) Apply(items []*model.Item) (*model.Promotions, error) {
	promotions := &model.Promotions{}
	itemCounts := countItemsByName(items)

	if n, ok := itemCounts["MacBook Pro"]; ok {
		it, err := p.db.GetItemByName(context.Background(), "Raspberry Pi B")
		if err != nil {
			return nil, err
		}
		if it.InventoryQuantity < n {
			return promotions, nil
		}
		for range n {
			promotions.AddedItems = append(promotions.AddedItems, it)
		}
	}

	return promotions, nil
}

// GoogleTVPromotion applies a "Buy 3 for the price of 2" discount.
type GoogleTVPromotion struct{}

func (p *GoogleTVPromotion) Apply(items []*model.Item) (*model.Promotions, error) {
	promotions := &model.Promotions{}
	itemCounts := countItemsByName(items)

	for _, item := range items {
		if item.Name == "Google TV" && itemCounts["Google TV"] >= 3 {
			discount := item.Price.Mul(decimal.NewFromInt(int64(itemCounts["Google TV"] / 3)))
			promotions.Deduction += discount.InexactFloat64()
		}
	}

	return promotions, nil
}

// AlexaSpeakerPromotion applies a 10% discount if more than 3 are bought.
type AlexaSpeakerPromotion struct{}

func (p *AlexaSpeakerPromotion) Apply(items []*model.Item) (*model.Promotions, error) {
	promotions := &model.Promotions{}
	itemCounts := countItemsByName(items)

	for _, item := range items {
		if item.Name == "Alexa Speaker" && itemCounts["Alexa Speaker"] > 3 {
			discount := item.Price.Mul(decimal.NewFromInt(int64(itemCounts["Alexa Speaker"]))).Mul(decimal.NewFromFloat(0.1))
			promotions.Deduction += discount.InexactFloat64()
		}
	}

	return promotions, nil
}

// countItemsByName counts the occurrences of each item by name.
func countItemsByName(items []*model.Item) map[string]int {
	itemCounts := make(map[string]int)
	for _, item := range items {
		itemCounts[item.Name]++
	}
	return itemCounts
}
