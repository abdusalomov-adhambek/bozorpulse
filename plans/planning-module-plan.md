# BozorPulse — Planning Module Implementation Plan (v1, implementation-ready)

> Status: PLAN ONLY. No source files were modified. This is the handoff document for the implementing agent.
> Product rules source of truth: `README.md`, `docs/AI-SPECS.md`, `docs/ROADMAP.md`, `docs/ARCHITECTURE.md`, `run.md`. The legacy `overview.md` brief is superseded and must NOT drive implementation.
> This plan was reviewed against the repository; the "Review deltas" section records fixes for issues found during review.

---

## 0. Decisions & assumptions (locked for the next agent)

| # | Decision | Detail |
| --- | --- | --- |
| D1 | **Scope = purchase-timing planning (confirmed with product owner)** | Customers are the buying side. They record upcoming product needs and see a predicted price, a conservative "buy now / wait / monitor" signal, and a budget estimate. Resale price / margin planning is explicitly NOT in scope. |
| D2 | **New bounded context `planning`** | Implement `backend/internal/modules/planning/{domain,application,infrastructure,interfaces}/`. Do not fold planning into `ai`; the user asked for a new module and the DDD boundary keeps plan ownership/arithmetic separate from forecasting. |
| D3 | **Reuse `forecast.Service`; do not build a second forecaster** | The existing `backend/internal/modules/ai/infrastructure/forecast/` already produces a conservative 30-day `PriceForecast` from `price_samples` + weather, and `domain.PriceForecast` is its output type. Inject the already-constructed instance from `cmd/api`. |
| D4 | **Extend the forecast output with a relative-change signal (review delta #1)** | The forecast's absolute `predicted` is derived from the product's base price, but customers pay **quantity-tier prices** (`product_warehouse_pricing`). Add `base_price` and `change_pct` to the forecast result so planning can apply the *relative* change to the best tier price for the requested quantity. Add tests for the new fields. |
| D5 | **Tier-aware "current price" (review delta #1)** | The planning adapter resolves the best applicable tier via the catalog service (`GET /products/{id}/pricing` equivalent). Budget = `quantity × tier_price × (1 + change_pct/100)`, with range bounds from the forecast's low/high ratio. Never multiply the raw forecast value by quantity. |
| D6 | **Nullable projections; insufficient ⇒ nulls (review delta #3)** | With < 3 samples `Forecast` returns `Predicted/Low/High = 0` with `DataStatusInsufficient`. Projection number fields MUST be pointers/nullable in the DTO and `null` in JSON when insufficient — never a fabricated `0.00`. UI renders "—". |
| D7 | **Horizon validation (review delta #2)** | Forecast horizon is fixed at 30 days. `target_date` is accepted only within `now … now+30d`; outside that window the projection is marked `insufficient` with a note "outside 30-day forecast range". |
| D8 | **One region + bounded fan-out + overall timeout (review delta #4)** | All items in a plan use one region (plan region, default = customer's company region). Forecast items concurrently with a small limit and a single ~10 s context timeout. Allow injecting a nil weather provider so planning can run fully offline/deterministically if configured. |
| D9 | **Dirty-check on update (review delta #5)** | `PATCH` recomputes projections only if `product_id`, `quantity`, `region_id`, or `target_date` changed; title/status-only edits keep the stored snapshot. |
| D10 | **Ownership = company, with fallback (review delta #7)** | `auth.Principal.CompanyID` comes from a LEFT JOIN and may be `""`. Use `company_id` when present, else fall back to `p.ID`. Never accept an owner id from the request body. |
| D11 | **Schema follows the deployed convention (review delta #6)** | Actual migrations use TEXT primary keys, company-scoped ownership, and mostly TEXT timestamps (not the BIGINT/user model in `DATA-MODEL.md`). New tables use TEXT ids, `companies` FKs, and `TIMESTAMPTZ` for new time columns. Add an explicit authority note to `DATA-MODEL.md`; put the detailed schema in `run.md`. |
| D12 | **Guardrails unchanged** | No order placement, no guaranteed prices, no LLM in the planning math, demo data visibly labeled. Backend owns all arithmetic. |

---

## 1. Current state (review findings)

- `backend/internal/modules/ai/infrastructure/forecast/forecast.go` exposes `Service.Forecast(ctx, productID, regionID) (domain.PriceForecast, error)`. It reads `products.effective_price` as `current`, the 24 most recent `price_samples` (`ORDER BY observed_at DESC`), blends median + current (0.8/0.2), clamps trend to ±10%, and nudges by weather (category sensitivity: `cat-03`/`cat-06` = 1.0, `cat-04` = 0.6, else 0.2). With 1–2 samples it returns an insufficient result with zeroed numbers and a basis string.
- `domain.PriceForecast` (`backend/internal/modules/ai/domain/ai.go`) currently carries `predicted`, `low`, `high`, `trend_pct`, `weather`, `samples`, `horizon_days`, `basis`, `data_status` — no base/relative-change fields (D4).
- The `price_forecast` tool is registered in `cmd/api/main.go` and the buying assistant already appends a "Price outlook" line in `buyingReply` (`ai/application/service.go`).
- Pricing tiers exist in `product_warehouse_pricing` (`min_qty`, `max_qty`, `unit_price`, `discount_percent`, `delivery_min_days`, `delivery_max_days`, `min_order`, `max_order`); `GET /api/v1/products/{id}/pricing` returns `{data:[...], total:N}` and `Market.vue` uses it to compute a "from …" best price. This must feed planning budgets (D5).
- Region ids are `reg-01` … `reg-05`, category ids `cat-01` … `cat-06` (see `backend/seeds/seed.sql`); the forecast/OpenMeteo adapters already use these exact ids.
- `cmd/api/main.go` already builds `forecaster := forecast.New(db, forecast.NewOpenMeteo(nil))` for the AI module.
- **Missing today:** no plan entity, no plan repository/table, no `/plans` API, no planning screen, no plan i18n keys/nav entry.

---

## 2. Target architecture

```
Vue customer/Planning.vue
        |  GET/POST/PATCH/DELETE /api/v1/plans…  (Bearer, role=customer, confirmed)
        v
planning/interfaces/http.go
        v
planning/application/service.go
   +-- ownership + validation (company id, qty>0, horizon)
   +-- tier price via catalog application interface (ProductReader / PricingReader)
   +-- projection rules (pure domain)
   +-- bounded parallel fan-out to Forecaster (overall ctx timeout)
        v
planning/infrastructure
   +-- repository.go        (plans / plan_items, Postgres)
   +-- forecast_adapter.go  (maps ai forecast.Service output -> planning DTO, adds base/change)
   +-- pricing_adapter.go   (maps catalog Service pricing -> best tier for qty)
        v
wired in cmd/api/main.go using the existing db + forecaster + catalogSvc
```

Hard rules:
1. `domain/` imports no `net/http`, `database/sql`, `internal/platform/postgres`, or config.
2. Cross-module calls go through application interfaces injected at the composition root; only narrow adapter files translate foreign DTOs.
3. All arithmetic, eligibility and ownership live in Go/SQL — the LLM is never involved in a plan.
4. Every projection carries `data_status` + `disclaimer`; "live" only when all contributing samples are `is_demo = false`.
5. All owner ids derive from `auth.FromContext(ctx)`.

---

## 3. Domain model (`backend/internal/modules/planning/domain/plan.go`)

```go
type PlanStatus string
const ( PlanDraft PlanStatus = "draft"; PlanActive = "active"; PlanArchived = "archived" )

type Recommendation string
const (
    RecBuyNow      Recommendation = "buy_now"
    RecWait        Recommendation = "wait"
    RecMonitor     Recommendation = "monitor"
    RecInsufficient Recommendation = "insufficient"
)

type Plan struct {
    ID, CompanyID, CustomerID, Title, RegionID string
    Status PlanStatus
    Items []PlanItem
    CreatedAt, UpdatedAt string // RFC3339
}

type PlanItem struct {
    ID, ProductID, ProductName, Unit, Note string
    Quantity int
    TargetDate string            // optional, date only (YYYY-MM-DD)
    Projection PriceProjection   // snapshot, recomputed on demand
}

// All fields nullable in JSON: insufficient forecasts must render null, never 0.
type PriceProjection struct {
    CurrentUnitPrice *float64 `json:"current_unit_price"` // best tier price for qty
    Predicted        *float64 `json:"predicted"`
    Low              *float64 `json:"low"`
    High             *float64 `json:"high"`
    ChangePct        *float64 `json:"change_pct"`
    TrendPct         float64  `json:"trend_pct"`
    HorizonDays      int      `json:"horizon_days"`
    EstimatedBudget  *float64 `json:"estimated_budget"`  // qty * predicted-tier-price
    BudgetLow        *float64 `json:"budget_low"`
    BudgetHigh       *float64 `json:"budget_high"`
    SavingVsNow      *float64 `json:"saving_vs_now"`      // qty * (tier - predicted tier)
    Recommendation   Recommendation `json:"recommendation"`
    Reason           string   `json:"reason"`
    DataStatus       string   `json:"data_status"`
    Basis            string   `json:"basis"`
    GeneratedAt      string   `json:"generated_at,omitempty"`
    Note             string   `json:"note,omitempty"` // e.g. "outside 30-day forecast range"
}
```

### Projection rules (pure, table-driven testable)

Inputs: tier price `c`, forecast `predicted/low/high/base/change_pct`, `data_status`, `target_date`.

1. If forecast `data_status == insufficient` **or** target date is outside `now..now+30d` → all numeric fields `nil`, `recommendation = insufficient`, reason "Not enough price history for a 30-day estimate." / "Target date is outside the 30-day forecast range.", `data_status = insufficient`.
2. Otherwise compute tier-adjusted prices: `p = c × (1 + change_pct/100)`, `lo = c × (low/base)`, `hi = c × (high/base)` (guard `base > 0`). Budget fields = `quantity × p/lo/hi`; `saving_vs_now = quantity × (c − p)`.
3. Signal (conservative — never claim a drop unless even the optimistic bound supports it):
   - `wait` if `hi < c × 0.98` → "Even the high estimate is below today's tier price; consider waiting."
   - `buy_now` if `lo > c × 1.02` → "Even the low estimate is above today's tier price; buying sooner may avoid an increase."
   - `monitor` otherwise → "No clear signal — compare live offers before buying."
4. `data_status` inherited from the forecast (`demo` if any demo sample); `disclaimer` from `AI-SPECS.md §4` when demo. Never phrase a projection as a quote.

---

## 4. Application layer (`backend/internal/modules/planning/application/`)

### `ports.go`

```go
type PlanRepository interface {
    Save(ctx, plan) error                 // upsert plan + items in one transaction
    Get(ctx, ownerKey, id string) (Plan, error)
    List(ctx, ownerKey string) ([]Plan, error)
    Delete(ctx, ownerKey, id string) error
}

type ProductReader interface {            // implemented by adapter over catalog.Service
    Product(ctx, productID string) (ProductInfo, error) // ID, Name, Unit, RegionID
    BestTierPrice(ctx, productID string, quantity int) (float64, bool, error)
}

type Forecaster interface {               // implemented by forecast_adapter.go
    Forecast(ctx, productID, regionID string) (ForecastOutcome, error)
}
```

`ForecastOutcome` is a planning-owned DTO (`Predicted/Low/High/Base/ChangePct/TrendPct/HorizonDays/Samples/DataStatus/Basis`) so the domain never imports `ai/domain`; the adapter translates.

### `service.go` use cases

- `Create(ctx, principal, input)` → validate, resolve owner (D10), for each item load product + tier price, fan out forecasts (D8), build projection (§3), persist.
- `List(ctx, principal)`, `Get(ctx, principal, id)` → owner-scoped; `shared.ErrNotFound` for missing, `shared.ErrForbidden` when the object exists under another owner.
- `Update(ctx, principal, id, input)` → dirty-check (D9); recompute only changed items; merge unchanged projections.
- `Delete(ctx, principal, id)` → owner-scoped.
- `Refresh(ctx, principal, id)` → recompute all item projections with fresh forecasts, update `generated_at`.
- `Preview(ctx, principal, input)` → same computation as `Create`, no persistence (UI what-if).
- Validation: title non-empty; `1 <= quantity <= 10000` (aligns with `compare_offers`); product must exist; role must be `customer`.

---

## 5. Persistence — `backend/migrations/007_planning.sql`

Follow the deployed convention (D11): TEXT ids, `companies` ownership, `TIMESTAMPTZ` for new time columns.

```sql
CREATE TABLE plans (
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
CREATE INDEX idx_plans_owner ON plans (company_id, updated_at DESC);

CREATE TABLE plan_items (
    id                TEXT PRIMARY KEY,
    plan_id           TEXT NOT NULL REFERENCES plans(id) ON DELETE CASCADE,
    product_id        TEXT NOT NULL REFERENCES products(id) ON DELETE RESTRICT,
    product_name      TEXT NOT NULL,
    unit              TEXT NOT NULL DEFAULT 'pcs',
    quantity          INTEGER NOT NULL CHECK (quantity > 0),
    target_date       DATE,
    note              TEXT NOT NULL DEFAULT '',
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
    generated_at       TIMESTAMPTZ,
    created_at         TIMESTAMPTZ NOT NULL DEFAULT now(),
    UNIQUE (plan_id, product_id)
);
CREATE INDEX idx_plan_items_plan ON plan_items (plan_id);
```

- IDs generated in the application, e.g. `plan-<hex>` / `planitem-<hex>` (crypto/rand), matching the existing TEXT-id style.
- Ownership is enforced **both** in the use case and in repository queries (`WHERE company_id = $ownerKey`).
- Optional `backend/seeds/008_planning_seed.sql`: one demo plan for `customer1` (`com-cus1`) with 2–3 items using seeded products, all projections labeled `is_demo`/`demo`, so the new screen is not empty. Every insert uses `ON CONFLICT (id) DO NOTHING`.
- Update `docker-compose.yml` init mounts (next filename after `006_demo_expansion.sql`) and `run.md §4` table + `§5` manual-apply commands. Remember init SQL runs only on an empty volume; existing volumes need manual `psql -f`.

---

## 6. HTTP API — `backend/internal/modules/planning/interfaces/http.go`

Guard: `auth.Chain(auth.Require(auth.RoleCustomer), auth.RequireConfirmed)`. All routes under `/api/v1`.

| Method | Route | Purpose |
| --- | --- | --- |
| `POST` | `/plans` | Create plan (compute + persist) |
| `GET` | `/plans` | List own plans (summaries) |
| `GET` | `/plans/{id}` | Get one plan with items |
| `PATCH` | `/plans/{id}` | Edit title/status/items (dirty-check recompute) |
| `DELETE` | `/plans/{id}` | Delete own plan (204) |
| `POST` | `/plans/{id}/refresh` | Recompute projections from fresh forecasts |
| `POST` | `/plans/preview` | Compute without saving (keep as a literal route; note it so a future `POST /plans/{id}` does not shadow id `preview` in Go's ServeMux) |

Conventions: thin handlers (decode DTO → one use case → `shared.WriteJSON`), standard error envelope via `shared.WriteError`, `403`/`404` semantics per `ARCHITECTURE.md §4`, `disclaimer` alongside `data_status` on every projection-bearing response.

---

## 7. Composition root — `backend/cmd/api/main.go`

- Add planning imports alongside the existing modules.
- Build `productReader := planningInfra.NewProductAdapter(catalogSvc)` and `forecasterAdapter := planningInfra.NewForecastAdapter(forecaster)` where `forecaster` is the variable already created for the AI module.
- `planningSvc := planningApp.New(planningInfra.NewRepository(db), productReader, forecasterAdapter, cfg)`; `planningInterfaces.NewHandler(planningSvc).RegisterRoutes(mux)`.
- If adding config flags (e.g. offline planning / no weather), add them to `backend/internal/platform/config/config.go`, `docker-compose.yml`, and `README.md` env table. Otherwise no config changes are required.

---

## 8. Frontend — Vue 3 platform

| File | Change |
| --- | --- |
| `platform/src/views/customer/Planning.vue` (new) | Plan list + create/edit form. Item editor: product picker (from `GET /products`, as in `Market.vue`), quantity, optional target date (date input, max = today+30d), note. Item table: current tier price, predicted + low–high band, trend, estimated budget, `saving_vs_now`, recommendation badge, `data_status` demo badge, "as of" generated_at. "Refresh forecast", "Preview", "Save plan", "Add to basket" (existing `basket` store — never place an order). Null projections render "—". Reuse `PriceChart.vue`, `StatCard.vue`, `DataTable.vue`, `StatusBadge.vue`, `AppIcon.vue`, `ui/*`. |
| `platform/src/router/index.js` | Add `Planning` to `NAV_CONFIG.customer` (use an existing icon name such as `trend` or `chart` — names like `dashboard`/`market` silently fall back to `box` in `AppIcon.vue`) and a `/customer/planning` child route with `meta.title`. |
| `platform/src/api/client.js` | Add `listPlans`, `getPlan`, `createPlan`, `updatePlan`, `deletePlan`, `refreshPlan`, `previewPlan` helpers (all use the existing `request()` with Bearer token). |
| `platform/src/i18n/index.js` | Add every new English source string to the **en, uz and ru** blocks. |

---

## 9. Tests

**Backend (Go, table-driven where applicable)**
- Projection rules: stable, rising, falling, insufficient (<3 samples → all nulls, never 0), target date outside 30-day horizon, tier price applied via `change_pct` (not absolute forecast), budget/low/high/saving arithmetic, demo/live status inheritance and disclaimer.
- Forecast adapter: maps `domain.PriceForecast` to `ForecastOutcome`, including new `base_price`/`change_pct`, and handles insufficient zero values.
- Pricing adapter: correct best tier for a quantity (default tier vs min-qty tier), missing product/tier.
- Use cases: owner scoping (cross-company → 403/404), `company_id == ""` fallback, quantity bounds, role guard, title-only `PATCH` does NOT call the forecaster (stub), bounded concurrency/timeout with a stub forecaster.
- Repository: save/get/list/delete round-trip with two owners; cascade delete of items.
- HTTP: 201/200/204 + error envelope codes; public getters are NOT exposed (all plan endpoints require auth).

**Frontend**
- `cd platform && npm run build` (template/type check).
- Manual smoke: seeded demo plan renders with demo badge; "—" shown for an insufficient item; preview does not persist; refresh updates `generated_at`.

---

## 10. Documentation updates

- `README.md`: add Planning to Features and the customer cabinet list; add a short `/plans` API example; mention the migration/seed.
- `docs/ARCHITECTURE.md`: add `planning` to the module map/diagram, the bounded-context responsibility table, and the role-authorization matrix (customer-only `/plans`).
- `docs/DATA-MODEL.md`: add the two tables/enums/indexes **with D11's authority note** (migrations are authoritative; new tables follow the deployed TEXT-id/company-ownership convention). Do not attempt a wholesale spec reconciliation in this change.
- `docs/ROADMAP.md`: record Planning as a controlled Phase-3 expansion (still within the out-of-scope guardrails).
- `run.md`: add `007_planning.sql` / `008_planning_seed.sql` to the seed table and manual-apply section; add a quick API check for `/plans`.
- `docs/AI-SPECS.md`: optional one-line note that price forecasts are also consumed by the planning module under the same labeling rules.

---

## 11. Implementation order (keep build green after each step)

1. Forecast DTO extension (`base_price`, `change_pct`) + tests; tier-aware price resolution adapter + tests.
2. Migration `007_planning.sql`, optional seed `008_planning_seed.sql`, compose mounts, `run.md` wiring.
3. Planning `domain/` (Plan, PlanItem, PriceProjection, pure rules) + unit tests.
4. Planning `application/` (ports, service, validation, dirty-check, fan-out) + use-case tests with stubs.
5. Planning `infrastructure/` (repository, forecast adapter, pricing adapter) + integration tests.
6. `interfaces/http.go` + `cmd/api/main.go` wiring + HTTP smoke tests.
7. Frontend view, router, api client, i18n + `npm run build`.
8. Docs updates + full verification: `cd backend && go test ./... && go vet ./...`, `cd platform && npm run build`, `docker compose config --quiet`, `curl` the new endpoints with a `customer1` token.

---

## 12. Non-goals / guardrails

- No order creation, no basket checkout, no auto-purchasing from a plan. "Add to basket" is the only action and is user-triggered.
- No price guarantees: projections are estimates with a range, a `basis` explanation, and `data_status`; "live" only for non-demo samples.
- No LLM in the planning path: backend owns filtering, arithmetic, ownership and labeling.
- No generic SQL/HTTP tools; no new external service (weather remains optional and cached as today).
- Demo data visibly labeled in API and UI; treat labeling failures as release blockers per `AI-SPECS.md §4`.

---

## 13. Open items for the implementing agent

- Decide whether planning uses the live OpenMeteo provider or a nil (offline) provider by default; D8 supports both. Using the existing `forecaster` keeps weather but adds a network dependency to plan creation.
- Confirm whether `target_date` should be required or optional (plan currently allows optional, validated against the 30-day horizon).
- If the product owner later asks for resale/margin planning, extend `PlanItem` with `target_sale_price` and a computed projected margin — this is out of scope for v1 (D1).
