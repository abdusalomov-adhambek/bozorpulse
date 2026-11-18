package infrastructure

import (
	"context"
	"database/sql"
	"encoding/json"
	"sort"

	"github.com/bozorpulse/bozorpulse-ai/backend/internal/modules/analytics/domain"
	"github.com/bozorpulse/bozorpulse-ai/backend/internal/platform/auth"
)

type Repository struct{ db *sql.DB }

func NewRepository(db *sql.DB) *Repository { return &Repository{db: db} }

func (r *Repository) Count(ctx context.Context, query string, args ...interface{}) int {
	var n int
	_ = r.db.QueryRowContext(ctx, query, args...).Scan(&n)
	return n
}

func (r *Repository) Market(ctx context.Context) domain.MarketSummary {
	rows, err := r.db.QueryContext(ctx, `SELECT effective_price FROM products`)
	if err != nil {
		return domain.MarketSummary{}
	}
	defer rows.Close()
	prices := make([]float64, 0)
	for rows.Next() {
		var p float64
		if rows.Scan(&p) == nil {
			prices = append(prices, p)
		}
	}
	s := domain.MarketSummary{Count: len(prices)}
	if len(prices) == 0 {
		return s
	}
	sort.Float64s(prices)
	sum := 0.0
	for _, v := range prices {
		sum += v
	}
	s.Min = prices[0]
	s.Avg = round2(sum / float64(len(prices)))
	if len(prices)%2 == 0 {
		s.Median = round2((prices[len(prices)/2-1] + prices[len(prices)/2]) / 2)
	} else {
		s.Median = round2(prices[len(prices)/2])
	}
	return s
}

func (r *Repository) RecentOrders(ctx context.Context, p auth.Principal) []interface{} {
	query := `SELECT id,status,total_amount,created_at FROM orders`
	var args []interface{}
	if p.Role == auth.RoleCustomer {
		args = append(args, p.CompanyID)
		query += " WHERE customer_id=$1"
	} else if p.Role == auth.RoleSupplier {
		args = append(args, p.CompanyID)
		query += " WHERE supplier_id=$1"
	}
	query += " ORDER BY created_at DESC LIMIT 5"
	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		return []interface{}{}
	}
	defer rows.Close()
	out := make([]interface{}, 0)
	for rows.Next() {
		var id, status, created string
		var total float64
		if err := rows.Scan(&id, &status, &total, &created); err != nil {
			continue
		}
		out = append(out, map[string]interface{}{"id": id, "status": status, "total_amount": total, "created_at": created})
	}
	return out
}

func round2(v float64) float64 { return float64(int(v*100+0.5)) / 100 }

var _ = json.Marshal
