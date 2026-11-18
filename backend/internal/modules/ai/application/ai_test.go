package application

import (
	"context"
	"encoding/json"
	"strings"
	"testing"
	"time"

	"github.com/bozorpulse/bozorpulse-ai/backend/internal/modules/ai/domain"
	"github.com/bozorpulse/bozorpulse-ai/backend/internal/platform/auth"
	"github.com/bozorpulse/bozorpulse-ai/backend/internal/platform/config"
	"github.com/bozorpulse/bozorpulse-ai/backend/internal/shared"
)

var testPrincipal = auth.Principal{ID: "u1", Role: auth.RoleCustomer, Status: auth.StatusConfirmed, CompanyID: "c1"}

// ---- fakes ----

type fakeStats struct {
	result domain.MarketStatsResult
	err    error
	gotQ   string
}

func (f *fakeStats) MarketStats(_ context.Context, q, _ string) (domain.MarketStatsResult, error) {
	f.gotQ = q
	return f.result, f.err
}

type fakeSearch struct {
	items []domain.ProductHit
	gotQ  string
}

func (f *fakeSearch) Search(_ context.Context, q, _ string, _ int) ([]domain.ProductHit, error) {
	f.gotQ = q
	return f.items, nil
}

type fakeOrders struct {
	order domain.OrderHit
	err   error
}

func (f *fakeOrders) Get(context.Context, string, auth.Principal) (domain.OrderHit, error) {
	return f.order, f.err
}

type fakeCompare struct {
	options     []domain.BuyingOption
	status      domain.DataStatus
	err         error
	gotQ        string
	gotQuantity int
	calls       int
}

func (f *fakeCompare) Compare(_ context.Context, q, _, _, _ string, quantity, _ int) ([]domain.BuyingOption, domain.DataStatus, []domain.Source, error) {
	f.gotQ = q
	f.gotQuantity = quantity
	f.calls++
	if f.err != nil {
		return nil, domain.DataStatusInsufficient, nil, f.err
	}
	sources := make([]domain.Source, 0, len(f.options))
	for _, o := range f.options {
		sources = append(sources, domain.Source{Type: "product", ID: o.ProductID, Label: o.ProductName, IsDemo: o.IsDemo})
	}
	return f.options, f.status, sources, nil
}

type fakeForecaster struct{ result domain.PriceForecast }

func (f *fakeForecaster) Forecast(context.Context, string, string) (domain.PriceForecast, error) {
	if f.result.HorizonDays == 0 {
		return domain.PriceForecast{
			Predicted: 6.99, Low: 6.57, High: 7.41, Samples: 12, HorizonDays: 30,
			TrendPct: 2.4, DataStatus: domain.DataStatusDemo, Basis: "Based on 12 past prices and weather in Bukhara.",
		}, nil
	}
	return f.result, nil
}

type fakeCandidates struct{ candidates []domain.CandidateInfo }

func (f *fakeCandidates) Candidates(context.Context, string) (float64, string, []domain.CandidateInfo, error) {
	return 3000, "1", f.candidates, nil
}
func (f *fakeCandidates) IsDemoProduct(context.Context, string) bool    { return true }
func (f *fakeCandidates) CustomerRegion(context.Context, string) string { return "1" }

type memoryStore struct {
	conversations map[string]domain.ConversationSummary
	messages      map[string][]domain.ChatMessage
}

func newMemoryStore() *memoryStore {
	return &memoryStore{conversations: map[string]domain.ConversationSummary{}, messages: map[string][]domain.ChatMessage{}}
}

func (s *memoryStore) FindOrCreate(_ context.Context, _ auth.Principal, id, _ string) (string, error) {
	if id == "" {
		id = "conv-1"
	}
	if _, ok := s.conversations[id]; !ok {
		s.conversations[id] = domain.ConversationSummary{}
	}
	return id, nil
}
func (s *memoryStore) Load(_ context.Context, _ auth.Principal, id string) ([]domain.ChatMessage, domain.ConversationSummary, error) {
	return s.messages[id], s.conversations[id], nil
}
func (s *memoryStore) Append(_ context.Context, _ auth.Principal, id string, msgs []domain.ChatMessage, summary domain.ConversationSummary, _ domain.Intent, _ domain.DataStatus, _ []domain.ToolCall, _ []domain.Source) error {
	s.messages[id] = append(s.messages[id], msgs...)
	s.conversations[id] = summary
	return nil
}
func (s *memoryStore) List(context.Context, auth.Principal) ([]ConversationInfo, error) {
	return nil, nil
}
func (s *memoryStore) Get(context.Context, auth.Principal, string) (ConversationDetail, error) {
	return ConversationDetail{}, shared.ErrNotFound
}
func (s *memoryStore) Delete(context.Context, auth.Principal, string) error { return nil }

