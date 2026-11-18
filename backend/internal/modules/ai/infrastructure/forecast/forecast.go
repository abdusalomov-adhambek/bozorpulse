// Package forecast produces conservative, explainable price outlooks from
// normalized price history. Weather is reported as context but contributes
// zero until a backtested category beta is configured (see calibration.json).
package forecast

import (
	"context"
	"database/sql"
	"fmt"
	"math"
	"sort"
	"strings"
	"time"

	"github.com/bozorpulse/bozorpulse-ai/backend/internal/modules/ai/domain"
)

// WeatherProvider supplies current conditions for a platform region.
type WeatherProvider interface {
	Current(ctx context.Context, regionID string) (domain.WeatherInfo, error)
}

type Service struct {
	db      *sql.DB
	weather WeatherProvider
	cal     Calibration
}

func New(db *sql.DB, weather WeatherProvider) *Service {
	cal, err := LoadCalibration()
	if err != nil {
		cal = Calibration{DefaultPriceType: "spot", Categories: map[string]CategoryCalibration{}}
	}
	return &Service{db: db, weather: weather, cal: cal}
}

var regionNames = map[string]string{
	"reg-01": "Tashkent", "reg-02": "Samarkand", "reg-03": "Bukhara",
	"reg-04": "Andijan", "reg-05": "Fergana",
}

func regionName(id string) string {
	if name, ok := regionNames[id]; ok {
		return name
	}
	return id
}

type priceRow struct {
	price  float64
	demo   bool
	at     string
	region string
}

