package query

import "testing"

func TestNormalize(t *testing.T) {
	tests := []struct {
		in, want string
	}{
		{"soat", "watch"},                   // the failing last chat
		{"Soat sotib olmoqchiman", "watch"}, // whole message also works
		{"menga ta soat kerak", "watch"},
		{"cement olmoqchiman", "cement"}, // English product + Uzbek verb
		{"wireless headphones", "headphone"},
		{"qo'l soati", "watch"},
		{"смарт-часы", "watch"},
		{"sement", "cement"},
		{"Цена цемента", "cement"},
		{"guruch", "rice"},
		{"naushniklar", "headphone"},
		{"cement", "cement"},       // already English
		{"Smart Watches", "watch"}, // canonical form still matches catalog
		{"watchs", "watch"},        // common misspelling still resolves
		{"CN-PC-050", "cn-pc-050"}, // SKU search still works
		{"", ""},
	}
	for _, tt := range tests {
		if got := Normalize(tt.in); got != tt.want {
			t.Errorf("Normalize(%q) = %q, want %q", tt.in, got, tt.want)
		}
	}
}

func TestNormalizeIdempotent(t *testing.T) {
	for _, in := range []string{"soat", "sement", "cement", "qo'l soati"} {
		once := Normalize(in)
		if twice := Normalize(once); twice != once {
			t.Errorf("Normalize(%q) = %q, then %q", in, once, twice)
		}
	}
}
