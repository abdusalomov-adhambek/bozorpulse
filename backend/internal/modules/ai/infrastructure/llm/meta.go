package llm

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"

	"github.com/bozorpulse/bozorpulse-ai/backend/internal/modules/ai/application"
	"github.com/bozorpulse/bozorpulse-ai/backend/internal/platform/config"
)

const maxResponseBytes = 1 << 20 // 1 MB

// MetaProvider calls the Meta Model API (https://api.meta.ai) through its
// OpenAI-compatible /chat/completions endpoint. No third-party SDK needed.
type MetaProvider struct {
	apiURL    string
	apiKey    string
	model     string
	maxTokens int
	client    *http.Client
}

var _ application.LLMProvider = (*MetaProvider)(nil)

func NewMetaProvider(cfg config.Config) *MetaProvider {
	return &MetaProvider{
		apiURL:    strings.TrimRight(cfg.LLMAPIURL, "/"),
		apiKey:    cfg.LLMAPIKey,
		model:     cfg.LLMModel,
		maxTokens: cfg.LLMMaxTokens,
		client:    &http.Client{Timeout: cfg.LLMTimeout},
	}
}

func (p *MetaProvider) SupportsTools() bool { return true }

func (p *MetaProvider) Complete(ctx context.Context, req application.CompletionRequest) (application.CompletionResponse, error) {
	model := req.Model
	if model == "" {
		model = p.model
	}
	maxTokens := req.MaxTokens
	if maxTokens <= 0 {
		maxTokens = p.maxTokens
	}
	body := openAIRequest{
		Model:       model,
		Messages:    toWireMessages(req.Messages),
		Tools:       toWireTools(req.Tools),
		MaxTokens:   maxTokens,
		Temperature: req.Temperature,
	}
	if len(body.Tools) > 0 {
		body.ToolChoice = "auto"
	}
	encoded, err := json.Marshal(body)
	if err != nil {
		return application.CompletionResponse{}, fmt.Errorf("encode llm request: %w", err)
	}

	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, p.apiURL+"/chat/completions", bytes.NewReader(encoded))
	if err != nil {
		return application.CompletionResponse{}, fmt.Errorf("build llm request: %w", err)
	}
	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Authorization", "Bearer "+p.apiKey)

	resp, err := p.client.Do(httpReq)
	if err != nil {
		return application.CompletionResponse{}, fmt.Errorf("llm request failed: %w", err)
	}
	defer resp.Body.Close()

	raw, err := io.ReadAll(io.LimitReader(resp.Body, maxResponseBytes+1))
	if err != nil {
		return application.CompletionResponse{}, fmt.Errorf("read llm response: %w", err)
	}
	if len(raw) > maxResponseBytes {
		return application.CompletionResponse{}, errors.New("llm response too large")
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return application.CompletionResponse{}, fmt.Errorf("llm api status %d", resp.StatusCode)
	}

	var parsed openAIResponse
	if err := json.Unmarshal(raw, &parsed); err != nil {
		return application.CompletionResponse{}, fmt.Errorf("decode llm response: %w", err)
	}
	if parsed.Error != nil {
		return application.CompletionResponse{}, fmt.Errorf("llm api error: %s", redact(parsed.Error.Message))
	}
	if len(parsed.Choices) == 0 {
		return application.CompletionResponse{}, errors.New("llm returned no choices")
	}
	choice := parsed.Choices[0]
	return application.CompletionResponse{
		Content:      choice.Message.Content,
		ToolCalls:    fromWireToolCalls(choice.Message.ToolCalls),
		FinishReason: choice.FinishReason,
	}, nil
}

// redact removes bearer tokens and key-like values from upstream error text so
// secrets never reach logs or replies.
func redact(s string) string {
	lower := strings.ToLower(s)
	for _, marker := range []string{"bearer ", "api_key", "api key", "apikey", "model_api_key", "sk-", "token"} {
		if idx := strings.Index(lower, marker); idx >= 0 {
			return strings.TrimSpace(s[:idx]) + " [redacted]"
		}
	}
	if len(s) > 300 {
		return s[:300] + "…"
	}
	return s
}
