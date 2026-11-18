package application

import (
	"context"
	"errors"

	"github.com/bozorpulse/bozorpulse-ai/backend/internal/modules/payments/domain"
	"github.com/bozorpulse/bozorpulse-ai/backend/internal/modules/payments/infrastructure"
	"github.com/bozorpulse/bozorpulse-ai/backend/internal/shared"
)

type Service struct{ repo *infrastructure.Repository }

func New(repo *infrastructure.Repository) *Service { return &Service{repo: repo} }

func (s *Service) Create(ctx context.Context, p domain.Payment) (domain.Payment, error) {
	if p.OrderID == "" || p.Amount <= 0 {
		return p, shared.WithStatus(errors.New("order_id and a positive amount are required"), 400)
	}
	if p.Method == "" {
		p.Method = "bank_transfer"
	}
	return s.repo.Create(ctx, p)
}

func (s *Service) Get(ctx context.Context, id string) (domain.Payment, error) {
	return s.repo.Get(ctx, id)
}
func (s *Service) List(ctx context.Context, limit, offset int) ([]domain.Payment, int, error) {
	return s.repo.List(ctx, limit, offset)
}
