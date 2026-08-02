package order

import "time"

type Item struct {
	SKU        string `json:"sku"`
	Quantity   int    `json:"quantity"`
	UnitPrice  int64  `json:"unit_price"` // cents
}

type Order struct {
	ID        string    `json:"id"`
	Items     []Item    `json:"items"`
	Total     int64     `json:"total"` // cents
	Status    string    `json:"status"`
	Timestamp time.Time `json:"timestamp"`
}

type CreateOrderRequest struct {
	Items []Item `json:"items"`
}

func ComputeTotal(items []Item) int64 {
	var total int64
	for _, item := range items {
		total += item.UnitPrice * int64(item.Quantity)
	}
	return total
}
