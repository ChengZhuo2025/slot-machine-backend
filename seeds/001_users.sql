-- 001_users.sql
-- 用户和管理员种子数据

-- 测试用户数据
INSERT INTO users (phone, openid, nickname, avatar, gender, member_level_id, points, status) VALUES
    ('13800138001', 'oXXXX_test_user_001', '张三', 'https://example.com/avatar/1.jpg', 1, 1, 100, 1),
    ('13800138002', 'oXXXX_test_user_002', '李四', 'https://example.com/avatar/2.jpg', 1, 2, 1500, 1),
    ('13800138003', 'oXXXX_test_user_003', '王五', 'https://example.com/avatar/3.jpg', 2, 3, 6000, 1),
    ('13800138004', 'oXXXX_test_user_004', '赵六', 'https://example.com/avatar/4.jpg', 1, 4, 25000, 1),
    ('13800138005', 'oXXXX_test_user_005', '小明', 'https://example.com/avatar/5.jpg', 1, 1, 50, 1),
    ('13800138006', 'oXXXX_test_user_006', '小红', 'https://example.com/avatar/6.jpg', 2, 1, 200, 1),
    ('13800138007', 'oXXXX_test_user_007', '小刚', NULL, 0, 1, 0, 1),
    ('13800138008', 'oXXXX_test_user_008', '小芳', NULL, 2, 2, 2000, 1),
    ('13800138009', NULL, '匿名用户', NULL, 0, 1, 0, 1),
    ('13800138010', 'oXXXX_test_user_010', '测试用户', 'https://example.com/avatar/10.jpg', 1, 1, 500, 1)
ON CONFLICT DO NOTHING;

-- 设置推荐关系（用户3推荐了用户5和用户6）
UPDATE users SET referrer_id = (SELECT id FROM users WHERE phone = '13800138003')
WHERE phone IN ('13800138005', '13800138006');

-- 创建用户钱包
INSERT INTO user_wallets (user_id, balance, frozen_balance, total_recharged, total_consumed)
SELECT id,
    CASE
        WHEN phone = '13800138001' THEN 100.00
        WHEN phone = '13800138002' THEN 500.00
        WHEN phone = '13800138003' THEN 1000.00
        WHEN phone = '13800138004' THEN 2000.00
        ELSE 0.00
    END,
    0.00,
    CASE
        WHEN phone = '13800138001' THEN 200.00
        WHEN phone = '13800138002' THEN 800.00
        WHEN phone = '13800138003' THEN 1500.00
        WHEN phone = '13800138004' THEN 3000.00
        ELSE 0.00
    END,
    CASE
        WHEN phone = '13800138001' THEN 100.00
        WHEN phone = '13800138002' THEN 300.00
        WHEN phone = '13800138003' THEN 500.00
        WHEN phone = '13800138004' THEN 1000.00
        ELSE 0.00
    END
FROM users
WHERE phone LIKE '138001380%'
ON CONFLICT DO NOTHING;

-- 测试用户收货地址
INSERT INTO addresses (user_id, receiver_name, receiver_phone, province, city, district, detail, is_default, tag)
SELECT id, '张三', '13800138001', '广东省', '深圳市', '南山区', '科技园南路100号', TRUE, 'company'
FROM users WHERE phone = '13800138001'
ON CONFLICT DO NOTHING;

INSERT INTO addresses (user_id, receiver_name, receiver_phone, province, city, district, detail, is_default, tag)
SELECT id, '张三', '13800138001', '广东省', '深圳市', '福田区', '福华路200号', FALSE, 'home'
FROM users WHERE phone = '13800138001'
ON CONFLICT DO NOTHING;

INSERT INTO addresses (user_id, receiver_name, receiver_phone, province, city, district, detail, is_default, tag)
SELECT id, '李四', '13800138002', '北京市', '朝阳区', '朝阳区', '建国路88号', TRUE, 'home'
FROM users WHERE phone = '13800138002'
ON CONFLICT DO NOTHING;