// Forecast combines normalized history, a seasonal baseline, market indicators
// and confidence. Cold-start levels and price-type guards follow the rules docs.
func (s *Service) Forecast(ctx context.Context, productID, regionID string) (domain.PriceForecast, error) {
	out := domain.PriceForecast{
		Status:            "insufficient_data",
		DataStatus:        domain.DataStatusInsufficient,
		HorizonDays:       30,
		AsOf:              time.Now().UTC().Format(time.RFC3339),
		CentralTendency:   "median",
		FallbackLevel:     "insufficient_data",
		ModelType:         "baseline_trend_seasonal",
		ModelVersion:      "2.0",
		PriceType:         s.cal.DefaultPriceType,
		Confidence:        "insufficient_data",
		ConfidenceReasons: []string{},
		Warnings:          []string{},
		AdjustmentsPct:    map[string]float64{"weather": 0, "total_after_caps": 0},
	}
	if productID == "" || s == nil || s.db == nil {
		return out, nil
	}

	var categoryID, unit string
	var current, effective float64
	if err := s.db.QueryRowContext(ctx,
		`SELECT COALESCE(category_id,''), COALESCE(unit,''), COALESCE(price,0), COALESCE(effective_price,0) FROM products WHERE id=$1`,
		productID,
	).Scan(&categoryID, &unit, &current, &effective); err != nil {
		return out, err
	}
	block := s.cal.For(categoryID)
	out.CanonicalCategory = block.CanonicalCategory
	out.MarketTags = block.MarketTags
	out.PriceType = s.cal.DefaultPriceType
	out.Regulated = s.cal.DefaultRegulated

	rows, err := s.db.QueryContext(ctx,
		`SELECT price, is_demo, observed_at, COALESCE(region_id,'') FROM price_samples
		 WHERE product_id=$1 AND price>0 ORDER BY observed_at DESC LIMIT 48`, productID)
	if err != nil {
		return out, err
	}
	all := readPriceRows(rows)
	if err := rows.Err(); err != nil {
		return out, err
	}

	// Region segmentation: use region-specific history when it is thick enough.
	samples := all
	if regionID != "" {
		regional := make([]priceRow, 0)
		for _, r := range all {
			if r.region == regionID {
				regional = append(regional, r)
			}
		}
		if len(regional) >= 3 {
			samples = regional
		} else if len(all) >= 3 {
			out.Warnings = append(out.Warnings, "region-specific price history is thin; used all regions")
		}
	}

	// Cold-start hierarchy (chatgpt v2 §10).
	level := "exact_product_history"
	if len(samples) < 3 {
		similar, err := s.similarHistory(ctx, categoryID, unit, productID)
		if err != nil {
			return out, err
		}
		if len(similar) >= 3 {
			samples, level = similar, "similar_products"
		} else if benchmark := s.categoryBenchmark(ctx, categoryID); benchmark > 0 {
			samples = []priceRow{{price: benchmark, demo: true}}
			level = "related_market_benchmarks"
		} else if asking := math.Max(current, effective); asking > 0 {
			samples = []priceRow{{price: asking, demo: true}}
			level = "supplier_offers"
			out.Warnings = append(out.Warnings, "estimate uses verified supplier asking prices, not transactions")
		} else {
			out.Warnings = append(out.Warnings, "insufficient data for a defensible estimate")
			out.Basis = "Not enough comparable observations."
			return out, nil
		}
	}
	out.FallbackLevel = level
	out.Samples = len(samples)

	prices := make([]float64, 0, len(samples))
	anyDemo := false
	oldest, newest := "", ""
	for _, r := range samples {
		prices = append(prices, r.price)
		anyDemo = anyDemo || r.demo
		if oldest == "" || r.at < oldest {
			oldest = r.at
		}
		if r.at > newest {
			newest = r.at
		}
	}
	sort.Float64s(prices)

	// Indicator set: quotes/listings and transactions stay separate.
	tx, err := s.transactionPrices(ctx, productID)
	if err != nil {
		return out, err
	}
	ind := buildIndicators(prices, tx)
	if len(tx) == 0 {
		out.Warnings = append(out.Warnings, "expected transaction price estimated from comparable offers; no completed transactions")
	}
	out.PriceIndicators = &ind

	// Price-type / regulation guard: no weather on fixed or contractual prices.
	if out.Regulated || out.PriceType == "contract" || out.PriceType == "list" {
		block.WeatherEnabled = false
		out.Warnings = append(out.Warnings, "weather layer disabled for "+out.PriceType+" or regulated price")
	}

	// Weather is always fetched for context, never applied without calibration.
	weather := domain.WeatherInfo{Region: regionName(regionID), Source: "unavailable"}
	weatherFailed := false
	if s.weather != nil {
		w, err := s.weather.Current(ctx, regionID)
		if err != nil {
			weatherFailed = true
		} else {
			weather = w
		}
	} else {
		weatherFailed = true
	}
	out.Weather = &weather
	if weatherFailed {
		out.Warnings = append(out.Warnings, "weather data unavailable; context only")
	}
	if !block.WeatherEnabled {
		out.Warnings = append(out.Warnings, "weather layer disabled for this category")
	}
	if !block.Calibrated() {
		out.Warnings = append(out.Warnings, "weather layer uncalibrated; zero weather adjustment applied")
	}

	targetMonth := time.Now().UTC().AddDate(0, 1, 0).Month()
	seasonal, seasonalApplied := seasonalFactor(samples, targetMonth)
	if !seasonalApplied && len(samples) >= 3 {
		out.Warnings = append(out.Warnings, "insufficient same-month history; seasonal factor not applied")
	}

	central, trend := compute(prices, math.Max(current, effective), block, seasonal)
	baseline := medianOf(prices)
	sigmaPct := 0.0
	if baseline > 0 {
		sigmaPct = stdDev(prices) / baseline
	}
	low, high, intervalType := predictionInterval(central, sigmaPct, out.Warnings)

	out.Status = "ok"
	out.DataStatus = domain.DataStatusDemo
	if !anyDemo {
		out.DataStatus = domain.DataStatusLive
	}
	out.Indicator = "market_median"
	if len(tx) > 0 {
		out.Indicator = "expected_transaction_price"
	}
	out.Predicted = round2(central)
	out.BaselinePrice = round2(baseline)
	out.Low = round2(low)
	out.High = round2(high)
	out.IntervalType = intervalType
	out.ReferencePrice = round2(baseline)
	out.ReferenceIndicator = "market_median"
	if baseline > 0 {
		out.PredictedChangePct = round2((central - baseline) / baseline * 100)
	} else {
		out.PredictedChangePct = 0
	}
	out.TrendPct = round2(trend * 100)
	out.AdjustmentsPct["weather"] = 0
	out.AdjustmentsPct["total_after_caps"] = round2(out.PredictedChangePct)

	spanDays := 0
	t0, e0 := time.Parse(time.RFC3339, oldest)
	t1, e1 := time.Parse(time.RFC3339, newest)
	if e0 == nil && e1 == nil {
		spanDays = int(t1.Sub(t0).Hours() / 24)
	}
	out.Confidence, out.ConfidenceReasons = confidenceLevel(len(samples), spanDays, level, out.Warnings)

	period := "last 30 days"
	if len(samples) > 0 && oldest != "" && newest != "" {
		period = oldest[:10] + " to " + newest[:10]
	}
	out.Basis = fmt.Sprintf("Based on %d comparable prices (%s), median %.2f, %+.1f%% recent trend, seasonality x%.2f. Estimate only — not a supplier quote.",
		len(samples), period, baseline, out.TrendPct, seasonal)
	return out, nil
}

