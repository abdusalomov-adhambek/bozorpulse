# BozorPulse — Roadmap

> Part of the current spec. See [README.md](../README.md), [ARCHITECTURE.md](ARCHITECTURE.md), [AI-SPECS.md](AI-SPECS.md), and [DATA-MODEL.md](DATA-MODEL.md). The legacy `overview.md` brief is superseded. This roadmap replaces any phasing described there.

## Scope guardrails

**MVP promise:** help suppliers and customers **find each other and create orders on the platform**, with market price visibility for customers and a direct sales channel for suppliers.

**In scope for MVP (Phases 0–2):** three roles and admin approval; catalog/categories/regions; product prices and discounts; market price statistics; supplier directories and deterministic recommendations; multiple warehouses; order creation and manual status changes; payment and inventory **records/movements**; role-aware dashboard; rule-based assistant; demo-data labeling; Vue 3 platform and Nuxt 3 website.

**Explicitly OUT of scope for the MVP (do not schedule as MVP work):**

- ❌ Payment processing (gateways, charging, payouts, reconciliation with a bank).
- ❌ Delivery tracking, logistics/carrier integrations, live shipment status.
- ❌ Automated purchasing, auto-approval of orders, replenishment bots, price negotiation.
- ❌ Any **order execution beyond creation** — the platform records an order and allows manual status changes; it does not move money or goods.
- ❌ Redis, message brokers, microservices, Kubernetes, production LLM dependency, email/SMS login, real-time chat between users.

Items in Phase 3 are **hardening and controlled expansion of the same MVP**, not the out-of-scope capabilities above.

## Phase overview

| Phase | Name | Outcome | Indicative effort |
| --- | --- | --- | --- |
| **0** | Foundation | Running API + schema + auth/approval; deployable shell of all five Compose services | 6 person-weeks |
| **1** | Marketplace MVP | Suppliers publish products, customers discover them and create orders; cabinets usable by all three roles | 14 person-weeks |
| **2** | Price visibility & AI | Market statistics, deterministic recommendations, assistant, analytics, demo labeling | 9 person-weeks |
| **3** | Hardening & scale | Payments/inventory completeness, i18n, optional LLM hook, performance, observability, launch readiness | 10 person-weeks |
| | | **Total** | **≈ 39 person-weeks** (≈ 13 calendar weeks with 3 engineers) |

Estimates assume three engineers (backend, frontend, full-stack/AI) working in parallel, an existing Compose scaffold, and demo data supplied by the team. They exclude user research, legal review of company data handling, and content production for the website.

---

## Phase 0 — Foundation

**Goal:** trust and connectivity. A new account can register, an admin can approve it, and the stack runs locally with one command.

| ID | Epic | Effort |
| --- | --- | --- |
| F0.1 | Go 1.25 module + composition root (`cmd/api`), config, `net/http` router with Go 1.22 method/path patterns, middleware (recovery, request ID, logging, JSON errors) | 1.5 pw |
| F0.2 | PostgreSQL 16 schema `001_init.sql` + `seed.sql` with all 13 tables, enums, indexes, and three demo accounts (admin/admin123, supplier1/sup123, customer1/cust123) | 1.5 pw |
| F0.3 | `identity` module: registration with INN/bank/company fields, username+password login, token sessions (`JWT_SECRET`, `TOKEN_TTL`), password hashing, `GET /auth/me` | 2 pw |
| F0.4 | Admin approve/reject + status guard: `POST /admin/users/{id}/approve\|reject`, `GET /admin/users?role=&status=`, 403 `account_not_confirmed` for non-confirmed users | 1 pw |
| F0.5 | Compose topology: postgres + api + platform + website + prototype, healthcheck, schema/seed init on first start; frontend shells with routing and Pinia auth store | — (shared with FE) |

**Acceptance criteria**

