-- Product media, order limits, delivery windows and warehouse/tier pricing.
ALTER TABLE products
  ADD COLUMN IF NOT EXISTS photos TEXT[] NOT NULL DEFAULT '{}',
  ADD COLUMN IF NOT EXISTS min_order_qty INTEGER NOT NULL DEFAULT 1,
  ADD COLUMN IF NOT EXISTS max_order_qty INTEGER NOT NULL DEFAULT 0,
  ADD COLUMN IF NOT EXISTS delivery_min_days INTEGER NOT NULL DEFAULT 0,
  ADD COLUMN IF NOT EXISTS delivery_max_days INTEGER NOT NULL DEFAULT 0;

-- Warehouse-specific and quantity-tiered price rules.
-- warehouse_id NULL = applies to every warehouse (default rule).
-- min_qty / max_qty define the order quantity band (max_qty 0 = unlimited).
CREATE TABLE IF NOT EXISTS product_warehouse_pricing (
  id                 TEXT PRIMARY KEY,
  product_id         TEXT NOT NULL REFERENCES products(id) ON DELETE CASCADE,
  warehouse_id       TEXT REFERENCES warehouses(id) ON DELETE CASCADE,
  min_qty            INTEGER NOT NULL DEFAULT 1,
  max_qty            INTEGER NOT NULL DEFAULT 0,
  unit_price         NUMERIC(14,2) NOT NULL CHECK (unit_price >= 0),
  discount_percent   NUMERIC(5,2) NOT NULL DEFAULT 0 CHECK (discount_percent BETWEEN 0 AND 90),
  delivery_min_days  INTEGER NOT NULL DEFAULT 0,
  delivery_max_days  INTEGER NOT NULL DEFAULT 0,
  min_order          INTEGER NOT NULL DEFAULT 1,
  max_order          INTEGER NOT NULL DEFAULT 0,
  created_at         TEXT NOT NULL,
  UNIQUE (product_id, warehouse_id, min_qty)
);

CREATE INDEX IF NOT EXISTS idx_pwp_product ON product_warehouse_pricing (product_id);
CREATE INDEX IF NOT EXISTS idx_pwp_warehouse ON product_warehouse_pricing (warehouse_id);