func newTestService(compare *fakeCompare, store ConversationStore) *Service {
	registry := NewRegistry(
		NewMarketStatsTool(&fakeStats{}),
		NewRecommendTool(NewRecommender(&fakeCandidates{})),
		NewSearchTool(&fakeSearch{}),
		NewGetOrderTool(&fakeOrders{}),
		NewCompareOffersTool(compare),
		NewPriceForecastTool(&fakeForecaster{}, &fakeSearch{}),
	)
	return New(&fakeCandidates{}, nil, registry, &fakeStats{}, &fakeSearch{}, &fakeOrders{}, compare, store, config.Config{AIMaxToolRounds: 5})
}

// ---- tool registry validation ----

func TestRegistryValidation(t *testing.T) {
	registry := NewRegistry(NewMarketStatsTool(&fakeStats{}), NewGetOrderTool(&fakeOrders{}), NewCompareOffersTool(&fakeCompare{}))
	tests := []struct {
		name string
		tool domain.ToolName
		args string
		ok   bool
	}{
		{"unknown tool", "drop_table", `{}`, false},
		{"unknown field", domain.ToolMarketStats, `{"bogus":1}`, false},
		{"wrong type", domain.ToolMarketStats, `{"q":123}`, false},
		{"missing order id", domain.ToolGetOrder, `{}`, false},
		{"compare limit too high", domain.ToolCompareOffers, `{"q":"cement","limit":99}`, false},
		{"compare quantity too high", domain.ToolCompareOffers, `{"q":"cement","quantity":99999}`, false},
		{"compare no selector", domain.ToolCompareOffers, `{}`, false},
		{"valid", domain.ToolMarketStats, `{"q":"cement"}`, true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := registry.Validate(tt.tool, json.RawMessage(tt.args))
			if (err == nil) != tt.ok {
				t.Fatalf("Validate() error = %v, want ok=%v", err, tt.ok)
			}
		})
	}
}

func TestGetOrderOwnershipPropagates(t *testing.T) {
	forbidden := shared.WithStatus(shared.ErrForbidden, 403)
	registry := NewRegistry(NewGetOrderTool(&fakeOrders{err: forbidden}))
	_, _, _, err := registry.Execute(context.Background(), testPrincipal, domain.ToolGetOrder, json.RawMessage(`{"order_id":"2"}`))
	if err == nil || !strings.Contains(err.Error(), "forbidden") {
		t.Fatalf("expected ownership error, got %v", err)
	}
}

// ---- groundedness ----

func TestGroundedReply(t *testing.T) {
	call := domain.ToolCall{Result: json.RawMessage(`{"options":[{"price":3050,"supplier":"A"}]}`)}
	tests := []struct {
		name   string
		answer string
		want   bool
	}{
		{"number from tool", "Price 3,050 per unit", true},
		{"fabricated number", "Price 9999 per unit", false},
		{"no numbers", "I need more information", true},
		{"PII", "Your phone is 123", false},
		{"markdown list ordinals are not facts", "1. First thing\n2. Second thing\n3. Third thing", true},
		{"citation markers", "Price 3050 [1] per bag", true},
		{"fabricated price in a list", "1. Price 9999 per bag", false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := groundedReply(tt.answer, []domain.ToolCall{call}); got != tt.want {
				t.Fatalf("groundedReply(%q) = %v, want %v", tt.answer, got, tt.want)
			}
		})
	}
}

// ---- classification ----

