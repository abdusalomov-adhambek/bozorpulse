package infrastructure

import (
	"context"

	catalogDomain "github.com/bozorpulse/bozorpulse-ai/backend/internal/modules/catalog/domain"
	"github.com/bozorpulse/bozorpulse-ai/backend/internal/modules/planning/application"
)

// catalogReader matches the catalog application service methods used here.
type catalogReader interface {
	Product(ctx context.Context, id string) (catalogDomain.Product, error)
	Pricing(ctx context.Context, productID string) ([]catalogDomain.PricingRule, error)
}

// ProductAdapter bridges the catalog application API into the planning port.
type ProductAdapter struct{ catalog catalogReader }

func NewProductAdapter(c catalogReader) *ProductAdapter { return &ProductAdapter{catalog: c} }

func (a *ProductAdapter) Product(ctx context.Context, productID string) (application.ProductInfo, error) {
	p, err := a.catalog.Product(ctx, productID)
	if err != nil {
		return application.ProductInfo{}, err
	}
	base := p.EffectivePrice
	if base <= 0 {
		base = p.Price
	}
	return application.ProductInfo{
		ID: p.ID, Name: p.Name, Unit: p.Unit,
		RegionID: p.RegionID, BasePrice: base,
	}, nil
}

// BestTierPrice returns the cheapest effective price among the quantity tiers
// that apply without a specific warehouse. Returns 0 when no tier matches so
// the application falls back to the product's base price.
func (a *ProductAdapter) BestTierPrice(ctx context.Context, productID string, quantity int) (float64, error) {
	rules, err := a.catalog.Pricing(ctx, productID)
	if err != nil {
		return 0, err
	}
	best := 0.0
	for _, r := range rules {
		if r.WarehouseID != "" {
			continue // planning has no warehouse context
		}
		if r.MinQty > 0 && quantity < r.MinQty {
			continue
		}
		if r.MaxQty > 0 && quantity > r.MaxQty {
			continue
		}
		price := r.EffectivePrice
		if price <= 0 {
			price = r.UnitPrice
		}
		if price <= 0 {
			continue
		}
		if best == 0 || price < best {
			best = price
		}
	}
	return best, nil
}
