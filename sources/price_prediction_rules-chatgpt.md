# BozorPulse — Product Price Intelligence & Prediction Rules

**Version:** 2.0 (supersedes Version 1.0)\
**Status:** Proposed executable specification — for backend, ML and AI implementation agents\
**Scope:** Price prediction and market-intelligence outputs for the BozorPulse B2B marketplace\
**Companion document:** `sources/price-prediction-rules-claude.md` — the **category calibration playbook** (price types, weather windows, caps, per-category blocks). This file is the **system specification** (architecture, price indicators, market dynamics, canonical output schema, evaluation and monitoring). Where the two overlap, this file is authoritative for system contracts and the companion is authoritative for per-category operational detail. Implementation configuration derived from both lives in `backend/internal/modules/ai/infrastructure/forecast/calibration.json`.

**Active policy decision:** until the calibration table contains backtested β values, the weather adjustment is **zero** and the contribution percentage is `null` — the forecast is a labeled analytical estimate, not a validated weather-adjusted model.

---

## What changed from Version 1.0 and why

Version 1.0 was a sound conceptual starting point: it established that weather is conditional,
history is the baseline, categories differ, uncertainty is mandatory, and the AI must not invent
data. A review found it was **not yet a production specification**. Version 2.0 closes those gaps:

| Review finding | Change in v2.0 |
| --- | --- |
| One "price" was assumed; supplier quotes and transactions were mixed | Separate, explicitly defined **price indicators** (§4) and **price-quality hierarchy** (§6.3) |
| No market segmentation in history handling | Mandatory normalization and segmentation by market, region, grade, unit, basis, source (§6) |
| No model-selection strategy | **Forecasting engine** section with baseline → category → product model ladder and selection rules (§9) |
| Qualitative confidence only | **Measurable confidence thresholds**, liquidity and fallback caps (§11) |
| No cold-start strategy | Five-level **cold-start hierarchy** with declared fallback level (§10) |
| Weather rules lacked quantitative treatment | **Anomalies, supply-share exposure, lags and attribution** (§8) |
| Supplier/buyer dynamics ignored | **Market dynamics** signals and derived indicators (§7) |
| LLM doing statistics was ambiguous | Hard **separation of forecasting engine and AI interpretation agent** (§3) |
| Category overlap (e.g. steel) | One **canonical category + multiple market/use-case tags** (§5) |
| Output schema too thin | Extended schema with `price_indicators`, `prediction`, `market_dynamics`, `weather_impact`, `model_metadata` (§14) |
| No operational monitoring | Evaluation, **7/30/90-day** tracking, drift, retraining and alerting (§13) |
| Manipulation, FX, regional bias unaddressed | Anomaly/manipulation detection (§12), regional and currency rules (§6.5–6.6) |

**Normative language:** MUST = required for any conforming implementation; SHOULD = expected unless
justified otherwise; MAY = optional. "Engine" means the statistical/ML forecasting engine; "agent"
means the LLM interpretation layer.

---

## 1. Mission and operating principles

Produce the most defensible, product-specific **set of price indicators, forecast ranges and market
signals** for a defined product, market, location, unit, currency and horizon — and explain them in
the user's language. BozorPulse must predict not only *what a product will cost*, but **how the
market is changing**: supplier competition, supply constraints, weather exposure and whether the
current price is unusual relative to history.

Core principles:

1. **Product-specific reasoning** — resolve category, supply chain, price mechanism, unit, grade and
   market before selecting signals.
2. **Evidence before narrative** — use measured relationships; never invent causes, prices, quotes,
   weather or coefficients.
3. **Weather is conditional** — a potential explanatory variable, never a universal price driver.
4. **History is the baseline** — every weather-aware model MUST be compared with a no-weather
   baseline for that product–market–horizon.
5. **Multiple price targets** — never silently substitute a supplier asking price for a market
   median or an expected transaction price (§4).
6. **Engine computes, agent explains** — all numbers come from the forecasting engine; the LLM
   MAY classify, interpret and phrase, but MUST NOT calculate forecasts or percentages itself (§3).
7. **Uncertainty is mandatory** — range, calibrated confidence, horizon and assumptions always.
8. **Comparable prices only** — normalize currency, unit, tax, packaging, grade, delivery terms,
   market level and source before comparison (§6).
