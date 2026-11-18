package application

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"strings"

	"github.com/bozorpulse/bozorpulse-ai/backend/internal/modules/ai/domain"
	"github.com/bozorpulse/bozorpulse-ai/backend/internal/modules/ai/internal/query"
	"github.com/bozorpulse/bozorpulse-ai/backend/internal/platform/auth"
)

// Tool is one allowlisted operation. Implementations never accept SQL, URLs
// or arbitrary filters, and always re-check the authenticated principal.
type Tool interface {
	Name() domain.ToolName
	Spec() ToolSpec
	validate(args json.RawMessage) error
	Execute(ctx context.Context, p auth.Principal, args json.RawMessage) (any, []domain.Source, domain.DataStatus, error)
}

type Registry struct {
	tools map[domain.ToolName]Tool
	order []Tool
}

func NewRegistry(tools ...Tool) *Registry {
	r := &Registry{tools: make(map[domain.ToolName]Tool, len(tools)), order: tools}
	for _, t := range tools {
		r.tools[t.Name()] = t
	}
	return r
}

func (r *Registry) Add(t Tool) {
	r.tools[t.Name()] = t
	r.order = append(r.order, t)
}

func (r *Registry) Specs() []ToolSpec {
	out := make([]ToolSpec, 0, len(r.order))
	for _, t := range r.order {
		out = append(out, t.Spec())
	}
	return out
}

// Validate rejects unknown tools and malformed arguments before execution.
func (r *Registry) Validate(name domain.ToolName, args json.RawMessage) error {
	t, ok := r.tools[name]
	if !ok {
		return fmt.Errorf("unknown tool %q", name)
	}
	return t.validate(args)
}

func (r *Registry) Execute(ctx context.Context, p auth.Principal, name domain.ToolName, args json.RawMessage) (any, []domain.Source, domain.DataStatus, error) {
	t, ok := r.tools[name]
	if !ok {
		return nil, nil, "", fmt.Errorf("unknown tool %q", name)
	}
	if err := t.validate(args); err != nil {
		return nil, nil, "", err
	}
	return t.Execute(ctx, p, args)
}

func decodeArgs(args json.RawMessage, dst any) error {
	if len(bytes.TrimSpace(args)) == 0 {
		args = json.RawMessage(`{}`)
	}
	dec := json.NewDecoder(bytes.NewReader(args))
	dec.DisallowUnknownFields()
	if err := dec.Decode(dst); err != nil {
		return fmt.Errorf("invalid tool arguments: %w", err)
	}
	if dec.More() {
		return errors.New("invalid tool arguments: extra JSON values")
	}
	// Guard against trailing garbage after the object.
	var extra any
	if err := dec.Decode(&extra); err != io.EOF {
		return errors.New("invalid tool arguments: trailing data")
	}
	return nil
}

func schema(s string) json.RawMessage { return json.RawMessage(s) }

// ---------- market_stats ----------

type marketStatsTool struct{ reader MarketStatsReader }

func NewMarketStatsTool(r MarketStatsReader) Tool { return &marketStatsTool{reader: r} }
func (t *marketStatsTool) Name() domain.ToolName  { return domain.ToolMarketStats }
func (t *marketStatsTool) Spec() ToolSpec {
	return ToolSpec{
		Name:        t.Name(),
		Description: "Get market price statistics (min, median, avg, count) from platform price samples.",
		InputSchema: schema(`{"type":"object","properties":{"category_id":{"type":"string"},"q":{"type":"string"}},"additionalProperties":false}`),
	}
}
func (t *marketStatsTool) validate(args json.RawMessage) error {
	var in struct {
		CategoryID string `json:"category_id"`
		Q          string `json:"q"`
	}
	return decodeArgs(args, &in)
}
func (t *marketStatsTool) Execute(ctx context.Context, _ auth.Principal, args json.RawMessage) (any, []domain.Source, domain.DataStatus, error) {
	var in struct {
		CategoryID string `json:"category_id"`
		Q          string `json:"q"`
	}
	if err := decodeArgs(args, &in); err != nil {
		return nil, nil, "", err
	}
	result, err := t.reader.MarketStats(ctx, query.Normalize(in.Q), in.CategoryID)
	if err != nil {
		return nil, nil, "", err
	}
	sources := make([]domain.Source, 0, len(result.Samples))
	for _, s := range result.Samples {
		sources = append(sources, domain.Source{Type: "price_sample", ID: s.ID, Label: s.ProductName, ObservedAt: s.CapturedAt, IsDemo: s.IsDemo})
	}
	return result, sources, result.DataStatus, nil
}

