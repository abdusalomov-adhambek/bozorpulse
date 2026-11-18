package domain

import (
	"fmt"
	"math"
	"strings"
	"time"
)

// DemoDisclaimer is attached to every demo-backed projection. It is derived
// from data_status on read, so it is always present even for plans persisted
// before the field was introduced.
const DemoDisclaimer = "Sample market data for demonstration. Not a live market quote."

// BuildProjection turns a forecast plus the price the customer can actually
// get for the requested quantity into a conservative projection.
//
// The forecast's absolute prediction is based on a market median, so it must
// never be multiplied by the planned quantity directly. Instead the forecast's
// relative change is applied to the quantity-specific tier price.
func BuildProjection(in ForecastInput, tierPrice float64, quantity int, now time.Time) PriceProjection {
	out := PriceProjection{
		HorizonDays: in.HorizonDays,
		TrendPct:    round2(in.TrendPct),
		DataStatus:  StatusInsufficient,
		Reason:      "Not enough price history for a 30-day estimate.",
		GeneratedAt: now.UTC().Format(time.RFC3339),
	}
	if out.HorizonDays <= 0 {
		out.HorizonDays = 30
	}
	if len(in.Warnings) > 0 {
		out.Note = strings.Join(in.Warnings, "; ")
	}

	// Guard: never invent a number from an insufficient or malformed forecast.
	base := in.Baseline
	if base <= 0 {
		base = in.Predicted // defensive fallback for older forecast payloads
	}
	usable := (in.DataStatus == StatusDemo || in.DataStatus == StatusLive) &&
		in.Predicted > 0 && base > 0 && in.Low > 0 && in.High > 0 &&
		tierPrice > 0 && quantity > 0
	if !usable || in.Samples < 3 {
		out.Recommendation = RecInsufficient
		out.Reason = "Not enough price history for a 30-day estimate. Compare live offers instead."
		out.DataStatus = StatusInsufficient
		out.Note = appendNote(out.Note, "Forecast does not provide a usable 30-day estimate.")
		return out
	}

	// Ratios from the forecast's own reference level, applied to the real tier
	// price. Clamping keeps a malformed upstream payload from producing absurd
	// budget figures; normal forecasts stay well inside this band.
	ratioP := clamp(in.Predicted/base, 0.5, 2)
	ratioL := clamp(in.Low/base, 0.5, 2)
	ratioH := clamp(in.High/base, 0.5, 2)

	predicted := tierPrice * ratioP
	low := tierPrice * ratioL
	high := tierPrice * ratioH
	changePct := (ratioP - 1) * 100

	out.CurrentUnitPrice = fptr(round2(tierPrice))
	out.Predicted = fptr(round2(predicted))
	out.Low = fptr(round2(low))
	out.High = fptr(round2(high))
	out.ChangePct = fptr(round2(changePct))
	out.EstimatedBudget = fptr(round2(predicted * float64(quantity)))
	out.BudgetLow = fptr(round2(low * float64(quantity)))
	out.BudgetHigh = fptr(round2(high * float64(quantity)))
	out.SavingVsNow = fptr(round2((tierPrice - predicted) * float64(quantity)))
	out.Basis = in.Basis
	out.DataStatus = in.DataStatus
	if in.DataStatus == StatusDemo {
		out.Disclaimer = DemoDisclaimer
	}

	// Conservative signal: only claim a drop when even the optimistic bound is
	// clearly below today's price, and only claim a rise when even the
	// pessimistic bound is clearly above it.
	switch {
	case high < tierPrice*0.98:
		out.Recommendation = RecWait
		out.Reason = "Even the high estimate is below today's tier price; consider waiting."
	case low > tierPrice*1.02:
		out.Recommendation = RecBuyNow
		out.Reason = "Even the low estimate is above today's tier price; buying sooner may avoid an increase."
	default:
		out.Recommendation = RecMonitor
		out.Reason = "No clear price signal — compare live offers before buying."
	}
	return out
}

// InsufficientProjection builds the null-valued projection used when a plan
// item cannot be projected at all (e.g. an out-of-range target date).
func InsufficientProjection(reason string, now time.Time) PriceProjection {
	return PriceProjection{
		HorizonDays:    30,
		Recommendation: RecInsufficient,
		Reason:         reason,
		DataStatus:     StatusInsufficient,
		GeneratedAt:    now.UTC().Format(time.RFC3339),
	}
}

func appendNote(existing, extra string) string {
	if existing == "" {
		return extra
	}
	return fmt.Sprintf("%s; %s", existing, extra)
}

func fptr(v float64) *float64 { return &v }

func round2(v float64) float64 { return math.Round(v*100) / 100 }

func clamp(v, low, high float64) float64 { return math.Max(low, math.Min(high, v)) }
