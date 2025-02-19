package model

type ItemsPriceRequest struct {
	SKUs []string `json:"skus"`
}

type PriceResponse struct {
	Items             []*Item     `json:"items"`
	Promotions        *Promotions `json:"promotions,omitempty"`
	TotalGross        float64     `json:"total_gross"`
	TotalWithDiscount float64     `json:"total_with_discount"`
}

type Promotions struct {
	Deduction  float64 `json:"deduction"`
	AddedItems []*Item `json:"added_items"`
}