// ---------- recommend_suppliers ----------

type recommendTool struct{ recommender SupplierRecommender }

func NewRecommendTool(r SupplierRecommender) Tool { return &recommendTool{recommender: r} }
func (t *recommendTool) Name() domain.ToolName    { return domain.ToolRecommendSuppliers }
func (t *recommendTool) Spec() ToolSpec {
	return ToolSpec{
		Name:        t.Name(),
		Description: "Rank suppliers offering products in the same category as a given product.",
		InputSchema: schema(`{"type":"object","properties":{"product_id":{"type":"string"},"limit":{"type":"integer","minimum":1,"maximum":20}},"required":["product_id"],"additionalProperties":false}`),
	}
}
func (t *recommendTool) validate(args json.RawMessage) error {
	var in struct {
		ProductID string `json:"product_id"`
		Limit     *int   `json:"limit"`
	}
	if err := decodeArgs(args, &in); err != nil {
		return err
	}
	if in.ProductID == "" {
		return errors.New("product_id is required")
	}
	if in.Limit != nil && (*in.Limit < 1 || *in.Limit > 20) {
		return errors.New("limit must be between 1 and 20")
	}
	return nil
}
func (t *recommendTool) Execute(ctx context.Context, p auth.Principal, args json.RawMessage) (any, []domain.Source, domain.DataStatus, error) {
	var in struct {
		ProductID string `json:"product_id"`
		Limit     *int   `json:"limit"`
	}
	if err := decodeArgs(args, &in); err != nil {
		return nil, nil, "", err
	}
	limit := 5
	if in.Limit != nil {
		limit = *in.Limit
	}
	set, err := t.recommender.RecommendSet(ctx, in.ProductID, p, limit)
	if err != nil {
		return nil, nil, "", err
	}
	sources := make([]domain.Source, 0, len(set.Items))
	for _, item := range set.Items {
		sources = append(sources, domain.Source{Type: "product", ID: item.ProductID, Label: item.ProductName, IsDemo: item.IsDemo})
	}
	return set, sources, set.DataStatus, nil
}

// ---------- search_products ----------

type searchTool struct{ searcher ProductSearcher }

func NewSearchTool(s ProductSearcher) Tool  { return &searchTool{searcher: s} }
func (t *searchTool) Name() domain.ToolName { return domain.ToolSearchProducts }
func (t *searchTool) Spec() ToolSpec {
	return ToolSpec{
		Name:        t.Name(),
		Description: "Search the product catalog by name or category.",
		InputSchema: schema(`{"type":"object","properties":{"q":{"type":"string"},"category_id":{"type":"string"},"limit":{"type":"integer","minimum":1,"maximum":20}},"additionalProperties":false}`),
	}
}
func (t *searchTool) validate(args json.RawMessage) error {
	var in struct {
		Q          string `json:"q"`
		CategoryID string `json:"category_id"`
		Limit      *int   `json:"limit"`
	}
	if err := decodeArgs(args, &in); err != nil {
		return err
	}
	if in.Limit != nil && (*in.Limit < 1 || *in.Limit > 20) {
		return errors.New("limit must be between 1 and 20")
	}
	return nil
}
func (t *searchTool) Execute(ctx context.Context, _ auth.Principal, args json.RawMessage) (any, []domain.Source, domain.DataStatus, error) {
	var in struct {
		Q          string `json:"q"`
		CategoryID string `json:"category_id"`
		Limit      *int   `json:"limit"`
	}
	if err := decodeArgs(args, &in); err != nil {
		return nil, nil, "", err
	}
	limit := 10
	if in.Limit != nil {
		limit = *in.Limit
	}
	items, err := t.searcher.Search(ctx, query.Normalize(in.Q), in.CategoryID, limit)
	if err != nil {
		return nil, nil, "", err
	}
	status := domain.DataStatusInsufficient
	if len(items) > 0 {
		status = domain.DataStatusDemo
		for _, item := range items {
			if !item.IsDemo {
				status = domain.DataStatusLive
				break
			}
		}
	}
	sources := make([]domain.Source, 0, len(items))
	for _, item := range items {
		sources = append(sources, domain.Source{Type: "product", ID: item.ID, Label: item.Name, IsDemo: item.IsDemo})
	}
	return map[string]any{"products": items, "count": len(items)}, sources, status, nil
}

