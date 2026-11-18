package sanitize

import (
	"strings"
	"testing"
)

func TestValueRemovesForbiddenNestedKeys(t *testing.T) {
	input := map[string]any{
		"name": "Supplier A",
		"owner": map[string]any{
			"inn": "123456789", "phone": "+7999", "bank_account": "KZ1", "username": "user1",
		},
		"items": []any{map[string]any{"password": "secret", "price": 12.5}},
	}
	out := Value(input).(map[string]any)
	owner := out["owner"].(map[string]any)
	for _, key := range []string{"inn", "phone", "bank_account"} {
		if _, ok := owner[key]; ok {
			t.Fatalf("forbidden key %q survived", key)
		}
	}
	if owner["username"] != "user1" {
		t.Fatal("allowed key was removed")
	}
	item := out["items"].([]any)[0].(map[string]any)
	if _, ok := item["password"]; ok {
		t.Fatal("password survived")
	}
	if item["price"] != 12.5 {
		t.Fatal("price was removed")
	}
}

func TestJSONRoundTrip(t *testing.T) {
	type payload struct {
		Name string `json:"name"`
		INN  string `json:"inn"`
	}
	raw, err := JSON(payload{Name: "Supplier", INN: "secret"})
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(raw), "secret") {
		t.Fatalf("PII in JSON: %s", raw)
	}
}

func TestText(t *testing.T) {
	if got := Text("Your phone is 123"); !strings.Contains(got, "[redacted]") {
		t.Fatalf("expected redaction, got %q", got)
	}
}
