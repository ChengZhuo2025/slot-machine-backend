-- 000007_create_products.down.sql
-- 回滚商品相关表

DROP TRIGGER IF EXISTS update_cart_items_updated_at ON cart_items;
DROP TRIGGER IF EXISTS update_products_updated_at ON products;

DROP TABLE IF EXISTS reviews;
DROP TABLE IF EXISTS cart_items;
DROP TABLE IF EXISTS product_skus;
DROP TABLE IF EXISTS products;
DROP TABLE IF EXISTS categories;
