// Package llm implements the LLMProvider port for the Meta Model API using
// its OpenAI-compatible Chat Completions wire format.
package llm

import (
	"encoding/json"

	"github.com/bozorpulse/bozorpulse-ai/backend/internal/modules/ai/application"
	"github.com/bozorpulse/bozorpulse-ai/backend/internal/modules/ai/domain"
)

type openAIRequest struct {
	Model       string          `json:"model"`
	Messages    []openAIMessage `json:"messages"`
	Tools       []openAITool    `json:"tools,omitempty"`
	ToolChoice  string          `json:"tool_choice,omitempty"`
	MaxTokens   int             `json:"max_tokens"`
	Temperature float64         `json:"temperature"`
}

type openAIMessage struct {
	Role       string           `json:"role"`
	Content    string           `json:"content,omitempty"`
	ToolCalls  []openAIToolCall `json:"tool_calls,omitempty"`
	ToolCallID string           `json:"tool_call_id,omitempty"`
}

type openAITool struct {
	Type     string         `json:"type"`
	Function openAIFunction `json:"function"`
}

type openAIFunction struct {
	Name        string          `json:"name"`
	Description string          `json:"description"`
	Parameters  json.RawMessage `json:"parameters"`
}

type openAIToolCall struct {
	ID       string `json:"id"`
	Type     string `json:"type"`
	Function struct {
		Name      string `json:"name"`
		Arguments string `json:"arguments"`
	} `json:"function"`
}

type openAIResponse struct {
	Choices []struct {
		Message struct {
			Content   string           `json:"content"`
			ToolCalls []openAIToolCall `json:"tool_calls"`
		} `json:"message"`
		FinishReason string `json:"finish_reason"`
	} `json:"choices"`
	Error *struct {
		Message string `json:"message"`
	} `json:"error"`
}

func toWireMessages(messages []application.LLMMessage) []openAIMessage {
	out := make([]openAIMessage, 0, len(messages))
	for _, m := range messages {
		wire := openAIMessage{Role: m.Role, Content: m.Content, ToolCallID: m.ToolCallID}
		for _, tc := range m.ToolCalls {
			call := openAIToolCall{ID: tc.ID, Type: "function"}
			call.Function.Name = string(tc.Name)
			call.Function.Arguments = string(tc.Arguments)
			wire.ToolCalls = append(wire.ToolCalls, call)
		}
		out = append(out, wire)
	}
	return out
}

func toWireTools(tools []application.ToolSpec) []openAITool {
	out := make([]openAITool, 0, len(tools))
	for _, t := range tools {
		out = append(out, openAITool{
			Type: "function",
			Function: openAIFunction{
				Name:        string(t.Name),
				Description: t.Description,
				Parameters:  t.InputSchema,
			},
		})
	}
	return out
}

func fromWireToolCalls(calls []openAIToolCall) []domain.ToolCall {
	out := make([]domain.ToolCall, 0, len(calls))
	for _, c := range calls {
		out = append(out, domain.ToolCall{
			ID:        c.ID,
			Name:      domain.ToolName(c.Function.Name),
			Arguments: json.RawMessage(c.Function.Arguments),
		})
	}
	return out
}
