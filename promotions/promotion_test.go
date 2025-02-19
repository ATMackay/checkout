package promotions

import (
	"testing"

	"github.com/ATMackay/checkout/model"
	"github.com/stretchr/testify/require"
)

func TestNewEngine(t *testing.T) {
	e := NewPromotionsEngine(&MacBookProPromotion{}, &GoogleTVPromotion{}, &AlexaSpeakerPromotion{})
	require.Len(t, e.promotions, 3)
}

func TestPromotions(t *testing.T) {
	e := NewPromotionsEngine(&MacBookProPromotion{}, &GoogleTVPromotion{}, &AlexaSpeakerPromotion{})
	t.Run("macbook-pro", func(t *testing.T) {
		items := []*model.Item{
			{Name: "MacBook Pro", SKU: "MacBookPro", Price: 5399.99},
			{Name: "Google TV", SKU: "GoogleTV", Price: 49.99},
			{Name: "Alexa Speaker", SKU: "AlexaSpeaker", Price: 109.50},
		}
		promotions := e.ApplyPromotions(items)
		require.NotNil(t, promotions)
		require.Equal(t, promotions.AddedItems, []*model.Item{{Name: "Raspberry Pi B", SKU: "43N23P", Price: 0}})
	})
}
