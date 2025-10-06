package promotions

import (
	"context"
	"testing"

	"github.com/ATMackay/checkout/database/mock"
	"github.com/ATMackay/checkout/model"
	"github.com/golang/mock/gomock"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/require"
)

func TestNewEngine(t *testing.T) {
	e := NewPromotionsEngine(&MacBookProPromotion{}, &GoogleTVPromotion{}, &AlexaSpeakerPromotion{})
	require.Len(t, e.promotions, 3)
}

func TestPromotions(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	db := mock.NewMockDatabase(ctrl)
	e := NewPromotionsEngine(NewMacBookProPromotion(db), &GoogleTVPromotion{}, &AlexaSpeakerPromotion{})
	t.Run("macbook-pro", func(t *testing.T) {
		it := &model.Item{Name: "Raspberry Pi B", SKU: "234234", Price: decimal.NewFromFloat(30.0), InventoryQuantity: 2}
		db.EXPECT().GetItemByName(context.Background(), "Raspberry Pi B").Return(it, nil)
		items := []*model.Item{
			{Name: "MacBook Pro", SKU: "MacBookPro", Price: decimal.NewFromFloat(5399.99)},
			{Name: "Google TV", SKU: "GoogleTV", Price: decimal.NewFromFloat(49.99)},
			{Name: "Alexa Speaker", SKU: "AlexaSpeaker", Price: decimal.NewFromFloat(109.50)},
		}
		promotions, err := e.ApplyPromotions(items)
		require.NoError(t, err)
		require.NotNil(t, promotions)
		require.Equal(t, []*model.Item{it}, promotions.AddedItems)
	})
}
