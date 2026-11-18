package infrastructure

import (
	"context"
	"sort"
	"strings"

	"github.com/bozorpulse/bozorpulse-ai/backend/internal/modules/ai/application"
	"github.com/bozorpulse/bozorpulse-ai/backend/internal/modules/ai/domain"
	catalogdomain "github.com/bozorpulse/bozorpulse-ai/backend/internal/modules/catalog/domain"
	companiesdomain "github.com/bozorpulse/bozorpulse-ai/backend/internal/modules/companies/domain"
	procurementdomain "github.com/bozorpulse/bozorpulse-ai/backend/internal/modules/procurement/domain"
	"github.com/bozorpulse/bozorpulse-ai/backend/internal/platform/auth"
)

// Narrow interfaces keep the adapters testable and avoid module cycles.
type catalogReader interface {
	Products(ctx context.Context, q, categoryID, supplierID string, minP, maxP float64, limit, offset int) ([]catalogdomain.Product, int, error)
	Product(ctx context.Context, id string) (catalogdomain.Product, error)
	Pricing(ctx context.Context, productID string) ([]catalogdomain.PricingRule, error)
	ResolvePrice(ctx context.Context, productID, warehouseID string, quantity int) (catalogdomain.ResolvedPrice, error)
}

type companyReader interface {
	Company(ctx context.Context, id string) (companiesdomain.Company, error)
}

type orderReader interface {
	Get(ctx context.Context, id string, p auth.Principal) (procurementdomain.Order, error)
}

type ProductSearchAdapter struct {
	catalog catalogReader
	repo    *Repository
}

var _ application.ProductSearcher = (*ProductSearchAdapter)(nil)

func NewProductSearchAdapter(c catalogReader, repo *Repository) *ProductSearchAdapter {
	return &ProductSearchAdapter{catalog: c, repo: repo}
}

func (a *ProductSearchAdapter) Search(ctx context.Context, q, categoryID string, limit int) ([]domain.ProductHit, error) {
	if limit <= 0 || limit > 20 {
		limit = 20
	}
	items, _, err := a.catalog.Products(ctx, q, categoryID, "", 0, 0, limit, 0)
	if err != nil {
		return nil, err
	}
	out := make([]domain.ProductHit, 0, len(items))
	for _, p := range items {
		out = append(out, domain.ProductHit{
			ID: p.ID, Name: p.Name, SKU: p.SKU, Unit: p.Unit,
			EffectivePrice: p.EffectivePrice, IsDemo: a.repo.IsDemoProduct(ctx, p.ID),
		})
	}
	return out, nil
}

type OrderGetAdapter struct{ orders orderReader }

var _ application.OrderGetter = (*OrderGetAdapter)(nil)

func NewOrderGetAdapter(o orderReader) *OrderGetAdapter { return &OrderGetAdapter{orders: o} }

func (a *OrderGetAdapter) Get(ctx context.Context, id string, p auth.Principal) (domain.OrderHit, error) {
	order, err := a.orders.Get(ctx, id, p)
	if err != nil {
		return domain.OrderHit{}, err
	}
	hit := domain.OrderHit{
		ID: order.ID, Status: order.Status, TotalAmount: order.TotalAmount,
		CreatedAt: order.CreatedAt, Items: make([]domain.OrderItemHit, 0, len(order.Items)),
	}
	for _, item := range order.Items {
		hit.Items = append(hit.Items, domain.OrderItemHit{
			ProductID: item.ProductID, Name: item.Name, Quantity: item.Quantity,
			UnitPrice: item.UnitPrice, LineTotal: item.LineTotal,
		})
	}
	return hit, nil
}

// OfferCompareAdapter ranks real catalog offers for the buying assistant (D6).
type OfferCompareAdapter struct {
	catalog    catalogReader
	companies  companyReader
	repo       *Repository
	forecaster application.PriceForecaster
}

var _ application.OfferComparer = (*OfferCompareAdapter)(nil)

func NewOfferCompareAdapter(c catalogReader, co companyReader, repo *Repository, forecaster application.PriceForecaster) *OfferCompareAdapter {
	return &OfferCompareAdapter{catalog: c, companies: co, repo: repo, forecaster: forecaster}
}