9. **Cold-start honesty** — declare which fallback level was used and cap confidence accordingly (§10).
10. **As-of discipline** — backtests use only data that would have been available at prediction time,
    including the weather *forecast* that existed then (§13.3).
11. **No false precision** — match decimal places to data quality; label demo/sample data visibly.

---

## 2. Define the request and the forecast target

Resolve the following before any calculation. Missing key fields MUST trigger a clarification or an
explicit, confidence-lowering assumption — never silent defaults.

- `product_id`, canonical `product_name`, `specification` (brand, model, grade, size, origin).
- `canonical_category` and `market_tags[]` (§5).
- `market` level: wholesale, distributor, manufacturer, retail, import/export.
- `location`: production/origin region(s) and delivery destination region.
- `currency` (ISO code) and `unit` (kg, tonne, litre, piece, bag, pallet, m², kWh, service-hour…).
- `price_basis`: ex-works, farm-gate, wholesale delivered, retail, Incoterm.
- `target_period_start`, `target_period_end`, `horizon_days` (7 / 30 / 90 primary).
- `as_of` timestamp and timezone; `data_source[]` allowed for this request.

### 2.1 Transaction and delivery conditions

All indicators and bounds in one response MUST refer to the **same** product specification, market
level, unit, currency, tax treatment and delivery terms. If the application requests both an asking
price and a transaction price, these MUST be separate indicators with separate labels — never
averaged together.

---

## 3. Architecture: two different jobs

Review finding: *"Separate the AI reasoning agent from the actual numerical forecasting engine."*

```
B2B Marketplace Data                 External Data
(transactions, quotes, inventory,   (weather observations + forecasts, FX,
 orders, suppliers, buyers)          commodity benchmarks, logistics)
        |                                    |
        v                                    v
Data Processing Layer — normalization, validation, deduplication, product matching,
segmentation, anomaly/manipulation flags, as-of availability timestamps
        |
        v
Historical Data Store — normalized prices, seasonality, trends, realized outcomes,
as-of feature snapshots, model versions
        |
        v
FORECASTING ENGINE (deterministic/statistical) — baseline models, category models,
product models, weather adjustments, intervals, confidence, cold-start fallback
        |
        v
AI INTERPRETATION AGENT (LLM) — economic reasoning, driver explanation, risk framing,
category/spec classification, multilingual phrasing
        |
        v
BozorPulse Price Intelligence API — indicators, prediction, ranges, confidence,
market dynamics, weather impact, model metadata, explanation
```

**Hard separation rules**

- The LLM MUST NOT compute predicted prices, intervals, percentage changes, confidence scores,
  weather contributions or any derived ratios from raw observations. It receives only the engine's
  structured, already-calculated output.
- The engine MUST NOT invent qualitative narrative. It emits structured values and reasons codes;
  the agent turns them into an explanation.
- If no engine result exists for a request, the agent MUST return `insufficient_data` rather than
  estimating a number from the prompt.
- The agent MUST answer in the **language of the user's question**, independent of the selected UI
  locale, and MUST NOT translate or alter the engine's numeric fields.

---

## 4. Price indicators — define the target before forecasting

Review finding: *"In a B2B marketplace there isn't necessarily one price for a product… Don't mix
these indicators in one forecasting model without explicitly defining the target."*

A conforming response MUST expose these indicators, with `null` when not computable:

| Indicator | Definition | Typical use |
| --- | --- | --- |
| `market_median_price` | Median of comparable, normalized observations in the defined market and period | Representative market level; robust to outliers |
| `market_average_price` | Volume- or reliability-weighted mean of comparable observations | Benchmark when weights are defensible |
| `market_min_price` | Lowest comparable available offer/price in the period | Buyer's best-available benchmark |
| `market_max_price` | Highest comparable available offer/price in the period | Outlier ceiling; context only |
| `expected_transaction_price` | Model estimate of the price at which buyer and supplier would actually transact, based on completed transactions where available | The central forecast for buying decisions |

Rules:

1. Each indicator MUST be computed from the **same normalized observation set** or the response MUST
   state that sets differ and why.
2. `expected_transaction_price` MUST be derived primarily from completed transactions. If only
   supplier quotes exist, it MUST be labeled as an asking-price-based estimate with reduced
   confidence — never presented as an executed price.
3. The indicator used as `reference_price` for `predicted_change_pct` MUST be named explicitly.
4. Scenario prices (§14) MUST state whether they are percentiles of a predictive distribution or
   assumption-driven scenarios, and of which indicator.