- [ ] `docker compose up --build` starts all five services; `GET /health` returns `{"status":"ok","service":"bozorpulse"}`.
- [ ] `POST /api/v1/auth/register` returns `{user, token}` with `user.status = "waiting"`; the token cannot access `/auth/me`-adjacent write routes until approval.
- [ ] `POST /api/v1/auth/login` with `admin/admin123`, `supplier1/sup123`, `customer1/cust123` succeeds; wrong password returns 401.
- [ ] Admin approves a waiting supplier; that supplier can then reach their cabinet; a rejected user gets 403 `account_not_confirmed`.
- [ ] Domain packages compile without importing `net/http`, `database/sql`, or `internal/platform/postgres` (import check in CI).
- [ ] README quickstart commands work from a clean checkout.

---

## Phase 1 — Marketplace MVP

**Goal:** the two-sided marketplace functions. Suppliers list products; customers browse and create orders against a chosen warehouse.

| ID | Epic | Effort |
| --- | --- | --- |
| F1.1 | `companies` + reference data: `/regions`, `/companies/me`, `/suppliers`, `/customers`, company verification flag | 2 pw |
| F1.2 | `catalog`: `/categories` CRUD, `/products` CRUD, `POST /products/{id}/discount`, generated `effective_price`, supplier ownership checks | 3 pw |
| F1.3 | `inventory` basics: `/warehouses` CRUD for supplier and customer, owner/role validation, `GET /inventory/stock` | 2 pw |
| F1.4 | `procurement`: `GET/POST /orders`, `GET /orders/{id}`, `PATCH /orders/{id}/status`, order items, computed `total_amount`, valid transitions, ownership checks | 3 pw |
| F1.5 | Platform cabinets: admin (Dashboard, Suppliers, Customers, Orders, Settings: Product categories/Products/Regions); supplier (Dashboard, Orders, Payments, Products, Warehouses, Cabinet); customer (Dashboard, Orders, Suppliers, Payments, Warehouses with Stock/Transfers/Supplies/Write-off/Returns, Cabinet) | 4 pw (FE) |
| F1.6 | Website (Nuxt 3): landing, value proposition, role explainer, links to platform; public catalog preview via public GETs | — (FE, parallel) |

**Acceptance criteria**

- [ ] Public GETs for products, categories, regions, suppliers work **without** a token; writes require a confirmed user.
- [ ] A confirmed supplier can create a product with `price` and `discount_percent`; `effective_price` is returned and stored correctly.
- [ ] A supplier cannot edit another supplier's product; a customer cannot create products (403).
- [ ] Both roles can create multiple warehouses; a customer selects a warehouse when creating an order.
- [ ] `POST /orders` creates an order with `status = "new"` and correct `total_amount`; invalid transitions (e.g. `delivered → new`) return 409.
- [ ] Admin sees all orders; supplier sees only orders where they are the seller; customer sees only their own.
- [ ] Every cabinet section listed in the product spec is reachable in the platform UI for the correct role.

---

## Phase 2 — Price visibility & AI

**Goal:** deliver the differentiators — honest market prices, explainable recommendations, and an assistant that cannot invent data.

