# Price Prediction Rules v2 (Weather + Historical Prices)

Rules for the AI component of a B2B platform that predicts product prices from historical prices and weather. Read Part A fully, then load only the category block (Part B) that matches the product.

**Role of this document:** category calibration playbook — price types, weather windows, caps, pass-through, per-category direction/modifiers and the calibration table. The system specification, five price indicators, market dynamics, canonical output schema, evaluation and monitoring live in `sources/price_prediction_rules-chatgpt.md`; where the two overlap, that file is authoritative for system contracts. Categories here are **category blocks/tags**; the platform still assigns exactly one canonical category per product as defined there. Implementation configuration lives in `backend/internal/modules/ai/infrastructure/forecast/calibration.json`.

**Active policy decision:** all `[calibrate]` values (β, caps, lags, pass-through) are inert until filled from backtesting. Until β exists, weather adjustment is **zero**, `fallback_used` is reserved for product-level cold start, and the output is labeled an analytical estimate — the z-score fallback formula in §6 is not applied.

Values marked **[calibrate]** are starting priors, not facts. Replace them with values learned from the platform's own data (Part A, Section 14).

---

# PART A: CORE RULES (apply to every product)

## 1. Roles: Who Does What

Numbers come from models. The AI selects rules, enforces limits, and explains.

| Component | Responsibility |
|---|---|
| **Statistical / ML model** | Baseline forecast, seasonality, weather coefficients (β), forecast intervals, ensemble spread |
| **AI (LLM) following these rules** | Classify the product, choose the weather windows and variables, check data quality, apply caps and safeguards, resolve conflicts, write the explanation, flag warnings |

Hard rules for the roles:
- The AI **must not invent** a price, coefficient, or weather value. If the model did not provide a number, use the fallback in Section 6 and label it `fallback_used: true`.
- The AI **may only cite drivers** that appear in the model output or input data.
- If no model output is available at all, the AI may only return a qualitative outlook (`up / down / neutral`, with confidence) and must not output price numbers.

---

## 2. Required Inputs

### 2.1 Product data
| Field | Description |
|---|---|
| `product_id`, `category`, `subcategory` | Identifiers and classification |
| `price_type` | `spot`, `contract`, `list`, `negotiated`, `auction` (see Section 4) |
| `unit`, `currency`, `incoterm` | Normalize before comparing |
| `quality_grade`, `volume_tier`, `min_order_qty`, `payment_terms` | Normalize or segment; never mix grades or tiers in one series |
| `price_history` | At least 24 months if seasonal, 90 days otherwise |
| `origin_regions` | Production or sourcing regions, with weights (e.g. 60% region A, 40% import) |
| `sale_regions` | Where it is sold |
| `route_regions` | Key transport corridors, ports, rivers |
| `perishability`, `storable_days` | Shelf life and stockpiling ability |
| `import_share`, `export_share` | Exposure to foreign weather and trade |
| `regulated` | `none`, `capped`, `fixed`, `contracted` |
| `weather_enabled` | Boolean; see Section 5 |
| `weather_windows` | Phenology/calendar windows (Section 7) |
| `hemisphere`, `climate_zone` | Needed to read seasons correctly |
| `water_source` | `rainfed`, `irrigated_river`, `irrigated_groundwater`, `greenhouse`, `none` |

### 2.2 Weather data
- Observed: temperature (min/max/mean), precipitation, humidity, wind, snow cover and snowpack, frost days, heat-wave days, soil moisture, river flow, reservoir levels, drought indices, extreme-event flags.
- Forecasts: short-term (1-14 days) and seasonal (1-6 months), **preferably as ensembles** (multiple members) so that spread can be measured.
- Climatology: at least 10 years per region and variable, used to compute anomalies.
- Weather for origin, sale, and route regions separately.

### 2.3 Optional context
FX rates, fuel and gas prices, holidays, inventory, trade policy, competitor prices, platform order volume.

---

## 3. Core Principles

