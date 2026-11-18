package infrastructure

import (
	"context"
	"crypto/rand"
	"database/sql"
	"encoding/hex"
	"errors"
	"time"

	"github.com/bozorpulse/bozorpulse-ai/backend/internal/modules/inventory/domain"
	"github.com/bozorpulse/bozorpulse-ai/backend/internal/shared"
)

type Repository struct{ db *sql.DB }

func NewRepository(db *sql.DB) *Repository { return &Repository{db: db} }

func (r *Repository) Owner(ctx context.Context, warehouseID string) (owner string, ok bool, err error) {
	err = r.db.QueryRowContext(ctx, `SELECT owner_id FROM warehouses WHERE id=$1`, warehouseID).Scan(&owner)
	if errors.Is(err, sql.ErrNoRows) {
		return "", false, nil
	}
	return owner, err == nil, err
}

func (r *Repository) ProductExists(ctx context.Context, id string) (bool, error) {
	var n int
	err := r.db.QueryRowContext(ctx, `SELECT count(*) FROM products WHERE id=$1`, id).Scan(&n)
	return n > 0, err
}

func (r *Repository) Stock(ctx context.Context, warehouseID, ownerID string, admin bool) ([]domain.StockItem, int, error) {
	cond := "1=1"
	var args []interface{}
	if !admin {
		args = append(args, ownerID)
		cond += " AND w.owner_id=$" + itoa(len(args))
	}
	if warehouseID != "" {
		args = append(args, warehouseID)
		cond += " AND s.warehouse_id=$" + itoa(len(args))
	}
	query := `SELECT s.product_id,COALESCE(p.name,s.product_id),s.warehouse_id,s.quantity,s.updated_at
		FROM stock_items s JOIN warehouses w ON w.id=s.warehouse_id LEFT JOIN products p ON p.id=s.product_id
		WHERE ` + cond + ` ORDER BY s.updated_at DESC`
	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()
	out := make([]domain.StockItem, 0)
	for rows.Next() {
		var s domain.StockItem
		if err := rows.Scan(&s.ProductID, &s.ProductName, &s.WarehouseID, &s.Quantity, &s.UpdatedAt); err != nil {
			return nil, 0, err
		}
		out = append(out, s)
	}
	return out, len(out), rows.Err()
}

// ApplyMovement records a movement and adjusts stock atomically. Quantity is
// signed: positive for inbound, negative for outbound.
func (r *Repository) ApplyMovement(ctx context.Context, m domain.Movement, signed int) (domain.Movement, error) {
	m.ID = "mov-" + newID()
	m.CreatedAt = time.Now().UTC().Format(time.RFC3339)
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return m, err
	}
	defer tx.Rollback()
	_, err = tx.ExecContext(ctx, `INSERT INTO stock_items (warehouse_id,product_id,quantity,updated_at)
		VALUES ($1,$2,$3,$4) ON CONFLICT (warehouse_id,product_id)
		DO UPDATE SET quantity=stock_items.quantity+EXCLUDED.quantity, updated_at=EXCLUDED.updated_at`,
		m.WarehouseID, m.ProductID, signed, m.CreatedAt)
	if err != nil {
		return m, err
	}
	_, err = tx.ExecContext(ctx, `INSERT INTO inventory_movements (id,warehouse_id,product_id,quantity,type,note,created_at)
		VALUES ($1,$2,$3,$4,$5,$6,$7)`, m.ID, m.WarehouseID, m.ProductID, m.Quantity, m.Type, m.Note, m.CreatedAt)
	if err != nil {
		return m, err
	}
	return m, tx.Commit()
}

func (r *Repository) Movements(ctx context.Context, warehouseID, ownerID string, admin bool) ([]domain.Movement, int, error) {
	cond := "1=1"
	var args []interface{}
	if !admin {
		args = append(args, ownerID)
		cond += " AND w.owner_id=$" + itoa(len(args))
	}
	if warehouseID != "" {
		args = append(args, warehouseID)
		cond += " AND m.warehouse_id=$" + itoa(len(args))
	}
	query := `SELECT m.id,m.warehouse_id,m.product_id,m.quantity,m.type,COALESCE(m.note,''),m.created_at
		FROM inventory_movements m JOIN warehouses w ON w.id=m.warehouse_id WHERE ` + cond + ` ORDER BY m.created_at DESC LIMIT 200`
	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()
	out := make([]domain.Movement, 0)
	for rows.Next() {
		var m domain.Movement
		if err := rows.Scan(&m.ID, &m.WarehouseID, &m.ProductID, &m.Quantity, &m.Type, &m.Note, &m.CreatedAt); err != nil {
			return nil, 0, err
		}
		out = append(out, m)
	}
	return out, len(out), rows.Err()
}

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
