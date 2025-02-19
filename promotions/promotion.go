package promotions

import "github.com/ATMackay/checkout/model"

// Promotion defines the interface for a promotion strategy.
type Promotion interface {
	Apply(items []*model.Item) *model.Promotions
}

// MacBookProPromotion adds a free Raspberry Pi B for each MacBook Pro.
type MacBookProPromotion struct{}

func (p *MacBookProPromotion) Apply(items []*model.Item) *model.Promotions {
	promotions := &model.Promotions{}
	itemCounts := countItemsByName(items)

	if n, ok := itemCounts["MacBook Pro"]; ok {
		for range n {
			promotions.AddedItems = append(promotions.AddedItems, &model.Item{
				Name:  "Raspberry Pi B",
				SKU:   "43N23P",
				Price: 0, // Added for free
			})
		}
	}

	return promotions
}

// GoogleTVPromotion applies a "Buy 3 for the price of 2" discount.
type GoogleTVPromotion struct{}

func (p *GoogleTVPromotion) Apply(items []*model.Item) *model.Promotions {
	promotions := &model.Promotions{}
	itemCounts := countItemsByName(items)

	for _, item := range items {
		if item.Name == "Google TV" && itemCounts["Google TV"] >= 3 {
			discount := float64(itemCounts["Google TV"]/3) * item.Price
			promotions.Deduction += discount
		}
	}

	return promotions
}

// AlexaSpeakerPromotion applies a 10% discount if more than 3 are bought.
type AlexaSpeakerPromotion struct{}

func (p *AlexaSpeakerPromotion) Apply(items []*model.Item) *model.Promotions {
	promotions := &model.Promotions{}
	itemCounts := countItemsByName(items)

	for _, item := range items {
		if item.Name == "Alexa Speaker" && itemCounts["Alexa Speaker"] > 3 {
			discount := 0.1 * item.Price * float64(itemCounts["Alexa Speaker"])
			promotions.Deduction += discount
		}
	}

	return promotions
}

// countItemsByName counts the occurrences of each item by name.
func countItemsByName(items []*model.Item) map[string]int {
	itemCounts := make(map[string]int)
	for _, item := range items {
		itemCounts[item.Name]++
	}
	return itemCounts
}