func TestClassifyIntentsMultilingual(t *testing.T) {
	tests := []struct {
		message string
		want    domain.Intent
	}{
		{"How much is cement?", domain.IntentMarketPrice},
		{"Narxi qancha?", domain.IntentMarketPrice},
		{"Цена цемента?", domain.IntentMarketPrice},
		{"Who sells cement?", domain.IntentSupplierDiscovery},
		{"Поставщики цемента", domain.IntentSupplierDiscovery},
		{"What should I buy?", domain.IntentBuyingAdvice},
		{"что купить?", domain.IntentBuyingAdvice},
		{"Where is order 42?", domain.IntentOrderStatus},
		{"Can you deliver tomorrow?", domain.IntentOutOfScope},
		{"How do I get approved?", domain.IntentOnboarding},
		{"Hi", domain.IntentSmallTalk},
		{"Soat sotib olmoqchiman", domain.IntentBuyingAdvice}, // "hi" inside "olmoqchiman" is not a greeting
		{"sotib olmoqchiman", domain.IntentBuyingAdvice},
		{"This is fine", domain.IntentFallback}, // nor is "hi" inside "This"
		{"qwertyuiop", domain.IntentFallback},
	}
	for _, tt := range tests {
		if got, _ := classify(tt.message, "general"); got != tt.want {
			t.Errorf("classify(%q) = %s, want %s", tt.message, got, tt.want)
		}
	}
}

// ---- query normalization into tools ----

func TestToolsNormalizeLocalLanguageQueries(t *testing.T) {
	stats, search, compare := &fakeStats{}, &fakeSearch{}, &fakeCompare{}
	registry := NewRegistry(NewMarketStatsTool(stats), NewSearchTool(search), NewCompareOffersTool(compare))
	for _, tc := range []struct {
		name domain.ToolName
		args string
	}{
		{domain.ToolMarketStats, `{"q":"soat"}`},
		{domain.ToolSearchProducts, `{"q":"soat","limit":20}`},
		{domain.ToolCompareOffers, `{"q":"soat","limit":10}`},
	} {
		if _, _, _, err := registry.Execute(context.Background(), testPrincipal, tc.name, json.RawMessage(tc.args)); err != nil {
			t.Fatalf("%s: %v", tc.name, err)
		}
	}
	if stats.gotQ != "watch" || search.gotQ != "watch" || compare.gotQ != "watch" {
		t.Fatalf("queries were not normalized: stats=%q search=%q compare=%q", stats.gotQ, search.gotQ, compare.gotQ)
	}
}

// ---- LLM tool status bookkeeping ----

type scriptedProvider struct {
	responses []CompletionResponse
	round     int
}

func (p *scriptedProvider) SupportsTools() bool { return true }

func (p *scriptedProvider) Complete(context.Context, CompletionRequest) (CompletionResponse, error) {
	if p.round < len(p.responses) {
		resp := p.responses[p.round]
		p.round++
		return resp, nil
	}
	return CompletionResponse{Content: "Here is what I found."}, nil
}

func newLLMService(provider LLMProvider, search *fakeSearch, stats *fakeStats, compare *fakeCompare) *Service {
	registry := NewRegistry(NewMarketStatsTool(stats), NewSearchTool(search), NewCompareOffersTool(compare),
		NewPriceForecastTool(&fakeForecaster{}, search))
	return New(&fakeCandidates{}, provider, registry, stats, search, &fakeOrders{}, compare, newMemoryStore(), config.Config{AIMaxToolRounds: 3})
}

func TestLLMReplyUpgradesStatusFromToolData(t *testing.T) {
	provider := &scriptedProvider{responses: []CompletionResponse{{
		ToolCalls: []domain.ToolCall{{ID: "call_1", Name: domain.ToolSearchProducts, Arguments: json.RawMessage(`{"q":"cement"}`)}},
	}}}
	search := &fakeSearch{items: []domain.ProductHit{{ID: "p1", Name: "Portland Cement", EffectivePrice: 6.8, IsDemo: true}}}
	svc := newLLMService(provider, search, &fakeStats{}, &fakeCompare{})

	reply, err := svc.Chat(context.Background(), testPrincipal, ChatInput{Message: "Do you have cement?", Locale: "en"})
	if err != nil {
		t.Fatal(err)
	}
	if reply.DataStatus != domain.DataStatusDemo {
		t.Fatalf("data_status = %s, want demo (tool returned demo data)", reply.DataStatus)
	}
	if reply.Disclaimer == "" {
		t.Fatal("expected demo disclaimer")
	}
}

func TestFuturePriceQuestionAlwaysGetsForecast(t *testing.T) {
	provider := &scriptedProvider{responses: []CompletionResponse{{Content: "Cement is around 7 per bag right now."}}}
	search := &fakeSearch{items: []domain.ProductHit{{ID: "prd-012", Name: "Portland Cement", Unit: "bag", IsDemo: true}}}
	svc := newLLMService(provider, search, &fakeStats{}, &fakeCompare{})
	reply, err := svc.Chat(context.Background(), testPrincipal, ChatInput{Message: "What will cement cost next month?", Locale: "en"})
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(reply.Reply, "Price outlook") || !strings.Contains(reply.Reply, "6.99") {
		t.Fatalf("forecast missing from reply: %q", reply.Reply)
	}
	if !hasToolCall(reply.ToolCalls, domain.ToolPriceForecast) {
		t.Fatalf("price_forecast tool call missing: %+v", reply.ToolCalls)
	}
}

