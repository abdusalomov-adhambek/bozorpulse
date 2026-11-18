// Package sanitize strips private fields before data is sent to an LLM or a log.
package sanitize

import (
	"encoding/json"
	"strings"
)

var forbiddenKeys = map[string]bool{
	"password": true, "password_hash": true, "inn": true,
	"bank_account": true, "phone": true, "token": true,
}

// Value recursively removes forbidden keys from any value.
func Value(v any) any {
	switch typed := v.(type) {
	case map[string]any:
		out := make(map[string]any, len(typed))
		for k, value := range typed {
			if forbiddenKeys[strings.ToLower(k)] {
				continue
			}
			out[k] = Value(value)
		}
		return out
	case map[string]string:
		out := make(map[string]string, len(typed))
		for k, value := range typed {
			if !forbiddenKeys[strings.ToLower(k)] {
				out[k] = value
			}
		}
		return out
	case []any:
		out := make([]any, 0, len(typed))
		for _, item := range typed {
			out = append(out, Value(item))
		}
		return out
	case []string:
		return append([]string(nil), typed...)
	default:
		return v
	}
}

// JSON converts a typed value to JSON with PII removed.
func JSON(v any) ([]byte, error) {
	raw, err := json.Marshal(v)
	if err != nil {
		return nil, err
	}
	var generic any
	if err := json.Unmarshal(raw, &generic); err != nil {
		return nil, err
	}
	return json.Marshal(Value(generic))
}

// Text removes key:value PII patterns from free text (e.g. LLM output).
func Text(s string) string {
	lower := strings.ToLower(s)
	for _, key := range []string{"password", "password_hash", "inn", "bank_account", "phone", "token"} {
		if idx := strings.Index(lower, key); idx >= 0 {
			return strings.TrimSpace(s[:idx]) + " [redacted]"
		}
	}
	return s
}
