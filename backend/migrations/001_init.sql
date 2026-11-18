-- BozorPulse initial schema (modular monolith / PostgreSQL 16).
-- Apply: psql "$DATABASE_URL" -f backend/migrations/001_init.sql

CREATE TABLE IF NOT EXISTS regions (
    id TEXT PRIMARY KEY,
    name TEXT NOT NULL UNIQUE,
    created_at TEXT NOT NULL
);

CREATE TABLE IF NOT EXISTS users (
    id TEXT PRIMARY KEY,
    username TEXT NOT NULL UNIQUE,
    password_hash TEXT NOT NULL,
    role TEXT NOT NULL CHECK (role IN ('admin','supplier','customer')),
    status TEXT NOT NULL DEFAULT 'waiting' CHECK (status IN ('waiting','confirmed','rejected')),
    inn TEXT NOT NULL DEFAULT '',
    bank_account TEXT NOT NULL DEFAULT '',
    company_name TEXT NOT NULL DEFAULT '',
    region_id TEXT REFERENCES regions(id),
    phone TEXT NOT NULL DEFAULT '',
    created_at TEXT NOT NULL
);
CREATE INDEX IF NOT EXISTS idx_users_role_status ON users (role, status);

CREATE TABLE IF NOT EXISTS sessions (
    token TEXT PRIMARY KEY,
    user_id TEXT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    expires_at TIMESTAMPTZ NOT NULL
);

CREATE TABLE IF NOT EXISTS companies (
    id TEXT PRIMARY KEY,
    owner_user_id TEXT NOT NULL UNIQUE REFERENCES users(id) ON DELETE CASCADE,
    name TEXT NOT NULL,
    inn TEXT NOT NULL DEFAULT '',
    bank_account TEXT NOT NULL DEFAULT '',
    region_id TEXT REFERENCES regions(id),
    created_at TEXT NOT NULL
);

CREATE TABLE IF NOT EXISTS categories (
    id TEXT PRIMARY KEY,
    name TEXT NOT NULL,
    parent_id TEXT REFERENCES categories(id),
    created_at TEXT NOT NULL
);

CREATE TABLE IF NOT EXISTS products (
    id TEXT PRIMARY KEY,
    supplier_id TEXT NOT NULL REFERENCES companies(id) ON DELETE CASCADE,
    category_id TEXT REFERENCES categories(id),
    name TEXT NOT NULL,
    sku TEXT NOT NULL DEFAULT '',
    description TEXT NOT NULL DEFAULT '',
    unit TEXT NOT NULL DEFAULT 'pcs',
    price NUMERIC(14,2) NOT NULL CHECK (price >= 0),
    discount_percent NUMERIC(5,2) NOT NULL DEFAULT 0 CHECK (discount_percent BETWEEN 0 AND 90),
    effective_price NUMERIC(14,2) NOT NULL,
    stock INTEGER NOT NULL DEFAULT 0,
    region_id TEXT REFERENCES regions(id),
    created_at TEXT NOT NULL,
    updated_at TEXT NOT NULL
);
CREATE INDEX IF NOT EXISTS idx_products_supplier ON products (supplier_id);
CREATE INDEX IF NOT EXISTS idx_products_category ON products (category_id);

CREATE TABLE IF NOT EXISTS warehouses (
    id TEXT PRIMARY KEY,
    owner_id TEXT NOT NULL REFERENCES companies(id) ON DELETE CASCADE,
    owner_role TEXT NOT NULL CHECK (owner_role IN ('supplier','customer','admin')),
    name TEXT NOT NULL,
    address TEXT NOT NULL DEFAULT '',
    city TEXT NOT NULL DEFAULT '',
    region_id TEXT REFERENCES regions(id),
    latitude NUMERIC(10,6) NOT NULL DEFAULT 0,
    longitude NUMERIC(10,6) NOT NULL DEFAULT 0,
    created_at TEXT NOT NULL
);
CREATE INDEX IF NOT EXISTS idx_warehouses_owner ON warehouses (owner_id);