func TestLLMReplyCapsLiveStatusWhenAnyToolIsDemo(t *testing.T) {
	provider := &scriptedProvider{responses: []CompletionResponse{{
		ToolCalls: []domain.ToolCall{
			{ID: "call_1", Name: domain.ToolSearchProducts, Arguments: json.RawMessage(`{"q":"cement"}`)},
			{ID: "call_2", Name: domain.ToolMarketStats, Arguments: json.RawMessage(`{"q":"cement"}`)},
		},
	}}}
	search := &fakeSearch{items: []domain.ProductHit{{ID: "p1", Name: "Portland Cement", EffectivePrice: 6.8}}}
	stats := &fakeStats{result: domain.MarketStatsResult{
		DataStatus: domain.DataStatusDemo,
		Stats:      domain.MarketStatsFigure{Count: 1},
		Samples:    []domain.MarketSampleHit{{ID: "s1", ProductName: "Portland Cement", Price: 6.8}},
	}}
	svc := newLLMService(provider, search, stats, &fakeCompare{})

	reply, err := svc.Chat(context.Background(), testPrincipal, ChatInput{Message: "How much is cement?", Locale: "en"})
	if err != nil {
		t.Fatal(err)
	}
	if reply.DataStatus != domain.DataStatusDemo {
		t.Fatalf("data_status = %s, want demo (one tool returned sample data)", reply.DataStatus)
	}
}

// ---- answer language follows the question ----

func TestDetectLanguageFromQuestion(t *testing.T) {
	tests := []struct {
		message, fallback, want string
	}{
		{"Soat olmoqchiman", "en", "uz"},
		{"Menga 100 ta soat kerak", "ru", "uz"},
		{"Narxi qancha?", "en", "uz"},
		{"Мне нужен цемент 100 кг", "en", "ru"},
		{"Цена цемента?", "uz", "ru"},
		{"I need 100 kg of cement", "uz", "en"},
		{"What will cement cost next month?", "ru", "en"},
		{"?", "ru", "ru"},
	}
	for _, tt := range tests {
		if got := detectLanguage(tt.message, tt.fallback); got != tt.want {
			t.Errorf("detectLanguage(%q, %q) = %q, want %q", tt.message, tt.fallback, got, tt.want)
		}
	}
}

func TestShoppingReplyUsesQuestionLanguageNotSelectedLocale(t *testing.T) {
	compare := &fakeCompare{
		options: []domain.BuyingOption{{ProductID: "p1", ProductName: "Smart Watch", SupplierName: "A", EffectivePrice: 8.22, LineTotal: 822, Unit: "pcs", IsDemo: true}},
		status:  domain.DataStatusDemo,
	}
	svc := newTestService(compare, newMemoryStore())
	uz, err := svc.Chat(context.Background(), testPrincipal, ChatInput{Message: "Menga 100 ta soat kerak", Mode: "shopping", Locale: "en"})
	if err != nil {
		t.Fatal(err)
	}
	if uz.Locale != "uz" || !strings.Contains(uz.Reply, "variant") || !strings.Contains(uz.Reply, "zaxirada") {
		t.Fatalf("Uzbek question got wrong language: %q (%s)", uz.Reply, uz.Locale)
	}

	ru, err := svc.Chat(context.Background(), testPrincipal, ChatInput{Message: "Мне нужен цемент 100 кг", Mode: "shopping", Locale: "uz"})
	if err != nil {
		t.Fatal(err)
	}
	if ru.Locale != "ru" || !strings.Contains(ru.Reply, "вариант") || !strings.Contains(ru.Reply, "доставка") {
		t.Fatalf("Russian question got wrong language: %q (%s)", ru.Reply, ru.Locale)
	}
}

// ---- buying assistant (D6) ----

