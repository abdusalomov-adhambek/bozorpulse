package domain

import (
	"encoding/json"
	"strings"
	"testing"
	"time"
)

var testNow = time.Date(2026, 11, 18, 12, 0, 0, 0, time.UTC)

func risingForecast() ForecastInput {
	return ForecastInput{
		Predicted: 110, Low: 103, High: 117, Baseline: 100,
		ChangePct: 10, TrendPct: 5, HorizonDays: 30, Samples: 12,
		DataStatus: StatusDemo, Basis: "Based on 12 prices",
	}
}

func TestBuildProjectionAppliesRatioToTierPrice(t *testing.T) {
	got := BuildProjection(risingForecast(), 200, 10, testNow)
	if got.Predicted == nil || *got.Predicted != 220 {
		t.Fatalf("expected tier price 200 scaled by 1.10 => 220, got %+v", got.Predicted)
	}
	if got.EstimatedBudget == nil || *got.EstimatedBudget != 2200 {
		t.Fatalf("budget should be 10 * 220 = 2200, got %+v", got.EstimatedBudget)
	}
	if got.CurrentUnitPrice == nil || *got.CurrentUnitPrice != 200 {
		t.Fatalf("current tier price lost: %+v", got.CurrentUnitPrice)
	}
	if got.Recommendation != RecBuyNow {
		t.Fatalf("rising forecast should recommend buying now, got %s", got.Recommendation)
	}
}

func TestBuildProjectionWaitSignal(t *testing.T) {
	in := risingForecast()
	in.Predicted, in.Low, in.High, in.ChangePct = 90, 84, 95, -10
	got := BuildProjection(in, 200, 5, testNow)
	if got.Recommendation != RecWait {
		t.Fatalf("falling forecast should recommend waiting, got %s", got.Recommendation)
	}
	if got.SavingVsNow == nil || *got.SavingVsNow <= 0 {
		t.Fatalf("expected positive saving vs now, got %+v", got.SavingVsNow)
	}
}

func TestBuildProjectionMonitorSignal(t *testing.T) {
	in := risingForecast()
	in.Predicted, in.Low, in.High, in.ChangePct = 101, 97, 104, 1
	got := BuildProjection(in, 200, 5, testNow)
	if got.Recommendation != RecMonitor {
		t.Fatalf("flat forecast should recommend monitoring, got %s", got.Recommendation)
	}
	if got.Reason == "" {
		t.Fatal("monitor must explain itself")
	}
}

func TestInsufficientForecastProducesNullsAndNoBudget(t *testing.T) {
	in := risingForecast()
	in.DataStatus = StatusInsufficient
	got := BuildProjection(in, 200, 10, testNow)
	if got.Predicted != nil || got.Low != nil || got.High != nil ||
		got.EstimatedBudget != nil || got.CurrentUnitPrice != nil || got.SavingVsNow != nil {
		t.Fatalf("insufficient projection must be all null: %+v", got)
	}
	if got.Recommendation != RecInsufficient {
		t.Fatalf("expected insufficient recommendation, got %s", got.Recommendation)
	}
	raw, err := json.Marshal(got)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(raw), `"predicted":null`) {
		t.Fatalf("JSON must serialize null, got %s", raw)
	}
}

func TestInsufficientSamplesDoNotInventNumbers(t *testing.T) {
	in := risingForecast()
	in.Samples = 2
	got := BuildProjection(in, 200, 10, testNow)
	if got.Predicted != nil || got.EstimatedBudget != nil {
		t.Fatalf("fewer than 3 samples must not produce numbers: %+v", got)
	}
}

func TestDemoStatusCarriesDisclaimer(t *testing.T) {
	got := BuildProjection(risingForecast(), 200, 10, testNow)
	if got.Disclaimer == "" || got.DataStatus != StatusDemo {
		t.Fatalf("demo projection needs label + disclaimer: %+v", got)
	}
	in := risingForecast()
	in.DataStatus = StatusLive
	live := BuildProjection(in, 200, 10, testNow)
	if live.Disclaimer != "" || live.DataStatus != StatusLive {
		t.Fatalf("live projection must not claim demo: %+v", live)
	}
}

func TestValidateTargetDate(t *testing.T) {
	if !ValidateTargetDate("", testNow) {
		t.Fatal("empty target date is valid")
	}
	if !ValidateTargetDate("2026-11-18", testNow) || !ValidateTargetDate("2026-12-18", testNow) {
		t.Fatal("dates inside the 30-day horizon must be accepted")
	}
	if ValidateTargetDate("2026-11-17", testNow) || ValidateTargetDate("2027-01-01", testNow) {
		t.Fatal("dates outside the 30-day horizon must be rejected")
	}
	if ValidateTargetDate("18/11/2026", testNow) {
		t.Fatal("invalid date format must be rejected")
	}
}
