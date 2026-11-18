package application

import (
	"context"
	"errors"
	"strings"

	"github.com/bozorpulse/bozorpulse-ai/backend/internal/modules/identity/domain"
	"github.com/bozorpulse/bozorpulse-ai/backend/internal/modules/identity/infrastructure"
	"github.com/bozorpulse/bozorpulse-ai/backend/internal/platform/auth"
	"github.com/bozorpulse/bozorpulse-ai/backend/internal/shared"
)

type Service struct {
	repo     *infrastructure.Repository
	sessions *auth.Sessions
}

func New(repo *infrastructure.Repository, sessions *auth.Sessions) *Service {
	return &Service{repo: repo, sessions: sessions}
}

func (s *Service) Register(ctx context.Context, in domain.RegisterInput) (domain.User, string, error) {
	in.Username = strings.TrimSpace(in.Username)
	in.INN = strings.TrimSpace(in.INN)
	in.BankAccount = strings.TrimSpace(in.BankAccount)
	if in.Username == "" || len(in.Password) < 4 {
		return domain.User{}, "", shared.WithStatus(errors.New("username and password (min 4 chars) are required"), 400)
	}
	if in.Role != auth.RoleSupplier && in.Role != auth.RoleCustomer {
		return domain.User{}, "", shared.WithStatus(errors.New("role must be supplier or customer"), 400)
	}
	if in.INN == "" || in.BankAccount == "" || in.CompanyName == "" {
		return domain.User{}, "", shared.WithStatus(errors.New("inn, bank_account and company_name are required"), 400)
	}
	if _, _, ok, err := s.repo.FindByUsername(ctx, in.Username); err != nil {
		return domain.User{}, "", err
	} else if ok {
		return domain.User{}, "", shared.WithStatus(errors.New("username already taken"), 409)
	}
	hash, err := auth.HashPassword(in.Password)
	if err != nil {
		return domain.User{}, "", err
	}
	u, err := s.repo.Create(ctx, in, hash, auth.StatusWaiting)
	if err != nil {
		return domain.User{}, "", err
	}
	token, err := s.sessions.Create(ctx, u.ID)
	return u, token, err
}

func (s *Service) Login(ctx context.Context, username, password string) (domain.User, string, error) {
	u, hash, ok, err := s.repo.FindByUsername(ctx, strings.TrimSpace(username))
	if err != nil {
		return domain.User{}, "", err
	}
	if !ok || !auth.CheckPassword(hash, password) {
		return domain.User{}, "", shared.WithStatus(errors.New("invalid username or password"), 401)
	}
	token, err := s.sessions.Create(ctx, u.ID)
	return u, token, err
}

func (s *Service) Me(ctx context.Context, id string) (domain.User, error) {
	return s.repo.FindByID(ctx, id)
}

func (s *Service) SetStatus(ctx context.Context, id, status string) (domain.User, error) {
	u, err := s.repo.FindByID(ctx, id)
	if err != nil {
		return domain.User{}, err
	}
	if u.Role == auth.RoleAdmin {
		return domain.User{}, shared.WithStatus(errors.New("admin accounts cannot be moderated"), 400)
	}
	if status != auth.StatusConfirmed && status != auth.StatusRejected {
		return domain.User{}, shared.WithStatus(errors.New("status must be confirmed or rejected"), 400)
	}
	return s.repo.UpdateStatus(ctx, id, status)
}

func (s *Service) List(ctx context.Context, role, status string, limit, offset int) ([]domain.User, int, error) {
	return s.repo.List(ctx, role, status, limit, offset)
}
