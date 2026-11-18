package infrastructure

import (
	"context"
	"crypto/rand"
	"database/sql"
	"encoding/hex"
	"errors"
	"time"

	"github.com/bozorpulse/bozorpulse-ai/backend/internal/modules/identity/domain"
	"github.com/bozorpulse/bozorpulse-ai/backend/internal/shared"
)

type Repository struct{ db *sql.DB }

func NewRepository(db *sql.DB) *Repository { return &Repository{db: db} }

const userCols = `id,username,role,status,COALESCE(inn,''),COALESCE(bank_account,''),COALESCE(company_name,''),COALESCE(region_id,''),COALESCE(phone,''),created_at`

func scanUser(row interface{ Scan(...interface{}) error }) (domain.User, error) {
	var u domain.User
	err := row.Scan(&u.ID, &u.Username, &u.Role, &u.Status, &u.INN, &u.BankAccount,
		&u.CompanyName, &u.RegionID, &u.Phone, &u.CreatedAt)
	return u, err
}

func (r *Repository) Create(ctx context.Context, in domain.RegisterInput, hash, status string) (domain.User, error) {
	u := domain.User{
		ID: "usr-" + newID(), Username: in.Username, Role: in.Role, Status: status,
		INN: in.INN, BankAccount: in.BankAccount, CompanyName: in.CompanyName,
		RegionID: in.RegionID, Phone: in.Phone, CreatedAt: time.Now().UTC().Format(time.RFC3339),
	}
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return u, err
	}
	defer tx.Rollback()
	if _, err = tx.ExecContext(ctx, `INSERT INTO users
		(id,username,password_hash,role,status,inn,bank_account,company_name,region_id,phone,created_at)
		VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11)`,
		u.ID, u.Username, hash, u.Role, u.Status, u.INN, u.BankAccount, u.CompanyName, u.RegionID, u.Phone, u.CreatedAt); err != nil {
		return u, err
	}
	companyID := "com-" + newID()
	if _, err = tx.ExecContext(ctx, `INSERT INTO companies
		(id,owner_user_id,name,inn,bank_account,region_id,created_at) VALUES ($1,$2,$3,$4,$5,$6,$7)`,
		companyID, u.ID, u.CompanyName, u.INN, u.BankAccount, u.RegionID, u.CreatedAt); err != nil {
		return u, err
	}
	if err = tx.Commit(); err != nil {
		return u, err
	}
	return u, nil
}

func (r *Repository) FindByUsername(ctx context.Context, username string) (domain.User, string, bool, error) {
	var hash string
	u, err := scanUser(r.db.QueryRowContext(ctx, `SELECT `+userCols+` FROM users WHERE username=$1`, username))
	if errors.Is(err, sql.ErrNoRows) {
		return u, "", false, nil
	}
	if err != nil {
		return u, "", false, err
	}
	err = r.db.QueryRowContext(ctx, `SELECT password_hash FROM users WHERE username=$1`, username).Scan(&hash)
	return u, hash, true, err
}

func (r *Repository) FindByID(ctx context.Context, id string) (domain.User, error) {
	u, err := scanUser(r.db.QueryRowContext(ctx, `SELECT `+userCols+` FROM users WHERE id=$1`, id))
	if errors.Is(err, sql.ErrNoRows) {
		return u, shared.ErrNotFound
	}
	return u, err
}

func (r *Repository) UpdateStatus(ctx context.Context, id, status string) (domain.User, error) {
	res, err := r.db.ExecContext(ctx, `UPDATE users SET status=$2 WHERE id=$1`, id, status)
	if err != nil {
		return domain.User{}, err
	}
	if n, _ := res.RowsAffected(); n == 0 {
		return domain.User{}, shared.ErrNotFound
	}
	return r.FindByID(ctx, id)
}

func (r *Repository) List(ctx context.Context, role, status string, limit, offset int) ([]domain.User, int, error) {
	cond := "1=1"
	var args []interface{}
	if role != "" {
		args = append(args, role)
		cond += " AND role=$" + itoa(len(args))
	}
	if status != "" {
		args = append(args, status)
		cond += " AND status=$" + itoa(len(args))
	}
	var total int
	if err := r.db.QueryRowContext(ctx, `SELECT count(*) FROM users WHERE `+cond, args...).Scan(&total); err != nil {
		return nil, 0, err
	}
	args = append(args, limit, offset)
	rows, err := r.db.QueryContext(ctx, `SELECT `+userCols+` FROM users WHERE `+cond+` ORDER BY created_at DESC LIMIT $`+itoa(len(args)-1)+` OFFSET $`+itoa(len(args)), args...)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()
	out := make([]domain.User, 0)
	for rows.Next() {
		u, err := scanUser(rows)
		if err != nil {
			return nil, 0, err
		}
		out = append(out, u)
	}
	return out, total, rows.Err()
}

func itoa(n int) string {
	if n == 0 {
		return "0"
	}
	out := ""
	for n > 0 {
		out = string(rune('0'+n%10)) + out
		n /= 10
	}
	return out
}

func newID() string {
	b := make([]byte, 6)
	_, _ = rand.Read(b)
	return hex.EncodeToString(b)
}
