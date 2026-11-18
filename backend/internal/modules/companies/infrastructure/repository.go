package infrastructure

import (
	"context"
	"crypto/rand"
	"database/sql"
	"encoding/hex"
	"errors"
	"time"

	"github.com/bozorpulse/bozorpulse-ai/backend/internal/modules/companies/domain"
	"github.com/bozorpulse/bozorpulse-ai/backend/internal/shared"
)

type Repository struct{ db *sql.DB }

func NewRepository(db *sql.DB) *Repository { return &Repository{db: db} }

func (r *Repository) List(ctx context.Context, role, status string, limit, offset int) ([]domain.Company, int, error) {
	cond := "u.role<>'admin'"
	var args []interface{}
	if role != "" {
		args = append(args, role)
		cond += " AND u.role=$" + itoa(len(args))
	}
	if status != "" {
		args = append(args, status)
		cond += " AND u.status=$" + itoa(len(args))
	}
	var total int
	if err := r.db.QueryRowContext(ctx, `SELECT count(*) FROM companies c JOIN users u ON u.id=c.owner_user_id WHERE `+cond, args...).Scan(&total); err != nil {
		return nil, 0, err
	}
	args = append(args, limit, offset)
	rows, err := r.db.QueryContext(ctx, `SELECT c.id,c.owner_user_id,u.role,c.name,c.inn,c.bank_account,
		COALESCE(c.region_id,''),COALESCE(rg.name,''),u.status,c.created_at
		FROM companies c JOIN users u ON u.id=c.owner_user_id LEFT JOIN regions rg ON rg.id=c.region_id
		WHERE `+cond+` ORDER BY c.created_at DESC LIMIT $`+itoa(len(args)-1)+` OFFSET $`+itoa(len(args)), args...)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()
	out := make([]domain.Company, 0)
	for rows.Next() {
		var c domain.Company
		if err := rows.Scan(&c.ID, &c.OwnerUserID, &c.Role, &c.Name, &c.INN, &c.BankAccount,
			&c.RegionID, &c.RegionName, &c.Status, &c.CreatedAt); err != nil {
			return nil, 0, err
		}
		out = append(out, c)
	}
	return out, total, rows.Err()
}

func (r *Repository) Get(ctx context.Context, id string) (domain.Company, error) {
	var c domain.Company
	err := r.db.QueryRowContext(ctx, `SELECT c.id,c.owner_user_id,u.role,c.name,c.inn,c.bank_account,
		COALESCE(c.region_id,''),COALESCE(rg.name,''),u.status,c.created_at
		FROM companies c JOIN users u ON u.id=c.owner_user_id LEFT JOIN regions rg ON rg.id=c.region_id WHERE c.id=$1`, id).
		Scan(&c.ID, &c.OwnerUserID, &c.Role, &c.Name, &c.INN, &c.BankAccount, &c.RegionID, &c.RegionName, &c.Status, &c.CreatedAt)
	if errors.Is(err, sql.ErrNoRows) {
		return c, shared.ErrNotFound
	}
	return c, err
}

func (r *Repository) Update(ctx context.Context, id string, name, inn, bank, regionID string) (domain.Company, error) {
	_, err := r.db.ExecContext(ctx, `UPDATE companies SET name=$2,inn=$3,bank_account=$4,region_id=NULLIF($5,'') WHERE id=$1`,
		id, name, inn, bank, regionID)
	if err != nil {
		return domain.Company{}, err
	}
	return r.Get(ctx, id)
}

const warehouseCols = `id,owner_id,owner_role,name,address,city,COALESCE(region_id,''),latitude,longitude,created_at`

func (r *Repository) ListWarehouses(ctx context.Context, ownerID string) ([]domain.Warehouse, error) {
	query := `SELECT ` + warehouseCols + ` FROM warehouses`
	var args []interface{}
	if ownerID != "" {
		args = append(args, ownerID)
		query += " WHERE owner_id=$1"
	}
	query += " ORDER BY created_at DESC"
	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := make([]domain.Warehouse, 0)
	for rows.Next() {
		var wh domain.Warehouse
		if err := rows.Scan(&wh.ID, &wh.OwnerID, &wh.OwnerRole, &wh.Name, &wh.Address, &wh.City,
			&wh.RegionID, &wh.Latitude, &wh.Longitude, &wh.CreatedAt); err != nil {
			return nil, err
		}
		out = append(out, wh)
	}
	return out, rows.Err()
}

func (r *Repository) GetWarehouse(ctx context.Context, id string) (domain.Warehouse, error) {
	var wh domain.Warehouse
	err := r.db.QueryRowContext(ctx, `SELECT `+warehouseCols+` FROM warehouses WHERE id=$1`, id).
		Scan(&wh.ID, &wh.OwnerID, &wh.OwnerRole, &wh.Name, &wh.Address, &wh.City, &wh.RegionID, &wh.Latitude, &wh.Longitude, &wh.CreatedAt)
	if errors.Is(err, sql.ErrNoRows) {
		return wh, shared.ErrNotFound
	}
	return wh, err
}

func (r *Repository) CreateWarehouse(ctx context.Context, wh domain.Warehouse) (domain.Warehouse, error) {
	wh.ID = "wh-" + newID()
	wh.CreatedAt = now()
	_, err := r.db.ExecContext(ctx, `INSERT INTO warehouses
		(id,owner_id,owner_role,name,address,city,region_id,latitude,longitude,created_at)
		VALUES ($1,$2,$3,$4,$5,$6,NULLIF($7,''),$8,$9,$10)`,
		wh.ID, wh.OwnerID, wh.OwnerRole, wh.Name, wh.Address, wh.City, wh.RegionID, wh.Latitude, wh.Longitude, wh.CreatedAt)
	return wh, err
}

func (r *Repository) UpdateWarehouse(ctx context.Context, wh domain.Warehouse) (domain.Warehouse, error) {
	_, err := r.db.ExecContext(ctx, `UPDATE warehouses SET name=$2,address=$3,city=$4,region_id=NULLIF($5,''),
		latitude=$6,longitude=$7 WHERE id=$1`, wh.ID, wh.Name, wh.Address, wh.City, wh.RegionID, wh.Latitude, wh.Longitude)
	if err != nil {
		return wh, err
	}
	return r.GetWarehouse(ctx, wh.ID)
}

func (r *Repository) DeleteWarehouse(ctx context.Context, id string) error {
	res, err := r.db.ExecContext(ctx, `DELETE FROM warehouses WHERE id=$1`, id)
	if err != nil {
		return err
	}
	if n, _ := res.RowsAffected(); n == 0 {
		return shared.ErrNotFound
	}
	return nil
}

func now() string { return time.Now().UTC().Format(time.RFC3339) }
func newID() string {
	b := make([]byte, 6)
	_, _ = rand.Read(b)
	return hex.EncodeToString(b)
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
