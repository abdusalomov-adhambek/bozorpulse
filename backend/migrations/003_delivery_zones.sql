-- Supplier delivery availability, price and period per region.
CREATE TABLE IF NOT EXISTS delivery_zones (
  id          TEXT PRIMARY KEY,
  supplier_id TEXT NOT NULL REFERENCES companies(id) ON DELETE CASCADE,
  region_id   TEXT NOT NULL REFERENCES regions(id) ON DELETE CASCADE,
  available   BOOLEAN NOT NULL DEFAULT true,
  price       NUMERIC(14,2) NOT NULL DEFAULT 0 CHECK (price >= 0),
  min_days    INTEGER NOT NULL DEFAULT 0,
  max_days    INTEGER NOT NULL DEFAULT 0,
  notes       TEXT NOT NULL DEFAULT '',
  created_at  TEXT NOT NULL,
  UNIQUE (supplier_id, region_id)
);

CREATE INDEX IF NOT EXISTS idx_delivery_zones_supplier ON delivery_zones (supplier_id);
