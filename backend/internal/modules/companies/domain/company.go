package domain

type Company struct {
	ID          string `json:"id"`
	OwnerUserID string `json:"owner_user_id"`
	Role        string `json:"role"`
	Name        string `json:"company_name"`
	INN         string `json:"inn"`
	BankAccount string `json:"bank_account"`
	RegionID    string `json:"region_id"`
	RegionName  string `json:"region_name"`
	Status      string `json:"status"`
	CreatedAt   string `json:"created_at"`
}

type Warehouse struct {
	ID        string  `json:"id"`
	OwnerID   string  `json:"owner_id"`
	OwnerRole string  `json:"owner_role"`
	Name      string  `json:"name"`
	Address   string  `json:"address"`
	City      string  `json:"city"`
	RegionID  string  `json:"region_id"`
	Latitude  float64 `json:"latitude"`
	Longitude float64 `json:"longitude"`
	CreatedAt string  `json:"created_at"`
}
