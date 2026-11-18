package infrastructure

import (
	"context"
	"crypto/rand"
	"database/sql"
	"encoding/hex"
	"errors"
	"time"

	"github.com/bozorpulse/bozorpulse-ai/backend/internal/modules/payments/domain"
	"github.com/bozorpulse/bozorpulse-ai/backend/internal/shared"
)

type Repository struct{ db *sql.DB }

func NewRepository(db *sql.DB) *Repository { return &Repository{db: db} }

func (r *Repository) Create(ctx context.Context, p domain.Payment) (domain.Payment, error) {
	p.ID = "pay-" + newID()
	if p.Status == "" {
		p.Status = "pending"
	}
	p.CreatedAt = time.Now().UTC().Format(time.RFC3339)
	_, err := r.db.ExecContext(ctx, `INSERT INTO payments (id,order_id,amount,method,bank_account,status,created_at)
		VALUES ($1,$2,$3,$4,$5,$6,$7)`, p.ID, p.OrderID, p.Amount, p.Method, p.BankAccount, p.Status, p.CreatedAt)
	return p, err
}

func (r *Repository) Get(ctx context.Context, id string) (domain.Payment, error) {
	var p domain.Payment
	err := r.db.QueryRowContext(ctx, `SELECT id,order_id,amount,method,bank_account,status,created_at FROM payments WHERE id=$1`, id).
		Scan(&p.ID, &p.OrderID, &p.Amount, &p.Method, &p.BankAccount, &p.Status, &p.CreatedAt)
	if errors.Is(err, sql.ErrNoRows) {
		return p, shared.ErrNotFound
	}
	return p, err
}

func (r *Repository) List(ctx context.Context, limit, offset int) ([]domain.Payment, int, error) {
	var total int
	if err := r.db.QueryRowContext(ctx, `SELECT count(*) FROM payments`).Scan(&total); err != nil {
		return nil, 0, err
	}
	rows, err := r.db.QueryContext(ctx, `SELECT id,order_id,amount,method,bank_account,status,created_at
		FROM payments ORDER BY created_at DESC LIMIT $1 OFFSET $2`, limit, offset)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()
	out := make([]domain.Payment, 0)
	for rows.Next() {
		var p domain.Payment
		if err := rows.Scan(&p.ID, &p.OrderID, &p.Amount, &p.Method, &p.BankAccount, &p.Status, &p.CreatedAt); err != nil {
			return nil, 0, err
		}
		out = append(out, p)
	}
	return out, total, rows.Err()
}

func newID() string {
	b := make([]byte, 6)
	_, _ = rand.Read(b)
	return hex.EncodeToString(b)
}