CREATE TABLE IF NOT EXISTS orders (
    id TEXT PRIMARY KEY,
    customer_id TEXT NOT NULL REFERENCES companies(id),
    supplier_id TEXT NOT NULL REFERENCES companies(id),
    warehouse_id TEXT REFERENCES warehouses(id),
    status TEXT NOT NULL DEFAULT 'new' CHECK (status IN ('new','confirmed','processing','shipped','delivered','cancelled')),
    total_amount NUMERIC(14,2) NOT NULL DEFAULT 0,
    items JSONB NOT NULL DEFAULT '[]',
    created_at TEXT NOT NULL,
    updated_at TEXT NOT NULL
);
CREATE INDEX IF NOT EXISTS idx_orders_customer ON orders (customer_id);
CREATE INDEX IF NOT EXISTS idx_orders_supplier ON orders (supplier_id);

CREATE TABLE IF NOT EXISTS order_items (
    id TEXT PRIMARY KEY,
    order_id TEXT NOT NULL REFERENCES orders(id) ON DELETE CASCADE,
    product_id TEXT NOT NULL REFERENCES products(id),
    quantity INTEGER NOT NULL,
    unit_price NUMERIC(14,2) NOT NULL,
    line_total NUMERIC(14,2) NOT NULL
);

CREATE TABLE IF NOT EXISTS payments (
    id TEXT PRIMARY KEY,
    order_id TEXT NOT NULL REFERENCES orders(id) ON DELETE CASCADE,
    amount NUMERIC(14,2) NOT NULL,
    method TEXT NOT NULL DEFAULT 'bank_transfer',
    bank_account TEXT NOT NULL DEFAULT '',
    status TEXT NOT NULL DEFAULT 'pending' CHECK (status IN ('pending','paid','failed')),
    created_at TEXT NOT NULL
);

CREATE TABLE IF NOT EXISTS stock_items (
    warehouse_id TEXT NOT NULL REFERENCES warehouses(id) ON DELETE CASCADE,
    product_id TEXT NOT NULL REFERENCES products(id) ON DELETE CASCADE,
    quantity INTEGER NOT NULL DEFAULT 0,
    updated_at TEXT NOT NULL,
    PRIMARY KEY (warehouse_id, product_id)
);

CREATE TABLE IF NOT EXISTS inventory_movements (
    id TEXT PRIMARY KEY,
    warehouse_id TEXT NOT NULL REFERENCES warehouses(id) ON DELETE CASCADE,
    product_id TEXT NOT NULL REFERENCES products(id),
    quantity INTEGER NOT NULL,
    type TEXT NOT NULL CHECK (type IN ('supply','transfer_in','transfer_out','write_off','return')),
    note TEXT NOT NULL DEFAULT '',
    created_at TEXT NOT NULL
);
CREATE INDEX IF NOT EXISTS idx_movements_warehouse ON inventory_movements (warehouse_id);

CREATE TABLE IF NOT EXISTS price_samples (
    id TEXT PRIMARY KEY,
    product_id TEXT NOT NULL REFERENCES products(id) ON DELETE CASCADE,
    supplier_id TEXT NOT NULL REFERENCES companies(id),
    region_id TEXT REFERENCES regions(id),
    price NUMERIC(14,2) NOT NULL,
    source_type TEXT NOT NULL DEFAULT 'supplier_offer',
    is_demo BOOLEAN NOT NULL DEFAULT true,
    observed_at TEXT NOT NULL
);

CREATE TABLE IF NOT EXISTS recommendations (
    id TEXT PRIMARY KEY,
    product_id TEXT NOT NULL REFERENCES products(id) ON DELETE CASCADE,
    supplier_id TEXT NOT NULL REFERENCES companies(id),
    score NUMERIC(5,2) NOT NULL,
    reasons JSONB NOT NULL DEFAULT '[]',
    created_at TEXT NOT NULL
);
