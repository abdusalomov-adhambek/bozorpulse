package domain

type Metric struct {
	Label string      `json:"label"`
	Value interface{} `json:"value"`
	Delta string      `json:"delta,omitempty"`
}

type Dashboard struct {
	Role          string        `json:"role"`
	Metrics       []Metric      `json:"metrics"`
	RecentOrders  []interface{} `json:"recent_orders"`
	MarketSummary MarketSummary `json:"market_summary"`
}

type MarketSummary struct {
	Min    float64 `json:"min"`
	Median float64 `json:"median"`
	Avg    float64 `json:"avg"`
	Count  int     `json:"count"`
}
