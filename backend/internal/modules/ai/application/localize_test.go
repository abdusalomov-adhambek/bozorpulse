package application

import (
	"fmt"
	"strings"
	"testing"
)

// Format strings live in a map, so `go vet` cannot check them. Format every
// template with representative arguments to catch placeholder/arg mismatches.
func TestReplyTemplatesFormatCorrectly(t *testing.T) {
	for _, locale := range []string{"en", "uz", "ru"} {
		T := text(locale)
		cases := map[string]string{
			"noOffers":        fmt.Sprintf(T.noOffers, "cement"),
			"top":             fmt.Sprintf(T.topOptions, 5),
			"header":          fmt.Sprintf(T.header, "Top 5", 100, "bag"),
			"line":            fmt.Sprintf(T.line, 1, "Cement", "Supplier", 6.8, "bag", 680.0, 100, 3, 7),
			"forecast":        fmt.Sprintf(T.forecast, 30, 6.99, "bag", 6.57, 7.41, "basis"),
			"forecastGeneric": fmt.Sprintf(T.forecastGeneric, 30, 6.99, 6.57, 7.41, "basis"),
			"unitNote":        fmt.Sprintf(T.unitNote, "Cement", "bag", "bags", "kg"),
			"orderStatus":     fmt.Sprintf(T.orderStatus, "42", "confirmed", 99.5),
			"supplierFound":   fmt.Sprintf(T.supplierFound, 5),
			"productFound":    fmt.Sprintf(T.productFound, 5),
			"productLine":     fmt.Sprintf(T.productLine, 1, "Cement", 6.8, "bag"),
			"marketStats":     fmt.Sprintf(T.marketStats, 5, 6.1, 7.4, 7.1, 6.8),
		}
		for name, got := range cases {
			if strings.Contains(got, "%!") {
				t.Errorf("%s/%s: bad format: %q", locale, name, got)
			}
		}
	}
}
