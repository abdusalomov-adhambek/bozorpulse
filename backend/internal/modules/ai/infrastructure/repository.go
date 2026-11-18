package infrastructure

import (
	"context"
	"database/sql"
	"math"
	"sort"
	"time"

	"github.com/bozorpulse/bozorpulse-ai/backend/internal/modules/ai/domain"
	"github.com/bozorpulse/bozorpulse-ai/backend/internal/modules/ai/internal/query"
)

type Repository struct{ db *sql.DB }

func NewRepository(db *sql.DB) *Repository { return &Repository{db: db} }

func (r *Repository) Count(ctx context.Context, q string, args ...interface{}) int {
	var n int
	_ = r.db.QueryRowContext(ctx, q, args...).Scan(&n)
	return n
}

func (r *Repository) Candidates(ctx context.Context, productID string) (targetPrice float64, targetRegion string, out []domain.CandidateInfo, err error) {
	err = r.db.QueryRowContext(ctx, `SELECT COALESCE(effective_price,price),COALESCE(region_id,'') FROM products WHERE id=$1`, productID).
		Scan(&targetPrice, &targetRegion)
	if err != nil {
		return 0, "", nil, err
	}
	rows, err := r.db.QueryContext(ctx, `SELECT p.id,p.name,p.supplier_id,co.name,COALESCE(p.region_id,''),
		p.effective_price,p.updated_at,COALESCE(co.inn,'')<>'',false,u.status='confirmed',
		EXISTS (SELECT 1 FROM price_samples ps WHERE ps.product_id=p.id AND ps.is_demo)
		FROM products p JOIN companies co ON co.id=p.supplier_id JOIN users u ON u.id=co.owner_user_id
		WHERE p.id<>$1 AND p.category_id=(SELECT category_id FROM products WHERE id=$1)
		AND u.status='confirmed' ORDER BY p.effective_price LIMIT 50`, productID)
	if err != nil {
		return 0, "", nil, err
	}
	defer rows.Close()
	for rows.Next() {
		var c domain.CandidateInfo
		if err := rows.Scan(&c.ProductID, &c.Name, &c.SupplierID, &c.SupplierName, &c.RegionID, &c.Price, &c.UpdatedAt,
			&c.HasINN, &c.Verified, &c.UserConfirmed, &c.IsDemo); err != nil {
			return 0, "", nil, err
		}
		out = append(out, c)
	}
	return targetPrice, targetRegion, out, rows.Err()
}

func (r *Repository) MinMaxPrice(ctx context.Context) (float64, float64, int) {
	var min, max float64
	var n int
	_ = r.db.QueryRowContext(ctx, `SELECT COALESCE(min(effective_price),0),COALESCE(max(effective_price),0),count(*) FROM products`).Scan(&min, &max, &n)
	return min, max, n
}

func (r *Repository) CustomerRegion(ctx context.Context, companyID string) string {
	var region string
	if err := r.db.QueryRowContext(ctx, `SELECT COALESCE(region_id,'') FROM warehouses WHERE owner_id=$1 ORDER BY created_at LIMIT 1`, companyID).Scan(&region); err != nil {
		return ""
	}
	return region
}

// IsDemoProduct reports whether any price sample for a product is demo data.
// The products table has no is_demo column, so default to true when unknown.
func (r *Repository) IsDemoProduct(ctx context.Context, productID string) bool {
	var demo bool
	err := r.db.QueryRowContext(ctx, `SELECT EXISTS (SELECT 1 FROM price_samples WHERE product_id=$1 AND is_demo)`, productID).Scan(&demo)
	if err != nil {
		return true
	}
	return demo
}

// MarketStats implements AI-SPECS §1: 90-day window, confirmed suppliers,
// latest sample per product, min/median/avg/count and conservative labeling.
func (r *Repository) MarketStats(ctx context.Context, q, categoryID string) (domain.MarketStatsResult, error) {
	result := domain.MarketStatsResult{
		Samples:     []domain.MarketSampleHit{},
		DataStatus:  domain.DataStatusInsufficient,
		GeneratedAt: time.Now().UTC().Format(time.RFC3339),
		Disclaimer:  "Not enough comparable market data yet.",
	}
	// Match the catalog's substring search: a bare LIKE without % would only
	// match an exact product name, so "cement" could never find "Portland Cement".
	searchQ := query.Normalize(q)
	if searchQ != "" {
		searchQ = "%" + searchQ + "%"
	}
	rows, err := r.db.QueryContext(ctx, `WITH dedup AS (
		SELECT DISTINCT ON (ps.product_id)
		       ps.id, ps.product_id, p.name AS product_name, p.unit, ps.region_id,
		       ps.observed_at, ps.price, ps.is_demo, p.category_id
		FROM price_samples ps
		JOIN products p ON p.id = ps.product_id
		JOIN companies co ON co.id = ps.supplier_id
		JOIN users u ON u.id = co.owner_user_id
		WHERE ps.observed_at::timestamptz >= now() - interval '90 days'
		  AND ps.price > 0 AND u.status = 'confirmed'
		  AND ($1 = '' OR LOWER(p.name) LIKE LOWER($1) OR LOWER(p.sku) LIKE LOWER($1))
		  AND ($2 = '' OR p.category_id = $2)
		ORDER BY ps.product_id, ps.observed_at DESC
	) SELECT id, product_id, product_name, unit, region_id, observed_at, price, is_demo FROM dedup`, searchQ, categoryID)
	if err != nil {
		return result, err
	}
	defer rows.Close()

	prices := make([]float64, 0)
	anyDemo := false
	for rows.Next() {
		var s domain.MarketSampleHit
		var captured string
		if err := rows.Scan(&s.ID, &s.ProductID, &s.ProductName, &s.Unit, &s.RegionID, &captured, &s.Price, &s.IsDemo); err != nil {
			return result, err
		}
		s.CapturedAt = captured
		result.Samples = append(result.Samples, s)
		prices = append(prices, s.Price)
		if s.IsDemo {
			anyDemo = true
		}
	}
	if err := rows.Err(); err != nil {
		return result, err
	}
	result.Stats.Count = len(prices)
	if len(prices) < 5 {
		result.DataStatus = domain.DataStatusInsufficient
		return result, nil
	}
	sort.Float64s(prices)
	min, median, avg := prices[0], prices[len(prices)/2], average(prices)
	if len(prices)%2 == 0 {
		median = (prices[len(prices)/2-1] + prices[len(prices)/2]) / 2
	}
	min, median, avg = roundMoney(min), roundMoney(median), roundMoney(avg)
	result.Stats.Min = &min
	result.Stats.Median = &median
	result.Stats.Avg = &avg
	if anyDemo {
		result.DataStatus = domain.DataStatusDemo
		result.Disclaimer = "Sample market data for demonstration. Not a live market quote."
	} else {
		result.DataStatus = domain.DataStatusLive
	}
	return result, nil
}

func roundMoney(v float64) float64 { return math.Round(v*100) / 100 }

func average(values []float64) float64 {
	if len(values) == 0 {
		return 0
	}
	sum := 0.0
	for _, v := range values {
		sum += v
	}
	return sum / float64(len(values))
}