1. **History first.** Build the baseline from historical prices (trend + seasonality) before using weather.
2. **Weather adjusts the baseline.** Use **anomalies** (deviation from normal for that time and place), never raw weather values.
3. **Respect lags.** Apply the lag defined for the category, not an instant effect (except demand-driven items with short lags).
4. **Origin drives supply; sale region drives demand; routes drive logistics.**
5. **Storability dampens shocks.** Perishable, non-storable goods react fast and strongly. Storable goods react slowly and weakly.
6. **No double counting.** Seasonality is in the baseline. Commodity weather effects are not re-applied to derived products beyond pass-through (Section 9).
7. **Conditional direction.** Direction of effect depends on modifiers (irrigation, stocks, trade, regulation). Apply the modifiers listed in the category block; do not use one-directional shortcuts.
8. **Less change when uncertain.** Weak evidence means staying close to the baseline and widening the range.
9. **Weather is one driver.** Do not credit weather for moves that other factors explain.

---

## 4. Price Types (B2B)

Treat each price type as its own series. Never merge them.

| Price type | How weather and market shifts reach it | Rule |
|---|---|---|
| **Spot** | Fast, direct | Full category lag and sensitivity apply |
| **List** | Slow, stepwise (repriced periodically) | Predict *when* the next repricing is likely and its size; no daily weather moves |
| **Contract** | Only at renewal, or via indexation clauses | Weather effect = 0 until renewal date or index trigger; forecast the renewal level, not interim prices |
| **Negotiated** | Moves with market, plus buyer/seller leverage | Use spot signal as a reference, widen the range |
| **Auction** | Highly sensitive to supply and demand shocks | Use wider ranges than spot |

Additional B2B adjustments (when data exists):
- **Volume tiers and MOQ**: forecast the base tier; apply tier ratios learned from history. Do not apply a tier discount twice.
- **Quality grade**: weather often hits quality (e.g. rain during harvest lowers grain grade) before it hits volume. Premium grades may rise while low grades fall in the same event.
- **Payment terms and incoterms**: normalize before modeling.

---

## 5. Per-Category On/Off Switch

Each category has `weather_enabled`. Use it as follows:

- `weather_enabled = false`: skip all weather adjustment. Forecast from baseline and non-weather factors. Output `weather_adj_pct = 0` and a warning "weather layer disabled for this category".
- Default: `true` for categories with sensitivity **Medium or higher**; `false` for **Low or Minimal** until backtests show an out-of-sample improvement (Section 14).
- Platform team changes the flag; the AI never changes it.
- Even when disabled, **extreme events** (flood at a port, hurricane at a refinery) may be reported as a `warnings` item with a qualitative risk note, without price numbers.

---

## 6. Prediction Procedure

1. **Check data and regulation.** Clean data; confirm `price_type` and `regulated`. If `fixed` or `contracted` (outside renewal), return the fixed price with an availability/supply-risk note and stop.
2. **Classify** the product (Part B). If several categories apply, use the one with higher weather sensitivity and note the second as secondary.
3. **Check `weather_enabled`.** If false, go to step 8.
4. **Select weather features:**
   - which variables (category block),
   - which windows (Section 7),
   - which regions and weights (origin, sale, route).
5. **Compute anomalies** (z-score or percentile vs climatology) per variable and window. Flag extreme events.
6. **Get the weather adjustment (magnitude):**
   - **Preferred:** use model output `weather_adj_pct = Σ β_k × anomaly_k` (β from the calibration table, Section 14), per horizon.
   - **Fallback** (only if no β exists): `weather_adj_pct = direction × cap_h × min(|z|, 3) / 3`, applied only when `|z| ≥ 1`, where `cap_h` is the final cap from Section 8 and `direction` comes from the category block after modifiers. Mark `fallback_used: true` and set confidence to at most `medium`.
