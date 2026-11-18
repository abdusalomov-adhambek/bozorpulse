package application

import (
	"context"
	"errors"

	"github.com/bozorpulse/bozorpulse-ai/backend/internal/modules/companies/domain"
	"github.com/bozorpulse/bozorpulse-ai/backend/internal/modules/companies/infrastructure"
	"github.com/bozorpulse/bozorpulse-ai/backend/internal/shared"
)

type Service struct{ repo *infrastructure.Repository }

func New(repo *infrastructure.Repository) *Service { return &Service{repo: repo} }

func (s *Service) Suppliers(ctx context.Context, limit, offset int) ([]domain.Company, int, error) {
	return s.repo.List(ctx, "supplier", "confirmed", limit, offset)
}
func (s *Service) Customers(ctx context.Context, status string, limit, offset int) ([]domain.Company, int, error) {
	return s.repo.List(ctx, "customer", status, limit, offset)
}
func (s *Service) Company(ctx context.Context, id string) (domain.Company, error) {
	return s.repo.Get(ctx, id)
}

func (s *Service) UpdateCompany(ctx context.Context, id string, in domain.Company) (domain.Company, error) {
	if in.Name == "" {
		return domain.Company{}, shared.WithStatus(errors.New("name is required"), 400)
	}
	return s.repo.Update(ctx, id, in.Name, in.INN, in.BankAccount, in.RegionID)
}

func (s *Service) Warehouses(ctx context.Context, ownerID string, admin bool) ([]domain.Warehouse, error) {
	if admin {
		return s.repo.ListWarehouses(ctx, "")
	}
	return s.repo.ListWarehouses(ctx, ownerID)
}

func (s *Service) Warehouse(ctx context.Context, id, ownerID string, admin bool) (domain.Warehouse, error) {
	wh, err := s.repo.GetWarehouse(ctx, id)
	if err != nil {
		return wh, err
	}
	if !admin && wh.OwnerID != ownerID {
		return wh, shared.ErrForbidden
	}
	return wh, nil
}

func (s *Service) CreateWarehouse(ctx context.Context, wh domain.Warehouse) (domain.Warehouse, error) {
	if wh.Name == "" || wh.City == "" {
		return wh, shared.WithStatus(errors.New("name and city are required"), 400)
	}
	return s.repo.CreateWarehouse(ctx, wh)
}

func (s *Service) UpdateWarehouse(ctx context.Context, wh domain.Warehouse) (domain.Warehouse, error) {
	if wh.Name == "" {
		return wh, shared.WithStatus(errors.New("name is required"), 400)
	}
	return s.repo.UpdateWarehouse(ctx, wh)
}

func (s *Service) DeleteWarehouse(ctx context.Context, id string) error {
	return s.repo.DeleteWarehouse(ctx, id)
}
