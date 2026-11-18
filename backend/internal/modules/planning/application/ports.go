package application

import (
	"context"

	"github.com/bozorpulse/bozorpulse-ai/backend/internal/modules/planning/domain"
)

// PlanRepository owns persistence for plans and their items.
type PlanRepository interface {
	Save(ctx context.Context, plan domain.Plan) error
	Get(ctx context.Context, ownerKey, id string) (domain.Plan, error)
	List(ctx context.Context, ownerKey string) ([]domain.Plan, error)
	Delete(ctx context.Context, ownerKey, id string) error
}

// ProductInfo is the narrow catalog view planning needs.
type ProductInfo struct {
	ID        string
	Name      string
	Unit      string
	RegionID  string
	BasePrice float64
}

// ProductReader is implemented by an adapter over the catalog application API.
type ProductReader interface {
	Product(ctx context.Context, productID string) (ProductInfo, error)
	// BestTierPrice returns the lowest effective price applicable to the
	// quantity, or 0 when no quantity tier exists (the caller then uses the
	// product's base price).
	BestTierPrice(ctx context.Context, productID string, quantity int) (float64, error)
}

// ForecastOutcome is the planning-owned copy of a price forecast. It avoids
// importing the AI module's domain type into this layer.
type ForecastOutcome struct {
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

// Forecaster is implemented by an adapter over the AI forecast service.
type Forecaster interface {
	Forecast(ctx context.Context, productID, regionID string) (ForecastOutcome, error)
}

// PlanInput is the request shape shared by Create, Update and Preview.
type PlanInput struct {
	Title    string
	RegionID string
	Status   string
	Items    []ItemInput
}

// ItemInput is one requested plan line.
type ItemInput struct {
	ProductID  string
	Quantity   int
	TargetDate string
	Note       string
}
