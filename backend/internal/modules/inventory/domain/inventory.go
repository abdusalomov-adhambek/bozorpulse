package domain

type StockItem struct {
	ProductID   string `json:"product_id"`
	ProductName string `json:"product_name"`
	WarehouseID string `json:"warehouse_id"`
	Quantity    int    `json:"quantity"`
	UpdatedAt   string `json:"updated_at"`
}

type Movement struct {
	ID          string `json:"id"`
	WarehouseID string `json:"warehouse_id"`
	ProductID   string `json:"product_id"`
	Quantity    int    `json:"quantity"`
	Type        string `json:"type"`
	Note        string `json:"note"`
	CreatedAt   string `json:"created_at"`
}

type MovementInput struct {
	WarehouseID   string `json:"warehouse_id"`
	ToWarehouseID string `json:"to_warehouse_id"`
	ProductID     string `json:"product_id"`
	Quantity      int    `json:"quantity"`
	Note          string `json:"note"`
	Reason        string `json:"reason"`
}
