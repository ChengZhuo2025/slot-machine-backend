-- 003_devices.sql
-- 商户、场地、设备种子数据

-- 商户数据
INSERT INTO merchants (name, contact_name, contact_phone, address, commission_rate, settlement_type, status) VALUES
    ('深圳智慧酒店管理有限公司', '王经理', '13800000001', '深圳市南山区科技园南路100号', 0.2000, 'monthly', 1),
    ('广州情趣生活连锁', '李总', '13800000002', '广州市天河区天河路200号', 0.1500, 'weekly', 1),
    ('北京浪漫时光商贸', '张总', '13800000003', '北京市朝阳区建国路88号', 0.1800, 'monthly', 1),
    ('上海都市生活服务', '赵经理', '13800000004', '上海市浦东新区陆家嘴环路100号', 0.2000, 'monthly', 1),
    ('成都休闲娱乐管理', '刘总', '13800000005', '成都市锦江区红星路88号', 0.2200, 'monthly', 1)
ON CONFLICT DO NOTHING;

-- 场地数据
INSERT INTO venues (merchant_id, name, type, province, city, district, address, longitude, latitude, contact_name, contact_phone, status)
SELECT id, '南山科技园智能柜点', 'office', '广东省', '深圳市', '南山区', '科技园南路100号A栋1楼', 113.9447, 22.5405, '张店长', '13811111111', 1
FROM merchants WHERE name = '深圳智慧酒店管理有限公司'
ON CONFLICT DO NOTHING;

INSERT INTO venues (merchant_id, name, type, province, city, district, address, longitude, latitude, contact_name, contact_phone, status)
SELECT id, '福田CBD智能柜点', 'mall', '广东省', '深圳市', '福田区', '福华路200号购物中心B1层', 114.0579, 22.5431, '李店长', '13811111112', 1
FROM merchants WHERE name = '深圳智慧酒店管理有限公司'
ON CONFLICT DO NOTHING;

INSERT INTO venues (merchant_id, name, type, province, city, district, address, longitude, latitude, contact_name, contact_phone, status)
SELECT id, '天河城智能柜点', 'mall', '广东省', '广州市', '天河区', '天河路208号天河城B2层', 113.3291, 23.1377, '王店长', '13811111113', 1
FROM merchants WHERE name = '广州情趣生活连锁'
ON CONFLICT DO NOTHING;

INSERT INTO venues (merchant_id, name, type, province, city, district, address, longitude, latitude, contact_name, contact_phone, status)
SELECT id, '浪漫酒店专柜', 'hotel', '广东省', '广州市', '越秀区', '环市东路333号浪漫酒店', 113.2765, 23.1436, '陈经理', '13811111114', 1
FROM merchants WHERE name = '广州情趣生活连锁'
ON CONFLICT DO NOTHING;

INSERT INTO venues (merchant_id, name, type, province, city, district, address, longitude, latitude, contact_name, contact_phone, status)
SELECT id, '国贸智能柜点', 'mall', '北京市', '朝阳区', '朝阳区', '建国门外大街1号国贸商城B1', 116.4551, 39.9084, '周店长', '13811111115', 1
FROM merchants WHERE name = '北京浪漫时光商贸'
ON CONFLICT DO NOTHING;

-- 设备数据
INSERT INTO devices (device_no, name, type, model, venue_id, qr_code, product_name, product_image, slot_count, available_slots, online_status, status)
SELECT
    'DEV-SZ-NAS-001',
    '南山科技园1号柜',
    'standard',
    'SL-1000',
    id,
    'https://qr.example.com/dev-sz-nas-001',
    '情趣按摩器',
    'https://img.example.com/product/1.jpg',
    4,
    4,
    1,
    1
FROM venues WHERE name = '南山科技园智能柜点'
ON CONFLICT DO NOTHING;

INSERT INTO devices (device_no, name, type, model, venue_id, qr_code, product_name, product_image, slot_count, available_slots, online_status, status)
SELECT
    'DEV-SZ-NAS-002',
    '南山科技园2号柜',
    'mini',
    'SL-500',
    id,
    'https://qr.example.com/dev-sz-nas-002',
    '成人情趣套装',
    'https://img.example.com/product/2.jpg',
    2,
    2,
    1,
    1
FROM venues WHERE name = '南山科技园智能柜点'
ON CONFLICT DO NOTHING;

INSERT INTO devices (device_no, name, type, model, venue_id, qr_code, product_name, product_image, slot_count, available_slots, online_status, status)
SELECT
    'DEV-SZ-FT-001',
    '福田CBD1号柜',
    'premium',
    'SL-2000',
    id,
    'https://qr.example.com/dev-sz-ft-001',
    '高端情趣用品',
    'https://img.example.com/product/3.jpg',
    6,
    6,
    1,
    1
FROM venues WHERE name = '福田CBD智能柜点'
ON CONFLICT DO NOTHING;

INSERT INTO devices (device_no, name, type, model, venue_id, qr_code, product_name, product_image, slot_count, available_slots, online_status, status)
SELECT
    'DEV-GZ-TH-001',
    '天河城1号柜',
    'standard',
    'SL-1000',
    id,
    'https://qr.example.com/dev-gz-th-001',
    '情趣内衣套装',
    'https://img.example.com/product/4.jpg',
    4,
    3,
    1,
    1
FROM venues WHERE name = '天河城智能柜点'
ON CONFLICT DO NOTHING;

INSERT INTO devices (device_no, name, type, model, venue_id, qr_code, product_name, product_image, slot_count, available_slots, online_status, status)
SELECT
    'DEV-GZ-LM-001',
    '浪漫酒店1号柜',
    'standard',
    'SL-1000',
    id,
    'https://qr.example.com/dev-gz-lm-001',
    '酒店专供情趣套装',
    'https://img.example.com/product/5.jpg',
    4,
    4,
    1,
    1
FROM venues WHERE name = '浪漫酒店专柜'
ON CONFLICT DO NOTHING;

INSERT INTO devices (device_no, name, type, model, venue_id, qr_code, product_name, product_image, slot_count, available_slots, online_status, rental_status, status)
SELECT
    'DEV-BJ-GM-001',
    '国贸1号柜',
    'premium',
    'SL-2000',
    id,
    'https://qr.example.com/dev-bj-gm-001',
    '高端震动按摩器',
    'https://img.example.com/product/6.jpg',
    6,
    5,
    1,
    1,
    1
FROM venues WHERE name = '国贸智能柜点'
ON CONFLICT DO NOTHING;

-- 更新商户关联的管理员
UPDATE admins SET merchant_id = (SELECT id FROM merchants WHERE name = '深圳智慧酒店管理有限公司')
WHERE username = 'platform';
