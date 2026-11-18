-- BozorPulse Planning module (purchase-timing plans).
-- Applied on first init in filename order, or manually with:
--   psql "$DATABASE_URL" -f backend/migrations/007_planning.sql
--
-- Convention note: the deployed schema uses TEXT ids and company-scoped
-- ownership (see 001_init.sql). New time columns use TIMESTAMPTZ.

CREATE TABLE IF NOT EXISTS plans (
    id           TEXT PRIMARY KEY,
    company_id   TEXT NOT NULL REFERENCES companies(id) ON DELETE CASCADE,
    customer_id  TEXT NOT NULL REFERENCES users(id) ON DELETE RESTRICT,
    title        TEXT NOT NULL,
    region_id    TEXT REFERENCES regions(id),
    status       TEXT NOT NULL DEFAULT 'active'
                 CHECK (status IN ('draft','active','archived')),
    created_at   TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at   TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE INDEX IF NOT EXISTS idx_plans_owner ON plans (company_id, updated_at DESC);

CREATE TABLE IF NOT EXISTS plan_items (
    id                 TEXT PRIMARY KEY,
    plan_id            TEXT NOT NULL REFERENCES plans(id) ON DELETE CASCADE,
    product_id         TEXT NOT NULL REFERENCES products(id) ON DELETE RESTRICT,
    product_name       TEXT NOT NULL,
    unit               TEXT NOT NULL DEFAULT 'pcs',
    quantity           INTEGER NOT NULL CHECK (quantity > 0),
    target_date        DATE,
    note               TEXT NOT NULL DEFAULT '',
    -- Price projection snapshot. Nullable: insufficient data => null, never 0.
    current_unit_price NUMERIC(14,2),
    predicted          NUMERIC(14,2),
    low                NUMERIC(14,2),
    high               NUMERIC(14,2),
    change_pct         NUMERIC(6,2),
    trend_pct          NUMERIC(6,2) NOT NULL DEFAULT 0,
    horizon_days       INTEGER NOT NULL DEFAULT 30,
    estimated_budget   NUMERIC(14,2),
    budget_low         NUMERIC(14,2),
    budget_high        NUMERIC(14,2),
    saving_vs_now      NUMERIC(14,2),
    recommendation     TEXT NOT NULL DEFAULT 'insufficient'
                       CHECK (recommendation IN ('buy_now','wait','monitor','insufficient')),
    reason             TEXT NOT NULL DEFAULT '',
    data_status        TEXT NOT NULL DEFAULT 'insufficient'
                       CHECK (data_status IN ('demo','live','insufficient')),
    basis              TEXT NOT NULL DEFAULT '',
    projection_note    TEXT NOT NULL DEFAULT '',
    generated_at       TIMESTAMPTZ,
    created_at         TIMESTAMPTZ NOT NULL DEFAULT now(),
    UNIQUE (plan_id, product_id)
);
CREATE INDEX IF NOT EXISTS idx_plan_items_plan ON plan_items (plan_id);
