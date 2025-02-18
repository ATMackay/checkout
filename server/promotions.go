package server

type Promotions struct {
	Deduction  float64 `json:"deduction"`
	AddedItems []*Item `json:"added_items"`
}

func applyPromotions(items []*Item) *Promotions {
	var promotions Promotions

	// Count the number of each item
	itemCounts := make(map[string]int)
	for _, item := range items {
		itemCounts[item.Name]++
	}

	// Apply promotions
	for _, item := range items {
		switch item.Name {
		case "MacBook Pro":
			// Add a free Raspberry Pi B for each MacBook Pro
			promotions.AddedItems = append(promotions.AddedItems, &Item{
				Name:  "Raspberry Pi B",
				SKU:   "RaspberryPiB",
				Price: 0, // Added for free
			})
		case "Google TV":
			// Buy 3 Google TVs for the price of 2
			if itemCounts["GoogleTV"] >= 3 {
				discount := float64(itemCounts["GoogleTV"]/3) * item.Price
				promotions.Deduction += discount
			}
		case "Alexa Speaker":
			// 10% discount on all Alexa Speakers if more than 3 are bought
			if itemCounts["AlexaSpeaker"] > 3 {
				discount := 0.1 * item.Price * float64(itemCounts["AlexaSpeaker"])
				promotions.Deduction += discount
			}
		}

	}

	return &promotions
}
