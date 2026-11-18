-- BozorPulse demo seed. All price data is labeled as sample/demo data.
-- Passwords: admin/admin123, supplier1/sup123, customer1/cust123.

INSERT INTO regions (id,name,created_at) VALUES
('reg-01','Tashkent','2026-01-01T00:00:00Z'),
('reg-02','Samarkand','2026-01-01T00:00:00Z'),
('reg-03','Bukhara','2026-01-01T00:00:00Z'),
('reg-04','Andijan','2026-01-01T00:00:00Z'),
('reg-05','Fergana','2026-01-01T00:00:00Z')
ON CONFLICT (id) DO NOTHING;

INSERT INTO users (id,username,password_hash,role,status,inn,bank_account,company_name,region_id,phone,created_at) VALUES
('usr-admin','admin','c1c5ac7fff20618cdd515f80b3bd477f:c606517ba6a2762690ea0a2dd3354f12c66a7e2d2f3fd54e8f9bd2c57adb3e97','admin','confirmed','','','BozorPulse Admin','reg-01','','2026-01-01T00:00:00Z'),
('usr-sup1','supplier1','9f5fc115d56c13b4e58ab1bf88503518:a0d7b2f68ad12b2563ad5c0cfe002297d31ed8f04fbeb882a00391e9f19e7bf7','supplier','confirmed','123456789',' account-demo-001 ','Atlas Wholesale LLC','reg-01','+998901111111','2026-01-02T00:00:00Z'),
('usr-cus1','customer1','ae31a89af8859d7367b9a02519e8d900:859e4522d2135417f9442095fef20f691e4c017ff09c77305c838550af961aa9','customer','confirmed','987654321','account-demo-002','Transit Retail Group','reg-01','+998902222222','2026-01-03T00:00:00Z'),
('usr-sup2','supplier2','cb6153f95b500eb81dcfa3cbf294372a:dbad1d3ea5f62d528f37eef0871435adf21a163af0a7f7e05e129911a4ca0cff','supplier','waiting','112233445','account-demo-003','Silk Road Supply','reg-02','+998903333333','2026-01-04T00:00:00Z'),
('usr-cus2','customer2','d51e0ed5a25b9c7304718c443f1323b5:831e3519034b3e3403955ee62d96fb8ab328a0de2430929ae97b28f4c70fc8b7','customer','waiting','556677889','account-demo-004','Steppe Foods LLC','reg-03','+998904444444','2026-01-05T00:00:00Z')
ON CONFLICT (id) DO NOTHING;

INSERT INTO companies (id,owner_user_id,name,inn,bank_account,region_id,created_at) VALUES
('com-adm1','usr-admin','BozorPulse Admin','','','reg-01','2026-01-01T00:00:00Z'),
('com-sup1','usr-sup1','Atlas Wholesale LLC','123456789','account-demo-001','reg-01','2026-01-02T00:00:00Z'),
('com-cus1','usr-cus1','Transit Retail Group','987654321','account-demo-002','reg-01','2026-01-03T00:00:00Z'),
('com-sup2','usr-sup2','Silk Road Supply','112233445','account-demo-003','reg-02','2026-01-04T00:00:00Z'),
('com-cus2','usr-cus2','Steppe Foods LLC','556677889','account-demo-004','reg-03','2026-01-05T00:00:00Z')
ON CONFLICT (id) DO NOTHING;

INSERT INTO categories (id,name,parent_id,created_at) VALUES
('cat-01','Electronics',NULL,'2026-01-01T00:00:00Z'),
('cat-02','Textiles',NULL,'2026-01-01T00:00:00Z'),
('cat-03','Food & Beverage',NULL,'2026-01-01T00:00:00Z'),
('cat-04','Construction',NULL,'2026-01-01T00:00:00Z'),
('cat-05','Home & Kitchen',NULL,'2026-01-01T00:00:00Z'),
('cat-06','Agriculture',NULL,'2026-01-01T00:00:00Z')
ON CONFLICT (id) DO NOTHING;

INSERT INTO products
    (id,supplier_id,category_id,name,sku,description,unit,price,discount_percent,effective_price,stock,region_id,created_at,updated_at)
