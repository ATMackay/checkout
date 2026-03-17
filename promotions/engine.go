package promotions

import (
	"context"

	"github.com/ATMackay/checkout/model"
)

// PromotionsEngine applies all registered promotions.
type PromotionsEngine struct {
	promotions []Promotion
}

// NewPromotionsEngine creates a new PromotionsEngine with the given promotions.
func NewPromotionsEngine(promotions ...Promotion) *PromotionsEngine {
	return &PromotionsEngine{
		promotions: promotions,
	}
}

// ApplyPromotions applies all registered promotions to the items.
func (e *PromotionsEngine) ApplyPromotions(ctx context.Context, items []*model.Item) (*model.Promotions, error) {
	result := &model.Promotions{}

	for _, promotion := range e.promotions {
		p, err := promotion.Apply(ctx, items)
		if err != nil {
			return nil, err
		}
		result.Deduction += p.Deduction
		result.AddedItems = append(result.AddedItems, p.AddedItems...)
	}

	return result, nil
}
