package domain

// Payment is a record of intended settlement. The MVP does not process money.
type Payment struct {
	ID          string  `json:"id"`
	OrderID     string  `json:"order_id"`
	Amount      float64 `json:"amount"`
	Method      string  `json:"method"`
	BankAccount string  `json:"bank_account"`
	Status      string  `json:"status"`
	CreatedAt   string  `json:"created_at"`
}
