package server

import (
	"github.com/ATMackay/checkout/model"
	"github.com/ATMackay/checkout/promotions"
)

var promotionsEngine = promotions.NewPromotionsEngine(
	&promotions.MacBookProPromotion{},
	&promotions.GoogleTVPromotion{},
	&promotions.AlexaSpeakerPromotion{}, // Add more deals/promotions to the engine
)

func applyPromotions(items []*model.Item) *model.Promotions {
	return promotionsEngine.ApplyPromotions(items)
}