// ---------- get_order ----------

type getOrderTool struct{ getter OrderGetter }

func NewGetOrderTool(g OrderGetter) Tool      { return &getOrderTool{getter: g} }
func (t *getOrderTool) Name() domain.ToolName { return domain.ToolGetOrder }
func (t *getOrderTool) Spec() ToolSpec {
	return ToolSpec{
		Name:        t.Name(),
		Description: "Get the status and total of one of the current user's orders.",
		InputSchema: schema(`{"type":"object","properties":{"order_id":{"type":"string"}},"required":["order_id"],"additionalProperties":false}`),
	}
}
func (t *getOrderTool) validate(args json.RawMessage) error {
	var in struct {
		OrderID string `json:"order_id"`
	}
	if err := decodeArgs(args, &in); err != nil {
		return err
	}
	if in.OrderID == "" {
		return errors.New("order_id is required")
	}
	return nil
}
func (t *getOrderTool) Execute(ctx context.Context, p auth.Principal, args json.RawMessage) (any, []domain.Source, domain.DataStatus, error) {
	var in struct {
		OrderID string `json:"order_id"`
	}
	if err := decodeArgs(args, &in); err != nil {
		return nil, nil, "", err
	}
	order, err := t.getter.Get(ctx, in.OrderID, p)
	if err != nil {
		return nil, nil, "", err
	}
	status := domain.DataStatusLive
	if order.ID == "" {
		status = domain.DataStatusInsufficient
	}
	return order, []domain.Source{{Type: "order", ID: order.ID, Label: "Order " + order.ID}}, status, nil
}

// ---------- compare_offers (D6) ----------

type compareOffersTool struct{ comparer OfferComparer }

func NewCompareOffersTool(c OfferComparer) Tool    { return &compareOffersTool{comparer: c} }
func (t *compareOffersTool) Name() domain.ToolName { return domain.ToolCompareOffers }
func (t *compareOffersTool) Spec() ToolSpec {
	return ToolSpec{
		Name:        t.Name(),
		Description: "Rank concrete supplier offers for a purchase: what to buy and from whom, with effective price, stock and delivery days.",
		InputSchema: schema(`{"type":"object","properties":{"q":{"type":"string"},"category_id":{"type":"string"},"product_id":{"type":"string"},"warehouse_id":{"type":"string"},"quantity":{"type":"integer","minimum":1,"maximum":10000},"limit":{"type":"integer","minimum":1,"maximum":10}},"additionalProperties":false}`),
	}
}
func (t *compareOffersTool) validate(args json.RawMessage) error {
	var in struct {
		Q           string `json:"q"`
		CategoryID  string `json:"category_id"`
		ProductID   string `json:"product_id"`
		WarehouseID string `json:"warehouse_id"`
		Quantity    *int   `json:"quantity"`
		Limit       *int   `json:"limit"`
	}
	if err := decodeArgs(args, &in); err != nil {
		return err
	}
	if in.Q == "" && in.CategoryID == "" && in.ProductID == "" {
		return errors.New("at least one of q, category_id or product_id is required")
	}
	if in.Quantity != nil && (*in.Quantity < 1 || *in.Quantity > 10000) {
		return errors.New("quantity must be between 1 and 10000")
	}
	if in.Limit != nil && (*in.Limit < 1 || *in.Limit > 10) {
		return errors.New("limit must be between 1 and 10")
	}
	return nil
}
func (t *compareOffersTool) Execute(ctx context.Context, _ auth.Principal, args json.RawMessage) (any, []domain.Source, domain.DataStatus, error) {
	var in struct {
		Q           string `json:"q"`
		CategoryID  string `json:"category_id"`
		ProductID   string `json:"product_id"`
		WarehouseID string `json:"warehouse_id"`
		Quantity    *int   `json:"quantity"`
		Limit       *int   `json:"limit"`
	}
	if err := decodeArgs(args, &in); err != nil {
		return nil, nil, "", err
	}
	qty := 1
	if in.Quantity != nil {
		qty = *in.Quantity
	}
	limit := 5
	if in.Limit != nil {
		limit = *in.Limit
	}
	options, status, sources, err := t.comparer.Compare(ctx, query.Normalize(in.Q), in.CategoryID, in.ProductID, in.WarehouseID, qty, limit)
	if err != nil {
		return nil, nil, "", err
	}
	return map[string]any{"options": options, "count": len(options), "data_status": status}, sources, status, nil
}