| ID | Epic | Effort |
| --- | --- | --- |
| F2.1 | `price_samples` ingestion + `GET /market/prices`: dedup, 90-day window, min/median/avg/count, `data_status`, sample-size badge, demo disclaimer | 2 pw |
| F2.2 | `ai` recommendations: `GET /ai/recommendations?product_id=`, weighted formula (0.40 price / 0.20 freshness / 0.20 region / 0.20 verification), reasons, tie-breakers, persisted snapshots | 2 pw |
| F2.3 | `ai` assistant: `POST /ai/chat`, 8 intents, typed tool registry (no SQL/HTTP tools), grounded templates, optional LLM provider interface **disabled by default** | 2.5 pw |
| F2.4 | `analytics`: `GET /analytics/dashboard` returning `{role, metrics, recent_orders, market_summary}`, wired into each Dashboard | 1 pw |
| F2.5 | Demo-data labeling end-to-end: `is_demo` flags, `data_status` in all AI/analytics responses, UI badges, evaluation fixtures and metric harness from [AI-SPECS.md §5](AI-SPECS.md#5-evaluation-metrics) | 1.5 pw |

**Acceptance criteria**

- [ ] `/market/prices?category_id=1&q=cement` returns `samples`, `stats {min, median, avg, count}`, and `data_status`; with fewer than 5 samples the stats are `null` and status is `insufficient` (no invented numbers).
- [ ] Seeded samples produce `data_status = "demo"` and the UI shows the "Demo data — sample values, not live market quotes" badge.
- [ ] Recommendations are reproducible for a fixed DB snapshot; `factors` and `reasons` are persisted and each reason maps to a factor.
- [ ] The assistant answers the 8 specified intents, calls only allowlisted tools, refuses payment/delivery promises honestly (`out_of_scope`), and reports 100% groundedness on the eval set.
- [ ] No `inn`, `bank_account`, `phone`, or password field appears in any AI request or log payload.
- [ ] Dashboards return a payload shaped by the caller's `role` and render without errors for all three roles.

---

## Phase 3 — Hardening & scale

**Goal:** launch readiness and a controlled path to more capability — still **without** the out-of-scope items listed above.

| ID | Epic | Effort |
| --- | --- | --- |
| F3.1 | `payments` records: `GET/POST /payments`, `GET /payments/{id}`, status transitions, owner checks; `stock_items` + all inventory movements (`supplies`, `transfers`, `write-offs`, `returns`) and `/inventory/movements` audit view | 3 pw |
| F3.2 | Observability & reliability: structured logs with `request_id`, `/health` readiness, error-rate dashboards, backup/restore drill, seed migration strategy for existing DBs | 2 pw |
| F3.3 | Performance: query `EXPLAIN ANALYZE` tuning, index review, read-replica routing for analytics/market reads, p95 targets from AI-SPECS | 2 pw |
| F3.4 | Multilingual: `ru` + `en` UI and assistant/i18n templates, then `uz` (Latin + Cyrillic); locale-aware number/date formatting | 2 pw |
| F3.5 | LLM hook — Meta Model API (OpenAI-compatible `/chat/completions`) enabled by default via `AI_LLM_ENABLED=true` + `MODEL_API_KEY`; tool selection from the allowlist, groundedness/PII guards, rule-based fallback; eval harness in CI | 1 pw |
| F3.6 | Planning (controlled expansion): customer purchase-timing plans, per-item projections from the existing price forecaster and quantity tiers, conservative `buy_now`/`wait`/`monitor` signals, `/plans` API and planning cabinet screen. Estimates only — no orders, no guarantees. | 2 pw |

**Acceptance criteria**

- [ ] Payment records and inventory movements are fully CRUD/auditable for the correct roles, with no gateway or carrier integration.
- [ ] Warehouse sub-tabs (Stock, Transfers, Supplies, Write-off, Returns) all persist and display movements.
- [ ] Market statistics p95 < 150 ms and recommendations p95 < 200 ms at 1,000 samples / 200 products in a load test.
- [ ] English, Russian, and Uzbek locales switch without changing scores or breaking layout; eval sets meet the §5 targets per language.
- [ ] With `AI_LLM_ENABLED=true` and `MODEL_API_KEY` set, the assistant uses the Meta Model API and preserves tool-call IDs across rounds.
- [ ] With `AI_LLM_ENABLED=false` **or** no API key (fallback), the API runs with no network egress and no key, and the rule-based assistant answers all supported intents.
- [ ] Restore drill recovers schema + data from backup within the agreed RTO.
- [ ] Customers can create/read/update/delete plans; projections use quantity-tier prices, null out on insufficient data, and are labeled `demo`/`live` with a disclaimer. No plan places an order.

---

## Risks and mitigations

| Risk | Impact | Likelihood | Mitigation |
| --- | --- | --- | --- |
| Admin approval becomes a bottleneck; users stuck as `waiting` | New suppliers/customers cannot use the platform | High | Make approval prominent in admin Suppliers/Customers lists, show pending counts on the Dashboard, add an SLA alert; keep resubmit/reject flows clear. |
| Demo prices mistaken for real market quotes | Loss of trust, misleading decisions | Medium | Enforce `is_demo` + `data_status` on every response, visible UI badges, "insufficient" state instead of extrapolation; treat labeling failures as release blockers. |
| Assistant invents numbers or oversteps scope | Wrong decisions, credibility damage | Medium | Typed tool allowlist, no SQL/HTTP tools, grounded templates, automated groundedness check (zero tolerance), honest `out_of_scope` refusals. |
| Scope creep into payments/delivery/automation | MVP slips indefinitely | High | Keep the out-of-scope list at the top of this document and in README; review any such request against the ordering-only promise. |
| Ownership bugs leak another user's orders/warehouses | Security and trust incident | Medium | Central status guard + object-level checks in use cases; 403 vs 404 tests for cross-tenant access in CI. |
| Poor data quality (duplicate SKUs, stale samples) | Unreliable market stats and rankings | Medium | UNIQUE SKU, 90-day window, dedup latest sample per product, `MIN_SAMPLES` threshold, verification factor in recommendations. |
| DB hot spots on market/dashboard queries | Latency at scale | Low in MVP | Indexes in DATA-MODEL.md, read replica and precomputation in F3.3; no premature cache ([ADR-003](ARCHITECTURE.md#adr-003-no-redis-in-the-mvp)). |
| LLM privacy/cost if enabled early | Data exposure, unpredictable spend | Low while disabled | Feature flag off by default, PII allowlist for prompts, zero-retention contract required for Phase C. |
| Frontend/backend version drift | Broken integration near launch | Medium | API contract lives in docs; contract smoke tests in Compose; pair FE/BE on endpoint changes. |

## KPIs

| KPI | Definition | MVP target | Measured at |
| --- | --- | --- | --- |
| Time to first order | Median time from confirmed supplier/customer to first completed order | < 7 days | `/analytics/dashboard`, order timestamps |
| Approval turnaround | Median time from `waiting` to `confirmed`/`rejected` | < 1 business day | `users.created_at` vs status change |
| Market price coverage | Categories with ≥ 5 usable samples (`data_status != insufficient`) | ≥ 80% of seeded categories | `/market/prices` eval sweep |
| Order creation success rate | Successful `POST /orders` / attempts | ≥ 98% | API logs |
| Two-sided activation | Confirmed suppliers with ≥ 1 product **and** customers with ≥ 1 order | ≥ 60% of confirmed accounts | admin dashboard |
| Demo-labeling coverage | Demo-backed responses carrying `data_status` + badge | 100% | automated API/UI checks |
| AI groundedness | Answers with only tool-sourced numbers/entities | 100% | AI eval harness |
| Recommendation determinism | Identical output for identical snapshot | 100% | CI test |
| API latency | p95 for `/market/prices` and `/ai/recommendations` | < 150 ms / < 200 ms | load test (F3.3) |
| Availability | API health-check success | ≥ 99.5% in launch month | monitoring |

## Effort summary

| Workstream | Phase 0 | Phase 1 | Phase 2 | Phase 3 | Total |
| --- | ---: | ---: | ---: | ---: | ---: |
| Backend | 6 pw | 10 pw | 7.5 pw | 7 pw | 30.5 pw |
| Frontend / platform / website | — (shell) | 4 pw | 1.5 pw (dashboards/badges) | 3 pw (i18n) | 8.5 pw |
| **Total** | **6 pw** | **14 pw** | **9 pw** | **10 pw** | **≈ 39 pw** |

Sequencing: Phase 0 unblocks everything; Phase 1 backend (F1.1–F1.4) can run in parallel with frontend shells (F1.5–F1.6); Phase 2 depends on products and `price_samples` from Phase 1; Phase 3 depends on a stable Phase 1–2 API contract.

> **Reminder:** all phases above build the *new* product described in this `docs/` set. `overview.md` is a legacy brief and is not a planning input.
