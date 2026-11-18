package infrastructure

import (
	"context"

	aiDomain "github.com/bozorpulse/bozorpulse-ai/backend/internal/modules/ai/domain"
	"github.com/bozorpulse/bozorpulse-ai/backend/internal/modules/planning/application"
	"github.com/bozorpulse/bozorpulse-ai/backend/internal/modules/planning/domain"
)

// aiForecaster matches the AI module's forecast service method. It is kept
// local so the adapter can be wired with any compatible implementation.
type aiForecaster interface {
	Forecast(ctx context.Context, productID, regionID string) (aiDomain.PriceForecast, error)
}

// ForecastAdapter translates an AI price forecast into the planning-owned DTO.
type ForecastAdapter struct{ forecaster aiForecaster }

func NewForecastAdapter(f aiForecaster) *ForecastAdapter { return &ForecastAdapter{forecaster: f} }

func (a *ForecastAdapter) Forecast(ctx context.Context, productID, regionID string) (application.ForecastOutcome, error) {
	fc, err := a.forecaster.Forecast(ctx, productID, regionID)
	if err != nil {
		return application.ForecastOutcome{}, err
	}
	base := fc.BaselinePrice
	if base <= 0 {
		base = fc.Predicted // defensive fallback for older forecast payloads
	}
	status := string(fc.DataStatus)
	if fc.Status == "insufficient_data" && status != domain.StatusInsufficient {
		status = domain.StatusInsufficient
	}
	return application.ForecastOutcome{
		Predicted:   fc.Predicted,
		Low:         fc.Low,
		High:        fc.High,
		Baseline:    base,
		ChangePct:   fc.PredictedChangePct,
		TrendPct:    fc.TrendPct,
		HorizonDays: fc.HorizonDays,
		Samples:     fc.Samples,
		DataStatus:  status,
		Basis:       fc.Basis,
		Warnings:    fc.Warnings,
	}, nil
}
