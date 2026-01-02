-- 005_products.sql
-- 商品分类、商品、SKU 种子数据

-- 商品分类
INSERT INTO categories (parent_id, name, icon, sort, level, is_active) VALUES
    (NULL, '情趣玩具', 'https://img.example.com/category/toys.png', 1, 1, TRUE),
    (NULL, '情趣内衣', 'https://img.example.com/category/lingerie.png', 2, 1, TRUE),
    (NULL, '安全护理', 'https://img.example.com/category/care.png', 3, 1, TRUE),
    (NULL, '润滑增强', 'https://img.example.com/category/lubricant.png', 4, 1, TRUE),
    (NULL, '情趣配件', 'https://img.example.com/category/accessories.png', 5, 1, TRUE)
ON CONFLICT DO NOTHING;

-- 二级分类
INSERT INTO categories (parent_id, name, icon, sort, level, is_active)
SELECT id, '震动棒', NULL, 1, 2, TRUE FROM categories WHERE name = '情趣玩具'
ON CONFLICT DO NOTHING;

INSERT INTO categories (parent_id, name, icon, sort, level, is_active)
SELECT id, '跳蛋', NULL, 2, 2, TRUE FROM categories WHERE name = '情趣玩具'
ON CONFLICT DO NOTHING;

INSERT INTO categories (parent_id, name, icon, sort, level, is_active)
SELECT id, '飞机杯', NULL, 3, 2, TRUE FROM categories WHERE name = '情趣玩具'
ON CONFLICT DO NOTHING;

INSERT INTO categories (parent_id, name, icon, sort, level, is_active)
SELECT id, '性感睡衣', NULL, 1, 2, TRUE FROM categories WHERE name = '情趣内衣'
ON CONFLICT DO NOTHING;

INSERT INTO categories (parent_id, name, icon, sort, level, is_active)
SELECT id, '制服诱惑', NULL, 2, 2, TRUE FROM categories WHERE name = '情趣内衣'
ON CONFLICT DO NOTHING;

INSERT INTO categories (parent_id, name, icon, sort, level, is_active)
SELECT id, '避孕套', NULL, 1, 2, TRUE FROM categories WHERE name = '安全护理'
ON CONFLICT DO NOTHING;

INSERT INTO categories (parent_id, name, icon, sort, level, is_active)
SELECT id, '润滑油', NULL, 1, 2, TRUE FROM categories WHERE name = '润滑增强'
ON CONFLICT DO NOTHING;

-- 商品数据
INSERT INTO products (category_id, name, subtitle, images, description, price, original_price, stock, sales, is_on_sale, is_hot, is_new, sort)
SELECT id, '智能震动按摩棒', '10种模式，静音设计，USB充电',
    '["https://img.example.com/product/p1/1.jpg", "https://img.example.com/product/p1/2.jpg"]',
    '这款智能震动按摩棒采用医用级硅胶材质，柔软亲肤。10种振动模式，满足不同需求。静音设计，隐私保护。USB充电，方便快捷。',
    299.00, 399.00, 100, 500, TRUE, TRUE, FALSE, 1
FROM categories WHERE name = '震动棒'
ON CONFLICT DO NOTHING;

INSERT INTO products (category_id, name, subtitle, images, description, price, original_price, stock, sales, is_on_sale, is_hot, is_new, sort)
SELECT id, '无线遥控跳蛋', '10米遥控，7种频率，防水设计',
    '["https://img.example.com/product/p2/1.jpg", "https://img.example.com/product/p2/2.jpg"]',
    '无线遥控设计，10米范围内自由控制。7种振动频率，IPX7级防水。静音马达，安心使用。',
    199.00, 259.00, 200, 800, TRUE, TRUE, TRUE, 2
FROM categories WHERE name = '跳蛋'
ON CONFLICT DO NOTHING;

INSERT INTO products (category_id, name, subtitle, images, description, price, original_price, stock, sales, is_on_sale, is_hot, is_new, sort)
SELECT id, '黑色蕾丝睡裙', '性感透视，舒适面料',
    '["https://img.example.com/product/p3/1.jpg", "https://img.example.com/product/p3/2.jpg"]',
    '精选优质蕾丝面料，柔软舒适。性感透视设计，展现迷人曲线。',
    99.00, 159.00, 150, 300, TRUE, FALSE, TRUE, 1