func TestBuyingAssistantClarifiesQuantity(t *testing.T) {
	compare := &fakeCompare{}
	svc := newTestService(compare, newMemoryStore())
	reply, err := svc.Chat(context.Background(), testPrincipal, ChatInput{
		Message: "I need cement in Tashkent, what should I buy?", Locale: "en", Mode: "shopping",
	})
	if err != nil {
		t.Fatal(err)
	}
	if reply.Intent != domain.IntentBuyingAdvice {
		t.Fatalf("intent = %s", reply.Intent)
	}
	if !strings.Contains(strings.ToLower(reply.Reply), "how much") {
		t.Fatalf("expected clarification, got %q", reply.Reply)
	}
	if len(reply.ToolCalls) != 0 {
		t.Fatal("no tool call expected before clarification")
	}
}

func TestBuyingAssistantReturnsGroundedShortlistAndActions(t *testing.T) {
	compare := &fakeCompare{
		options: []domain.BuyingOption{
			{ProductID: "18", ProductName: "Cement M400", SupplierName: "Cement Supply LLP", EffectivePrice: 3050, Unit: "bag", LineTotal: 30500, Stock: 500, DeliveryMinDays: 2, DeliveryMaxDays: 3, IsDemo: true, Why: []string{"Lowest total for this quantity"}},
			{ProductID: "19", ProductName: "Cement M500", SupplierName: "Other LLP", EffectivePrice: 3300, Unit: "bag", LineTotal: 33000, Stock: 100, DeliveryMinDays: 1, DeliveryMaxDays: 2, IsDemo: true, Why: []string{"Fast delivery"}},
		},
		status: domain.DataStatusDemo,
	}
	svc := newTestService(compare, newMemoryStore())
	reply, err := svc.Chat(context.Background(), testPrincipal, ChatInput{
		Message: "I need 1000 kg of cement, what should I buy?", Locale: "en", Mode: "shopping",
	})
	if err != nil {
		t.Fatal(err)
	}
	if reply.DataStatus != domain.DataStatusDemo {
		t.Fatalf("status = %s", reply.DataStatus)
	}
	if !strings.Contains(reply.Reply, "3050.00") || !strings.Contains(reply.Reply, "Cement Supply LLP") {
		t.Fatalf("reply not grounded in tool data: %q", reply.Reply)
	}
	if len(reply.Actions) < 3 {
		t.Fatalf("expected actions, got %+v", reply.Actions)
	}
	kinds := map[string]bool{}
	for _, a := range reply.Actions {
		kinds[a.Kind] = true
	}
	if !kinds["link"] || !kinds["add_to_basket"] || !kinds["go_to_orders"] {
		t.Fatalf("missing action kinds: %+v", kinds)
	}
	for _, a := range reply.Actions {
		if strings.Contains(strings.ToLower(a.Kind), "order_create") {
			t.Fatal("AI must never create an order")
		}
	}
	if len(reply.Sources) == 0 {
		t.Fatal("expected sources")
	}
}

func TestBuyingFollowUpUsesSummaryShortlist(t *testing.T) {
	compare := &fakeCompare{
		options: []domain.BuyingOption{{ProductID: "18", ProductName: "Cement", SupplierName: "A", EffectivePrice: 3050, Unit: "bag", LineTotal: 3050, IsDemo: true}},
		status:  domain.DataStatusDemo,
	}
	store := newMemoryStore()
	svc := newTestService(compare, store)
	_, err := svc.Chat(context.Background(), testPrincipal, ChatInput{Message: "I need 1000 kg of cement", Mode: "shopping", Locale: "en"})
	if err != nil {
		t.Fatal(err)
	}
	summary := store.conversations["conv-1"]
	if summary.Quantity != 1000 || summary.Intent != domain.IntentBuyingAdvice {
		t.Fatalf("summary not persisted: %+v", summary)
	}
}

func TestBuyingFollowUpKeepsProductQuery(t *testing.T) {
	compare := &fakeCompare{
		options: []domain.BuyingOption{{ProductID: "18", ProductName: "Cement M400", SupplierName: "A", EffectivePrice: 3050, Unit: "bag", LineTotal: 3050, IsDemo: true}},
		status:  domain.DataStatusDemo,
	}
	store := newMemoryStore()
	svc := newTestService(compare, store)
	if _, err := svc.Chat(context.Background(), testPrincipal, ChatInput{Message: "I need 1000 kg of cement", Mode: "shopping", Locale: "en"}); err != nil {
		t.Fatal(err)
	}
	if _, err := svc.Chat(context.Background(), testPrincipal, ChatInput{Message: "and the cheaper one?", Mode: "shopping", Locale: "en"}); err != nil {
		t.Fatal(err)
	}
	if got := store.conversations["conv-1"].Q; !strings.Contains(got, "cement") {
		t.Fatalf("product query lost on follow-up: %q", got)
	}
}

