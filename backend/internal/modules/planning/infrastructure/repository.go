package infrastructure

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"github.com/bozorpulse/bozorpulse-ai/backend/internal/modules/planning/domain"
	"github.com/bozorpulse/bozorpulse-ai/backend/internal/shared"
)

// Repository owns the plans and plan_items tables.
type Repository struct{ db *sql.DB }

func NewRepository(db *sql.DB) *Repository { return &Repository{db: db} }

const itemColumns = `id,plan_id,product_id,product_name,unit,quantity,target_date,note,
		current_unit_price,predicted,low,high,change_pct,trend_pct,horizon_days,
		estimated_budget,budget_low,budget_high,saving_vs_now,
		recommendation,reason,data_status,basis,projection_note,generated_at,created_at`

func (r *Repository) Save(ctx context.Context, plan domain.Plan) error {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	_, err = tx.ExecContext(ctx, `
		INSERT INTO plans (id,company_id,customer_id,title,region_id,status,created_at,updated_at)
		VALUES ($1,$2,$3,$4,$5,$6,$7,$8)
		ON CONFLICT (id) DO UPDATE SET
			company_id=EXCLUDED.company_id, customer_id=EXCLUDED.customer_id,
			title=EXCLUDED.title, region_id=EXCLUDED.region_id, status=EXCLUDED.status,
			updated_at=EXCLUDED.updated_at`,
		plan.ID, plan.CompanyID, plan.CustomerID, plan.Title, nullable(plan.RegionID),
		string(plan.Status), nullable(plan.CreatedAt), nullable(plan.UpdatedAt))
	if err != nil {
		return err
	}
	if _, err := tx.ExecContext(ctx, `DELETE FROM plan_items WHERE plan_id=$1`, plan.ID); err != nil {
		return err
	}
	for _, item := range plan.Items {
		p := item.Projection
		_, err := tx.ExecContext(ctx, `
			INSERT INTO plan_items (`+itemColumns+`)
			VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14,$15,$16,$17,$18,$19,$20,$21,$22,$23,$24,$25,$26)`,
			item.ID, plan.ID, item.ProductID, item.ProductName, item.Unit, item.Quantity,
			nullable(item.TargetDate), item.Note,
			p.CurrentUnitPrice, p.Predicted, p.Low, p.High, p.ChangePct,
			p.TrendPct, p.HorizonDays, p.EstimatedBudget, p.BudgetLow, p.BudgetHigh,
			p.SavingVsNow, string(p.Recommendation), p.Reason, p.DataStatus,
			p.Basis, p.Note, nullable(p.GeneratedAt), nullable(time.Now().UTC().Format(time.RFC3339)))
		if err != nil {
			return fmt.Errorf("save plan item %s: %w", item.ID, err)
		}
	}
	return tx.Commit()
}

func (r *Repository) Get(ctx context.Context, ownerKey, id string) (domain.Plan, error) {
	var plan domain.Plan
	var region, created, updated sql.NullString
	var status string
	err := r.db.QueryRowContext(ctx, `
		SELECT id,company_id,customer_id,title,region_id,status,
		       COALESCE(created_at::text,''),COALESCE(updated_at::text,'')
		FROM plans WHERE id=$1`, id).Scan(
		&plan.ID, &plan.CompanyID, &plan.CustomerID, &plan.Title, &region, &status, &created, &updated)
	if errors.Is(err, sql.ErrNoRows) {
		return domain.Plan{}, shared.ErrNotFound
	}
	if err != nil {
		return domain.Plan{}, err
	}
	if plan.CompanyID != ownerKey {
		return domain.Plan{}, shared.ErrForbidden
	}
	plan.RegionID, plan.Status, plan.CreatedAt, plan.UpdatedAt = region.String, domain.PlanStatus(status), created.String, updated.String
	items, err := r.items(ctx, plan.ID)
	if err != nil {
		return domain.Plan{}, err
	}
	plan.Items = items
	return plan, nil
}

