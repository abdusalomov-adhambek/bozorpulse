package domain

import "encoding/json"

// DataStatus is the provenance label attached to every AI-facing response.
type DataStatus string

const (
	DataStatusLive         DataStatus = "live"
	DataStatusDemo         DataStatus = "demo"
	DataStatusInsufficient DataStatus = "insufficient"
)

// Intent is the classified purpose of a chat message.
type Intent string

const (
	IntentMarketPrice       Intent = "market_price"
	IntentSupplierDiscovery Intent = "supplier_discovery"
	IntentProductSearch     Intent = "product_search"
	IntentOrderStatus       Intent = "order_status"
	IntentOnboarding        Intent = "onboarding"
	IntentSmallTalk         Intent = "small_talk"
	IntentOutOfScope        Intent = "out_of_scope"
	IntentFallback          Intent = "fallback"
	IntentBuyingAdvice      Intent = "buying_advice"
)

// ToolName is an allowlisted tool the assistant may call.
type ToolName string

const (
	ToolMarketStats        ToolName = "market_stats"
	ToolRecommendSuppliers ToolName = "recommend_suppliers"
	ToolSearchProducts     ToolName = "search_products"
	ToolGetOrder           ToolName = "get_order"
	ToolCompareOffers      ToolName = "compare_offers"
	ToolPriceForecast      ToolName = "price_forecast"
)

// ToolCall records one tool execution for audit and UI evidence.
type ToolCall struct {
	ID         string          `json:"id"`
	Name       ToolName        `json:"name"`
	Arguments  json.RawMessage `json:"arguments"`
	Result     json.RawMessage `json:"result,omitempty"`
	Error      string          `json:"error,omitempty"`
	DurationMS int64           `json:"duration_ms"`
}

// Source points at the platform record backing a number in a reply.
type Source struct {
	Type       string `json:"type"`
	ID         string `json:"id"`
	Label      string `json:"label,omitempty"`
	ObservedAt string `json:"observed_at,omitempty"`
	IsDemo     bool   `json:"is_demo"`
}

type ChatMessage struct {
	Role       string     `json:"role"`
	Content    string     `json:"content"`
	ToolCalls  []ToolCall `json:"tool_calls,omitempty"`
	Sources    []Source   `json:"sources,omitempty"`
	DataStatus DataStatus `json:"data_status,omitempty"`
	Intent     Intent     `json:"intent,omitempty"`
}

// ChatAction is a UI-only next step. The AI never executes it.
type ChatAction struct {
	Label  string            `json:"label"`
	Kind   string            `json:"kind"`
	Target string            `json:"target"`
	Meta   map[string]string `json:"meta,omitempty"`
}

// ChatReply is the public response of POST /ai/chat.
type ChatReply struct {
	Reply          string       `json:"reply"`
	Intent         Intent       `json:"intent"`
	ToolCalls      []ToolCall   `json:"tool_calls"`
	DataStatus     DataStatus   `json:"data_status"`
	Sources        []Source     `json:"sources"`
	Disclaimer     string       `json:"disclaimer,omitempty"`
	Suggestions    []string     `json:"suggestions"`
	Actions        []ChatAction `json:"actions,omitempty"`
	Locale         string       `json:"locale"`
	ConversationID string       `json:"conversation_id,omitempty"`
}

// Factors are the deterministic sub-scores of a recommendation.
type Factors struct {
	Price        float64 `json:"price"`
	Freshness    float64 `json:"freshness"`
	Region       float64 `json:"region"`
	Verification float64 `json:"verification"`
}

type Recommendation struct {
	SupplierID   string     `json:"supplier_id"`
	SupplierName string     `json:"supplier_name"`
	ProductID    string     `json:"product_id"`
	ProductName  string     `json:"product_name"`
	Price        float64    `json:"price"`
	Score        float64    `json:"score"`
	Factors      Factors    `json:"factors"`
	Reasons      []string   `json:"reasons"`
	IsDemo       bool       `json:"is_demo"`
	DataStatus   DataStatus `json:"data_status"`
}

type RecommendationSet struct {
	ProductID  string           `json:"product_id"`
	Items      []Recommendation `json:"items"`
	DataStatus DataStatus       `json:"data_status"`
	Disclaimer string           `json:"disclaimer,omitempty"`
}

// Narrow hit types keep the ai module decoupled from catalog/procurement domains.
type ProductHit struct {
	ID             string  `json:"id"`
	Name           string  `json:"name"`
	SKU            string  `json:"sku"`
	Unit           string  `json:"unit"`
	EffectivePrice float64 `json:"effective_price"`
	IsDemo         bool    `json:"is_demo"`
}

type OrderItemHit struct {
	ProductID string  `json:"product_id"`
	Name      string  `json:"name"`
	Quantity  int     `json:"quantity"`
	UnitPrice float64 `json:"unit_price"`
	LineTotal float64 `json:"line_total"`
}

type OrderHit struct {
	ID          string         `json:"id"`
	Status      string         `json:"status"`
	TotalAmount float64        `json:"total_amount"`
	CreatedAt   string         `json:"created_at"`
	Items       []OrderItemHit `json:"items"`
}

type MarketSampleHit struct {
	ID          string  `json:"id"`
	ProductID   string  `json:"product_id"`
	ProductName string  `json:"product_name"`
	Price       float64 `json:"price"`
	Unit        string  `json:"unit"`
	RegionID    string  `json:"region_id"`
	CapturedAt  string  `json:"captured_at"`
	IsDemo      bool    `json:"is_demo"`
}

