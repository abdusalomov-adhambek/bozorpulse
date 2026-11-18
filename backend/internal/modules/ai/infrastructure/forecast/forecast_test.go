package forecast

import (
	"strings"
	"testing"
	"time"

	"github.com/bozorpulse/bozorpulse-ai/backend/internal/modules/ai/domain"
)

func calibrationForTest(t *testing.T) Calibration {
	t.Helper()
	cal, err := LoadCalibration()
	if err != nil {
		t.Fatal(err)
	}
	return cal
}

func TestComputeStablePrices(t *testing.T) {
	block := calibrationForTest(t).For("cat-01")
	prices := []float64{10, 10.5, 9.8, 10.2, 10.1, 10}
	central, trend := compute(prices, 0, block, 1)
	if central < 9 || central > 11 {
		t.Fatalf("stable prices should stay near the median, got %.2f", central)
	}
	if trend > 0.05 {
		t.Fatalf("unexpected trend %.2f", trend)
	}
}

func TestComputeRisingTrend(t *testing.T) {
	block := calibrationForTest(t).For("cat-01")
	prices := []float64{8, 8, 8, 8, 12, 12, 12, 12}
	central, trend := compute(prices, 0, block, 1)
	if trend <= 0 || central <= 10 {
		t.Fatalf("rising prices should lift the estimate: central=%.2f trend=%.2f", central, trend)
	}
}

func TestWeatherHasNoEffectUntilCalibrated(t *testing.T) {
	cal := calibrationForTest(t)
	prices := []float64{10, 10, 10, 10, 10, 10}
	food := cal.For("cat-06")
	electronics := cal.For("cat-01")
	if food.Calibrated() || electronics.Calibrated() {
		t.Fatal("no category should be calibrated in the shipped config")
	}
	foodCentral, _ := compute(prices, 0, food, 1)
	techCentral, _ := compute(prices, 0, electronics, 1)
	if foodCentral != techCentral {
		t.Fatalf("uncalibrated weather must not change the estimate: food=%.2f tech=%.2f", foodCentral, techCentral)
	}
}

func TestCalibratedWeatherAppliesWithCap(t *testing.T) {
	block := calibrationForTest(t).For("cat-06")
	beta := 4.0
	block.BetaPctPerSD = &beta
	central, _ := compute([]float64{10, 10, 10, 10}, 0, block, 1)
	if central < 10.35 || central > 10.45 {
		t.Fatalf("beta 4%% should lift 10.00 to about 10.40, got %.2f", central)
	}
	huge := 100.0
	block.BetaPctPerSD = &huge
	capped, _ := compute([]float64{10, 10, 10, 10}, 0, block, 1)
	if capped > 12.5 {
		t.Fatalf("weather adjustment must respect the category cap, got %.2f", capped)
	}
}

func TestBuildIndicatorsSeparatesTransactionPrices(t *testing.T) {
	ind := buildIndicators([]float64{8, 10, 12}, []float64{9, 11})
	if ind.MarketMedian != 10 || ind.MarketAverage != 10 || ind.MarketMin != 8 || ind.MarketMax != 12 {
		t.Fatalf("unexpected indicators: %+v", ind)
	}
	if ind.ExpectedTransaction != 10 || !strings.Contains(ind.Basis, "transaction") {
		t.Fatalf("transaction indicator missing: %+v", ind)
	}
	quotesOnly := buildIndicators([]float64{8, 10, 12}, nil)
	if !strings.Contains(quotesOnly.Basis, "approximated") {
		t.Fatalf("quote-only estimate must be labeled: %+v", quotesOnly)
	}
}

func TestSeasonalFactorNeedsEnoughObservations(t *testing.T) {
	month := time.Now().UTC().AddDate(0, 1, 0).Month()
	stamp := func(m time.Month) string { return time.Date(2026, m, 15, 0, 0, 0, 0, time.UTC).Format(time.RFC3339) }
	samples := []priceRow{
		{price: 10, at: stamp(month)}, {price: 12, at: stamp(month)},
		{price: 10, at: stamp(time.January)}, {price: 10, at: stamp(time.February)},
	}
	f, ok := seasonalFactor(samples, month)
	if !ok || f < 0.8 || f > 1.2 {
		t.Fatalf("unexpected seasonal factor %.2f (%v)", f, ok)
	}
	_, ok = seasonalFactor(samples[:2], month)
	if ok {
		t.Fatal("seasonality must not be inferred from one-month history")
	}
}

func TestConfidenceUsesHistoryAndFallback(t *testing.T) {
	high, _ := confidenceLevel(40, 400, "exact_product_history", nil)
	if high != "high" {
		t.Fatalf("expected high, got %s", high)
	}
	medium, _ := confidenceLevel(15, 120, "similar_products", nil)
	if medium != "medium" {
		t.Fatalf("expected medium, got %s", medium)
	}
	low, _ := confidenceLevel(3, 30, "supplier_offers", []string{"asking"})
	if low != "low" {
		t.Fatalf("expected low, got %s", low)
	}
}

func TestPriceForecastInsufficientShape(t *testing.T) {
	s := &Service{}
	out, err := s.Forecast(t.Context(), "", "")
	if err != nil {
		t.Fatal(err)
	}
	if out.Status != "insufficient_data" || out.DataStatus != domain.DataStatusInsufficient || out.Predicted != 0 {
		t.Fatalf("bad insufficient shape: %+v", out)
	}
}
