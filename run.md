# BozorPulse — Run & Seed Guide

> Read this first if you need to start the project or understand the demo data.
> Product rules: `docs/AI-SPECS.md`, `docs/ROADMAP.md`, `README.md`. Implementation handoff: `plans/ai-integration-plan.md`.
> `overview.md` is a legacy brief — do NOT use it for decisions.

---

## 1. Prerequisites

- Docker + Docker Compose v2
- Go 1.25 (only for running the API outside Docker)
- Node.js 20+ (only for running frontends outside Docker)
- PostgreSQL 16 is provided by Compose

The root `.env` is committed for the demo and already contains a working `MODEL_API_KEY` for the Meta Model API, so the LLM assistant works out of the box. **That key is shared and public — rotate it before any real deployment.**

---

## 2. Fastest start (recommended)

From the repository root:

```bash
docker compose up --build
```

This starts all five services:

| Service | URL | Notes |
| --- | --- | --- |
| Platform (Vue 3) | http://localhost:3000 | Customer / supplier / admin cabinets |
| Website (Nuxt 3) | http://localhost:3001 | Public marketing site |
| API | http://localhost:8080 | `GET /health` → `{"status":"ok",...}` |
| Prototype | http://localhost:4173 | Static visual prototype |
| PostgreSQL 16 | localhost:55433 | user/pass/db: `bozorpulse` / `bozorpulse` / `bozorpulse` |

API base path: `http://localhost:8080/api/v1`. The platform proxies `/api` to the API on port 8080 with a 120 s timeout (LLM calls can take ~20–30 s).

Stop with `docker compose down`. Add `-v` to **drop the database volume and re-run all migrations/seeds from scratch**.

### Fresh database vs existing volume

PostgreSQL init scripts (`/docker-entrypoint-initdb.d/*.sql`) run **only on an empty data volume**. If you already have a `pgdata` volume, new migration/seed files are ignored. To rebuild demo data:

```bash
docker compose down -v
docker compose up --build
```

If you must keep the existing volume, apply the new files manually (see §5).

---

## 3. Demo accounts

Login = password for all demo accounts. All are `confirmed`.

| Username | Password | Role |
| --- | --- | --- |
| `admin` | `admin123` | admin |
| `customer1` … `customer4` | same as username | customer |
| `supplier1` … `supplier10` | same as username | supplier |

Newly registered accounts still start as `waiting` and need admin approval.

Quick check:

```bash
BASE=http://localhost:8080/api/v1
curl -s -X POST "$BASE/auth/login" -H 'Content-Type: application/json' \
  -d '{"username":"customer1","password":"customer1"}'
```

---

## 4. Seed files (what creates what)

Applied in filename order on a fresh volume:

| Mount name | Source file | Purpose |
| --- | --- | --- |
| `001_init.sql` | `backend/migrations/001_init.sql` | Base schema: users, sessions, companies, regions, categories, products, warehouses, orders, payments, stock, price_samples, recommendations |
| `002_pricing.sql` | `backend/migrations/002_product_pricing.sql` | Product media, order limits, delivery windows, `product_warehouse_pricing` |
| `003_delivery.sql` | `backend/migrations/003_delivery_zones.sql` | Supplier `delivery_zones` per region |
| `004_ai.sql` | `backend/migrations/004_ai_conversation.sql` | `ai_conversations`, `ai_messages` for assistant history |
| `005_seed.sql` | `backend/seeds/seed.sql` | Base demo: 1 admin, supplier1/2, customer1/2, 12 products, one order/payment, 12 months of demo price samples, categories, regions |
| `006_demo_expansion.sql` | `backend/seeds/demo_expansion.sql` | Expanded demo dataset (see below) |
| `007_planning.sql` | `backend/migrations/007_planning.sql` | Purchase-timing plans: `plans`, `plan_items` |
| `008_planning_seed.sql` | `backend/seeds/008_planning_seed.sql` | One demo plan for `customer1` with 2 demo-labeled items |