type MarketStatsFigure struct {
	Min    *float64 `json:"min"`
	Median *float64 `json:"median"`
	Avg    *float64 `json:"avg"`
	Count  int      `json:"count"`
}

type MarketStatsResult struct {
	Samples     []MarketSampleHit `json:"samples"`
	Stats       MarketStatsFigure `json:"stats"`
	DataStatus  DataStatus        `json:"data_status"`
	GeneratedAt string            `json:"generated_at"`
	Disclaimer  string            `json:"disclaimer,omitempty"`
}

// WeatherInfo is the current weather used as one input to a price outlook.
type WeatherInfo struct {
	Region          string  `json:"region"`
	TemperatureC    float64 `json:"temperature_c"`
	PrecipitationMM float64 `json:"precipitation_mm"`
	ObservedAt      string  `json:"observed_at,omitempty"`
	Source          string  `json:"source"`
}

// PriceIndicators are separate, explicitly defined price targets. A single
// "price" is never silently substituted for another indicator.
type PriceIndicators struct {
	MarketMedian        float64 `json:"market_median"`
	MarketAverage       float64 `json:"market_average"`
	MarketMin           float64 `json:"market_min"`
	MarketMax           float64 `json:"market_max"`
	ExpectedTransaction float64 `json:"expected_transaction_price"`
	Basis               string  `json:"basis"`
}

// PriceForecast is a deliberately conservative, explainable estimate. It is
// never a quote. The leading fields are the wire contract used by chat cards;
// the remaining fields implement the extended v2.0 intelligence schema.
type PriceForecast struct {
	Predicted   float64      `json:"predicted"`
	Low         float64      `json:"low"`
	High        float64      `json:"high"`
	TrendPct    float64      `json:"trend_pct"`
	Weather     *WeatherInfo `json:"weather,omitempty"`
	Samples     int          `json:"samples"`
	HorizonDays int          `json:"horizon_days"`
	Basis       string       `json:"basis"`
	DataStatus  DataStatus   `json:"data_status"`

	Status             string             `json:"status"`
	Indicator          string             `json:"indicator,omitempty"`
	CentralTendency    string             `json:"central_tendency,omitempty"`
	IntervalType       string             `json:"interval_type,omitempty"`
	ReferencePrice     float64            `json:"reference_price,omitempty"`
	ReferenceIndicator string             `json:"reference_indicator,omitempty"`
	PredictedChangePct float64            `json:"predicted_change_pct,omitempty"`
	AsOf               string             `json:"as_of,omitempty"`
	PriceIndicators    *PriceIndicators   `json:"price_indicators,omitempty"`
	Confidence         string             `json:"confidence,omitempty"`
	ConfidenceReasons  []string           `json:"confidence_reasons,omitempty"`
	FallbackLevel      string             `json:"fallback_level,omitempty"`
	ModelType          string             `json:"model_type,omitempty"`
	ModelVersion       string             `json:"model_version,omitempty"`
	BaselinePrice      float64            `json:"baseline_price,omitempty"`
	AdjustmentsPct     map[string]float64 `json:"adjustments_pct,omitempty"`
	PriceType          string             `json:"price_type,omitempty"`
	Regulated          bool               `json:"regulated,omitempty"`
	CanonicalCategory  string             `json:"canonical_category,omitempty"`
	MarketTags         []string           `json:"market_tags,omitempty"`
	Warnings           []string           `json:"warnings,omitempty"`
}

// BuyingOption is one ranked "what/where to buy" answer from compare_offers.
type BuyingOption struct {
	ProductID       string         `json:"product_id"`
	ProductName     string         `json:"product_name"`
	SupplierID      string         `json:"supplier_id"`
	SupplierName    string         `json:"supplier_name"`
	Unit            string         `json:"unit"`
	UnitPrice       float64        `json:"unit_price"`
	EffectivePrice  float64        `json:"effective_price"`
	LineTotal       float64        `json:"line_total"`
	Quantity        int            `json:"quantity"`
	Stock           int            `json:"stock"`
	MinOrder        int            `json:"min_order"`
	MaxOrder        int            `json:"max_order"`
	DeliveryMinDays int            `json:"delivery_min_days"`
	DeliveryMaxDays int            `json:"delivery_max_days"`
	RegionID        string         `json:"region_id"`
	IsDemo          bool           `json:"is_demo"`
	Why             []string       `json:"why"`
	Forecast        *PriceForecast `json:"forecast,omitempty"`
}

// CandidateInfo is a same-category product considered for recommendations.
type CandidateInfo struct {
	ProductID, Name, SupplierID, SupplierName, RegionID string
	Price                                               float64
	UpdatedAt                                           string
	HasINN                                              bool
	Verified                                            bool
	UserConfirmed                                       bool
	IsDemo                                              bool
}

// ConversationSummary is the compact context carried between turns.
type ConversationSummary struct {
	Intent         Intent         `json:"intent"`
	ProductID      string         `json:"product_id"`
	Q              string         `json:"q"`
	OrderID        string         `json:"order_id"`
	Locale         string         `json:"locale"`
	Quantity       int            `json:"quantity"`
	WarehouseID    string         `json:"warehouse_id"`
	RegionID       string         `json:"region_id"`
	Budget         float64        `json:"budget"`
	Shortlist      []BuyingOption `json:"shortlist,omitempty"`
	LastDataStatus DataStatus     `json:"last_data_status,omitempty"`
}
