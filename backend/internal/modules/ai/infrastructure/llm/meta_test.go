package llm

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/bozorpulse/bozorpulse-ai/backend/internal/modules/ai/application"
	"github.com/bozorpulse/bozorpulse-ai/backend/internal/platform/config"
)

func testConfig(url string, key string) config.Config {
	return config.Config{
		LLMAPIURL: url, LLMAPIKey: key, LLMModel: "muse-spark-1.3",
		LLMMaxTokens: 800, LLMTimeout: 3 * time.Second,
	}
}

func TestMetaProviderRequestAndToolCalls(t *testing.T) {
	var gotPath, gotAuth, gotBody string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		gotAuth = r.Header.Get("Authorization")
		raw, _ := io.ReadAll(r.Body)
		gotBody = string(raw)
		w.Header().Set("Content-Type", "application/json")
		_, _ = io.WriteString(w, `{"choices":[{"finish_reason":"tool_calls","message":{"content":"","tool_calls":[{"id":"call_1","type":"function","function":{"name":"market_stats","arguments":"{\"q\":\"cement\"}"}}]}}]}`)
	}))
	defer srv.Close()

	p := NewMetaProvider(testConfig(srv.URL+"/v1", "test-secret"))
	resp, err := p.Complete(context.Background(), application.CompletionRequest{
		Messages:    []application.LLMMessage{{Role: "user", Content: "How much is cement?"}},
		Tools:       []application.ToolSpec{{Name: "market_stats", InputSchema: json.RawMessage(`{"type":"object"}`)}},
		MaxTokens:   800,
		Temperature: 0,
	})
	if err != nil {
		t.Fatal(err)
	}
	if gotPath != "/v1/chat/completions" {
		t.Fatalf("path = %q", gotPath)
	}
	if gotAuth != "Bearer test-secret" {
		t.Fatalf("auth = %q", gotAuth)
	}
	if !strings.Contains(gotBody, `"model":"muse-spark-1.3"`) || !strings.Contains(gotBody, `"tool_choice":"auto"`) {
		t.Fatalf("unexpected body: %s", gotBody)
	}
	if len(resp.ToolCalls) != 1 || resp.ToolCalls[0].ID != "call_1" || resp.ToolCalls[0].Name != "market_stats" {
		t.Fatalf("tool calls = %+v", resp.ToolCalls)
	}
}

func TestMetaProviderErrorsDoNotLeakKey(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(401)
		_, _ = io.WriteString(w, `{"error":{"message":"invalid api key sk-test-secret-123"}}`)
	}))
	defer srv.Close()
	p := NewMetaProvider(testConfig(srv.URL, "test-secret"))
	_, err := p.Complete(context.Background(), application.CompletionRequest{})
	if err == nil {
		t.Fatal("expected error")
	}
	if strings.Contains(err.Error(), "test-secret") || strings.Contains(err.Error(), "sk-test") {
		t.Fatalf("key leaked: %v", err)
	}
}

func TestMetaProviderMalformedJSON(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = io.WriteString(w, `{not json`)
	}))
	defer srv.Close()
	p := NewMetaProvider(testConfig(srv.URL, "k"))
	if _, err := p.Complete(context.Background(), application.CompletionRequest{}); err == nil {
		t.Fatal("expected decode error")
	}
}