7. **Apply the seasonal timing rule** (Section 10) and cross-product effects (Section 11), if relevant.
8. **Apply other factors** (Section 12): FX, energy, policy, etc.
9. **Apply caps** (Section 8): per factor, then total.
10. **Combine:** `expected = baseline × (1 + weather_adj + seasonal_timing_adj + cross_product_adj + fx_adj + energy_adj + other_adj)`, after caps. The output numbers must satisfy this identity.
11. **Uncertainty:** build low/expected/high from the model intervals and widen using the rules in Section 13.
12. **Explain** using only drivers present in the data.
13. **Self-check** (Section 15) before returning.

---

## 7. Weather Windows (Which Period Counts)

Weather matters only inside the **windows** relevant to the product. Each product must carry `weather_windows`. Example schema:

```json
{
  "weather_windows": [
    {
      "name": "flowering_frost",
      "region": "origin",
      "variable": "min_temp_anomaly",
      "start": "03-20",
      "end": "05-10",
      "hemisphere_adjusted": true,
      "lag_days": [14, 60]
    },
    {
      "name": "harvest_rain",
      "region": "origin",
      "variable": "precip_anomaly",
      "start": "09-01",
      "end": "10-31",
      "hemisphere_adjusted": true,
      "lag_days": [0, 21]
    }
  ]
}
```

Rules:
- Dates are **local to the region and hemisphere**. In the southern hemisphere, shift calendar seasons by six months.
- Use local **phenology**: windows must follow the actual crop or product calendar of the origin region, not generic month ranges. If a crop calendar is missing, use the category default window in Part B and set confidence to at most `medium`.
- If today is outside all windows for a product, weather features default to **neutral** (zero adjustment), except for forecast extreme events, demand-driven categories, and logistics.
- Accumulated indicators (soil moisture, degree days, reservoir level, snowpack) can carry effects across windows; treat them as state variables and not as one-off anomalies.
- Source of water matters:
  - `rainfed`: rainfall and soil moisture anomalies are primary.
  - `irrigated_river`: snowpack, upstream river flow, and reservoir levels are primary; local rainfall is secondary.
  - `irrigated_groundwater`: rainfall matters little in the short run; heat stress and energy cost matter more.
  - `greenhouse`: outdoor weather matters mainly through **heating/cooling energy cost** and light.

---

## 8. Limits and Safeguards

### 8.1 Per-factor weather cap
Start from the sensitivity cap, then scale by the product's own volatility.

| Sensitivity | Base cap (normal conditions) **[calibrate]** |
|---|---|
| Very High | ±25% |
| High | ±15% |
| Medium | ±8% |
| Low | ±3% |
| Minimal | ±1% |

```
cap_h = min( base_cap, k × σ_h )
```
- `σ_h`: standard deviation of the product's historical percentage price changes over horizon `h`.
- `k`: 2 **[calibrate]** (so the cap equals two typical moves for that product).
- For products with very low volatility, `cap_h` becomes small automatically; for volatile products it stays near the base cap.

### 8.2 Combined cap
- The **sum of all adjustments** (weather, seasonal timing, cross-product, FX, energy, other) must stay within `total_cap_h = 1.5 × cap_h` **[calibrate]** unless an extreme event or a confirmed structural change (policy, devaluation) is flagged.
- If the sum exceeds `total_cap_h`, scale all adjustments down proportionally and add a warning.

### 8.3 Extreme-event exception
- If a confirmed extreme event exists (flood, frost during flowering, major storm, long heat wave), caps may be exceeded, but the output must set `extreme_event: true`, widen the range, and state the event.
- Confirmation requires at least one of: observed data meeting an extreme threshold, an official weather warning, or a trusted reported event. A forecast alone, beyond 7 days, is not confirmation.

### 8.4 Other safeguards
- **Regulated / contracted prices:** weather impact is zero (Section 4 and step 1 of Section 6).
- **Sparse data:** less than 90 days of history (or less than two seasons for seasonal goods) means half the weather weight, wider range, confidence at most `low`.
- **Conflicting signals:** if the weather signal and history disagree strongly, do not average blindly; explain the conflict, keep the baseline-weighted result, and widen the range.
- **No look-ahead:** in backtests use only information available at the forecast date, including forecasts as issued at that time.