5. A forecast of "the price" with no indicator named is non-conforming.

---

## 5. Product taxonomy: one canonical category, many tags

Review finding: *"Steel appears under both construction materials and metals. Use one canonical
product category and multiple market/use-case tags."*

- Every product has exactly one `canonical_category` from the marketplace taxonomy. It determines
  the default model family, base weather relevance and evaluation peer group.
- Products MAY carry `market_tags[]` such as `construction_demand`, `food_processing`,
  `export_commodity`, `cold_chain`, `seasonal_retail`, `industrial_input`. Tags select additional
  features and category overlays; they MUST NOT create a second competing category.
- If classification is ambiguous, choose the most specific supported canonical category, record the
  ambiguity in `explanation.assumptions`, and lower confidence.
- Category-wide coefficients MUST NOT be applied mechanically to every product. Product-level and
  region-level estimates take precedence wherever data supports them.

The v1 category table (agriculture, fresh produce, livestock, food processing, construction, metals,
energy, chemicals, textiles, electronics, machinery, timber/paper, pharma, consumer, automotive,
services) remains the starting taxonomy. Each category entry MUST now declare: `canonical_category`,
allowed `market_tags[]`, baseline weather relevance, typical transmission channels and typical lag
range (§8.4). Detailed per-category profiles live in the companion document.

---

## 6. Data preparation, normalization and market segmentation

Review finding: *"Historical prices — good preprocessing, missing market segmentation."*

### 6.1 Required normalization pipeline

1. Validate timestamps, units, currencies, locations, product identifiers and price basis.
2. Deduplicate using source + observation time + supplier + price; retain the raw record.
3. Normalize package size and unit to the canonical unit; record the conversion factor.
4. Convert currencies with a **time-aligned** exchange rate; store both original and converted value.
5. Record VAT, duties, delivery, insurance and commission inclusion.
6. Record grade, brand, origin, contract size and delivery terms.
7. Separate source types: completed transaction, invoice, confirmed quote, market report, listing,
   indicative estimate.
8. Flag outliers with robust, product-specific rules; retain genuine shocks, quarantine erroneous or
   incomparable observations.
9. Record missing values, source reliability and data-quality flags.
10. Preserve the original observation and its normalized version for auditability.

### 6.2 Segmentation — mandatory before comparison

Observations MUST be grouped by all of: `canonical_product_id + specification`, `market_level`,
`region/origin`, `currency`, `unit`, `price_basis`, `grade`, `source_type`. Cross-segment pooling is
allowed only through a documented hierarchical model or explicit, recorded assumption.

### 6.3 Price-quality hierarchy (transaction target)

1. Verified completed B2B transactions with comparable terms.
2. Verified supplier invoices / purchase records.
3. Confirmed supplier quotes.
4. Reliable wholesale or commodity market reports.
5. Public catalogue / listing prices.
6. Indicative or unverified user-submitted prices.

Advertised-price changes MUST NOT be treated as market-price changes. Prefer completed transactions
for `expected_transaction_price`; track quote-to-transaction conversion as a market signal (§7).

### 6.4 Outliers, structural breaks and manipulation

- Detect unusual changes with robust methods and product-specific thresholds; investigate against
  source, unit, currency, quality, policy and event data.
- Detect structural breaks in tax, regulation, product specification, supplier set or currency
  regime; do not assume old relationships persist.
- Flag manipulation patterns (P2): duplicate listings, suspiciously low offers, coordinated price
  moves across suppliers, rapid re-posting, and single-supplier dominance in a thin market.
  Flagged observations are excluded from baseline statistics unless a human override records why.

### 6.5 Regional differences

Do not combine Tashkent wholesale prices with Fergana supplier prices as if identical. Where origin
and destination differ, model a **regional basis** (transport cost, local market premium/discount,
availability) and record it. Weather exposure is computed over **supply-source regions**, not the
buyer's city (§8.2).

### 6.6 Currency and commodity separation

Underlying commodity movement and UZS exchange-rate movement MUST be separable features. A price
rise caused solely by FX depreciation must not be reported as a real market tightening. FX and
commodity benchmarks are external data series with their own as-of timestamps.

---

## 7. Market dynamics and supplier behavior

Review finding: *"Historical prices alone don't tell the whole story."*

The engine SHOULD compute the following signals where the marketplace has the data, and MUST report
them as trends/levels rather than inventing magnitudes:

