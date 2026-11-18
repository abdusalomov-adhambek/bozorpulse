package application

import (
	"context"
	"encoding/json"

	"github.com/bozorpulse/bozorpulse-ai/backend/internal/modules/ai/domain"
	"github.com/bozorpulse/bozorpulse-ai/backend/internal/platform/auth"
)

// LLMProvider abstracts the model call. RuleProvider needs no network;
// MetaProvider speaks the OpenAI-compatible Chat Completions wire format.
type LLMProvider interface {
	Complete(ctx context.Context, req CompletionRequest) (CompletionResponse, error)
	SupportsTools() bool
}

// LLMMessage carries the tool-call continuation state required by Chat
// Completions: assistant tool_calls plus tool results with matching IDs.
type LLMMessage struct {
	Role       string
	Content    string
	ToolCalls  []domain.ToolCall
	ToolCallID string
}

type CompletionRequest struct {
	Model       string
	Messages    []LLMMessage
	Tools       []ToolSpec
	MaxTokens   int
	Temperature float64
}

type CompletionResponse struct {
	Content      string
	ToolCalls    []domain.ToolCall
	FinishReason string
}

type ToolSpec struct {
	Name        domain.ToolName
	Description string
	InputSchema json.RawMessage
}

type ProductSearcher interface {
	Search(ctx context.Context, q, categoryID string, limit int) ([]domain.ProductHit, error)
}

type OrderGetter interface {
	Get(ctx context.Context, id string, p auth.Principal) (domain.OrderHit, error)
}

type MarketStatsReader interface {
	MarketStats(ctx context.Context, q, categoryID string) (domain.MarketStatsResult, error)
}

// CandidateSource supplies same-category products for recommendations.
type CandidateSource interface {
	Candidates(ctx context.Context, productID string) (targetPrice float64, targetRegion string, out []domain.CandidateInfo, err error)
	IsDemoProduct(ctx context.Context, productID string) bool
	CustomerRegion(ctx context.Context, companyID string) string
}

type SupplierRecommender interface {
	RecommendSet(ctx context.Context, productID string, p auth.Principal, limit int) (domain.RecommendationSet, error)
}

// PriceForecaster estimates a future price from past samples and weather.
type PriceForecaster interface {
	Forecast(ctx context.Context, productID, regionID string) (domain.PriceForecast, error)
}

// OfferComparer ranks concrete offers the customer can actually buy (D6).
type OfferComparer interface {
	Compare(ctx context.Context, q, categoryID, productID, warehouseID string, quantity int, limit int) ([]domain.BuyingOption, domain.DataStatus, []domain.Source, error)
}

// ConversationStore persists chat history and the compact summary with
// company-level access control.
type ConversationStore interface {
	FindOrCreate(ctx context.Context, p auth.Principal, conversationID, locale string) (string, error)
	Load(ctx context.Context, p auth.Principal, conversationID string) ([]domain.ChatMessage, domain.ConversationSummary, error)
	Append(ctx context.Context, p auth.Principal, conversationID string, messages []domain.ChatMessage, summary domain.ConversationSummary, intent domain.Intent, status domain.DataStatus, toolCalls []domain.ToolCall, sources []domain.Source) error
	List(ctx context.Context, p auth.Principal) ([]ConversationInfo, error)
	Get(ctx context.Context, p auth.Principal, conversationID string) (ConversationDetail, error)
	Delete(ctx context.Context, p auth.Principal, conversationID string) error
}

type ConversationInfo struct {
	ID         string `json:"id"`
	Locale     string `json:"locale"`
	Intent     string `json:"intent"`
	DataStatus string `json:"data_status"`
	Preview    string `json:"preview,omitempty"`
	CreatedAt  string `json:"created_at"`
	UpdatedAt  string `json:"updated_at"`
}

type ConversationDetail struct {
	ConversationInfo
	Messages []domain.ChatMessage `json:"messages"`
}
