package llm

import (
	"context"

	"github.com/bozorpulse/bozorpulse-ai/backend/internal/modules/ai/application"
)

// RuleProvider signals that no model call is available; the orchestrator then
// runs the deterministic rule-based assistant.
type RuleProvider struct{}

var _ application.LLMProvider = (*RuleProvider)(nil)

func NewRuleProvider() *RuleProvider { return &RuleProvider{} }

func (p *RuleProvider) SupportsTools() bool { return false }

func (p *RuleProvider) Complete(context.Context, application.CompletionRequest) (application.CompletionResponse, error) {
	return application.CompletionResponse{}, nil
}
