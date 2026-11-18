# BozorPulse

> **Documentation status — read this first.** This README and the documents in [`docs/`](docs/) are the **current source of truth** for BozorPulse after the major product redesign. [`overview.md`](overview.md) is a **legacy brief** kept only for historical context and is **superseded** by the new specification in this repository's `docs/`. Do not use `overview.md` to plan, build, or review features; use [ARCHITECTURE.md](docs/ARCHITECTURE.md), [AI-SPECS.md](docs/AI-SPECS.md), [DATA-MODEL.md](docs/DATA-MODEL.md), and [ROADMAP.md](docs/ROADMAP.md) instead.

BozorPulse is an AI-assisted **B2B marketplace** that connects **suppliers** and **customers**. It is built around two simple ideas: customers should see real market price levels before they buy, and suppliers should have a direct channel to more buyers.

## Contents

- [What BozorPulse does](#what-bozorpulse-does)
- [Product principle](#product-principle)
- [Roles and approval workflow](#roles-and-approval-workflow)
- [Features](#features)
- [Repository layout](#repository-layout)
- [Quickstart](#quickstart)
- [Demo accounts](#demo-accounts)
- [API examples](#api-examples)
- [Environment variables](#environment-variables)
- [Testing](#testing)
- [Documentation](#documentation)

---

## What BozorPulse does

| Side | Problem today | What the platform provides |
| --- | --- | --- |
| **Customer** | Does not know the real market price of a product; cannot compare offers fairly. | Market price visibility (min / median / average with sample size) and a catalog of confirmed suppliers. |
| **Supplier** | Cannot reach enough buyers or show products with transparent pricing. | A direct sales channel: a product catalog, prices and discounts, warehouses, and incoming orders. |
| **Admin** | Needs trust and order in the marketplace. | Approval of new accounts, management of categories, products, regions, and visibility of all orders. |

The **MVP** goal is deliberately narrow: **help both sides find each other and create orders on the platform.**

**Explicitly out of scope for the MVP:** payment processing, delivery tracking, automated purchasing, and any order execution beyond order *creation*. Payments are recorded as data only; order status may be advanced manually by authorized users, but the platform does not move money or goods.

## Product principle

1. **Price transparency before commitment.** A customer can query `/api/v1/market/prices` for a category or search term and see `min`, `median`, `avg`, `count`, and a `data_status` label before placing an order.
2. **Direct supplier channel.** A supplier publishes products with `price`, `discount_percent`, and a derived `effective_price`; customers discover those products and order from them without intermediaries.
3. **Trust by approval.** Every new supplier/customer account starts as `waiting` and is approved or rejected by an admin. Only `confirmed` users can enter their cabinet.
4. **Honest AI.** Market statistics, supplier recommendations, and the assistant are deterministic and grounded in platform data. Demo/sample data is always labeled; nothing fabricated is presented as a real market quote. See [AI-SPECS.md](docs/AI-SPECS.md).

## Roles and approval workflow

There are **exactly three roles**: `admin`, `supplier`, `customer`.

During registration a supplier or customer provides: **username, password, INN, bank account number, company name**, plus optional `region_id` and `phone`. Login uses **username + password only** — there is no email login.

```mermaid
stateDiagram-v2
    [*] --> waiting: POST /auth/register
    waiting --> confirmed: POST /admin/users/ID/approve
    waiting --> rejected: POST /admin/users/ID/reject
    rejected --> waiting: resubmit (admin action)
    confirmed --> [*]: cabinet access granted
    rejected --> [*]: cabinet access denied
```

- A new account has status `waiting` and receives a token from `POST /auth/register`, but the **status guard** blocks cabinet/order/catalog-write access until an admin confirms it.
- **Public endpoints** (no token required): `POST /auth/register`, `POST /auth/login`, and `GET` on products, categories, regions, suppliers, and market prices.
- Everything else requires `Authorization: Bearer <token>`; admin endpoints additionally require `role = admin`.

| Role | Cabinet sections |
| --- | --- |
| **admin** | Dashboard, Suppliers, Customers, Orders, Settings (Product categories, Products, Regions). Approvals happen in the Suppliers / Customers lists. |
| **customer** | Dashboard, Orders, Market, **Planning**, Basket, Suppliers, Payments, Warehouses (**Stock, Transfers, Supplies, Write-off, Returns**), Cabinet. Customers can have multiple warehouses and pick one when ordering. |
| **supplier** | Dashboard, Orders, Payments, Products, Warehouses, Cabinet. Suppliers can have multiple warehouses and manage products with prices and discounts. |

## Features

- **Identity & approval** — registration with company details, username/password login, token sessions, admin approve/reject gate.
- **Catalog** — categories, regions, products, `effective_price` from price + discount, supplier-scoped product management.
- **Market price visibility** — `/market/prices` returns `samples`, `stats {min, median, avg, count}`, and `data_status`.
- **Discovery** — supplier and customer directories, supplier detail pages, deterministic supplier recommendations with reasons.
- **Orders** — create an order for a supplier and a chosen warehouse, list/read orders, manual status changes (`new → confirmed → processing → shipped → delivered`, or `cancelled`).
- **Warehouses & inventory** — multiple warehouses per owner, stock view, supply / transfer / write-off / return movements.
- **Payments (records only)** — create and read payment records tied to orders; no money is moved.
- **Analytics & AI** — role-aware dashboard, honest market-price statistics, deterministic supplier recommendations, and an assistant with a typed tool registry backed by the **Meta Model API** (LLM on by default, with an automatic rule-based fallback when no key is configured). Customers also get a **buying assistant** ("what to buy / where to buy") on the Market page with a ranked shortlist and one-click "add to basket" actions.
- **Planning (purchase timing)** — customers save upcoming product needs and see a conservative 30-day price outlook for each item: current tier price, predicted price with a range, estimated budget, and a deterministic `buy_now` / `wait` / `monitor` signal. Forecasts are estimates (never quotes), demo data is labeled, and plans never place orders.

## Repository layout

```
bozorpulse-ai/
├── backend/                              # Go 1.25 API, module github.com/bozorpulse/bozorpulse-ai/backend
│   ├── cmd/api/                          # entrypoint: cd backend && go run ./cmd/api   (:8080)
│   ├── internal/
│   │   ├── modules/                      # bounded-context modules (DDD layers inside each)
│   │   │   ├── identity/                 # auth, users, roles, approval workflow
│   │   │   ├── companies/                # company profiles, regions, verification
│   │   │   ├── catalog/                  # categories, products, prices, discounts
│   │   │   ├── procurement/              # orders, order items, order status
│   │   │   ├── payments/                 # payment records (no processing)
│   │   │   ├── planning/                 # purchase-timing plans on price forecasts
│   │   │   ├── inventory/                # warehouses, stock, inventory movements
│   │   │   ├── analytics/                # dashboards and aggregate metrics
│   │   │   ├── ai/                       # market stats, recommendations, assistant
│   │   │   └── <each module>/{domain,application,infrastructure,interfaces}/
│   │   ├── platform/                     # shared technical layer
│   │   │   ├── auth/                     # token issuing/validation, password hashing
│   │   │   ├── config/                   # environment configuration
│   │   │   ├── http/                     # router, middleware, response/error helpers
│   │   │   └── postgres/                 # connection pool and transaction helpers
│   │   └── shared/                       # shared kernel: IDs, money, errors, enums
│   ├── migrations/001_init.sql           # initial PostgreSQL schema
│   ├── seeds/seed.sql                    # demo data (regions, categories, users, products, samples)
│   └── go.mod                            # go 1.25
├── platform/                             # Vue 3 + Vite + Pinia customer/supplier/admin SPA
│   └── src/{api,assets,components,layouts,router,stores,views}/
├── website/                              # Nuxt 3 marketing website
│   └── {assets,components,pages}/
├── prototype/                            # static visual prototype (vendored Vue global build)
│   └── vendor/vue.global.prod.js
├── docs/
│   ├── ARCHITECTURE.md
│   ├── AI-SPECS.md
│   ├── DATA-MODEL.md
│   └── ROADMAP.md
├── overview.md                           # LEGACY brief — superseded by docs/ (see note above)
└── docker-compose.yml                    # postgres + api + platform + website + prototype
```

Backend routing uses the **Go standard library** (`net/http` method/path patterns, Go 1.22+), for example `mux.HandleFunc("GET /api/v1/products/{id}", handler)`. There is **no third-party web framework and no Redis** in the MVP.

## Quickstart

Requirements: **Go 1.25**, **PostgreSQL 16**, **Node.js 20+**, and **Docker + Docker Compose** (for the easiest path).

### Everything with Docker Compose (recommended)

```bash
docker compose up --build
```

This starts the full stack:

| Service | URL | Notes |
| --- | --- | --- |
| API | http://localhost:8080 | health: `GET /health` → `{"status":"ok","service":"bozorpulse"}` |
| Platform (Vue 3) | http://localhost:5173 | operator and customer/supplier cabinets |
| Website (Nuxt 3) | http://localhost:3000 | public marketing site |
| Prototype (static) | http://localhost:4173 | static visual prototype |
| PostgreSQL 16 | localhost:55433 | schema + demo seed applied on first init from `backend/migrations/001_init.sql` and `backend/seeds/seed.sql` |

Stop with `docker compose down` (add `-v` to drop the database volume).

### Backend API only

Bring up just the database, then run the API against the local connection string:

```bash
docker compose up -d postgres          # applies migration + seed on first start

cd backend
DATABASE_URL='postgres://bozorpulse:bozorpulse@localhost:55433/bozorpulse?sslmode=disable' \
APP_ENV=development \
BP_ADDR=:8080 \
go run ./cmd/api
# API on http://localhost:8080, base path /api/v1
```

If you manage PostgreSQL yourself, apply `backend/migrations/001_init.sql` and then `backend/seeds/seed.sql` before starting the API.

### Platform (Vue 3 + Vite + Pinia)

```bash
cd platform
npm install
npm run dev        # http://localhost:5173
npm run build      # production bundle in dist/
```

### Website (Nuxt 3)

```bash
cd website
npm install
npm run dev        # http://localhost:3000
npm run build      # production build
```

## Demo accounts

The seeds (`backend/seeds/seed.sql` + `backend/seeds/demo_expansion.sql`) create the demo accounts below. Passwords are for local/demo use only.

| Username | Password | Role | Status |
| --- | --- | --- | --- |
| `admin` | `admin123` | admin | confirmed |
| `customer1` … `customer4` | same as the username (e.g. `customer1`) | customer | confirmed |
| `supplier1` … `supplier10` | same as the username (e.g. `supplier1`) | supplier | confirmed |

The expansion seed adds **120 products** (12 per supplier) and **four months** of clearly labeled demo price observations (`is_demo = true`, Aug–Nov 2026), so the Market page, `/market/prices` statistics and the AI buying assistant have realistic data on a fresh database.

Any newly registered account starts as `waiting` and needs admin approval before its cabinet works.

> The root `.env` includes a shared demo `MODEL_API_KEY` so `docker compose up --build` works with the LLM enabled out of the box. Treat that key as public: rotate it before deploying or publishing this repository.

## API examples

All endpoints are under `http://localhost:8080/api/v1`, use JSON, and (except public endpoints) require `Authorization: Bearer <token>`.

```bash
BASE=http://localhost:8080/api/v1
```

### Register (public)

```bash
curl -sS -X POST "$BASE/auth/register" \
  -H 'Content-Type: application/json' \
  -d '{
        "username": "supplier2",
        "password": "sup123",
        "role": "supplier",
        "inn": "770700000111",
        "bank_account": "KZ123456789012345678",
        "company_name": "New Supply LLP",
        "region_id": 1,
        "phone": "+7 700 000 00 00"
      }'
# -> { "user": {..., "status": "waiting", ...}, "token": "..." }
```

### Login (public)

```bash
ADMIN_TOKEN=$(curl -sS -X POST "$BASE/auth/login" \
  -H 'Content-Type: application/json' \
  -d '{"username":"admin","password":"admin123"}' | jq -r .token)

SUPPLIER_TOKEN=$(curl -sS -X POST "$BASE/auth/login" \
  -H 'Content-Type: application/json' \
  -d '{"username":"supplier1","password":"sup123"}' | jq -r .token)

CUSTOMER_TOKEN=$(curl -sS -X POST "$BASE/auth/login" \
  -H 'Content-Type: application/json' \
  -d '{"username":"customer1","password":"cust123"}' | jq -r .token)
```

### Approve a waiting user (admin)

```bash
curl -sS -X POST "$BASE/admin/users/4/approve" \
  -H "Authorization: Bearer $ADMIN_TOKEN"
# -> User with status "confirmed"

# Filter the lists: GET "$BASE/admin/users?role=supplier&status=waiting"
```

### Create a product (confirmed supplier)

```bash
curl -sS -X POST "$BASE/products" \
  -H "Authorization: Bearer $SUPPLIER_TOKEN" \
  -H 'Content-Type: application/json' \
  -d '{
        "category_id": 1,
        "name": "Portland cement M400",
        "sku": "CEM-M400-50",
        "description": "50 kg bag",
        "unit": "bag",
        "price": 3200,
        "discount_percent": 5,
        "stock": 500,
        "region_id": 1
      }'
# -> Product (effective_price is derived: price * (1 - discount_percent/100))
```

### Get market prices (public)

```bash
curl -sS "$BASE/market/prices?category_id=1&q=cement"
# -> {
#      "samples": [ ... ],
#      "stats": { "min": 2900, "median": 3100, "avg": 3125, "count": 12 },
#      "data_status": "demo"
#    }
```

### Create an order (confirmed customer)

```bash
curl -sS -X POST "$BASE/orders" \
  -H "Authorization: Bearer $CUSTOMER_TOKEN" \
  -H 'Content-Type: application/json' \
  -d '{
        "supplier_id": 1,
        "warehouse_id": 1,
        "items": [
          {"product_id": 1, "quantity": 10}
        ]
      }'
# -> Order with status "new" and computed total_amount
# Advance manually: PATCH "$BASE/orders/{id}/status"  {"status":"confirmed"}
```

### Plan upcoming purchases (confirmed customer)

```bash
curl -sS -X POST "$BASE/plans" \
  -H "Authorization: Bearer $CUSTOMER_TOKEN" \
  -H 'Content-Type: application/json' \
  -d '{
        "title": "Q4 restock",
        "region_id": "reg-01",
        "items": [{"product_id": "prd-001", "quantity": 50, "target_date": "2026-12-05"}]
      }'
# -> Plan with per-item projection: current_unit_price, predicted (+ low/high),
#    estimated_budget, recommendation (buy_now|wait|monitor|insufficient),
#    data_status and disclaimer. Estimates only — no order is created.
# Other routes: GET /plans, GET|PATCH|DELETE /plans/{id},
#               POST /plans/{id}/refresh, POST /plans/preview
```

## Environment variables

Configuration is read from the environment (the API container defaults are in `docker-compose.yml`).

| Variable | Required | Default | Description |
| --- | --- | --- | --- |
| `DATABASE_URL` | yes | — | PostgreSQL DSN, e.g. `postgres://bozorpulse:bozorpulse@localhost:55433/bozorpulse?sslmode=disable`. |
| `BP_ADDR` | no | `:8080` | Address the HTTP server listens on. |
| `APP_ENV` | no | `development` | `development` \| `staging` \| `production`. Controls logging and error detail. |
| `JWT_SECRET` | yes in production | dev-only fallback | Secret used to sign bearer tokens. Must be set to the same value across API replicas. |
| `TOKEN_TTL` | no | `24h` | Lifetime of issued tokens (Go duration string). |
| `CORS_ORIGIN` | no | platform + website dev URLs | Comma-separated allowed origins for browser clients. |
| `LOG_LEVEL` | no | `info` | `debug` \| `info` \| `warn` \| `error`. |
| `AI_LLM_ENABLED` | no | `true` | Enables the LLM-backed assistant. Set `false` to force the rule-based assistant (no network calls). |
| `MODEL_API_KEY` | yes for LLM | — | Meta Model API key (https://dev.meta.ai). Aliases: `LLM_API_KEY`, `META_API_KEY`. Without a key the app falls back to rule-based mode. |
| `LLM_API_URL` | no | `https://api.meta.ai/v1` | OpenAI-compatible Chat Completions base URL. Override to use a different provider/gateway. |
| `LLM_MODEL` | no | `muse-spark-1.3` | Model ID sent to the provider. |
| `LLM_MAX_TOKENS` | no | `1600` | Max tokens per completion. `muse-spark-1.3` is a reasoning model, so leave room for reasoning tokens. |
| `LLM_TIMEOUT` | no | `15s` | Per-request timeout for the model call. |
| `AI_MAX_TOOL_ROUNDS` | no | `5` | Maximum tool-call rounds per chat request. |

> **No Redis in the MVP.** Market statistics and recommendations are computed from PostgreSQL data on demand; there is no cache dependency to configure. `REDIS_URL` is ignored by the new stack.

## Testing

```bash
# Backend: all Go packages
cd backend
go test ./...
go test ./... -race          # race detector
go vet ./...

# Platform
cd platform
npm run build                # type/template check via Vite build

# Website
cd website
npm run build

# Full stack smoke test
docker compose up --build -d
curl -sS http://localhost:8080/health
```

## Documentation

- [ARCHITECTURE.md](docs/ARCHITECTURE.md) — goals, context/module diagrams, DDD layer rules, request flow, authorization, error handling, deployment, scaling, ADRs.
- [AI-SPECS.md](docs/AI-SPECS.md) — market price statistics, deterministic recommendation formula, assistant intents/tools, demo labeling, evaluation, LLM and multilingual roadmaps, privacy.
- [DATA-MODEL.md](docs/DATA-MODEL.md) — full PostgreSQL schema, indexes, relationships, ER diagram, status enums.
- [ROADMAP.md](docs/ROADMAP.md) — phases 0–3, epics, acceptance criteria, risks, KPIs, effort, and MVP out-of-scope list.
# bozorpulse
# bozorpulse
