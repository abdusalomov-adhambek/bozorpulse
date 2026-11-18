package infrastructure

import (
	"context"
	"crypto/rand"
	"database/sql"
	"encoding/hex"
	"errors"
	"time"

	"github.com/bozorpulse/bozorpulse-ai/backend/internal/modules/catalog/domain"
	"github.com/bozorpulse/bozorpulse-ai/backend/internal/shared"
	"github.com/lib/pq"
)

type Repository struct{ db *sql.DB }

func NewRepository(db *sql.DB) *Repository { return &Repository{db: db} }

const productCols = `id,supplier_id,COALESCE(category_id,''),name,COALESCE(sku,''),COALESCE(description,''),
	COALESCE(unit,'pcs'),price,discount_percent,effective_price,stock,COALESCE(region_id,''),
	COALESCE(photos,'{}'),min_order_qty,max_order_qty,delivery_min_days,delivery_max_days,created_at,updated_at`

const productColsP = `p.id,p.supplier_id,COALESCE(p.category_id,''),p.name,COALESCE(p.sku,''),COALESCE(p.description,''),
	COALESCE(p.unit,'pcs'),p.price,p.discount_percent,p.effective_price,p.stock,COALESCE(p.region_id,''),
	COALESCE(p.photos,'{}'),p.min_order_qty,p.max_order_qty,p.delivery_min_days,p.delivery_max_days,p.created_at,p.updated_at`

func scanProduct(row interface{ Scan(...interface{}) error }) (domain.Product, error) {
	var p domain.Product
	err := row.Scan(&p.ID, &p.SupplierID, &p.CategoryID, &p.Name, &p.SKU, &p.Description, &p.Unit,
		&p.Price, &p.DiscountPercent, &p.EffectivePrice, &p.Stock, &p.RegionID, pq.Array(&p.Photos),
		&p.MinOrderQty, &p.MaxOrderQty, &p.DeliveryMinDays, &p.DeliveryMaxDays, &p.CreatedAt, &p.UpdatedAt)
	return p, err
}