func TestSlotsFromIgnoresFollowUpFillers(t *testing.T) {
	tests := map[string]string{
		"Which is faster?":        "",
		"And the cheaper one?":    "",
		"Cement olmoqchiman":      "cement",
		"menga 100 ta soat kerak": "soat",
		"100 pcs":                 "",
		"5 bags":                  "",
		"1pcs":                    "",
		"10bags":                  "",
	}
	for message, want := range tests {
		if got := slotsFrom(message).q; got != want {
			t.Errorf("slotsFrom(%q).q = %q, want %q", message, got, want)
		}
	}
}

func TestBuyingFollowUpWhichIsFasterKeepsProductAndRanksBySpeed(t *testing.T) {
	compare := &fakeCompare{
		options: []domain.BuyingOption{
			{ProductID: "cheap", ProductName: "Cheap slow", SupplierName: "A", EffectivePrice: 6.0, LineTotal: 600, DeliveryMinDays: 9, IsDemo: true},
			{ProductID: "fast", ProductName: "Fast one", SupplierName: "B", EffectivePrice: 9.0, LineTotal: 900, DeliveryMinDays: 1, IsDemo: true},
		},
		status: domain.DataStatusDemo,
	}
	store := newMemoryStore()
	svc := newTestService(compare, store)
	first, err := svc.Chat(context.Background(), testPrincipal, ChatInput{Message: "I need 100 kg of cement", Mode: "shopping", Locale: "en"})
	if err != nil || !strings.Contains(first.Reply, "Top ") {
		t.Fatalf("initial answer: %q, %v", first.Reply, err)
	}
	follow, err := svc.Chat(context.Background(), testPrincipal, ChatInput{Message: "Which is faster?", Mode: "shopping", Locale: "en"})
	if err != nil {
		t.Fatal(err)
	}
	if compare.gotQ != "cement" || compare.gotQuantity != 100 {
		t.Fatalf("follow-up tool args: q=%q qty=%d", compare.gotQ, compare.gotQuantity)
	}
	if store.conversations["conv-1"].Q != "cement" {
		t.Fatalf("stored query overwritten: %q", store.conversations["conv-1"].Q)
	}
	if !strings.Contains(follow.Reply, "Fastest of") || !strings.Contains(follow.Reply, "1. Fast one") {
		t.Fatalf("follow-up did not rank by delivery: %q", follow.Reply)
	}
	if follow.DataStatus != domain.DataStatusDemo {
		t.Fatalf("data_status = %s, want demo", follow.DataStatus)
	}
}

func TestBuyingBareQuantityDoesNotBecomeSearchTerm(t *testing.T) {
	compare := &fakeCompare{}
	svc := newTestService(compare, newMemoryStore())
	_, err := svc.Chat(context.Background(), testPrincipal, ChatInput{Message: "What should I buy?", Mode: "shopping", Locale: "en"})
	if err != nil {
		t.Fatal(err)
	}
	reply, err := svc.Chat(context.Background(), testPrincipal, ChatInput{Message: "1,000 kg", Mode: "shopping", Locale: "en"})
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(strings.ToLower(reply.Reply), "what product") {
		t.Fatalf("expected a product clarification, got %q", reply.Reply)
	}
	if compare.calls != 0 {
		t.Fatalf("unexpected compare_offers call with q=%q", compare.gotQ)
	}
}

func TestBuyingUzbekProductAfterBareQuantity(t *testing.T) {
	compare := &fakeCompare{
		options: []domain.BuyingOption{{ProductID: "p1", ProductName: "Portland Cement", SupplierName: "A", EffectivePrice: 6.8, LineTotal: 6800, IsDemo: true}},
		status:  domain.DataStatusDemo,
	}
	svc := newTestService(compare, newMemoryStore())
	for _, msg := range []string{"What should I buy?", "1,000 kg", "Cement olmoqchiman"} {
		if _, err := svc.Chat(context.Background(), testPrincipal, ChatInput{Message: msg, Mode: "shopping", Locale: "uz"}); err != nil {
			t.Fatal(err)
		}
	}
	if compare.gotQ != "cement" || compare.gotQuantity != 1000 {
		t.Fatalf("tool args: q=%q qty=%d", compare.gotQ, compare.gotQuantity)
	}
}

