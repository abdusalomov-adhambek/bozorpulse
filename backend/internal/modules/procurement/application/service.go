package application

import (
	"context"
	"errors"
	"fmt"

	"github.com/bozorpulse/bozorpulse-ai/backend/internal/modules/procurement/domain"
	"github.com/bozorpulse/bozorpulse-ai/backend/internal/modules/procurement/infrastructure"
	"github.com/bozorpulse/bozorpulse-ai/backend/internal/platform/auth"
	"github.com/bozorpulse/bozorpulse-ai/backend/internal/shared"
)

type Service struct{ repo *infrastructure.Repository }

func New(repo *infrastructure.Repository) *Service { return &Service{repo: repo} }

func (s *Service) Create(ctx context.Context, customerID string, in domain.CreateInput) (domain.Order, error) {
	if len(in.Items) == 0 {
		return domain.Order{}, shared.WithStatus(errors.New("at least one item is required"), 400)
	}
	if in.WarehouseID != "" {
		owner, ok, err := s.repo.WarehouseOwner(ctx, in.WarehouseID)
		if err != nil {
			return domain.Order{}, err
		}
		if !ok || owner != customerID {
			return domain.Order{}, shared.WithStatus(errors.New("warehouse does not belong to the customer"), 403)
		}
	}
	o := domain.Order{CustomerID: customerID, SupplierID: in.SupplierID, WarehouseID: in.WarehouseID, Items: make([]domain.OrderItem, 0)}
	for _, item := range in.Items {
		if item.Quantity <= 0 {
			return domain.Order{}, shared.WithStatus(errors.New("quantity must be positive"), 400)
		}
		name, price, supplierID, ok, err := s.repo.ProductInfo(ctx, item.ProductID)
		if err != nil {
			return domain.Order{}, err
		}
		if !ok {
			return domain.Order{}, shared.WithStatus(errors.New("product "+item.ProductID+" not found"), 404)
		}
		if o.SupplierID == "" {
			o.SupplierID = supplierID
		}
		if supplierID != o.SupplierID {
			return domain.Order{}, shared.WithStatus(errors.New("all items must come from one supplier"), 400)
		}
		line := domain.OrderItem{
			ProductID: item.ProductID, Name: name, Quantity: item.Quantity,
			UnitPrice: price, LineTotal: round2(price * float64(item.Quantity)),
		}
		o.Items = append(o.Items, line)
		o.TotalAmount += line.LineTotal
	}
	o.TotalAmount = round2(o.TotalAmount)
	return s.repo.Create(ctx, o)
}

func (s *Service) List(ctx context.Context, p auth.Principal, limit, offset int) ([]domain.Order, int, error) {
	switch p.Role {
	case auth.RoleAdmin:
		return s.repo.List(ctx, "", "", true, limit, offset)
	case auth.RoleSupplier:
		return s.repo.List(ctx, "", p.CompanyID, false, limit, offset)
	default:
		return s.repo.List(ctx, p.CompanyID, "", false, limit, offset)
	}
}

func (s *Service) Get(ctx context.Context, id string, p auth.Principal) (domain.Order, error) {
	o, err := s.repo.Get(ctx, id)
	if err != nil {
		return o, err
	}
	if p.Role == auth.RoleSupplier && o.SupplierID != p.CompanyID {
		return o, shared.ErrForbidden
	}
	if p.Role == auth.RoleCustomer && o.CustomerID != p.CompanyID {
		return o, shared.ErrForbidden
	}
	return o, nil
}

func (s *Service) SetStatus(ctx context.Context, id, status string, p auth.Principal) (domain.Order, error) {
	// Only the supplier who owns the order may change its status. Admins
	// can inspect orders but never approve or advance them.
	if p.Role != auth.RoleSupplier {
		return domain.Order{}, shared.ErrForbidden
	}
	switch status {
	case "new", "confirmed", "processing", "shipped", "delivered", "cancelled":
	default:
		return domain.Order{}, shared.WithStatus(fmt.Errorf("invalid status"), 400)
	}
	order, err := s.repo.Get(ctx, id)
	if err != nil {
		return domain.Order{}, err
	}
	if order.SupplierID != p.CompanyID {
		return domain.Order{}, shared.ErrForbidden
	}
	return s.repo.SetStatus(ctx, id, status)
}

func round2(v float64) float64 { return float64(int(v*100+0.5)) / 100 }