### `demo_expansion.sql` in detail

- Aligns `customer1`, `customer2`, `supplier1`, `supplier2` to the **login = password** convention and confirms them.
- Inserts `customer3`, `customer4` and `supplier3` … `supplier10` (all confirmed), each with a company and a warehouse.
- Adds **120 products** (`prd-demo-001` … `prd-demo-120`), 12 per supplier, spread across the 6 existing categories. Total products after both seeds: **132**.
- Adds **240 pricing tiers** (`pwp-demo-*`): a default tier (min qty 1) and a supplier-warehouse tier (min qty 50) per product.
- Adds **stock items** for the supplier warehouses.
- Adds **480 price samples** (`ps-demo-*`): four monthly observations per product for **Aug, Sep, Oct, Nov 2026**, all `is_demo = true`. Total samples after both seeds: **624**.

All IDs are deterministic and every insert uses `ON CONFLICT (id) DO NOTHING` (users use `DO UPDATE` for password/status), so the seed is safe to re-run.

Verify seed contents:

```bash
docker exec bozorpulse-ai-postgres-1 psql -U bozorpulse -d bozorpulse \
  -c "SELECT role,status,count(*) FROM users GROUP BY 1,2;" \
  -c "SELECT count(*) AS products FROM products;" \
  -c "SELECT count(*) AS samples, min(observed_at), max(observed_at) FROM price_samples;"
```

Expected: 10 confirmed suppliers, ≥4 confirmed customers, 132 products, 624 samples with `observed_at` through `2026-11-01`.

---

## 5. Applying new migrations/seeds to an existing volume

```bash
# AI conversation tables
docker exec -i bozorpulse-ai-postgres-1 psql -U bozorpulse -d bozorpulse \
  < backend/migrations/004_ai_conversation.sql

# Expanded demo data
docker exec -i bozorpulse-ai-postgres-1 psql -U bozorpulse -d bozorpulse \
  < backend/seeds/demo_expansion.sql

# Planning module
docker exec -i bozorpulse-ai-postgres-1 psql -U bozorpulse -d bozorpulse \
  < backend/migrations/007_planning.sql
docker exec -i bozorpulse-ai-postgres-1 psql -U bozorpulse -d bozorpulse \
  < backend/seeds/008_planning_seed.sql
```

---

## 6. Environment variables

Read from the environment by `backend/internal/platform/config/config.go`. Root `.env` feeds Docker Compose; `backend/.env` is a copy for direct `go run`.

| Variable | Default | Meaning |
| --- | --- | --- |
| `DATABASE_URL` | — | Postgres DSN (set by Compose) |
| `BP_ADDR` | `:8080` | API listen address |
| `APP_ENV` | `development` | Logging/error detail |
| `AI_LLM_ENABLED` | `true` | Enable LLM-backed assistant |
| `MODEL_API_KEY` | — | Meta Model API key (aliases: `LLM_API_KEY`, `META_API_KEY`) |
| `LLM_API_URL` | `https://api.meta.ai/v1` | OpenAI-compatible Chat Completions base URL |
| `LLM_MODEL` | `muse-spark-1.3` | Model ID (reasoning model — keep `LLM_MAX_TOKENS` high) |
| `LLM_MAX_TOKENS` | `1600` | Max completion tokens |
| `LLM_TIMEOUT` | `45s` | Per-call timeout |
| `AI_MAX_TOOL_ROUNDS` | `5` | Max tool-call rounds per chat |

If no key is set (or `AI_LLM_ENABLED=false`), the API logs a warning and uses the deterministic rule-based assistant — no network egress.

---

## 7. Running parts individually

```bash
# API only (uses existing Postgres on 55433)
docker compose up -d postgres
cd backend
set -a && . ../.env && set +a      # loads MODEL_API_KEY etc. into this shell
go run ./cmd/api                   # http://localhost:8080

# Platform
cd platform && npm install && npm run dev     # http://localhost:3000

# Website
cd website && npm install && npm run dev      # http://localhost:3001 (Nuxt default)
```

