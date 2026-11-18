package application

import (
	"context"

	"github.com/bozorpulse/bozorpulse-ai/backend/internal/modules/analytics/domain"
	"github.com/bozorpulse/bozorpulse-ai/backend/internal/modules/analytics/infrastructure"
	"github.com/bozorpulse/bozorpulse-ai/backend/internal/platform/auth"
)

type Service struct{ repo *infrastructure.Repository }

func New(repo *infrastructure.Repository) *Service { return &Service{repo: repo} }

func (s *Service) Dashboard(ctx context.Context, p auth.Principal) domain.Dashboard {
	d := domain.Dashboard{Role: p.Role, Metrics: make([]domain.Metric, 0), RecentOrders: s.repo.RecentOrders(ctx, p), MarketSummary: s.repo.Market(ctx)}
	switch p.Role {
	case auth.RoleAdmin:
		d.Metrics = append(d.Metrics,
			domain.Metric{Label: "Waiting approval", Value: s.repo.Count(ctx, `SELECT count(*) FROM users WHERE status='waiting'`)},
			domain.Metric{Label: "Confirmed users", Value: s.repo.Count(ctx, `SELECT count(*) FROM users WHERE status='confirmed'`)},
			domain.Metric{Label: "Suppliers", Value: s.repo.Count(ctx, `SELECT count(*) FROM users WHERE role='supplier'`)},
			domain.Metric{Label: "Customers", Value: s.repo.Count(ctx, `SELECT count(*) FROM users WHERE role='customer'`)},
			domain.Metric{Label: "Orders", Value: s.repo.Count(ctx, `SELECT count(*) FROM orders`)},
			domain.Metric{Label: "Products", Value: s.repo.Count(ctx, `SELECT count(*) FROM products`)},
		)
	case auth.RoleSupplier:
		d.Metrics = append(d.Metrics,
			domain.Metric{Label: "My products", Value: s.repo.Count(ctx, `SELECT count(*) FROM products WHERE supplier_id=$1`, p.CompanyID)},
			domain.Metric{Label: "My orders", Value: s.repo.Count(ctx, `SELECT count(*) FROM orders WHERE supplier_id=$1`, p.CompanyID)},
			domain.Metric{Label: "My warehouses", Value: s.repo.Count(ctx, `SELECT count(*) FROM warehouses WHERE owner_id=$1`, p.CompanyID)},
		)
	default:
		d.Metrics = append(d.Metrics,
			domain.Metric{Label: "My orders", Value: s.repo.Count(ctx, `SELECT count(*) FROM orders WHERE customer_id=$1`, p.CompanyID)},
			domain.Metric{Label: "Suppliers available", Value: s.repo.Count(ctx, `SELECT count(*) FROM users WHERE role='supplier' AND status='confirmed'`)},
			domain.Metric{Label: "My warehouses", Value: s.repo.Count(ctx, `SELECT count(*) FROM warehouses WHERE owner_id=$1`, p.CompanyID)},
		)
	}
	return d
}