func TestBareUnitSuggestionKeepsProductQuery(t *testing.T) {
	compare := &fakeCompare{
		options: []domain.BuyingOption{{ProductID: "p1", ProductName: "Smart Watch", SupplierName: "A", EffectivePrice: 8.22, LineTotal: 822, Unit: "pcs", IsDemo: true}},
		status:  domain.DataStatusDemo,
	}
	store := newMemoryStore()
	svc := newTestService(compare, store)
	first, err := svc.Chat(context.Background(), testPrincipal, ChatInput{Message: "I need watchs", Mode: "shopping", Locale: "en"})
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(strings.ToLower(first.Reply), "how much") {
		t.Fatalf("expected quantity question, got %q", first.Reply)
	}
	if store.conversations["conv-1"].Q != "watchs" {
		t.Fatalf("product query not stored: %q", store.conversations["conv-1"].Q)
	}
	second, err := svc.Chat(context.Background(), testPrincipal, ChatInput{Message: "100 pcs", Mode: "shopping", Locale: "en"})
	if err != nil {
		t.Fatal(err)
	}
	if store.conversations["conv-1"].Q != "watchs" {
		t.Fatalf("unit word overwrote product query: %q", store.conversations["conv-1"].Q)
	}
	if compare.gotQ != "watch" || compare.gotQuantity != 100 {
		t.Fatalf("tool args: q=%q qty=%d, want watch/100", compare.gotQ, compare.gotQuantity)
	}
	if !strings.Contains(second.Reply, "Smart Watch") || second.DataStatus != domain.DataStatusDemo {
		t.Fatalf("offers were not returned: %q (%s)", second.Reply, second.DataStatus)
	}
}

func TestCompactUnitAnswersKeepProductAndUpdateQuantity(t *testing.T) {
	compare := &fakeCompare{
		options: []domain.BuyingOption{{ProductID: "p1", ProductName: "Smart Watch", SupplierName: "A", EffectivePrice: 8.22, LineTotal: 822, Unit: "pcs", IsDemo: true}},
		status:  domain.DataStatusDemo,
	}
	store := newMemoryStore()
	svc := newTestService(compare, store)
	first, err := svc.Chat(context.Background(), testPrincipal, ChatInput{Message: "Soat olmoqchiman", Mode: "shopping", Locale: "uz"})
	if err != nil || first.Locale != "uz" || !strings.Contains(strings.ToLower(first.Reply), "qancha") {
		t.Fatalf("initial answer should be Uzbek: %q (%s), %v", first.Reply, first.Locale, err)
	}
	for i, msg := range []string{"1pcs", "5 pcs", "10 pcs"} {
		reply, err := svc.Chat(context.Background(), testPrincipal, ChatInput{Message: msg, Mode: "shopping", Locale: "uz"})
		if err != nil {
			t.Fatal(err)
		}
		if got := store.conversations["conv-1"].Q; got != "soat" {
			t.Fatalf("%s: product query overwritten: %q", msg, got)
		}
		if compare.gotQ != "watch" || compare.gotQuantity != []int{1, 5, 10}[i] {
			t.Fatalf("%s: tool args q=%q qty=%d", msg, compare.gotQ, compare.gotQuantity)
		}
		if !strings.Contains(reply.Reply, "Smart Watch") || reply.DataStatus != domain.DataStatusDemo {
			t.Fatalf("%s: offers missing: %q (%s)", msg, reply.Reply, reply.DataStatus)
		}
	}
}

// ---- product units ----

func TestSlotsFromCapturesExplicitUnit(t *testing.T) {
	tests := map[string]string{
		"I need 100 kg of cement":     "kg",
		"I need 100 bags of cement":   "bag",
		"100 кг цемента":              "kg",
		"I need 100 pcs of watches":   "pcs",
		"I need 100 of those watches": "",
		"1pcs":                        "pcs",
		"10bags":                      "bag",
	}
	for message, want := range tests {
		if got := slotsFrom(message).unit; got != want {
			t.Errorf("slotsFrom(%q).unit = %q, want %q", message, got, want)
		}
	}
}