FROM categories WHERE name = '性感睡衣'
ON CONFLICT DO NOTHING;

INSERT INTO products (category_id, name, subtitle, images, description, price, original_price, stock, sales, is_on_sale, is_hot, is_new, sort)
SELECT id, '超薄避孕套', '超薄0.01，热感设计',
    '["https://img.example.com/product/p4/1.jpg"]',
    '超薄设计，如肌肤般亲密。热感材质，提升体验。',
    59.00, 79.00, 500, 2000, TRUE, TRUE, FALSE, 1
FROM categories WHERE name = '避孕套'
ON CONFLICT DO NOTHING;

INSERT INTO products (category_id, name, subtitle, images, description, price, original_price, stock, sales, is_on_sale, is_hot, is_new, sort)
SELECT id, '水溶性润滑油', '温和无刺激，易清洗',
    '["https://img.example.com/product/p5/1.jpg"]',
    '水溶性配方，温和无刺激。易清洗，不留残留。',
    39.00, 59.00, 300, 1500, TRUE, FALSE, FALSE, 1
FROM categories WHERE name = '润滑油'
ON CONFLICT DO NOTHING;

-- 商品 SKU
INSERT INTO product_skus (product_id, sku_code, attributes, price, stock, is_active)
SELECT id, 'SKU-VIB-PINK', '{"颜色": "粉色"}', 299.00, 50, TRUE
FROM products WHERE name = '智能震动按摩棒'
ON CONFLICT DO NOTHING;

INSERT INTO product_skus (product_id, sku_code, attributes, price, stock, is_active)
SELECT id, 'SKU-VIB-PURPLE', '{"颜色": "紫色"}', 299.00, 50, TRUE
FROM products WHERE name = '智能震动按摩棒'
ON CONFLICT DO NOTHING;

INSERT INTO product_skus (product_id, sku_code, attributes, price, stock, is_active)
SELECT id, 'SKU-EGG-PINK', '{"颜色": "粉色"}', 199.00, 100, TRUE
FROM products WHERE name = '无线遥控跳蛋'
ON CONFLICT DO NOTHING;

INSERT INTO product_skus (product_id, sku_code, attributes, price, stock, is_active)
SELECT id, 'SKU-EGG-RED', '{"颜色": "红色"}', 199.00, 100, TRUE
FROM products WHERE name = '无线遥控跳蛋'
ON CONFLICT DO NOTHING;

INSERT INTO product_skus (product_id, sku_code, attributes, price, stock, is_active)
SELECT id, 'SKU-LACE-S', '{"尺码": "S"}', 99.00, 50, TRUE
FROM products WHERE name = '黑色蕾丝睡裙'
ON CONFLICT DO NOTHING;

INSERT INTO product_skus (product_id, sku_code, attributes, price, stock, is_active)
SELECT id, 'SKU-LACE-M', '{"尺码": "M"}', 99.00, 50, TRUE
FROM products WHERE name = '黑色蕾丝睡裙'
ON CONFLICT DO NOTHING;

INSERT INTO product_skus (product_id, sku_code, attributes, price, stock, is_active)
SELECT id, 'SKU-LACE-L', '{"尺码": "L"}', 99.00, 50, TRUE
FROM products WHERE name = '黑色蕾丝睡裙'
ON CONFLICT DO NOTHING;

INSERT INTO product_skus (product_id, sku_code, attributes, price, stock, is_active)
SELECT id, 'SKU-CONDOM-10', '{"规格": "10只装"}', 59.00, 300, TRUE
FROM products WHERE name = '超薄避孕套'
ON CONFLICT DO NOTHING;

INSERT INTO product_skus (product_id, sku_code, attributes, price, stock, is_active)
SELECT id, 'SKU-CONDOM-20', '{"规格": "20只装"}', 99.00, 200, TRUE
FROM products WHERE name = '超薄避孕套'
ON CONFLICT DO NOTHING;