| Signal | Why it matters | Output field |
| --- | --- | --- |
| Active supplier count | More competition can reduce prices | `market_dynamics.supplier_count`, `competition_level` |
| Active buyer count / order trend | Higher demand can raise prices | `buyer_demand_trend` |
| Supplier inventory / stock-to-use | Low inventory signals supply pressure | `available_supply_trend` |
| Available quantity per offer | 10 tonnes ≠ 10,000 tonnes | `available_quantity` in offer context |
| Quote frequency | Market activity | `quote_activity` |
| Quote changes | Supplier reaction to conditions | `quote_change_trend` |
| Transaction volume | Separates asking from executed demand | `transaction_volume_trend` |
| Quote-to-transaction ratio | Conversion / realism of asking prices | `quote_to_transaction_ratio` |
| Supplier concentration | Dependence on few suppliers | `supplier_concentration` |
| Delivery lead time | Shortages / logistics constraints | `lead_time_trend` |

Rules:

1. Trends MUST be computed from observed snapshots with as-of timestamps; `increasing` /
   `decreasing` / `stable` / `unknown` are the allowed labels.
2. Market dynamics MUST NOT be used to override the statistical forecast without a backtested
   relationship; they may adjust confidence, produce risk flags and feed validated features.
3. A supplier changing an advertised price is **not** proof the market price changed. Completed
   transactions take precedence for market-level inference.

---

## 8. Weather analysis — quantitative treatment

Review finding: *"Good principles, insufficient quantitative rules."*

### 8.1 Weather anomaly, not raw weather

Compare current and forecast conditions with **historical seasonal normals** for the relevant
location and date: `anomaly = current value − seasonal normal` (or standardized anomaly). A
temperature of 35 °C is only notable relative to what is normal there and then. Use observations for
historical analysis and forecast data for future periods; keep the two distinct and record issue
time.

### 8.2 Exposure weighting by supply share

For a product sourced from multiple regions, aggregate exposure as a weighted contribution:

```
exposure = Σ_i ( supply_share_i × relevance_i × anomaly_i × lag_factor_i )
Σ supply_share_i = 1 ; relevance_i ∈ {0, low, medium, high} mapped by a documented calibration
```

Example (illustrative weights only): Region A 50% supply share, high relevance; Region B 30%, medium;
Region C 20%, low. The engine computes the weighted exposure; the agent explains it. Weights MUST
come from sourcing data or recorded assumptions, never from the LLM's intuition.

### 8.3 Transmission channels (retained from v1)

Production/yield; livestock/aquaculture; storage/spoilage; energy demand/supply; logistics and
infrastructure; construction and outdoor services; seasonal demand. For each product, only channels
in its causal map (§4 of v1, retained) may be used.

### 8.4 Delayed effects

Weather impact is rarely instantaneous. Typical lag ranges (examples, not universal assumptions —
estimate actual lags from historical data):

| Product type | Typical lag |
| --- | --- |
| Fresh vegetables / herbs | days to weeks |
| Grain / oilseed crops | weeks to months, crop-stage dependent |
| Livestock feed / dairy | weeks to months |
| Construction materials | immediate logistics effect, delayed demand effect |
| Electronics / machinery | indirect, potentially long supply-chain lag |
| Energy | immediate demand effect, variable supply effect |

Lag factors MUST be validated; if unvalidated, weather contribution stays `null`.

### 8.5 Attribution — separate the drivers

For any forecast where weather is used, the engine MUST report — when its model supports attribution
— a decomposition into:

- movement explained by the **historical trend / baseline**,
- movement associated with **weather** (anomaly × exposure × lag),
- movement associated with **other market factors** (§7, inputs, FX, dynamics).

`weather_impact.contribution_estimate_pct` MUST remain `null` unless a validated attribution method
(e.g. ablation against the no-weather baseline, Shapley-style decomposition for tree models) produces
it. The LLM MUST NOT invent these percentages, and MUST NOT double-count a weather effect already
inside the combined forecast.

### 8.6 Weather relevance classes

`high` (direct, evidenced channel), `medium` (indirect/conditional), `low` (monitoring only),
`unknown` (insufficient evidence). These qualitative classes MUST NOT be converted directly into
percentage price changes.

### 8.7 Safeguards (retained and extended)

