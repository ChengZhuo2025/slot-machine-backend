-- 000009_create_marketing.down.sql
-- 回滚营销相关表

ALTER TABLE orders DROP CONSTRAINT IF EXISTS fk_order_coupon;

DROP TABLE IF EXISTS member_packages;
DROP TABLE IF EXISTS campaigns;
DROP TABLE IF EXISTS user_coupons;
DROP TABLE IF EXISTS coupons;