func TestBuyingReplyUsesProductUnit(t *testing.T) {
	compare := &fakeCompare{
		options: []domain.BuyingOption{
			{ProductID: "p1", ProductName: "Portland Cement", SupplierName: "A", EffectivePrice: 6.8, LineTotal: 680, Unit: "bag", DeliveryMinDays: 3, IsDemo: true},
			{ProductID: "p2", ProductName: "Wireless Headphones", SupplierName: "B", EffectivePrice: 28.5, LineTotal: 2850, Unit: "pcs", DeliveryMinDays: 1, IsDemo: true},
		},
		status: domain.DataStatusDemo,
	}
	svc := newTestService(compare, newMemoryStore())
	reply, err := svc.Chat(context.Background(), testPrincipal, ChatInput{Message: "I need 100 of cement", Mode: "shopping", Locale: "en"})
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(reply.Reply, "for about 100 bags:") {
		t.Fatalf("header should use the product's unit: %q", reply.Reply)
	}
	if !strings.Contains(reply.Reply, "per bag") || !strings.Contains(reply.Reply, "per pcs") {
		t.Fatalf("per-unit prices missing: %q", reply.Reply)
	}
}

func TestBuyingReplyNotesMismatchedUnit(t *testing.T) {
	compare := &fakeCompare{
		options: []domain.BuyingOption{{ProductID: "p1", ProductName: "Portland Cement", SupplierName: "A", EffectivePrice: 6.8, LineTotal: 680, Unit: "bag", IsDemo: true}},
		status:  domain.DataStatusDemo,
	}
	svc := newTestService(compare, newMemoryStore())
	reply, err := svc.Chat(context.Background(), testPrincipal, ChatInput{Message: "I need 100 kg of cement", Mode: "shopping", Locale: "en"})
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(reply.Reply, "sold per bag") || !strings.Contains(reply.Reply, "amounts below are in bags") {
		t.Fatalf("expected the product unit to be stated, got %q", reply.Reply)
	}
	if !strings.Contains(reply.Reply, "for about 100 bags:") {
		t.Fatalf("totals must be calculated in the product unit: %q", reply.Reply)
	}
	for _, s := range reply.Suggestions {
		if strings.Contains(s, "kg") {
			t.Fatalf("suggestion still uses kg: %q", s)
		}
	}
}

func TestQuantitySuggestionsUseProductUnit(t *testing.T) {
	svc := &Service{search: &fakeSearch{items: []domain.ProductHit{{ID: "p1", Name: "Wireless Headphones", Unit: "pcs"}}}}
	got := svc.quantitySuggestions("headphones")
	if len(got) != 3 || got[0] != "100 pcs" || got[2] != "5,000 pcs" {
		t.Fatalf("suggestions = %v, want pcs amounts", got)
	}

	svc = &Service{search: &fakeSearch{items: []domain.ProductHit{{ID: "p2", Name: "Cooking Pot", Unit: "set"}}}}
	got = svc.quantitySuggestions("pot")
	if len(got) != 3 || got[0] != "100 sets" {
		t.Fatalf("suggestions = %v, want sets amounts", got)
	}
}

// ---- recommendation determinism ----

func TestRecommendationDeterminism(t *testing.T) {
	now := time.Now().UTC().Format(time.RFC3339)
	candidates := []domain.CandidateInfo{
		{ProductID: "2", Name: "Cement B", SupplierID: "s2", SupplierName: "Supplier B", RegionID: "1", Price: 2900, UpdatedAt: now, UserConfirmed: true, IsDemo: true},
		{ProductID: "3", Name: "Cement C", SupplierID: "s3", SupplierName: "Supplier C", RegionID: "1", Price: 3100, UpdatedAt: now, UserConfirmed: true, IsDemo: true},
	}
	r := NewRecommender(&fakeCandidates{candidates: candidates})
	a, err := r.RecommendSet(context.Background(), "1", testPrincipal, 5)
	if err != nil {
		t.Fatal(err)
	}
	b, err := r.RecommendSet(context.Background(), "1", testPrincipal, 5)
	if err != nil {
		t.Fatal(err)
	}
	rawA, _ := json.Marshal(a)
	rawB, _ := json.Marshal(b)
	if string(rawA) != string(rawB) {
		t.Fatalf("non-deterministic recommendations:\n%s\n%s", rawA, rawB)
	}
	if len(a.Items) != 2 || a.DataStatus != domain.DataStatusDemo {
		t.Fatalf("unexpected set: %s", rawA)
	}
	for _, item := range a.Items {
		if len(item.Reasons) == 0 || item.Score <= 0 {
			t.Fatalf("incomplete recommendation: %+v", item)
		}
	}
}