func readPriceRows(rows *sql.Rows) []priceRow {
	out := []priceRow{}
	for rows.Next() {
		var r priceRow
		if err := rows.Scan(&r.price, &r.demo, &r.at, &r.region); err != nil {
			return out
		}
		out = append(out, r)
	}
	return out
}

func (s *Service) similarHistory(ctx context.Context, categoryID, unit, productID string) ([]priceRow, error) {
	rows, err := s.db.QueryContext(ctx,
		`SELECT ps.price, ps.is_demo, ps.observed_at, COALESCE(ps.region_id,'')
		 FROM price_samples ps JOIN products p ON p.id=ps.product_id
		 WHERE p.category_id=$1 AND p.id<>$2 AND ps.price>0 AND ($3='' OR p.unit=$3)
		 ORDER BY ps.observed_at DESC LIMIT 48`, categoryID, productID, unit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return readPriceRows(rows), nil
}

func (s *Service) categoryBenchmark(ctx context.Context, categoryID string) float64 {
	var v float64
	_ = s.db.QueryRowContext(ctx,
		`SELECT percentile_cont(0.5) WITHIN GROUP (ORDER BY effective_price)
		 FROM products WHERE category_id=$1 AND effective_price>0`, categoryID).Scan(&v)
	return v
}

func (s *Service) transactionPrices(ctx context.Context, productID string) ([]float64, error) {
	rows, err := s.db.QueryContext(ctx,
		`SELECT oi.unit_price FROM order_items oi JOIN orders o ON o.id=oi.order_id
		 WHERE oi.product_id=$1 AND oi.unit_price>0 ORDER BY o.created_at DESC LIMIT 24`, productID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []float64{}
	for rows.Next() {
		var v float64
		if err := rows.Scan(&v); err != nil {
			return out, err
		}
		out = append(out, v)
	}
	return out, rows.Err()
}

func buildIndicators(prices, tx []float64) domain.PriceIndicators {
	sorted := append([]float64(nil), prices...)
	sort.Float64s(sorted)
	sum := 0.0
	for _, p := range sorted {
		sum += p
	}
	ind := domain.PriceIndicators{
		MarketMedian:  medianOf(sorted),
		MarketAverage: round2(sum / float64(len(sorted))),
		MarketMin:     sorted[0],
		MarketMax:     sorted[len(sorted)-1],
		Basis:         "normalized comparable price samples; per catalog unit",
	}
	if len(tx) > 0 {
		t := append([]float64(nil), tx...)
		sort.Float64s(t)
		ind.ExpectedTransaction = round2(medianOf(t))
		ind.Basis += "; transaction indicator from recorded order lines"
	} else {
		ind.ExpectedTransaction = round2(ind.MarketMedian)
		ind.Basis += "; transaction indicator approximated from comparable offers"
	}
	return ind
}

// compute is pure: history trend and seasonality, with a weather term that is
// zero until a calibrated beta exists.
func compute(prices []float64, current float64, block CategoryCalibration, seasonal float64) (float64, float64) {
	sorted := append([]float64(nil), prices...)
	sort.Float64s(sorted)
	median := medianOf(sorted)

	half := len(sorted) / 2
	older, recent := medianOf(sorted[:half]), medianOf(sorted[len(sorted)-half:])
	trend := 0.0
	if older > 0 {
		trend = clamp((recent-older)/older, -0.10, 0.10)
	}
	base := median
	if current > 0 {
		base = 0.8*median + 0.2*current
	}

	weatherAdj := 0.0
	if block.Calibrated() {
		// Future hook: beta is a percentage per +1 SD anomaly. With all
		// calibrations currently null this branch never executes.
		weatherAdj = clamp(*block.BetaPctPerSD/100, -block.BaseCapPct/100, block.BaseCapPct/100)
	}
	return base * (1 + 0.5*trend) * seasonal * (1 + weatherAdj), trend
}

func seasonalFactor(samples []priceRow, targetMonth time.Month) (float64, bool) {
	monthPrices := []float64{}
	all := []float64{}
	for _, r := range samples {
		t, err := time.Parse(time.RFC3339, r.at)
		if err != nil || r.price <= 0 {
			continue
		}
		all = append(all, r.price)
		if t.Month() == targetMonth {
			monthPrices = append(monthPrices, r.price)
		}
	}
	if len(monthPrices) < 2 || len(all) < 3 {
		return 1, false
	}
	sort.Float64s(monthPrices)
	sort.Float64s(all)
	overall := medianOf(all)
	if overall <= 0 {
		return 1, false
	}
	return clamp(medianOf(monthPrices)/overall, 0.8, 1.2), true
}

func predictionInterval(central, sigmaPct float64, warnings []string) (float64, float64, string) {
	half := clamp(1.28*sigmaPct, 0.03, 0.12)
	for _, w := range warnings {
		if strings.Contains(w, "thin") || strings.Contains(w, "sparse") ||
			strings.Contains(w, "uncalibrated") || strings.Contains(w, "unavailable") ||
			strings.Contains(w, "asking") || strings.Contains(w, "insufficient") {
			half = clamp(half+0.02, 0.03, 0.20)
			break
		}
	}
	return central * (1 - half), central * (1 + half), "80_percent_prediction_interval"
}

func confidenceLevel(samples, spanDays int, level string, warnings []string) (string, []string) {
	reasons := []string{fmt.Sprintf("%d comparable observations", samples)}
	switch {
	case samples >= 30 && spanDays >= 300 && level == "exact_product_history":
		reasons = append(reasons, "long, product-specific history")
	case samples >= 10 && spanDays >= 90:
		reasons = append(reasons, "usable history")
	default:
		reasons = append(reasons, "limited or cold-start history")
	}
	confidence := "high"
	switch level {
	case "similar_products":
		confidence = "medium"
	case "related_market_benchmarks", "supplier_offers":
		confidence = "low"
	}
	if samples < 10 || spanDays < 90 {
		if confidence == "high" {
			confidence = "medium"
		}
	}
	if samples < 3 {
		confidence = "low"
	}
	if len(warnings) >= 3 && confidence == "high" {
		confidence = "medium"
	}
	reasons = append(reasons, "fallback level: "+level)
	return confidence, reasons
}

func medianOf(sorted []float64) float64 {
	if len(sorted) == 0 {
		return 0
	}
	mid := len(sorted) / 2
	if len(sorted)%2 == 1 {
		return sorted[mid]
	}
	return (sorted[mid-1] + sorted[mid]) / 2
}

func stdDev(values []float64) float64 {
	if len(values) == 0 {
		return 0
	}
	mean := 0.0
	for _, v := range values {
		mean += v
	}
	mean /= float64(len(values))
	sum := 0.0
	for _, v := range values {
		sum += (v - mean) * (v - mean)
	}
	return math.Sqrt(sum / float64(len(values)))
}

func clamp(v, low, high float64) float64 { return math.Max(low, math.Min(high, v)) }
func round2(v float64) float64           { return math.Round(v*100) / 100 }