func (a *OfferCompareAdapter) Compare(ctx context.Context, q, categoryID, productID, warehouseID string, quantity int, limit int) ([]domain.BuyingOption, domain.DataStatus, []domain.Source, error) {
	if quantity <= 0 {
		quantity = 1
	}
	if limit <= 0 || limit > 10 {
		limit = 5
	}
	searchQ, searchCategory := q, categoryID
	if productID != "" {
		target, err := a.catalog.Product(ctx, productID)
		if err != nil {
			return nil, domain.DataStatusInsufficient, nil, err
		}
		searchCategory = target.CategoryID
		if searchQ == "" {
			searchQ = target.Name
		}
	}
	products, _, err := a.catalog.Products(ctx, searchQ, searchCategory, "", 0, 0, 20, 0)
	if err != nil {
		return nil, domain.DataStatusInsufficient, nil, err
	}

	options := make([]domain.BuyingOption, 0, len(products))
	anyDemo := false
	for _, p := range products {
		if p.Stock > 0 && p.Stock < quantity {
			continue
		}
		unitPrice, effectivePrice := p.Price, p.EffectivePrice
		minDays, maxDays := p.DeliveryMinDays, p.DeliveryMaxDays
		minOrder, maxOrder := p.MinOrderQty, p.MaxOrderQty
		if warehouseID != "" {
			resolved, err := a.catalog.ResolvePrice(ctx, p.ID, warehouseID, quantity)
			if err == nil {
				unitPrice, effectivePrice = resolved.UnitPrice, resolved.EffectivePrice
				if resolved.DeliveryMinDays > 0 {
					minDays = resolved.DeliveryMinDays
				}
				if resolved.DeliveryMaxDays > 0 {
					maxDays = resolved.DeliveryMaxDays
				}
			}
		} else if rules, err := a.catalog.Pricing(ctx, p.ID); err == nil && len(rules) > 0 {
			best := rules[0]
			for _, rule := range rules[1:] {
				if rule.EffectivePrice < best.EffectivePrice {
					best = rule
				}
			}
			unitPrice, effectivePrice = best.UnitPrice, best.EffectivePrice
			if best.DeliveryMinDays > 0 {
				minDays = best.DeliveryMinDays
			}
			if best.DeliveryMaxDays > 0 {
				maxDays = best.DeliveryMaxDays
			}
			if best.MinOrder > 0 {
				minOrder = best.MinOrder
			}
			if best.MaxOrder > 0 {
				maxOrder = best.MaxOrder
			}
		}
		supplierName := "Supplier"
		if co, err := a.companies.Company(ctx, p.SupplierID); err == nil {
			supplierName = co.Name
		}
		demo := a.repo.IsDemoProduct(ctx, p.ID)
		if demo {
			anyDemo = true
		}
		option := domain.BuyingOption{
			ProductID: p.ID, ProductName: p.Name, SupplierID: p.SupplierID,
			SupplierName: supplierName, Unit: p.Unit, UnitPrice: unitPrice,
			EffectivePrice: effectivePrice, LineTotal: effectivePrice * float64(quantity),
			Quantity: quantity, Stock: p.Stock, MinOrder: minOrder, MaxOrder: maxOrder,
			DeliveryMinDays: minDays, DeliveryMaxDays: maxDays,
			RegionID: p.RegionID, IsDemo: demo, Why: []string{},
		}
		options = append(options, option)
	}
	sort.SliceStable(options, func(i, j int) bool {
		if options[i].LineTotal != options[j].LineTotal {
			return options[i].LineTotal < options[j].LineTotal
		}
		return options[i].DeliveryMinDays < options[j].DeliveryMinDays
	})
	if len(options) > limit {
		options = options[:limit]
	}
	// Attach an explainable price outlook to the shortlisted offers. Weather
	// is cached per region by the provider, so this stays cheap.
	if a.forecaster != nil {
		for i := range options {
			fc, err := a.forecaster.Forecast(ctx, options[i].ProductID, options[i].RegionID)
			if err == nil && fc.Samples > 0 && fc.Predicted > 0 {
				options[i].Forecast = &fc
			}
		}
	}
	for i := range options {
		o := &options[i]
		if i == 0 {
			o.Why = append(o.Why, "Lowest total for this quantity")
		}
		if o.DeliveryMaxDays > 0 && o.DeliveryMaxDays <= 3 {
			o.Why = append(o.Why, "Fast delivery")
		}
		if o.Stock >= o.Quantity && o.Quantity > 1 {
			o.Why = append(o.Why, "Enough stock available")
		}
		if len(o.Why) == 0 {
			o.Why = append(o.Why, "Comparable offer in the catalog")
		}
	}

	status := domain.DataStatusInsufficient
	if len(options) > 0 {
		status = domain.DataStatusDemo
		if !anyDemo {
			status = domain.DataStatusLive
		}
	}
	sources := make([]domain.Source, 0, len(options))
	for _, o := range options {
		sources = append(sources, domain.Source{Type: "product", ID: o.ProductID, Label: o.ProductName + " — " + o.SupplierName, IsDemo: o.IsDemo})
	}
	return options, status, sources, nil
}

// unused import guard for strings (kept for future reason templates)
var _ = strings.TrimSpace
