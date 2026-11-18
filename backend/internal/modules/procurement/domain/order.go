package domain

type OrderItem struct {
	ProductID string  `json:"product_id"`
	Name      string  `json:"name"`
	Quantity  int     `json:"quantity"`
	UnitPrice float64 `json:"unit_price"`
	LineTotal float64 `json:"line_total"`
}

type Order struct {
	ID          string      `json:"id"`
	CustomerID  string      `json:"customer_id"`
	SupplierID  string      `json:"supplier_id"`
	WarehouseID string      `json:"warehouse_id"`
	Status      string      `json:"status"`
	TotalAmount float64     `json:"total_amount"`
	Items       []OrderItem `json:"items"`
	CreatedAt   string      `json:"created_at"`
	UpdatedAt   string      `json:"updated_at"`
}

type CreateInput struct {
	SupplierID  string `json:"supplier_id"`
	WarehouseID string `json:"warehouse_id"`
	Items       []struct {
		ProductID string `json:"product_id"`
		Quantity  int    `json:"quantity"`
	} `json:"items"`
}
