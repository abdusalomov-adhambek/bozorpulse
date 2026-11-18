package domain

// Category is a product classification node.
type Category struct {
	ID        string `json:"id"`
	Name      string `json:"name"`
	ParentID  string `json:"parent_id"`
	CreatedAt string `json:"created_at"`
}

// Region is a geography used for warehouses, suppliers, and market prices.
type Region struct {
	ID        string `json:"id"`
	Name      string `json:"name"`
	CreatedAt string `json:"created_at"`
}

// Product is a supplier's listed good with its own price and discount.
type Product struct {
	ID              string   `json:"id"`
	SupplierID      string   `json:"supplier_id"`
	CategoryID      string   `json:"category_id"`
	Name            string   `json:"name"`
	SKU             string   `json:"sku"`
	Description     string   `json:"description"`
	Unit            string   `json:"unit"`
	Price           float64  `json:"price"`
	DiscountPercent float64  `json:"discount_percent"`
	EffectivePrice  float64  `json:"effective_price"`
	Stock           int      `json:"stock"`
	RegionID        string   `json:"region_id"`
	Photos          []string `json:"photos"`
	MinOrderQty     int      `json:"min_order_qty"`
	MaxOrderQty     int      `json:"max_order_qty"`
	DeliveryMinDays int      `json:"delivery_min_days"`
	DeliveryMaxDays int      `json:"delivery_max_days"`
	CreatedAt       string   `json:"created_at"`
	UpdatedAt       string   `json:"updated_at"`
}

// PricingRule is a warehouse and/or quantity tier for a product.
type PricingRule struct {
	ID              string  `json:"id"`
	ProductID       string  `json:"product_id"`
	WarehouseID     string  `json:"warehouse_id"`
	WarehouseName   string  `json:"warehouse_name,omitempty"`
	MinQty          int     `json:"min_qty"`
	MaxQty          int     `json:"max_qty"`
	UnitPrice       float64 `json:"unit_price"`
	DiscountPercent float64 `json:"discount_percent"`
	EffectivePrice  float64 `json:"effective_price"`
	DeliveryMinDays int     `json:"delivery_min_days"`
	DeliveryMaxDays int     `json:"delivery_max_days"`
	MinOrder        int     `json:"min_order"`
	MaxOrder        int     `json:"max_order"`
	CreatedAt       string  `json:"created_at"`
}

// DeliveryZone describes whether a supplier delivers to a region, the fee and
// the delivery window for that region.
type DeliveryZone struct {
	ID         string  `json:"id"`
	SupplierID string  `json:"supplier_id"`
	RegionID   string  `json:"region_id"`
	RegionName string  `json:"region_name,omitempty"`
	Available  bool    `json:"available"`
	Price      float64 `json:"price"`
	MinDays    int     `json:"min_days"`
	MaxDays    int     `json:"max_days"`
	Notes      string  `json:"notes"`
	CreatedAt  string  `json:"created_at"`
}

// ResolvedPrice is the price that applies to a product for a given warehouse
// and quantity, together with its delivery window and order limits.
type ResolvedPrice struct {
	UnitPrice       float64 `json:"unit_price"`
	DiscountPercent float64 `json:"discount_percent"`
	EffectivePrice  float64 `json:"effective_price"`
	DeliveryMinDays int     `json:"delivery_min_days"`
	DeliveryMaxDays int     `json:"delivery_max_days"`
	MinOrder        int     `json:"min_order"`
	MaxOrder        int     `json:"max_order"`
	Source          string  `json:"source"` // product | warehouse | tier
	RuleID          string  `json:"rule_id,omitempty"`
	Note            string  `json:"note,omitempty"`
}

// PriceHistoryPoint is one observed price sample for a product.
type PriceHistoryPoint struct {
	ID         string  `json:"id"`
	Price      float64 `json:"price"`
	SourceType string  `json:"source_type"`
	IsDemo     bool    `json:"is_demo"`
	ObservedAt string  `json:"observed_at"`
	RegionID   string  `json:"region_id"`
}

// PriceHistory is a product's observation list plus a small summary.
type PriceHistory struct {
	ProductID   string              `json:"product_id"`
	ProductName string              `json:"product_name"`
	Samples     []PriceHistoryPoint `json:"samples"`
	Min         *float64            `json:"min"`
	Max         *float64            `json:"max"`
	Avg         *float64            `json:"avg"`
	Count       int                 `json:"count"`
	DataStatus  string              `json:"data_status"`
	Disclaimer  string              `json:"disclaimer,omitempty"`
}
