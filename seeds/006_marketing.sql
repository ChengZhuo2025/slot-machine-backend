-- 006_marketing.sql
-- 优惠券、活动种子数据

-- 优惠券模板
INSERT INTO coupons (name, type, value, min_amount, max_discount, applicable_type, total_count, issued_count, used_count, per_user_limit, start_time, end_time, validity_days, status) VALUES
    ('新用户专享10元券', 'fixed', 10.00, 50.00, NULL, 'all', 10000, 500, 100, 1, '2026-01-01 00:00:00+08', '2026-12-31 23:59:59+08', 30, 1),
    ('满100减20', 'fixed', 20.00, 100.00, NULL, 'mall', 5000, 1000, 300, 3, '2026-01-01 00:00:00+08', '2026-06-30 23:59:59+08', NULL, 1),
    ('租借9折券', 'percent', 0.90, 0.00, 50.00, 'rental', 3000, 800, 200, 2, '2026-01-01 00:00:00+08', '2026-06-30 23:59:59+08', 15, 1),
    ('酒店预订8折券', 'percent', 0.80, 100.00, 100.00, 'hotel', 2000, 300, 80, 1, '2026-01-01 00:00:00+08', '2026-03-31 23:59:59+08', NULL, 1),
    ('满200减50大额券', 'fixed', 50.00, 200.00, NULL, 'all', 1000, 100, 20, 1, '2026-01-15 00:00:00+08', '2026-02-28 23:59:59+08', 7, 1),
    ('会员专享95折', 'percent', 0.95, 0.00, 30.00, 'all', 99999, 2000, 500, 99, '2026-01-01 00:00:00+08', '2026-12-31 23:59:59+08', NULL, 1)
ON CONFLICT DO NOTHING;

-- 用户优惠券
INSERT INTO user_coupons (user_id, coupon_id, status, expire_at)
SELECT u.id, c.id, 0, c.end_time
FROM users u, coupons c
WHERE u.phone = '13800138001' AND c.name = '新用户专享10元券'
ON CONFLICT DO NOTHING;

INSERT INTO user_coupons (user_id, coupon_id, status, expire_at)
SELECT u.id, c.id, 0, c.end_time
FROM users u, coupons c
WHERE u.phone = '13800138001' AND c.name = '满100减20'
ON CONFLICT DO NOTHING;

INSERT INTO user_coupons (user_id, coupon_id, status, expire_at)
SELECT u.id, c.id, 0, c.end_time
FROM users u, coupons c
WHERE u.phone = '13800138002' AND c.name = '租借9折券'
ON CONFLICT DO NOTHING;

INSERT INTO user_coupons (user_id, coupon_id, status, expire_at)
SELECT u.id, c.id, 0, c.end_time
FROM users u, coupons c
WHERE u.phone = '13800138003' AND c.name = '会员专享95折'
ON CONFLICT DO NOTHING;

-- 营销活动
INSERT INTO campaigns (name, type, description, rules, start_time, end_time, status) VALUES
    ('春节大促', 'discount', '春节期间全场商品8折优惠',
     '{"discount": 0.8, "applicable": "all", "exclude_categories": []}',
     '2026-01-20 00:00:00+08', '2026-02-10 23:59:59+08', 1),
    ('新品上市送积分', 'points', '购买新品双倍积分',
     '{"points_multiplier": 2, "applicable": "new_products"}',
     '2026-01-01 00:00:00+08', '2026-03-31 23:59:59+08', 1),
    ('分享有礼', 'gift', '分享商品到朋友圈获得优惠券',
     '{"coupon_id": 1, "share_count": 3}',
     '2026-01-01 00:00:00+08', '2026-06-30 23:59:59+08', 1)
ON CONFLICT DO NOTHING;

-- 会员套餐
INSERT INTO member_packages (name, target_level_id, price, duration_days, bonus_points, description, is_active)
SELECT '银卡会员月卡', id, 29.00, 30, 100, '开通银卡会员，享98折优惠', TRUE
FROM member_levels WHERE name = '银卡会员'
ON CONFLICT DO NOTHING;

INSERT INTO member_packages (name, target_level_id, price, duration_days, bonus_points, description, is_active)
SELECT '银卡会员季卡', id, 79.00, 90, 300, '开通银卡会员季卡，享98折优惠，赠送300积分', TRUE
FROM member_levels WHERE name = '银卡会员'
ON CONFLICT DO NOTHING;

INSERT INTO member_packages (name, target_level_id, price, duration_days, bonus_points, description, is_active)
SELECT '金卡会员月卡', id, 59.00, 30, 200, '开通金卡会员，享95折优惠', TRUE
FROM member_levels WHERE name = '金卡会员'
ON CONFLICT DO NOTHING;

INSERT INTO member_packages (name, target_level_id, price, duration_days, bonus_points, description, is_active)
SELECT '金卡会员年卡', id, 499.00, 365, 2000, '开通金卡会员年卡，享95折优惠，赠送2000积分', TRUE
FROM member_levels WHERE name = '金卡会员'
ON CONFLICT DO NOTHING;

INSERT INTO member_packages (name, target_level_id, price, duration_days, bonus_points, description, is_active)
SELECT '钻石会员年卡', id, 999.00, 365, 5000, '开通钻石会员年卡，享9折优惠，赠送5000积分，专属客服', TRUE
FROM member_levels WHERE name = '钻石会员'
ON CONFLICT DO NOTHING;
