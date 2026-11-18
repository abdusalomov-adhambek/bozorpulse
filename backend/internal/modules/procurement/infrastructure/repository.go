package infrastructure

import (
	"context"
	"crypto/rand"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"errors"
	"time"

	"github.com/bozorpulse/bozorpulse-ai/backend/internal/modules/procurement/domain"
	"github.com/bozorpulse/bozorpulse-ai/backend/internal/shared"
)

type Repository struct{ db *sql.DB }

func NewRepository(db *sql.DB) *Repository { return &Repository{db: db} }

func (r *Repository) ProductInfo(ctx context.Context, id string) (name string, price float64, supplierID string, ok bool, err error) {
	err = r.db.QueryRowContext(ctx, `SELECT name,effective_price,supplier_id FROM products WHERE id=$1`, id).
		Scan(&name, &price, &supplierID)
	if errors.Is(err, sql.ErrNoRows) {
		return "", 0, "", false, nil
	}
	return name, price, supplierID, err == nil, err
}

func (r *Repository) WarehouseOwner(ctx context.Context, id string) (owner string, ok bool, err error) {
	err = r.db.QueryRowContext(ctx, `SELECT owner_id FROM warehouses WHERE id=$1`, id).Scan(&owner)
	if errors.Is(err, sql.ErrNoRows) {
		return "", false, nil
	}
	return owner, err == nil, err
}

func (r *Repository) Create(ctx context.Context, o domain.Order) (domain.Order, error) {
	o.ID = "ord-" + newID()
	o.Status = "new"
	o.CreatedAt, o.UpdatedAt = now(), now()
	itemsJSON, _ := json.Marshal(o.Items)
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return o, err
	}
	defer tx.Rollback()
	_, err = tx.ExecContext(ctx, `INSERT INTO orders
		(id,customer_id,supplier_id,warehouse_id,status,total_amount,items,created_at,updated_at)
		VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9)`,
		o.ID, o.CustomerID, o.SupplierID, o.WarehouseID, o.Status, o.TotalAmount, itemsJSON, o.CreatedAt, o.UpdatedAt)
	if err != nil {
		return o, err
	}
	return o, tx.Commit()
}

func scanOrder(row interface{ Scan(...interface{}) error }) (domain.Order, error) {
	var o domain.Order
	var raw []byte
	err := row.Scan(&o.ID, &o.CustomerID, &o.SupplierID, &o.WarehouseID, &o.Status, &o.TotalAmount, &raw, &o.CreatedAt, &o.UpdatedAt)
	if err != nil {
		return o, err
	}
	_ = json.Unmarshal(raw, &o.Items)
	return o, nil
}

func (r *Repository) Get(ctx context.Context, id string) (domain.Order, error) {
	o, err := scanOrder(r.db.QueryRowContext(ctx, `SELECT id,customer_id,supplier_id,warehouse_id,status,
		total_amount,items,created_at,updated_at FROM orders WHERE id=$1`, id))
	if errors.Is(err, sql.ErrNoRows) {
		return o, shared.ErrNotFound
	}
	return o, err
}

func (r *Repository) List(ctx context.Context, customerID, supplierID string, admin bool, limit, offset int) ([]domain.Order, int, error) {
	cond := "1=1"
	var args []interface{}
	if !admin {
		if customerID != "" {
			args = append(args, customerID)
			cond += " AND customer_id=$" + itoa(len(args))
		}
		if supplierID != "" {
			args = append(args, supplierID)
			cond += " AND supplier_id=$" + itoa(len(args))
		}
	}
	var total int
	if err := r.db.QueryRowContext(ctx, `SELECT count(*) FROM orders WHERE `+cond, args...).Scan(&total); err != nil {
		return nil, 0, err
	}
	args = append(args, limit, offset)
	rows, err := r.db.QueryContext(ctx, `SELECT id,customer_id,supplier_id,warehouse_id,status,total_amount,
		items,created_at,updated_at FROM orders WHERE `+cond+` ORDER BY created_at DESC LIMIT $`+itoa(len(args)-1)+` OFFSET $`+itoa(len(args)), args...)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()
	out := make([]domain.Order, 0)
	for rows.Next() {
		o, err := scanOrder(rows)
		if err != nil {
			return nil, 0, err
		}
		out = append(out, o)
	}
	return out, total, rows.Err()
}

func (r *Repository) SetStatus(ctx context.Context, id, status string) (domain.Order, error) {
	_, err := r.db.ExecContext(ctx, `UPDATE orders SET status=$2,updated_at=$3 WHERE id=$1`, id, status, now())
	if err != nil {
		return domain.Order{}, err
	}
	return r.Get(ctx, id)
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

var _ = shared.ErrNotFound
