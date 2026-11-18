package application

import (
	"context"
	"errors"

	"github.com/bozorpulse/bozorpulse-ai/backend/internal/modules/catalog/domain"
	"github.com/bozorpulse/bozorpulse-ai/backend/internal/modules/catalog/infrastructure"
	"github.com/bozorpulse/bozorpulse-ai/backend/internal/shared"
)

type Service struct{ repo *infrastructure.Repository }

func New(repo *infrastructure.Repository) *Service { return &Service{repo: repo} }

func (s *Service) Categories(ctx context.Context) ([]domain.Category, error) {
	return s.repo.ListCategories(ctx)
}
func (s *Service) Regions(ctx context.Context) ([]domain.Region, error) {
	return s.repo.ListRegions(ctx)
}

func (s *Service) CreateCategory(ctx context.Context, name, parent string) (domain.Category, error) {
	if name == "" {
		return domain.Category{}, shared.WithStatus(errors.New("name is required"), 400)
	}
	return s.repo.CreateCategory(ctx, name, parent)
}
func (s *Service) UpdateCategory(ctx context.Context, id, name, parent string) (domain.Category, error) {
	return s.repo.UpdateCategory(ctx, id, name, parent)
}
func (s *Service) DeleteCategory(ctx context.Context, id string) error {
	return s.repo.DeleteCategory(ctx, id)
}

func (s *Service) CreateRegion(ctx context.Context, name string) (domain.Region, error) {
	if name == "" {
		return domain.Region{}, shared.WithStatus(errors.New("name is required"), 400)
	}
	return s.repo.CreateRegion(ctx, name)
}

func (s *Service) Products(ctx context.Context, q, categoryID, supplierID string, minP, maxP float64, limit, offset int) ([]domain.Product, int, error) {
	return s.repo.ListProducts(ctx, q, categoryID, supplierID, minP, maxP, limit, offset)
}

func (s *Service) Product(ctx context.Context, id string) (domain.Product, error) {
	return s.repo.GetProduct(ctx, id)
}

func (s *Service) PriceHistory(ctx context.Context, id string, limit int) (domain.PriceHistory, error) {
	return s.repo.PriceHistory(ctx, id, limit)
}

func (s *Service) CreateProduct(ctx context.Context, p domain.Product) (domain.Product, error) {
	if p.Name == "" || p.Price <= 0 {
		return p, shared.WithStatus(errors.New("name and a positive price are required"), 400)
	}
	if p.DiscountPercent < 0 || p.DiscountPercent > 90 {
		return p, shared.WithStatus(errors.New("discount must be between 0 and 90"), 400)
	}
	if p.Unit == "" {
		p.Unit = "pcs"
	}
	return s.repo.CreateProduct(ctx, p)
}

func (s *Service) UpdateProduct(ctx context.Context, p domain.Product) (domain.Product, error) {
	return s.repo.UpdateProduct(ctx, p)
}

func (s *Service) DeleteProduct(ctx context.Context, id string) error {
	return s.repo.DeleteProduct(ctx, id)
}

func (s *Service) SetDiscount(ctx context.Context, id string, discount float64) (domain.Product, error) {
	return s.repo.SetDiscount(ctx, id, discount)
}

func (s *Service) Pricing(ctx context.Context, productID string) ([]domain.PricingRule, error) {
	return s.repo.ListPricing(ctx, productID)
}

func (s *Service) CreatePricing(ctx context.Context, rule domain.PricingRule) (domain.PricingRule, error) {
	if rule.ProductID == "" || rule.UnitPrice <= 0 {
		return rule, shared.WithStatus(errors.New("product_id and a positive price are required"), 400)
	}
	if rule.MinQty <= 0 || (rule.MaxQty > 0 && rule.MaxQty < rule.MinQty) {
		return rule, shared.WithStatus(errors.New("invalid quantity band"), 400)
	}
	if rule.DiscountPercent < 0 || rule.DiscountPercent > 90 {
		return rule, shared.WithStatus(errors.New("discount must be between 0 and 90"), 400)
	}
	return s.repo.CreatePricing(ctx, rule)
}

func (s *Service) DeletePricing(ctx context.Context, id string) error {
	return s.repo.DeletePricing(ctx, id)
}

func (s *Service) DeliveryZones(ctx context.Context, supplierID string) ([]domain.DeliveryZone, error) {
	return s.repo.ListDeliveryZones(ctx, supplierID)
}

func (s *Service) CreateDeliveryZone(ctx context.Context, zone domain.DeliveryZone) (domain.DeliveryZone, error) {
	if zone.SupplierID == "" || zone.RegionID == "" {
		return zone, shared.WithStatus(errors.New("supplier_id and region_id are required"), 400)
	}
	if zone.Price < 0 || zone.MinDays < 0 || zone.MaxDays < 0 {
		return zone, shared.WithStatus(errors.New("price and days cannot be negative"), 400)
	}
	return s.repo.CreateDeliveryZone(ctx, zone)
}

func (s *Service) UpdateDeliveryZone(ctx context.Context, zone domain.DeliveryZone) (domain.DeliveryZone, error) {
	return s.repo.UpdateDeliveryZone(ctx, zone)
}

func (s *Service) DeleteDeliveryZone(ctx context.Context, id, supplierID string) error {
	return s.repo.DeleteDeliveryZone(ctx, id, supplierID)
}

func (s *Service) ResolvePrice(ctx context.Context, productID, warehouseID string, quantity int) (domain.ResolvedPrice, error) {
	if quantity <= 0 {
		quantity = 1
	}
	result, err := s.repo.ResolvePrice(ctx, productID, warehouseID, quantity)
	if err != nil {
		return result, err
	}
	if quantity < result.MinOrder || (result.MaxOrder > 0 && quantity > result.MaxOrder) {
		return result, shared.WithStatus(errors.New("quantity is outside the allowed order range"), 400)
	}
	return result, nil
}