func (r *Repository) ListCategories(ctx context.Context) ([]domain.Category, error) {
	rows, err := r.db.QueryContext(ctx, `SELECT id,name,COALESCE(parent_id,''),created_at FROM categories ORDER BY name`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := make([]domain.Category, 0)
	for rows.Next() {
		var c domain.Category
		if err := rows.Scan(&c.ID, &c.Name, &c.ParentID, &c.CreatedAt); err != nil {
			return nil, err
		}
		out = append(out, c)
	}
	return out, rows.Err()
}

func (r *Repository) CreateCategory(ctx context.Context, name, parentID string) (domain.Category, error) {
	c := domain.Category{ID: "cat-" + newID(), Name: name, ParentID: parentID, CreatedAt: now()}
	_, err := r.db.ExecContext(ctx, `INSERT INTO categories (id,name,parent_id,created_at) VALUES ($1,$2,NULLIF($3,''),$4)`,
		c.ID, c.Name, c.ParentID, c.CreatedAt)
	return c, err
}

func (r *Repository) UpdateCategory(ctx context.Context, id, name, parentID string) (domain.Category, error) {
	_, err := r.db.ExecContext(ctx, `UPDATE categories SET name=$2,parent_id=NULLIF($3,'') WHERE id=$1`, id, name, parentID)
	if err != nil {
		return domain.Category{}, err
	}
	cats, _ := r.ListCategories(ctx)
	for _, c := range cats {
		if c.ID == id {
			return c, nil
		}
	}
	return domain.Category{}, shared.ErrNotFound
}

func (r *Repository) DeleteCategory(ctx context.Context, id string) error {
	res, err := r.db.ExecContext(ctx, `DELETE FROM categories WHERE id=$1`, id)
	if err != nil {
		return err
	}
	if n, _ := res.RowsAffected(); n == 0 {
		return shared.ErrNotFound
	}
	return nil
}

func (r *Repository) ListRegions(ctx context.Context) ([]domain.Region, error) {
	rows, err := r.db.QueryContext(ctx, `SELECT id,name,created_at FROM regions ORDER BY name`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := make([]domain.Region, 0)
	for rows.Next() {
		var rg domain.Region
		if err := rows.Scan(&rg.ID, &rg.Name, &rg.CreatedAt); err != nil {
			return nil, err
		}
		out = append(out, rg)
	}
	return out, rows.Err()
}

func (r *Repository) CreateRegion(ctx context.Context, name string) (domain.Region, error) {
	rg := domain.Region{ID: "reg-" + newID(), Name: name, CreatedAt: now()}
	_, err := r.db.ExecContext(ctx, `INSERT INTO regions (id,name,created_at) VALUES ($1,$2,$3)`, rg.ID, rg.Name, rg.CreatedAt)
	return rg, err
}

func (r *Repository) ListProducts(ctx context.Context, q, categoryID, supplierID string, minPrice, maxPrice float64, limit, offset int) ([]domain.Product, int, error) {
	cond := "1=1"
	var args []interface{}
	if q != "" {
		args = append(args, "%"+q+"%")
		cond += " AND (LOWER(p.name) LIKE LOWER($" + itoa(len(args)) + ") OR LOWER(p.sku) LIKE LOWER($" + itoa(len(args)) + "))"
	}
	if categoryID != "" {
		args = append(args, categoryID)
		cond += " AND p.category_id=$" + itoa(len(args))
	}
	if supplierID != "" {
		args = append(args, supplierID)
		cond += " AND p.supplier_id=$" + itoa(len(args))
	}
	if minPrice > 0 {
		args = append(args, minPrice)
		cond += " AND p.effective_price >= $" + itoa(len(args))
	}
	if maxPrice > 0 {
		args = append(args, maxPrice)
		cond += " AND p.effective_price <= $" + itoa(len(args))
	}
	var total int
	if err := r.db.QueryRowContext(ctx, `SELECT count(*) FROM products p WHERE `+cond, args...).Scan(&total); err != nil {
		return nil, 0, err
	}
	args = append(args, limit, offset)
	rows, err := r.db.QueryContext(ctx, `SELECT `+productColsP+` FROM products p WHERE `+cond+
		` ORDER BY p.created_at DESC LIMIT $`+itoa(len(args)-1)+` OFFSET $`+itoa(len(args)), args...)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()
	out := make([]domain.Product, 0)
	for rows.Next() {
		p, err := scanProduct(rows)
		if err != nil {
			return nil, 0, err
		}
		out = append(out, p)
	}
	return out, total, rows.Err()
}

func (r *Repository) GetProduct(ctx context.Context, id string) (domain.Product, error) {
	p, err := scanProduct(r.db.QueryRowContext(ctx, `SELECT `+productCols+` FROM products WHERE id=$1`, id))
	if errors.Is(err, sql.ErrNoRows) {
		return p, shared.ErrNotFound
	}
	return p, err
}

func (r *Repository) CreateProduct(ctx context.Context, p domain.Product) (domain.Product, error) {
	p.ID = "prd-" + newID()
	p.CreatedAt, p.UpdatedAt = now(), now()
	p.EffectivePrice = effective(p.Price, p.DiscountPercent)
	if p.MinOrderQty <= 0 {
		p.MinOrderQty = 1
	}
	if p.Photos == nil {
		p.Photos = []string{}
	}
	_, err := r.db.ExecContext(ctx, `INSERT INTO products
		(id,supplier_id,category_id,name,sku,description,unit,price,discount_percent,effective_price,stock,region_id,
		 photos,min_order_qty,max_order_qty,delivery_min_days,delivery_max_days,created_at,updated_at)
		VALUES ($1,$2,NULLIF($3,''),$4,$5,$6,$7,$8,$9,$10,$11,NULLIF($12,''),$13,$14,$15,$16,$17,$18,$19)`,
		p.ID, p.SupplierID, p.CategoryID, p.Name, p.SKU, p.Description, p.Unit, p.Price, p.DiscountPercent,
		p.EffectivePrice, p.Stock, p.RegionID, pq.Array(p.Photos), p.MinOrderQty, p.MaxOrderQty,
		p.DeliveryMinDays, p.DeliveryMaxDays, p.CreatedAt, p.UpdatedAt)
	return p, err
}

func (r *Repository) UpdateProduct(ctx context.Context, p domain.Product) (domain.Product, error) {
	old, err := r.GetProduct(ctx, p.ID)
	if err != nil {
		return p, err
	}
	if p.Price == 0 {
		p.Price = old.Price
	}
	if p.Name == "" {
		p.Name = old.Name
	}
	if p.SKU == "" {
		p.SKU = old.SKU
	}
	if p.Description == "" {
		p.Description = old.Description
	}
	if p.Unit == "" {
		p.Unit = old.Unit
	}
	if p.CategoryID == "" {
		p.CategoryID = old.CategoryID
	}
	if p.Stock == 0 {
		p.Stock = old.Stock
	}
	if p.DeliveryMinDays == 0 {
		p.DeliveryMinDays = old.DeliveryMinDays
	}
	if p.DeliveryMaxDays == 0 {
		p.DeliveryMaxDays = old.DeliveryMaxDays
	}
	p.UpdatedAt = now()
	p.EffectivePrice = effective(p.Price, p.DiscountPercent)
	if p.MinOrderQty <= 0 {
		p.MinOrderQty = old.MinOrderQty
	}
	if p.Photos == nil {
		p.Photos = old.Photos
	}
	_, err = r.db.ExecContext(ctx, `UPDATE products SET name=$2,category_id=NULLIF($3,''),sku=$4,description=$5,
		unit=$6,price=$7,discount_percent=$8,effective_price=$9,stock=$10,region_id=NULLIF($11,''),
		photos=$12,min_order_qty=$13,max_order_qty=$14,delivery_min_days=$15,delivery_max_days=$16,updated_at=$17 WHERE id=$1`,
		p.ID, p.Name, p.CategoryID, p.SKU, p.Description, p.Unit, p.Price, p.DiscountPercent, p.EffectivePrice,
		p.Stock, p.RegionID, pq.Array(p.Photos), p.MinOrderQty, p.MaxOrderQty, p.DeliveryMinDays,
		p.DeliveryMaxDays, p.UpdatedAt)
	return p, err
}

func (r *Repository) SetDiscount(ctx context.Context, id string, discount float64) (domain.Product, error) {
	if discount < 0 || discount > 90 {
		return domain.Product{}, shared.WithStatus(errors.New("discount must be between 0 and 90"), 400)
	}
	_, err := r.db.ExecContext(ctx, `UPDATE products SET discount_percent=$2,effective_price=price*(1-$2/100.0),updated_at=$3 WHERE id=$1`,
		id, discount, now())
	if err != nil {
		return domain.Product{}, err
	}
	return r.GetProduct(ctx, id)
}

func (r *Repository) DeleteProduct(ctx context.Context, id string) error {
	res, err := r.db.ExecContext(ctx, `DELETE FROM products WHERE id=$1`, id)
	if err != nil {
		return err
	}
	if n, _ := res.RowsAffected(); n == 0 {
		return shared.ErrNotFound
	}
	return nil
}

// PriceHistory returns the product's price observations oldest-first.
func (r *Repository) PriceHistory(ctx context.Context, productID string, limit int) (domain.PriceHistory, error) {
	out := domain.PriceHistory{ProductID: productID, Samples: []domain.PriceHistoryPoint{}, DataStatus: "insufficient"}
	if limit <= 0 || limit > 200 {
		limit = 100
	}
	err := r.db.QueryRowContext(ctx, `SELECT name FROM products WHERE id=$1`, productID).Scan(&out.ProductName)
	if err != nil {
		return out, err
	}
	rows, err := r.db.QueryContext(ctx, `SELECT id,price,source_type,is_demo,observed_at,COALESCE(region_id,'')
		FROM price_samples WHERE product_id=$1 ORDER BY observed_at DESC, id DESC LIMIT $2`, productID, limit)
	if err != nil {
		return out, err
	}
	defer rows.Close()
	for rows.Next() {
		var p domain.PriceHistoryPoint
		var observed string
		if err := rows.Scan(&p.ID, &p.Price, &p.SourceType, &p.IsDemo, &observed, &p.RegionID); err != nil {
			return out, err
		}
		p.ObservedAt = observed
		out.Samples = append(out.Samples, p)
	}
	if err := rows.Err(); err != nil {
		return out, err
	}
	// Reverse to chronological order for charts.
	for i, j := 0, len(out.Samples)-1; i < j; i, j = i+1, j-1 {
		out.Samples[i], out.Samples[j] = out.Samples[j], out.Samples[i]
	}
	out.Count = len(out.Samples)
	if out.Count == 0 {
		out.Disclaimer = "No price observations recorded for this product yet."
		return out, nil
	}
	min, max, sum := out.Samples[0].Price, out.Samples[0].Price, 0.0
	for _, p := range out.Samples {
		if p.Price < min { min = p.Price }
		if p.Price > max { max = p.Price }
		sum += p.Price
	}
	avg := sum / float64(out.Count)
	out.Min, out.Max, out.Avg = &min, &max, &avg
	out.DataStatus = "live"
	for _, p := range out.Samples {
		if p.IsDemo {
			out.DataStatus = "demo"
			out.Disclaimer = "Sample market data for demonstration. Not a live market quote."
			break
		}
	}
	return out, nil
}

func (r *Repository) ListPricing(ctx context.Context, productID string) ([]domain.PricingRule, error) {
	rows, err := r.db.QueryContext(ctx, `SELECT pr.id,pr.product_id,COALESCE(pr.warehouse_id,''),COALESCE(w.name,''),
		pr.min_qty,pr.max_qty,pr.unit_price,pr.discount_percent,pr.delivery_min_days,pr.delivery_max_days,
		pr.min_order,pr.max_order,pr.created_at
		FROM product_warehouse_pricing pr LEFT JOIN warehouses w ON w.id=pr.warehouse_id
		WHERE pr.product_id=$1 ORDER BY pr.warehouse_id NULLS FIRST, pr.min_qty`, productID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := make([]domain.PricingRule, 0)
	for rows.Next() {
		var rule domain.PricingRule
		if err := rows.Scan(&rule.ID, &rule.ProductID, &rule.WarehouseID, &rule.WarehouseName, &rule.MinQty,
			&rule.MaxQty, &rule.UnitPrice, &rule.DiscountPercent, &rule.DeliveryMinDays, &rule.DeliveryMaxDays,
			&rule.MinOrder, &rule.MaxOrder, &rule.CreatedAt); err != nil {
			return nil, err
		}
		rule.EffectivePrice = effective(rule.UnitPrice, rule.DiscountPercent)
		out = append(out, rule)
	}
	return out, rows.Err()
}

func (r *Repository) CreatePricing(ctx context.Context, rule domain.PricingRule) (domain.PricingRule, error) {
	rule.ID = "prc-" + newID()
	rule.CreatedAt = now()
	if rule.MinQty <= 0 {
		rule.MinQty = 1
	}
	if rule.MinOrder <= 0 {
		rule.MinOrder = rule.MinQty
	}
	_, err := r.db.ExecContext(ctx, `INSERT INTO product_warehouse_pricing
		(id,product_id,warehouse_id,min_qty,max_qty,unit_price,discount_percent,delivery_min_days,delivery_max_days,min_order,max_order,created_at)
		VALUES ($1,$2,NULLIF($3,''),$4,$5,$6,$7,$8,$9,$10,$11,$12)`,
		rule.ID, rule.ProductID, rule.WarehouseID, rule.MinQty, rule.MaxQty, rule.UnitPrice, rule.DiscountPercent,
		rule.DeliveryMinDays, rule.DeliveryMaxDays, rule.MinOrder, rule.MaxOrder, rule.CreatedAt)
	if err != nil {
		return rule, err
	}
	rule.EffectivePrice = effective(rule.UnitPrice, rule.DiscountPercent)
	return rule, nil
}

func (r *Repository) DeletePricing(ctx context.Context, id string) error {
	res, err := r.db.ExecContext(ctx, `DELETE FROM product_warehouse_pricing WHERE id=$1`, id)
	if err != nil {
		return err
	}
	if n, _ := res.RowsAffected(); n == 0 {
		return shared.ErrNotFound
	}
	return nil
}

func (r *Repository) ResolvePrice(ctx context.Context, productID, warehouseID string, quantity int) (domain.ResolvedPrice, error) {
	product, err := r.GetProduct(ctx, productID)
	if err != nil {
		return domain.ResolvedPrice{}, err
	}
	result := domain.ResolvedPrice{
		UnitPrice: product.Price, DiscountPercent: product.DiscountPercent,
		EffectivePrice:  product.EffectivePrice,
		DeliveryMinDays: product.DeliveryMinDays, DeliveryMaxDays: product.DeliveryMaxDays,
		MinOrder: product.MinOrderQty, MaxOrder: product.MaxOrderQty,
		Source: "product",
	}
	rules, err := r.ListPricing(ctx, productID)
	if err != nil {
		return result, err
	}
	best := -1
	bestScore := -1
	for i, rule := range rules {
		if rule.WarehouseID != "" && rule.WarehouseID != warehouseID {
			continue
		}
		if rule.MinQty > 0 && quantity < rule.MinQty {
			continue
		}
		if rule.MaxQty > 0 && quantity > rule.MaxQty {
			continue
		}
		score := 0
		if rule.WarehouseID != "" {
			score += 10
		}
		if rule.MaxQty > 0 {
			score += 3
		}
		if score > bestScore || (score == bestScore && best >= 0 && rule.EffectivePrice < rules[best].EffectivePrice) {
			best, bestScore = i, score
		}
	}
	if best >= 0 {
		rule := rules[best]
		result.UnitPrice = rule.UnitPrice
		result.DiscountPercent = rule.DiscountPercent
		result.EffectivePrice = rule.EffectivePrice
		if rule.DeliveryMinDays > 0 || rule.DeliveryMaxDays > 0 {
			result.DeliveryMinDays = rule.DeliveryMinDays
			result.DeliveryMaxDays = rule.DeliveryMaxDays
		}
		if rule.MinOrder > 0 {
			result.MinOrder = rule.MinOrder
		}
		if rule.MaxOrder > 0 {
			result.MaxOrder = rule.MaxOrder
		}
		result.Source = "product"
		if rule.WarehouseID != "" {
			result.Source = "warehouse"
		}
		if rule.MaxQty > 0 {
			result.Source += "+tier"
		}
		result.RuleID = rule.ID
	}
	if result.MinOrder <= 0 {
		result.MinOrder = 1
	}
	return result, nil
}

func (r *Repository) ListDeliveryZones(ctx context.Context, supplierID string) ([]domain.DeliveryZone, error) {
	rows, err := r.db.QueryContext(ctx, `SELECT dz.id,dz.supplier_id,dz.region_id,COALESCE(rg.name,''),
		dz.available,dz.price,dz.min_days,dz.max_days,dz.notes,dz.created_at
		FROM delivery_zones dz LEFT JOIN regions rg ON rg.id=dz.region_id
		WHERE dz.supplier_id=$1 ORDER BY rg.name`, supplierID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := make([]domain.DeliveryZone, 0)
	for rows.Next() {
		var z domain.DeliveryZone
		if err := rows.Scan(&z.ID, &z.SupplierID, &z.RegionID, &z.RegionName, &z.Available, &z.Price,
			&z.MinDays, &z.MaxDays, &z.Notes, &z.CreatedAt); err != nil {
			return nil, err
		}
		out = append(out, z)
	}
	return out, rows.Err()
}

func (r *Repository) CreateDeliveryZone(ctx context.Context, z domain.DeliveryZone) (domain.DeliveryZone, error) {
	z.ID = "dz-" + newID()
	z.CreatedAt = now()
	_, err := r.db.ExecContext(ctx, `INSERT INTO delivery_zones
		(id,supplier_id,region_id,available,price,min_days,max_days,notes,created_at)
		VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9)`,
		z.ID, z.SupplierID, z.RegionID, z.Available, z.Price, z.MinDays, z.MaxDays, z.Notes, z.CreatedAt)
	return z, err
}

func (r *Repository) UpdateDeliveryZone(ctx context.Context, z domain.DeliveryZone) (domain.DeliveryZone, error) {
	_, err := r.db.ExecContext(ctx, `UPDATE delivery_zones SET available=$2,price=$3,min_days=$4,max_days=$5,notes=$6
		WHERE id=$1 AND supplier_id=$7`, z.ID, z.Available, z.Price, z.MinDays, z.MaxDays, z.Notes, z.SupplierID)
	if err != nil {
		return z, err
	}
	zones, err := r.ListDeliveryZones(ctx, z.SupplierID)
	if err != nil {
		return z, err
	}
	for _, item := range zones {
		if item.ID == z.ID {
			return item, nil
		}
	}
	return z, shared.ErrNotFound
}

func (r *Repository) DeleteDeliveryZone(ctx context.Context, id, supplierID string) error {
	res, err := r.db.ExecContext(ctx, `DELETE FROM delivery_zones WHERE id=$1 AND supplier_id=$2`, id, supplierID)
	if err != nil {
		return err
	}
	if n, _ := res.RowsAffected(); n == 0 {
		return shared.ErrNotFound
	}
	return nil
}

func effective(price, discount float64) float64 {
	return round2(price * (1 - discount/100))
}

func round2(v float64) float64 { return float64(int(v*100+0.5)) / 100 }

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