func (r *Repository) List(ctx context.Context, ownerKey string) ([]domain.Plan, error) {
	rows, err := r.db.QueryContext(ctx, `
		SELECT id,company_id,customer_id,title,COALESCE(region_id,''),status,
		       COALESCE(created_at::text,''),COALESCE(updated_at::text,'')
		FROM plans WHERE company_id=$1 ORDER BY updated_at DESC`, ownerKey)
	if err != nil {
		return nil, err
	}
	out := []domain.Plan{}
	for rows.Next() {
		var p domain.Plan
		var status string
		if err := rows.Scan(&p.ID, &p.CompanyID, &p.CustomerID, &p.Title, &p.RegionID, &status, &p.CreatedAt, &p.UpdatedAt); err != nil {
			rows.Close()
			return nil, err
		}
		p.Status = domain.PlanStatus(status)
		out = append(out, p)
	}
	if err := rows.Err(); err != nil {
		rows.Close()
		return nil, err
	}
	rows.Close()
	for i := range out {
		items, err := r.items(ctx, out[i].ID)
		if err != nil {
			return nil, err
		}
		out[i].Items = items
	}
	return out, nil
}

func (r *Repository) Delete(ctx context.Context, ownerKey, id string) error {
	plan, err := r.Get(ctx, ownerKey, id)
	if err != nil {
		return err
	}
	res, err := r.db.ExecContext(ctx, `DELETE FROM plans WHERE id=$1 AND company_id=$2`, plan.ID, ownerKey)
	if err != nil {
		return err
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return shared.ErrNotFound
	}
	return nil
}

func (r *Repository) items(ctx context.Context, planID string) ([]domain.PlanItem, error) {
	// Cast temporal columns to text so lib/pq's time.Time values scan cleanly.
	const selectCols = `id,product_id,product_name,unit,quantity,target_date::text,note,
		current_unit_price,predicted,low,high,change_pct,trend_pct,horizon_days,
		estimated_budget,budget_low,budget_high,saving_vs_now,
		recommendation,reason,data_status,basis,projection_note,generated_at::text,created_at::text`
	rows, err := r.db.QueryContext(ctx, `SELECT `+selectCols+` FROM plan_items WHERE plan_id=$1 ORDER BY created_at, id`, planID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []domain.PlanItem{}
	for rows.Next() {
		var item domain.PlanItem
		var p domain.PriceProjection
		var target, note, basis, projectionNote, generated, created sql.NullString
		var current, predicted, low, high, change, budget, budgetLow, budgetHigh, saving sql.NullFloat64
		var rec, status string
		err := rows.Scan(
			&item.ID, &item.ProductID, &item.ProductName, &item.Unit, &item.Quantity,
			&target, &note, &current, &predicted, &low, &high, &change, &p.TrendPct,
			&p.HorizonDays, &budget, &budgetLow, &budgetHigh, &saving,
			&rec, &p.Reason, &status, &basis, &projectionNote, &generated, &created)
		if err != nil {
			return nil, err
		}
		item.TargetDate, item.Note = target.String, note.String
		p.CurrentUnitPrice, p.Predicted, p.Low, p.High = floatPtr(current), floatPtr(predicted), floatPtr(low), floatPtr(high)
		p.ChangePct, p.EstimatedBudget, p.BudgetLow = floatPtr(change), floatPtr(budget), floatPtr(budgetLow)
		p.BudgetHigh, p.SavingVsNow = floatPtr(budgetHigh), floatPtr(saving)
		p.Recommendation, p.DataStatus = domain.Recommendation(rec), status
		p.Basis, p.Note, p.GeneratedAt = basis.String, projectionNote.String, generated.String
		item.Projection = p
		out = append(out, item)
	}
	return out, rows.Err()
}

func nullable(s string) any {
	if s == "" {
		return nil
	}
	return s
}

func floatPtr(v sql.NullFloat64) *float64 {
	if !v.Valid {
		return nil
	}
	f := v.Float64
	return &f
}
