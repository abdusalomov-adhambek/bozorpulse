// Package domain defines the identity aggregate.
package domain

// User is a registered account. Suppliers and customers start in "waiting"
// until an admin approves them.
type User struct {
	ID          string `json:"id"`
	Username    string `json:"username"`
	Role        string `json:"role"`
	Status      string `json:"status"`
	INN         string `json:"inn"`
	BankAccount string `json:"bank_account"`
	CompanyName string `json:"company_name"`
	RegionID    string `json:"region_id"`
	Phone       string `json:"phone"`
	CreatedAt   string `json:"created_at"`
}

type RegisterInput struct {
	Username    string `json:"username"`
	Password    string `json:"password"`
	Role        string `json:"role"`
	INN         string `json:"inn"`
	BankAccount string `json:"bank_account"`
	CompanyName string `json:"company_name"`
	RegionID    string `json:"region_id"`
	Phone       string `json:"phone"`
}

func ValidRole(role string) bool {
	return role == "admin" || role == "supplier" || role == "customer"
}