// ---------- price_forecast ----------

type priceForecastTool struct {
	forecaster PriceForecaster
	searcher   ProductSearcher
}

func NewPriceForecastTool(f PriceForecaster, s ProductSearcher) Tool {
	return &priceForecastTool{forecaster: f, searcher: s}
}

func (t *priceForecastTool) Name() domain.ToolName { return domain.ToolPriceForecast }

func (t *priceForecastTool) Spec() ToolSpec {
	return ToolSpec{
		Name:        t.Name(),
		Description: "Estimate a product's likely price over the next 30 days from past price samples and current weather. An estimate, not a supplier quote.",
		InputSchema: schema(`{"type":"object","properties":{"product_id":{"type":"string"},"q":{"type":"string"},"region_id":{"type":"string"}},"additionalProperties":false}`),
	}
}

func (t *priceForecastTool) validate(args json.RawMessage) error {
	var in struct {
		ProductID string `json:"product_id"`
		Q         string `json:"q"`
		RegionID  string `json:"region_id"`
	}
	if err := decodeArgs(args, &in); err != nil {
		return err
	}
	if in.ProductID == "" && strings.TrimSpace(in.Q) == "" {
		return errors.New("product_id or q is required")
	}
	return nil
}

func (t *priceForecastTool) Execute(ctx context.Context, _ auth.Principal, args json.RawMessage) (any, []domain.Source, domain.DataStatus, error) {
	var in struct {
		ProductID string `json:"product_id"`
		Q         string `json:"q"`
		RegionID  string `json:"region_id"`
	}
	if err := decodeArgs(args, &in); err != nil {
		return nil, nil, "", err
	}
	productID := in.ProductID
	sources := []domain.Source{}
	if productID == "" {
		hits, err := t.searcher.Search(ctx, query.Normalize(in.Q), "", 1)
		if err != nil {
			return nil, nil, "", err
		}
		if len(hits) == 0 {
			return domain.PriceForecast{HorizonDays: 30, DataStatus: domain.DataStatusInsufficient, Basis: "No matching product found."},
				sources, domain.DataStatusInsufficient, nil
		}
		productID = hits[0].ID
		sources = append(sources, domain.Source{Type: "product", ID: productID, Label: hits[0].Name})
	}
	regionID := in.RegionID
	if regionID == "" {
		regionID = "reg-01" // Tashkent — a stated default beats silently omitting weather
	}
	fc, err := t.forecaster.Forecast(ctx, productID, regionID)
	if err != nil {
		return nil, nil, "", err
	}
	if in.RegionID == "" {
		fc.Basis += " Assumed Tashkent because no region was given."
	}
	if len(sources) == 0 {
		sources = append(sources, domain.Source{Type: "product", ID: productID, Label: "Price forecast"})
	}
	return fc, sources, fc.DataStatus, nil
}