Never apply one weather coefficient to all products; never assume hot weather raises every food price
or rain always reduces supply (irrigation, greenhouses, imports can offset). Reduce weather influence
as horizon grows. Avoid double-counting correlated variables (heatwave, max-temperature anomaly,
cooling degree days). Use the weather forecast available at the original prediction time in
backtests (§13.3).

---

## 9. Forecasting engine and model selection

Review finding: *"Good model suggestions, missing model-selection strategy."*

### 9.1 Model ladder

| Level | Model | When used |
| --- | --- | --- |
| L0 | Seasonal naive / last observed comparable price | Minimal data; always computed as a floor baseline |
| L1 | Moving average / exponential smoothing (ETS) | Sparse but regular history |
| L2 | SARIMA / regression with calendar terms | ≥ ~2 seasonal cycles of clean history |
| L3 | Category-specific / hierarchical panel model | Related products or regions can share strength |
| L4 | Gradient-boosted trees or ensemble with validated features | Sufficient data and proven out-of-sample gain |
| L5 | Weather-adjusted combined model | Only if §8 gate passes for that product–market–horizon |

Complexity MUST be justified by out-of-sample performance. Prefer a simple, calibrated model when
data is sparse. Every prediction MUST report `model_type` and `model_version` (§14).

### 9.2 Selection procedure (engine, deterministic)

1. Compute L0 baseline. Try increasingly complex candidates using **time-ordered backtests**.
2. Candidate is promoted only if it beats the current champion on the primary metric (WAPE/MAE) for
   that horizon, at ≥ a documented improvement threshold, with calibrated intervals.
3. Test weather features via ablation: combined model vs same model without weather (§9.3). Promote
   weather only on measured improvement or a recorded, defensible physical mechanism with
   conservative caps.
4. Lock the champion per `canonical_category + market + horizon` (and per product where data allows).
   Changes require a version bump and backtest record.

### 9.3 Weather inclusion gate

The engine MUST be able to conclude: weather is not a driver for this product; the signal is too
uncertain; data is insufficient; or other factors dominate. In those cases the baseline forecast is
returned with `weather_impact.relevance` set accordingly and `contribution_estimate_pct = null`.

### 9.4 Horizons

Primary horizons are **7, 30 and 90 days**, evaluated separately (§13). Very-short-term (1–7 days)
relies on recent quotes/inventory/short-range weather; medium term on seasonality, production cycles
and broader drivers; beyond 90 days use climate normals or seasonal outlooks only, clearly
distinguished from deterministic forecasts.

---

## 10. Cold-start prediction

Review finding: *"No cold-start strategy."*

When exact history is missing or too short, the engine walks this hierarchy and MUST report the level
used in `model_metadata.fallback_level`:

1. `exact_product_history` — same product, specification and market.
2. `similar_products` — comparable specifications, quality and supply chain.
3. `related_market_benchmarks` — upstream commodity, manufacturer or import prices.
4. `supplier_offers` — verified comparable quotes, explicitly labeled as asking prices.
5. `insufficient_data` — return an indicative estimate only if defensible; otherwise `status:
   insufficient_data` with all numeric fields `null`.

Confidence caps by fallback level:

| Fallback level | Maximum confidence |
| --- | --- |
| `exact_product_history` | high (subject to other thresholds) |
| `similar_products` | medium |
| `related_market_benchmarks` | low |
| `supplier_offers` | low |
| `insufficient_data` | insufficient_data |

