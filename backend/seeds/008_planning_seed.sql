-- Demo planning data for customer1. All projections are labeled demo data.
--   psql "$DATABASE_URL" -f backend/seeds/008_planning_seed.sql
INSERT INTO plans (id,company_id,customer_id,title,region_id,status,created_at,updated_at) VALUES
 ('pln-demo-001','com-cus1','usr-cus1','Q4 restock plan','reg-01','active',
  '2026-11-17T09:00:00Z','2026-11-18T09:00:00Z')
ON CONFLICT (id) DO NOTHING;

INSERT INTO plan_items (
 id,plan_id,product_id,product_name,unit,quantity,target_date,note,
 current_unit_price,predicted,low,high,change_pct,trend_pct,horizon_days,
 estimated_budget,budget_low,budget_high,saving_vs_now,
 recommendation,reason,data_status,basis,projection_note,generated_at) VALUES
 ('pli-demo-001','pln-demo-001','prd-001','Wireless Headphones','pcs',120,'2026-12-10','Before the holiday season',
  27.08,28.44,26.98,29.90,5.02,2.50,30,
  3412.80,3237.60,3588.00,-163.20,
  'monitor','No clear price signal — compare live offers before buying.','demo',
  'Based on demo price samples for demonstration. Estimate only — not a supplier quote.',
  'Demo projection for the planning module.','2026-11-18T09:00:00Z'),
 ('pli-demo-002','pln-demo-001','prd-004','Basmati Rice 5kg','bag',300,'2026-12-01','Seasonal stock-up',
  8.40,7.98,7.55,8.40,-5.00,-3.00,30,
  2394.00,2265.00,2520.00,126.00,
  'monitor','No clear price signal — compare live offers before buying.','demo',
  'Based on demo price samples for demonstration. Estimate only — not a supplier quote.',
  'Demo projection for the planning module.','2026-11-18T09:00:00Z')
ON CONFLICT (id) DO NOTHING;
