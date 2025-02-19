package model

type Order struct {
	Reference string   `json:"reference"`
	SKUs      []string `json:"skus"`
	Cost      float64  `json:"cost"`
}

type Orders []Order