The agent MUST explain which level was used in plain language (e.g. "based on similar products, not
this product's own history").

---

## 11. Confidence and uncertainty — measurable thresholds

Review finding: *"Qualitative, needs measurable thresholds."*

Confidence is produced by the engine from historical validation, never by the LLM. The platform team
MUST calibrate the following illustrative thresholds with backtest results:

| Rule | high | medium | low | insufficient_data |
| --- | --- | --- | --- | --- |
| Comparable observations (30-day window) | ≥ 30 | 10–29 | 3–9 | < 3 |
| History span | ≥ 12 months | 3–12 months | < 3 months | none |
| Interval coverage (backtest) | ≥ nominal (e.g. 80% for 80% interval) | within −5pp | below −5pp or unvalidated | not measurable |
| Liquidity / transactions | ≥ 100 comparable | 20–99 | 2–19 | < 2 |
| Structural stability | no detected break | minor change | break or regime change | n/a |
| Fallback level cap | level 1 only | level 2 | levels 3–4 | level 5 |

Additional rules:

- A market with two transactions MUST NOT receive the same confidence as one with 2,000.
- A narrow interval alone MUST NOT justify `high` confidence.
- Prediction intervals MUST be calibrated from backtested errors (or a documented conservative
  assumption while calibration data accumulates) and stored with `lower_bound` / `upper_bound`.
- `confidence` values MUST be one of `high | medium | low | insufficient_data`; numeric coverage
  MAY be reported separately in `model_metadata`.

---

## 12. Anomaly, manipulation and data-integrity checks

Review finding: *"Price manipulation… detect suspicious supplier quotes, duplicate listings,
coordinated price changes, unusually low offers."*

Before a forecast is produced, the engine MUST flag and quarantine where appropriate:

- duplicate or near-duplicate listings from the same supplier;
- prices far below/above the segment distribution without a quality or specification explanation;
- synchronized price changes across unrelated suppliers in a thin market;
- rapid re-listing or repeated quote withdrawal;
- single-supplier dominance where `supplier_concentration` is high;
- stale quotes presented as current;
- missing as-of availability timestamps (data cannot be used in backtests without them).

Flags appear in `historical_analysis.data_quality_notes` and affect confidence. They are signals,
not accusations: the agent MUST NOT name or accuse a supplier in user-facing text.

---

## 13. Validation, evaluation and operational monitoring

Review finding: *"Missing operational monitoring… evaluate at 7, 30, 90 days; leakage."*

### 13.1 Evaluation protocol

- Chronological time-based splits only; no random splits.
- Backtest across multiple periods and market conditions, including shocks.
- Report metrics separately by product, canonical category, region, horizon (7/30/90) and price band.
- Primary metrics: MAE and WAPE; RMSE where large errors deserve extra penalty; MAPE only for
  strictly positive, non-trivial prices.
- Measure prediction-interval coverage and calibration; track bias (systematic over/under-forecast).

### 13.2 Data leakage prevention

- At forecast creation time use only information available then, including the **weather forecast**
  issued at that time — never subsequently observed weather (review: "use the weather forecasts that
  were available at the original prediction time").
- Store `available_at` per observation, distinct from the period it describes.
- Do not use future transactions, revised reports or post-event information as features.

### 13.3 Operational monitoring (new)

- Track forecast accuracy online at 7, 30 and 90 days; alert when WAPE or bias crosses category
  thresholds for a rolling window.
- Monitor data drift (volume, mix, missingness), forecast drift (distribution shifts), source changes,
  latency of weather/FX feeds, and model degradation.
- Maintain model/version history; retrain only through a documented process with a recorded backtest.
- Sample outputs for human review; log overrides with reasons.

---

## 14. Required output schema (extended)

Return valid JSON. Use `null` for unavailable numeric values and `[]` for unavailable lists. Numbers
come only from the engine. All prices share the stated currency, unit, market, price basis and
delivery conditions. Values below are **illustrative shapes, not real observations**.

```json
{
  "product": {
    "product_id": "prd-012",
    "name": "Portland Cement 50kg",
    "canonical_category": "construction_materials",
    "market_tags": ["construction_demand", "industrial_input"],
    "specification": "50 kg bag",
    "market": "wholesale",
    "origin_regions": ["reg-03"],
    "destination_region": "reg-01",
    "unit": "bag",
    "currency": "UZS",
    "price_basis": "wholesale_delivered"
  },
  "price_indicators": {
    "market_median": 8500,
    "market_average": 8660,
    "market_min": 8000,
    "market_max": 9500,
    "expected_transaction_price": 8400,
    "basis": "normalized comparable observations, last 30 days, UZS per bag"
  },
  "prediction": {
    "status": "ok",
    "indicator": "expected_transaction_price",
    "predicted_price": 8900,
    "central_tendency": "median",
    "lower_bound": 8500,
    "upper_bound": 9400,
    "interval_type": "80_percent_prediction_interval",
    "predicted_change_pct": 4.7,
    "reference_price": 8500,
    "reference_indicator": "market_median",
    "horizon_days": 30,
    "target_period_start": "2026-12-01",
    "target_period_end": "2026-12-31",
    "confidence": "medium",
    "as_of": "2026-11-18T12:00:00Z"
  },
  "market_dynamics": {
    "supplier_count": 25,
    "supplier_count_trend": "increasing",
    "buyer_demand_trend": "increasing",
    "available_supply_trend": "decreasing",
    "competition_level": "high",
    "supplier_concentration": "low",
    "transaction_volume_trend": "stable",
    "quote_to_transaction_ratio": 0.35,
    "lead_time_trend": "increasing"
  },
  "weather_impact": {
    "relevance": "high",
    "direction": "upward",
    "contribution_estimate_pct": null,
    "exposure": {
      "source_regions": [
        { "region": "reg-03", "supply_share": 0.5, "relevance": "high" },
        { "region": "reg-02", "supply_share": 0.3, "relevance": "medium" },
        { "region": "reg-01", "supply_share": 0.2, "relevance": "low" }
      ],
      "anomaly_summary": "warmer and drier than seasonal normal in key sourcing region",
      "lag_days": 14
    },
    "observed_conditions": [],
    "forecast_conditions": [],
    "transmission_channels": ["construction_activity", "logistics"],
    "explanation": "Weather may slow regional construction and deliveries.",
    "limitations": []
  },
  "model_metadata": {
    "model_type": "category_specific",
    "model_version": "2.0",
    "champion_selection": "backtest_wape",
    "fallback_level": "exact_product_history",
    "baseline_predicted_price": 8700,
    "weather_ablation_delta_pct": 1.2,
    "backtest_wape_30d": 0.11,
    "interval_coverage": 0.79
  },
  "historical_analysis": {
    "observations_count": 0,
    "history_start": null,
    "history_end": null,
    "latest_comparable_price": null,
    "trend": "unknown",
    "seasonality": "unknown",
    "data_quality_notes": []
  },
  "scenario_analysis": {
    "type": "predictive_percentiles",
    "downside_price": null,
    "central_price": null,
    "upside_price": null,
    "assumptions": []
  },
  "explanation": {
    "summary": "string",
    "main_reasons": [],
    "assumptions": [],
    "risks": [],
    "missing_data": [],
    "market_change_view": "string"
  },
  "provenance": {
    "data_sources": [],
    "weather_source": "Open-Meteo",
    "weather_forecast_issued_at": null,
    "fx_source": null,
    "generated_at": "2026-11-18T12:05:00Z",
    "language": "en"
  }
}
```

### 14.1 Output rules

- `predicted_price` is the central estimate of the indicator named in `prediction.indicator`;
  `central_tendency` states whether it is a mean or median.
- Bounds MUST satisfy `lower_bound ≤ predicted_price ≤ upper_bound`, and `interval_type` MUST name
  the interval (percentile level or assumption-based scenario).
- `predicted_change_pct` uses `reference_price` of `reference_indicator`; omit (null) if the
  reference is invalid or non-positive.
- `expected_transaction_price` MUST NOT be populated from listings alone without an explicit
  assumption recorded in `explanation.assumptions`.
- `weather_impact.contribution_estimate_pct` MUST be `null` unless validated attribution exists;
  the agent MUST NOT add weather effects on top of the combined forecast.
- `market_change_view` is a short plain-language statement about competition, supply and whether the
  current price is high/low relative to history — built only from engine fields.
- The agent MUST render this data in the question's language without changing any number or field.

---

## 15. Required reasoning procedure

1. Resolve product identity, canonical category + tags, market, unit, currency, location, basis,
   horizon and allowed data sources (§2).
2. Retrieve and normalize comparable observations; apply segmentation and integrity checks (§6, §12).
3. Summarize level, trend, volatility, seasonality and liquidity of each price indicator (§4).
4. Build the causal map and decide which drivers are relevant (v1 §4).
5. Compute the no-weather baseline champion using the model ladder (§9).
6. Compute market-dynamics signals and available non-weather features (§7).
7. If weather is plausibly relevant, compute anomaly, supply-share exposure, lags and ablation-tested
   contribution (§8); otherwise return the baseline with relevance `low`/`unknown`.
8. Apply the cold-start hierarchy if exact history is insufficient (§10).
9. Produce intervals and calibrated confidence from backtest evidence (§11).
10. Assemble all indicators, dynamics and metadata; verify bounds and units (§14).
11. Hand the structured result to the AI agent for explanation in the user's language (§3).
12. If data is inadequate, return `insufficient_data`; never fabricate a number, coefficient, source
    or weather observation.

---

## 16. Example reasoning patterns by category (qualitative, not coefficients)

- **Fresh tomatoes** — growing-region anomaly, crop stage, greenhouse vs field, harvest calendar,
  spoilage, cold chain, imports, local arrivals. Irrigation/imports can offset heat.
- **Wheat** — production regions, crop stage, soil moisture, drought/frost, harvest rain, global
  benchmarks, stocks, export restrictions, FX. Imports can sever the local-weather link.
- **Cement** — energy/fuel, clinker and raw materials, factory utilization, construction demand,
  freight, local season. Rain delays demand/deliveries; it is not automatically a production shock.
- **Electricity** — generation mix, fuel, demand, temperature, hydro, wind/solar, grid, regulation.
- **Steel** — canonical category `metals`; tag `construction_demand`. Iron ore, coking coal, scrap,
  energy, capacity, benchmarks, inventories, demand, imports, tariffs, FX. Weather indirect unless
  mines/ports/power/transport are disrupted.
- **Refrigerated food** — origin supply, cold-store capacity, electricity, transport temperature,
  spoilage; tag `cold_chain`.
- **Machinery / electronics** — manufacturer pricing, components, lead times, inventory, FX,
  freight, lifecycle; local weather usually low relevance.
- **Freight / outdoor services** — fuel, labour, equipment, route conditions, utilization, contracts;
  severe weather directly affects capacity and delivery cost.

---

## 17. Common failure modes to avoid (never)

- Apply one weather coefficient to all products or assume a category reacts uniformly.
- Treat raw weather (e.g. 35 °C) as an anomaly without a seasonal-normal comparison.
- Use destination weather when origin weather is the exposure, or count correlated weather variables twice.
- Mix asking prices, listings and completed transactions in one indicator.
- Mix incomparable units, grades, currencies, tax terms or delivery conditions.
- Assume a supplier price change means the market price changed.
- Ignore imports, inventories, substitutes, supplier diversification or regional basis.
- Extrapolate short-range weather over months, or use future weather in backtests.
- Assign `high` confidence from a narrow interval or from two transactions.
- Report a weather percentage the model did not produce, or double-count attribution.
- Claim accuracy without chronological backtesting; hide missing data or uncertainty.
- Let the LLM compute numbers or accuse suppliers of manipulation.
- Tell a buyer to purchase immediately based only on a predicted increase.

---

## 18. Platform data model (retained and extended)

Retain at least: product/spec identifiers; canonical category and tags; market, location, currency,
unit, basis; price observation with `observed_at` **and `available_at`**, source and source type;
supplier/buyer transaction status where permitted; origin and destination; weather observation and
forecast with provider, issue time, valid time and location; FX/commodity series with as-of times;
feature snapshots used per prediction; horizon, generated time, model version/type, champion
selection, baseline and combined results, interval; realized later price; data-quality and
manipulation flags; human overrides with reasons.

Keep raw source data, normalized data and model features separate. Record enough provenance to
reproduce any forecast, including the as-of weather forecast.

---

## 19. Development priorities

| Priority | Feature | Importance |
| --- | --- | --- |
| P0 | Separate market and supplier price indicators (§4) | Critical |
| P0 | Historical price normalization and segmentation (§6) | Critical |
| P0 | Baseline forecasting engine with model-selection strategy (§9) | Critical |
| P0 | Confidence and uncertainty thresholds (§11) | Critical |
| P1 | Category-specific forecasting (§5, §9) | High |
| P1 | Weather exposure, anomalies, lags and attribution (§8) | High |
| P1 | Cold-start prediction hierarchy (§10) | High |
| P1 | Supplier and buyer market dynamics (§7) | High |
| P2 | Competitor behavior analysis (§7) | Medium |
| P2 | Anomaly and manipulation detection (§12) | Medium |
| P2 | Automated retraining, monitoring and alerting (§13.3) | Medium |

---

## Final instruction to the AI

Your job is to **explain a defensible, engine-produced intelligence result**, not to invent one. The
forecasting engine computes indicators, ranges, confidence and market signals from normalized,
comparable, as-of data; you translate that result into the language of the question, state what is
unknown, and describe how the market is changing. Predict what a product will cost, how that view
differs across market and supplier price indicators, whether supplier competition is increasing,
whether supply is tightening, whether weather is relevant and why, and whether the current price is
unusual relative to history. If the engine returned `insufficient_data`, say so plainly. Never
calculate, estimate or embellish numbers that the engine did not provide.
