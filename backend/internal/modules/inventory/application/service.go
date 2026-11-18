package application

import (
	"context"
	"errors"

	"github.com/bozorpulse/bozorpulse-ai/backend/internal/modules/inventory/domain"
	"github.com/bozorpulse/bozorpulse-ai/backend/internal/modules/inventory/infrastructure"
	"github.com/bozorpulse/bozorpulse-ai/backend/internal/platform/auth"
	"github.com/bozorpulse/bozorpulse-ai/backend/internal/shared"
)

type Service struct{ repo *infrastructure.Repository }

func New(repo *infrastructure.Repository) *Service { return &Service{repo: repo} }

func (s *Service) Stock(ctx context.Context, p auth.Principal, warehouseID string) ([]domain.StockItem, int, error) {
	return s.repo.Stock(ctx, warehouseID, p.CompanyID, p.Role == auth.RoleAdmin)
}

func (s *Service) Movements(ctx context.Context, p auth.Principal, warehouseID string) ([]domain.Movement, int, error) {
	return s.repo.Movements(ctx, warehouseID, p.CompanyID, p.Role == auth.RoleAdmin)
}

func (s *Service) checkOwnership(ctx context.Context, p auth.Principal, warehouseID string) error {
	if p.Role == auth.RoleAdmin {
		return nil
	}
	owner, ok, err := s.repo.Owner(ctx, warehouseID)
	if err != nil {
		return err
	}
	if !ok || owner != p.CompanyID {
		return shared.WithStatus(errors.New("warehouse does not belong to your company"), 403)
	}
	return nil
}

func (s *Service) checkProduct(ctx context.Context, productID string) error {
	ok, err := s.repo.ProductExists(ctx, productID)
	if err != nil {
		return err
	}
	if !ok {
		return shared.WithStatus(errors.New("product not found"), 404)
	}
	return nil
}

func (s *Service) Supply(ctx context.Context, p auth.Principal, in domain.MovementInput) (domain.Movement, error) {
	if in.Quantity <= 0 || in.WarehouseID == "" || in.ProductID == "" {
		return domain.Movement{}, shared.WithStatus(errors.New("warehouse_id, product_id and positive quantity are required"), 400)
	}
	if err := s.checkOwnership(ctx, p, in.WarehouseID); err != nil {
		return domain.Movement{}, err
	}
	if err := s.checkProduct(ctx, in.ProductID); err != nil {
		return domain.Movement{}, err
	}
	m := domain.Movement{WarehouseID: in.WarehouseID, ProductID: in.ProductID, Quantity: in.Quantity, Type: "supply", Note: in.Note}
	return s.repo.ApplyMovement(ctx, m, in.Quantity)
}

func (s *Service) Transfer(ctx context.Context, p auth.Principal, in domain.MovementInput) ([]domain.Movement, error) {
	if in.Quantity <= 0 || in.WarehouseID == "" || in.ToWarehouseID == "" || in.ProductID == "" {
		return nil, shared.WithStatus(errors.New("from/to warehouses, product and positive quantity are required"), 400)
	}
	if in.WarehouseID == in.ToWarehouseID {
		return nil, shared.WithStatus(errors.New("source and destination warehouses must differ"), 400)
	}
	if err := s.checkOwnership(ctx, p, in.WarehouseID); err != nil {
		return nil, err
	}
	if err := s.checkOwnership(ctx, p, in.ToWarehouseID); err != nil {
		return nil, err
	}
	if err := s.checkProduct(ctx, in.ProductID); err != nil {
		return nil, err
	}
	out, err := s.repo.ApplyMovement(ctx, domain.Movement{WarehouseID: in.WarehouseID, ProductID: in.ProductID,
		Quantity: in.Quantity, Type: "transfer_out", Note: in.Note}, -in.Quantity)
	if err != nil {
		return nil, err
	}
	inb, err := s.repo.ApplyMovement(ctx, domain.Movement{WarehouseID: in.ToWarehouseID, ProductID: in.ProductID,
		Quantity: in.Quantity, Type: "transfer_in", Note: in.Note}, in.Quantity)
	if err != nil {
		return nil, err
	}
	return []domain.Movement{out, inb}, nil
}

func (s *Service) WriteOff(ctx context.Context, p auth.Principal, in domain.MovementInput) (domain.Movement, error) {
	if in.Quantity <= 0 || in.WarehouseID == "" || in.ProductID == "" {
		return domain.Movement{}, shared.WithStatus(errors.New("warehouse_id, product_id and positive quantity are required"), 400)
	}
	if err := s.checkOwnership(ctx, p, in.WarehouseID); err != nil {
		return domain.Movement{}, err
	}
	if err := s.checkProduct(ctx, in.ProductID); err != nil {
		return domain.Movement{}, err
	}
	note := in.Reason
	if note == "" {
		note = in.Note
	}
	m := domain.Movement{WarehouseID: in.WarehouseID, ProductID: in.ProductID, Quantity: in.Quantity, Type: "write_off", Note: note}
	return s.repo.ApplyMovement(ctx, m, -in.Quantity)
}

func (s *Service) Return(ctx context.Context, p auth.Principal, in domain.MovementInput) (domain.Movement, error) {
	if in.Quantity <= 0 || in.WarehouseID == "" || in.ProductID == "" {
		return domain.Movement{}, shared.WithStatus(errors.New("warehouse_id, product_id and positive quantity are required"), 400)
	}
	if err := s.checkOwnership(ctx, p, in.WarehouseID); err != nil {
		return domain.Movement{}, err
	}
	if err := s.checkProduct(ctx, in.ProductID); err != nil {
		return domain.Movement{}, err
	}
	m := domain.Movement{WarehouseID: in.WarehouseID, ProductID: in.ProductID, Quantity: in.Quantity, Type: "return", Note: in.Note}
	return s.repo.ApplyMovement(ctx, m, in.Quantity)
}