---

## 8. AI features — quick API checks

All `/ai/*` endpoints require `Authorization: Bearer <token>` and a confirmed account. `/market/prices` is public.

```bash
BASE=http://localhost:8080/api/v1
TOKEN=$(curl -s -X POST "$BASE/auth/login" -H 'Content-Type: application/json' \
  -d '{"username":"customer1","password":"customer1"}' | python3 -c 'import sys,json;print(json.load(sys.stdin)["token"])')

# Honest market stats (demo labeled, min/median/avg/count)
curl -s "$BASE/market/prices"

# Price history of one product (public)
curl -s "$BASE/products/prd-012/price-history"

# General assistant (LLM chooses tools from the allowlist)
curl -s -X POST "$BASE/ai/chat" -H "Authorization: Bearer $TOKEN" \
  -H 'Content-Type: application/json' \
  -d '{"message":"How much is Portland Cement?","locale":"en"}'

# Customer buying assistant: what to buy / where to buy
curl -s -X POST "$BASE/ai/chat" -H "Authorization: Bearer $TOKEN" \
  -H 'Content-Type: application/json' \
  -d '{"message":"I need 100 kg of cement, what should I buy?","mode":"shopping","locale":"en"}'
# Reuse returned "conversation_id" on the next call; ask "and the cheaper one?"

# Deterministic recommendations
curl -s "$BASE/ai/recommendations?product_id=prd-012&limit=5" -H "Authorization: Bearer $TOKEN"

# Conversation history
curl -s "$BASE/ai/conversations" -H "Authorization: Bearer $TOKEN"

# Purchase-timing plans (customer only)
curl -s "$BASE/plans" -H "Authorization: Bearer $TOKEN"
curl -s -X POST "$BASE/plans" -H "Authorization: Bearer $TOKEN" \
  -H 'Content-Type: application/json' \
  -d '{"title":"Q4 restock","items":[{"product_id":"prd-001","quantity":50,"target_date":"2026-12-05"}]}'
# Also: GET/PATCH/DELETE /plans/{id}, POST /plans/{id}/refresh, POST /plans/preview
```

Allowlisted tools: `market_stats`, `recommend_suppliers`, `search_products`, `get_order`, `compare_offers`. There is no SQL/HTTP tool and the assistant never creates orders.

UI: open http://localhost:3000 → log in as `customer1 / customer1` → **Market → Ask AI what to buy**. The floating chat bubble opens the assistant from any cabinet.

---

## 9. Tests / verification

```bash
cd backend && go test ./... && go vet ./...
cd platform && npm run build
docker compose config --quiet
curl -s http://localhost:8080/health
```

Go test files cover tool validation, provider wire format, PII sanitization, groundedness, EN/RU/UZ intents, the buying flow and recommendation determinism. Eval prompts live at
`backend/internal/modules/ai/testdata/eval_prompts.json`.

---

## 10. Troubleshooting

| Symptom | Cause / fix |
| --- | --- |
| `unknown field "locale"` on `/ai/chat` | An old API binary is running on port 8080. Rebuild/restart it (`go run ./cmd/api` or `docker compose up --build api`). |
| `no such table: ai_conversations` / missing tables | Init scripts don't rerun on an existing volume. Apply §5, or `docker compose down -v` then up. |
| Assistant replies are generic templates | Either no `MODEL_API_KEY` (rule-based fallback) or groundedness rejected the reply. Check the API startup log for `Meta Model API enabled`. |
| `HTTP 500` from the platform proxy on chat | Vite proxy timeout — it is set to 120 s in `platform/vite.config.js`; restart the dev server after editing. |
| `data_status: insufficient` from `/market/prices` | With < 5 deduplicated samples the stats are intentionally `null` — no invented numbers. Run the expansion seed to get 132 samples. |
| Login fails for `customer3` / `supplier7` | Apply `demo_expansion.sql` (§5) or recreate the volume. |
