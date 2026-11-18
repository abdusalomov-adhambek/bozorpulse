package forecast

import (
	_ "embed"
	"encoding/json"
	"fmt"
)

// calibration.json is the implementation of the two rules documents: category
// blocks from price-prediction-rules-claude.md and canonical categories/tags
// from price_prediction_rules-chatgpt.md. It is embedded so the API has no
// runtime file dependency; rebuild after editing it.
//
//go:embed calibration.json
var calibrationRaw []byte

type CategoryCalibration struct {
	CanonicalCategory string   `json:"canonical_category"`
	MarketTags        []string `json:"market_tags"`
	WeatherEnabled    bool     `json:"weather_enabled"`
	Sensitivity       string   `json:"sensitivity"`
	BaseCapPct        float64  `json:"base_cap_pct"`
	K                 float64  `json:"k"`
	BetaPctPerSD      *float64 `json:"beta_pct_per_sd"`
	LagDays           int      `json:"lag_days"`
	DefaultWindows    []string `json:"default_windows"`
	Direction         string   `json:"direction"`
	Notes             string   `json:"notes"`
}

type Calibration struct {
	Version          string                         `json:"version"`
	Note             string                         `json:"note"`
	DefaultPriceType string                         `json:"default_price_type"`
	DefaultRegulated bool                           `json:"default_regulated"`
	Categories       map[string]CategoryCalibration `json:"categories"`
}

func LoadCalibration() (Calibration, error) {
	var c Calibration
	if err := json.Unmarshal(calibrationRaw, &c); err != nil {
		return c, fmt.Errorf("forecast calibration: %w", err)
	}
	if c.DefaultPriceType == "" {
		c.DefaultPriceType = "spot"
	}
	return c, nil
}

// For returns the category block, with a conservative default for unknown IDs.
func (c Calibration) For(categoryID string) CategoryCalibration {
	if block, ok := c.Categories[categoryID]; ok {
		return block
	}
	return CategoryCalibration{
		CanonicalCategory: "unclassified",
		WeatherEnabled:    false,
		Sensitivity:       "unknown",
		BaseCapPct:        3,
		K:                 2,
		Direction:         "neutral",
		Notes:             "category not classified; weather layer disabled",
	}
}

// Calibrated is true only when a backtested beta exists. Until then the
// weather adjustment must remain zero (active policy decision).
func (b CategoryCalibration) Calibrated() bool {
	return b.BetaPctPerSD != nil && *b.BetaPctPerSD != 0 && b.WeatherEnabled
}
