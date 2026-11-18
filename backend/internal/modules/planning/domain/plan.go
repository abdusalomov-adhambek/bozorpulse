// Package domain owns the planning entities and the pure projection rules.
// It must not import net/http, database/sql or any infrastructure package.
package domain

import "time"

// PlanStatus is the lifecycle state of a saved plan.
type PlanStatus string

const (
	PlanDraft    PlanStatus = "draft"
	PlanActive   PlanStatus = "active"
	PlanArchived PlanStatus = "archived"
)

// Recommendation is the conservative, deterministic signal shown to a customer.
type Recommendation string

const (
	RecBuyNow       Recommendation = "buy_now"
	RecWait         Recommendation = "wait"
	RecMonitor      Recommendation = "monitor"
	RecInsufficient Recommendation = "insufficient"
)

// Data status values mirror the AI module's labels (kept as a string so this
// domain stays independent of other modules).
const (
	StatusDemo         = "demo"
	StatusLive         = "live"
	StatusInsufficient = "insufficient"
)

// Plan is a customer's collection of upcoming purchase needs.
type Plan struct {
	ID         string     `json:"id"`
	CompanyID  string     `json:"company_id"`
	CustomerID string     `json:"customer_id"`
	Title      string     `json:"title"`
	RegionID   string     `json:"region_id,omitempty"`
	Status     PlanStatus `json:"status"`
	Items      []PlanItem `json:"items"`
	CreatedAt  string     `json:"created_at"`
	UpdatedAt  string     `json:"updated_at"`
}

// PlanItem is one planned product need together with a forecast snapshot.
type PlanItem struct {
	ID          string          `json:"id"`
	ProductID   string          `json:"product_id"`
	ProductName string          `json:"product_name"`
	Unit        string          `json:"unit"`
	Quantity    int             `json:"quantity"`
	TargetDate  string          `json:"target_date,omitempty"`
	Note        string          `json:"note,omitempty"`
	Projection  PriceProjection `json:"projection"`
}

// PriceProjection is a snapshot derived from a price forecast. Every number is
// a pointer: an insufficient forecast must serialize as null, never as 0.
type PriceProjection struct {
	CurrentUnitPrice *float64       `json:"current_unit_price"`
	Predicted        *float64       `json:"predicted"`
	Low              *float64       `json:"low"`
	High             *float64       `json:"high"`
	ChangePct        *float64       `json:"change_pct"`
	TrendPct         float64        `json:"trend_pct"`
	HorizonDays      int            `json:"horizon_days"`
	EstimatedBudget  *float64       `json:"estimated_budget"`
	BudgetLow        *float64       `json:"budget_low"`
	BudgetHigh       *float64       `json:"budget_high"`
	SavingVsNow      *float64       `json:"saving_vs_now"`
	Recommendation   Recommendation `json:"recommendation"`
	Reason           string         `json:"reason"`
	DataStatus       string         `json:"data_status"`
	Disclaimer       string         `json:"disclaimer,omitempty"`
	Basis            string         `json:"basis,omitempty"`
	Note             string         `json:"note,omitempty"`
	GeneratedAt      string         `json:"generated_at,omitempty"`
}

// ForecastInput is the narrow subset of a price forecast the projection needs.
// It keeps this domain decoupled from the AI module's wire type.
type ForecastInput struct {
	Predicted   float64
	Low         float64
	High        float64
	Baseline    float64
	ChangePct   float64
	TrendPct    float64
	HorizonDays int
	Samples     int
	DataStatus  string
	Basis       string
	Warnings    []string
}

// ValidateStatus returns whether s is a known plan status.
func ValidateStatus(s string) bool {
	switch PlanStatus(s) {
	case PlanDraft, PlanActive, PlanArchived:
		return true
	}
	return false
}

// ValidateTargetDate accepts an optional YYYY-MM-DD date inside the forecast
// horizon (today .. today+30 days). An empty date is always valid.
func ValidateTargetDate(target string, now time.Time) bool {
	if target == "" {
		return true
	}
	t, err := time.Parse("2006-01-02", target)
	if err != nil {
		return false
	}
	today := time.Date(now.UTC().Year(), now.UTC().Month(), now.UTC().Day(), 0, 0, 0, 0, time.UTC)
	last := today.AddDate(0, 0, 30)
	return !t.Before(today) && !t.After(last)
}