VALUES
('prd-001','com-sup1','cat-01','Wireless Headphones','EL-WH-100','Over-ear wireless headphones','pcs',28.50,5,27.08,2400,'reg-01','2026-09-01T00:00:00Z','2026-10-15T00:00:00Z'),
('prd-002','com-sup1','cat-02','Cotton T-Shirts','TX-CT-220','100% cotton bulk t-shirts','pcs',6.20,0,6.20,12000,'reg-01','2026-09-02T00:00:00Z','2026-10-16T00:00:00Z'),
('prd-003','com-sup1','cat-05','LED Desk Lamps','HK-LD-310','Adjustable LED desk lamp','pcs',14.80,10,13.32,3200,'reg-01','2026-09-03T00:00:00Z','2026-10-14T00:00:00Z'),
('prd-004','com-sup1','cat-03','Basmati Rice 5kg','FB-BR-500','Premium basmati rice 5kg bag','bag',8.40,0,8.40,9000,'reg-02','2026-09-04T00:00:00Z','2026-10-12T00:00:00Z'),
('prd-005','com-sup1','cat-04','Ceramic Tiles','CN-CT-600','Glazed ceramic floor tiles','sqm',3.90,8,3.59,45000,'reg-01','2026-09-05T00:00:00Z','2026-10-10T00:00:00Z'),
('prd-006','com-sup1','cat-01','Bluetooth Speakers','EL-BS-150','Portable bluetooth speaker','pcs',21.00,5,19.95,4100,'reg-01','2026-09-06T00:00:00Z','2026-10-11T00:00:00Z'),
('prd-007','com-sup1','cat-06','Fresh Apples','AG-FA-100','Seasonal fresh apples','kg',1.40,0,1.40,30000,'reg-04','2026-09-07T00:00:00Z','2026-10-17T00:00:00Z'),
('prd-008','com-sup1','cat-03','Cooking Oil 1L','FB-CO-100','Refined sunflower oil','bottle',2.80,3,2.72,15000,'reg-01','2026-09-08T00:00:00Z','2026-10-13T00:00:00Z'),
('prd-009','com-sup2','cat-02','Woven Fabric Rolls','TX-WF-410','Cotton woven fabric','meter',12.30,0,12.30,15000,'reg-02','2026-09-09T00:00:00Z','2026-10-09T00:00:00Z'),
('prd-010','com-sup2','cat-05','Kitchen Utensil Sets','HK-KU-090','Stainless steel utensil set','set',11.20,5,10.64,2800,'reg-02','2026-09-10T00:00:00Z','2026-10-08T00:00:00Z'),
('prd-011','com-sup2','cat-01','Smart Watches','EL-SW-200','Fitness smart watch','pcs',42.00,7,39.06,1800,'reg-02','2026-09-11T00:00:00Z','2026-10-07T00:00:00Z'),
('prd-012','com-sup2','cat-04','Portland Cement','CN-PC-050','Construction cement 50kg','bag',6.80,0,6.80,80000,'reg-03','2026-09-12T00:00:00Z','2026-10-06T00:00:00Z')
ON CONFLICT (id) DO NOTHING;

UPDATE products SET delivery_min_days = 3, delivery_max_days = 7
WHERE delivery_min_days = 0 AND delivery_max_days = 0;

INSERT INTO warehouses (id,owner_id,owner_role,name,address,city,region_id,latitude,longitude,created_at) VALUES
('wh-cus1-a','com-cus1','customer','Tashkent Central Warehouse','12 Amir Timur St','Tashkent','reg-01',41.2995,69.2401,'2026-10-01T00:00:00Z'),
('wh-cus1-b','com-cus1','customer','Samarkand Warehouse','5 Registan St','Samarkand','reg-02',39.6270,66.9750,'2026-10-02T00:00:00Z'),
('wh-sup1-a','com-sup1','supplier','Atlas Main Depot','30 Navoi St','Tashkent','reg-01',41.3111,69.2797,'2026-10-03T00:00:00Z'),
('wh-sup1-b','com-sup1','supplier','Atlas Samarkand Depot','8 Bukhara Rd','Samarkand','reg-02',39.6501,66.9603,'2026-10-04T00:00:00Z')
ON CONFLICT (id) DO NOTHING;

INSERT INTO orders (id,customer_id,supplier_id,warehouse_id,status,total_amount,items,created_at,updated_at) VALUES
('ord-001','com-cus1','com-sup1','wh-cus1-a','confirmed',4594.00,
 '[{"product_id":"prd-001","name":"Wireless Headphones","quantity":100,"unit_price":27.08,"line_total":2708.00},{"product_id":"prd-006","name":"Bluetooth Speakers","quantity":100,"unit_price":19.95,"line_total":1995.00}]'::jsonb,
 '2026-10-20T00:00:00Z','2026-10-21T00:00:00Z')
ON CONFLICT (id) DO NOTHING;

INSERT INTO order_items (id,order_id,product_id,quantity,unit_price,line_total) VALUES
('oi-001','ord-001','prd-001',100,27.08,2708.00),
('oi-002','ord-001','prd-006',100,19.95,1995.00)
ON CONFLICT (id) DO NOTHING;

INSERT INTO payments (id,order_id,amount,method,bank_account,status,created_at) VALUES
('pay-001','ord-001',4594.00,'bank_transfer','account-demo-002','pending','2026-10-21T00:00:00Z')
ON CONFLICT (id) DO NOTHING;

INSERT INTO stock_items (warehouse_id,product_id,quantity,updated_at) VALUES
('wh-sup1-a','prd-001',2400,'2026-10-15T00:00:00Z'),
('wh-sup1-a','prd-002',12000,'2026-10-16T00:00:00Z'),
('wh-sup1-b','prd-004',9000,'2026-10-12T00:00:00Z'),
('wh-cus1-a','prd-001',100,'2026-10-21T00:00:00Z')
ON CONFLICT (warehouse_id,product_id) DO NOTHING;

INSERT INTO inventory_movements (id,warehouse_id,product_id,quantity,type,note,created_at) VALUES
('mov-001','wh-sup1-a','prd-001',2500,'supply','Initial stock — demo data','2026-10-01T00:00:00Z'),
('mov-002','wh-cus1-a','prd-001',100,'supply','Received from ord-001','2026-10-21T00:00:00Z')
ON CONFLICT (id) DO NOTHING;

-- Clearly labeled sample price observations for charts and market statistics.
INSERT INTO price_samples (id,product_id,supplier_id,region_id,price,source_type,is_demo,observed_at)
SELECT 'ps-' || p.id || '-' || k, p.id, p.supplier_id, p.region_id,
       round((p.price * (1 + 0.004*k))::numeric, 2), 'demo', true,
       to_char(date '2025-11-01' + k * interval '1 month', 'YYYY-MM') || '-01T00:00:00Z'
FROM products p CROSS JOIN generate_series(0,11) AS k
ON CONFLICT (id) DO NOTHING;