---

## 9. Cost Pass-Through (Derived Products)

For product B made partly from commodity A:

```
price_change_B = cost_share_A × pass_through × price_change_A
```

- `cost_share_A`: share of B's total cost that A represents (0-1).
- `pass_through`: share of A's cost change that B's price actually reflects (0-1), typically 0.3-0.8 **[calibrate]**.
- These are **two separate factors**; do not use a number that already combines them.
- Apply with a lag of 1-6 months depending on inventory and contracts.
- Increases pass through faster than decreases (downward stickiness). Use a decrease pass-through about 0.5-0.8× the increase value **[calibrate]**.
- If A's change was weather-driven, label weather as an **indirect** driver of B. Do not add a separate direct weather adjustment for B unless B also has its own weather exposure (e.g. beverage demand and temperature).

---

## 10. Seasonality and Timing

- Normal seasonal patterns belong in the **baseline**. Do not add them again.
- `seasonal_timing_adj_pct` is allowed only when the **timing or intensity of the season deviates from normal**, for example:
  - an early or late cold onset shifting heating and winter goods demand,
  - an early harvest or late harvest shifting the seasonal price trough,
  - an unusually long summer.
- Compute it as a shift of the seasonal curve (days early/late × the curve's slope), not as a new level adjustment.
- If the season is on time, `seasonal_timing_adj_pct = 0`.

---

## 11. Cross-Product, Cold-Start, and Intermittent Items

**Substitution and linked goods**
- If a close substitute or complement moved sharply because of weather (e.g. tomatoes up 30%), apply a **partial** cross-effect to related products (cucumbers, peppers) using learned cross-elasticities: `cross_product_adj = ε × related_price_change`, with `ε` typically 0.1-0.4 **[calibrate]**.
- Only apply if both products are in the same use case and market. Cap by Section 8.

**Cold start (no or little history)**
- Use the category or subcategory average seasonality and the region's price level as a proxy; set confidence to `low`; widen the range; set `weather_adj` to at most half of the normal cap.
- Return a warning "limited history".

**Intermittent or thin markets**
- If a product has frequent gaps (few transactions), forecast using the last observed price with an explicit uncertainty band; do not interpolate prices as if they were observed. Weather adjustment only for categories with Very High or High sensitivity.

---

## 12. Non-Weather Factors (Always Check)

Where data exists, consider:
- **FX rates** (imports, exports)
- **Fuel and energy prices** (production and transport cost)
- **Holidays and cultural events**
- **Government policy:** tariffs, export bans, subsidies, price controls, taxes, strategic stocks
- **Supply chain events:** strikes, border closures, port congestion
- **Inventory and platform order volume**
- **Competitor and supplier prices**
- **General inflation**

If a non-weather factor clearly explains a price move, do not credit weather for it. Each factor gets its own line in `adjustments`.

---

## 13. Uncertainty Rules

- Base the range on model intervals. If weather forecast **ensembles** are available, include the ensemble spread of the weather feature: wide spread means a wide price range.
- Widen the range when: horizon is long, data is sparse, price type is negotiated/auction, extreme events are flagged, fallback was used, or signals conflict.
- Weather forecast skill drops after about 10 days. For horizons beyond that, use climatology and seasonal outlooks, and widen the range.
- Never predict a specific weather-driven spike more than 2 weeks ahead with high confidence.
- Confidence labels:
  - `high`: ≥2 seasons of history, model β available, short horizon, no conflicts
  - `medium`: one or two weaknesses (fallback used, moderate data, medium horizon)
  - `low`: sparse data, long horizon, conflicts, cold start, or disabled weather with an extreme event risk

---

## 14. Calibration Table (Platform Team Fills This In)

The model learns these from platform data via backtesting. The AI reads them from configuration and must not change them.

| Category | `weather_enabled` | Key variable(s) | β per +1 SD anomaly (%) | Lag (days) | Max `cap_h` | Notes |
|---|---|---|---|---|---|---|
| Fresh produce | TBD | min_temp, heat_days, precip | TBD | TBD | TBD | |
| Grains / oilseeds / pulses | TBD | soil_moisture, heat, harvest_rain | TBD | TBD | TBD | |
| Livestock / dairy / eggs / fish | TBD | heat_stress, pasture_drought | TBD | TBD | TBD | |
| Processed / packaged food | TBD | (via raw material) | TBD | TBD | TBD | |
| Energy | TBD | HDD, CDD, hydro_level | TBD | TBD | TBD | |
| Construction materials | TBD | precip, freeze_days | TBD | TBD | TBD | |
| Metals / mining | TBD | extreme_event flag | TBD | TBD | TBD | |
| Textiles / fibers | TBD | precip, heat | TBD | TBD | TBD | |
| Apparel / seasonal retail | TBD | temp_anomaly (sale region) | TBD | TBD | TBD | |
| Appliances / climate equipment | TBD | heat_wave, cold_snap | TBD | TBD | TBD | |
| Agricultural inputs | TBD | planting_window_weather | TBD | TBD | TBD | |
| Chemicals / plastics / packaging | TBD | extreme_event flag | TBD | TBD | TBD | |
| Electronics / machinery / auto | TBD | seasonal items only | TBD | TBD | TBD | |
| Pharma / medical / hygiene | TBD | seasonal demand | TBD | TBD | TBD | |
| Furniture / household | TBD | (via materials) | TBD | TBD | TBD | |
| Transport / logistics | TBD | storm, snow, river_level | TBD | TBD | TBD | |

Evaluation process:
- Rolling-origin backtests **per category and price type**.
- Metrics: sMAPE, bias, and interval coverage (the 80% range should contain about 80% of actuals).
- Keep the weather layer only if it **beats the baseline-only model out of sample**; otherwise set `weather_enabled = false`.
- Recalibrate β, lags, caps, pass-through, and cross-elasticities quarterly or after regime changes.
- Log all predictions with inputs, adjustments, and model version for audits. Manually review extreme-event cases.

---

## 15. Output Format

Return one JSON object per product, price type, and horizon:

```json
{
  "forecast_date": "YYYY-MM-DD",
  "model_version": "string",
  "product_id": "string",
  "category": "string",
  "price_type": "spot | contract | list | negotiated | auction",
  "horizon": "7d | 30d | 90d | 12m",
  "currency": "string",
  "unit": "string",
  "data_freshness": {
    "last_price_date": "YYYY-MM-DD",
    "last_weather_update": "YYYY-MM-DDTHH:MM",
    "history_length_days": 0
  },
  "baseline_price": 0.0,
  "predicted_price": { "low": 0.0, "expected": 0.0, "high": 0.0 },
  "adjustments_pct": {
    "weather": 0.0,
    "seasonal_timing": 0.0,
    "cross_product": 0.0,
    "fx": 0.0,
    "energy": 0.0,
    "other": 0.0,
    "total_after_caps": 0.0
  },
  "weather_signal": {
    "weather_enabled": true,
    "regions_used": [{ "region": "string", "role": "origin | sale | route", "weight": 0.0 }],
    "windows_active": ["string"],
    "key_anomalies": ["e.g. min temp -3.2 C below normal for 4 days in flowering window"],
    "direction": "up | down | neutral",
    "lag_applied_days": 0,
    "ensemble_spread": "low | medium | high | not_available",
    "extreme_event": false,
    "fallback_used": false
  },
  "confidence": "low | medium | high",
  "confidence_reasons": ["short reasons"],
  "top_drivers": ["ordered list, most important first, only from provided data"],
  "explanation": "2-4 plain sentences understandable by a buyer or seller",
  "warnings": ["e.g. regulated price, contract price until renewal, sparse data, weather layer disabled"]
}
```

Output rules:
- Consistency: `expected = baseline_price × (1 + total_after_caps)`, and `low ≤ expected ≤ high`.
- `explanation` is for business users: no jargon, no invented facts, mention uncertainty honestly.
- Always include `confidence` with at least one reason.
- Add a `warnings` entry whenever any safeguard triggered.
- For `contract` and `list` price types, state the next repricing or renewal date if known.

---

## 16. Self-Check Before Returning

- [ ] Roles respected: all numbers from the model or declared fallback, none invented?
- [ ] `price_type` and `regulated` handled (contract/fixed prices not moved by weather)?
- [ ] `weather_enabled` checked?
- [ ] Baseline built before weather; seasonality not double counted?
- [ ] Correct regions (origin / sale / route) and correct windows for the date and hemisphere?
- [ ] Anomalies used, not raw values; correct lag applied?
- [ ] Direction adjusted by category modifiers (irrigation, stocks, trade, regulation)?
- [ ] Per-factor cap, volatility-scaled cap, and combined cap applied (or extreme event flagged)?
- [ ] Pass-through and cross-product effects not double counted?
- [ ] Non-weather factors considered?
- [ ] Quality grade, volume tier, and incoterm consistent?
- [ ] Uncertainty widened where required; confidence label justified?
- [ ] Output identity holds and all warnings included?

---

## 17. Hard Rules (Never Violate)

1. Never present a prediction as guaranteed.
2. Never invent weather data, prices, coefficients, or events. If data is missing, say so and lower confidence.
3. Never produce price numbers without model output or a declared fallback.
4. Never apply weather to contract or fixed prices before the renewal or index trigger.
5. Never exceed per-factor or combined caps without an extreme-event or structural-change flag.
6. Never apply a weather effect without a category rule, window, and lag.
7. Never credit weather for price changes explained by other factors.
8. Never change `weather_enabled` or calibration values.
9. Always return uncertainty ranges, a confidence level, and warnings.

---

# PART B: CATEGORY BLOCKS

Each block is self-contained. To reduce prompt size, load Part A plus only the matching block. Fields: **sensitivity**, **default `weather_enabled`**, **variables**, **default windows** (used when no local crop calendar exists), **lag**, **direction and modifiers**, **notes**.

## B1. Fresh Produce (vegetables, fruits, greens, berries, melons)
- **Sensitivity:** Very High | **Default enabled:** true
- **Variables:** min temp (frost), max temp (heat), hail, heavy rain, drought/soil moisture, wind (damage).
- **Default windows:** frost risk during flowering and fruit set; heat stress during fruit development; harvest-period rain. Dates from the local crop calendar.
- **Lag:** 0-3 weeks for open-field harvest effects; 1-6 weeks for damage during growth.
- **Direction and modifiers:**
  - Adverse extreme at origin → price **up**.
  - Good conditions and large harvest → price **down**, strongest at harvest peak.
  - `irrigated_*`: drought matters less; heat and water-supply limits matter more.
  - `greenhouse`: outdoor weather matters mainly via heating energy cost.
  - High `import_share`: use exporting-region weather and route disruptions.
- **Notes:** very strong seasonality (learn from history); no stockpiling, so shocks pass through fast and fade fast; check quality grade effects (hail and rain damage grade before volume).

## B2. Grains, Oilseeds, Pulses (wheat, corn, rice, barley, sunflower, soy, lentils)
- **Sensitivity:** High for supply, Medium for price | **Default enabled:** true
- **Variables:** soil moisture, rainfall, heat during flowering and grain fill, winter frost kill, harvest-time rain, snowpack and river flow (irrigated areas).
- **Default windows:** planting moisture, flowering and grain fill, harvest.
- **Lag:** 1-6 months for growing-season effects; weeks for harvest rain.
- **Direction and modifiers:**
  - Drought or heat stress at key stages → price **up**; excellent moisture → price **down**.
  - Weakened by: large **stocks**, import availability, government reserves, export bans.
  - `irrigated_*`: use water supply indicators instead of rainfall.
  - Global producers' weather can dominate if the product is traded internationally.
- **Notes:** FX and global prices matter strongly; policy often overrides weather signals.

## B3. Livestock, Meat, Dairy, Eggs, Fish
- **Sensitivity:** Medium | **Default enabled:** true
- **Variables:** heat stress index, severe cold, pasture drought, feed-crop conditions, sea state (fishing).
- **Default windows:** summer heat stress; winter cold; pasture season.
- **Lag:** dairy yield 1-4 weeks; feed cost 1-3 months; herd size 3-12 months; fishing days to weeks.
- **Direction and modifiers:**
  - Heat and drought → supply and yields down, feed cost up → price **up**.
  - Link to grain and feed prices (Section 9).
  - Demand effects of weather on meat are uncertain; apply only if the platform's own data shows a stable relationship.
- **Notes:** disease outbreaks and import rules often matter more than weather.

## B4. Processed and Packaged Food (flour, bread, oil, sugar, canned food, beverages)
- **Sensitivity:** Low to Medium (indirect) | **Default enabled:** false (true for beverages and ice cream, which have direct temperature demand)
- **Variables:** raw-material prices (via pass-through); sale-region temperature anomaly for beverages and frozen goods.
- **Lag:** 1-6 months from raw material; 0-3 weeks for direct temperature demand.
- **Direction and modifiers:** pass-through per Section 9 (partial, delayed, sticky downward); hot weather → beverage demand **up**.
- **Notes:** contracts, inventory, and margins smooth changes.

## B5. Energy (electricity, gas, coal, fuel, heating oil)
- **Sensitivity:** High for demand, Medium to High for supply | **Default enabled:** true
- **Variables:** heating and cooling degree days, hydro reservoir level and inflow, wind and solar output, storms affecting infrastructure.
- **Default windows:** heating season, cooling season, hydro inflow season.
- **Lag:** demand 0-14 days; hydro 1-6 months.
- **Direction and modifiers:**
  - Extreme cold or heat → price **up**; low hydro/wind or infrastructure damage → price **up**; mild weather → price **down**.
  - `regulated = fixed/capped`: weather effect on price is zero; report availability risk instead.
- **Notes:** many energy prices are administratively set; check this first.

## B6. Construction Materials (cement, bricks, sand, gravel, lumber, rebar, glass, paint)
- **Sensitivity:** Medium | **Default enabled:** true
- **Variables:** temperature, precipitation, freeze days (building season), floods, storms and disasters.
- **Default windows:** building season onset and end; post-disaster 1-6 month window.
- **Lag:** weeks for seasonal demand; 1-6 months for rebuilding surges.
- **Direction and modifiers:**
  - Early good weather → demand **up**; harsh winter → demand and price **down or flat**.
  - Disaster → repair materials **up**.
  - Energy-intensive materials (cement, glass, bricks) also follow energy prices.
- **Notes:** timing effects belong to Section 10 (seasonal timing), not a level shift.

## B7. Metals and Mining Products (steel, aluminum, copper, ores)
- **Sensitivity:** Low to Medium | **Default enabled:** false
- **Variables:** extreme events at mines, ports, rail; hydropower availability for smelters; energy prices.
- **Lag:** days to weeks for disruption; months for energy cost.
- **Direction and modifiers:** supply disruption → price **up**; macro and global market dominate.
- **Notes:** report extreme-event risk as a warning even when disabled.

## B8. Textiles, Fibers, Leather (cotton, wool, fabrics, yarn)
- **Sensitivity:** Medium to High for raw fiber, Low for finished fabric | **Default enabled:** true for raw fiber, false for finished goods
- **Variables:** rainfall, heat, and water supply during growth; harvest rain; livestock conditions for wool and leather.
- **Lag:** 1-6 months for fiber; 3-9 months for finished goods.
- **Direction and modifiers:** adverse growing weather → fiber **up**, passing slowly to fabric; consider irrigation and stocks as in B2.

## B9. Apparel, Footwear, Seasonal Retail Goods
- **Sensitivity:** Medium (demand-driven) | **Default enabled:** true
- **Variables:** temperature anomaly in the **sale region**, season onset timing, snow, rain.
- **Lag:** 0-4 weeks.
- **Direction and modifiers:**
  - Colder than normal at winter season start → winter goods demand **up**; warm winter → demand **down**, discounts likely. Reverse for summer goods.
  - Apply through `seasonal_timing_adj_pct` (Section 10), not as a level change.
  - Mismatched season → overstock → clearance **down** in 1-3 months.

## B10. Appliances and Climate Equipment (AC, heaters, fans, water heaters, generators)
- **Sensitivity:** High (short-term demand) | **Default enabled:** true
- **Variables:** heat waves, cold snaps, storms, power-outage events (sale region).
- **Lag:** 0-3 weeks.
- **Direction and modifiers:** extreme heat → AC/fans **up**; extreme cold → heaters/generators **up**; off-season → **down**.
- **Notes:** check stock levels; high inventory dampens the effect.

## B11. Agricultural Inputs (fertilizer, seeds, pesticides, feed, irrigation equipment)
- **Sensitivity:** Medium | **Default enabled:** true
- **Variables:** planting-season weather, drought, pest-favorable conditions; natural gas and energy prices for fertilizer.
- **Lag:** demand peaks 0-8 weeks around planting; drought affects next-season demand.
- **Direction and modifiers:** good planting conditions → demand and price **up** at season start; drought → irrigation equipment **up**; warm, humid conditions → pesticide demand may rise.
- **Notes:** fertilizer follows gas and global markets strongly.

## B12. Chemicals, Plastics, Packaging
- **Sensitivity:** Low | **Default enabled:** false
- **Variables:** storms and floods affecting plants and ports; energy prices; seasonal packaging demand (harvest, holidays).
- **Direction and modifiers:** disruption → price **up**; otherwise follow trend and energy.

## B13. Electronics, Machinery, Vehicles, Auto Parts
- **Sensitivity:** Minimal to Low | **Default enabled:** false (true only for seasonal items: winter tires, batteries, antifreeze, wipers)
- **Variables:** extreme events at factories and ports; cold-season onset for seasonal parts.
- **Lag:** disruption 2-12 weeks; seasonal demand 0-6 weeks.
- **Direction and modifiers:** seasonal parts rise before and during cold onset (via Section 10); otherwise follow history, FX, and supply chain signals.

## B14. Pharmaceuticals, Medical Supplies, Hygiene
- **Sensitivity:** Low to Medium (seasonal demand) | **Default enabled:** false
- **Variables:** temperature swings, season (cold and flu, allergy, heat-related).
- **Lag:** 0-6 weeks.
- **Direction and modifiers:** seasonal demand surges may raise prices of specific items; **check regulation first**, since many prices are capped.

## B15. Furniture, Household Goods, Home Decor
- **Sensitivity:** Low | **Default enabled:** false
- **Variables:** timber and material costs; spring/summer demand for outdoor furniture.
- **Direction and modifiers:** mostly trend and seasonality; weather acts indirectly through materials (Section 9).

## B16. Transport, Logistics, Delivery Services
- **Sensitivity:** High (short term) | **Default enabled:** true
- **Variables:** storms, snow, ice, floods, fog, low river levels, extreme heat on rails and roads (route regions).
- **Lag:** 0-14 days.
- **Direction and modifiers:** disruption → freight and delivery prices **up**, delivery times **longer**; pass the cost into products dependent on that route (Section 9).

## B17. Default (unclassified)
- **Sensitivity:** Unknown | **Default enabled:** false
- Use baseline and non-weather factors only. If weather is enabled by the platform, cap the adjustment at ±3% **[calibrate]**, set confidence to `low`, and add the warning "category not classified".
