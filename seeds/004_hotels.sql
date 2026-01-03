-- 004_hotels.sql
-- 酒店、房间、时段价格种子数据

-- 酒店数据
INSERT INTO hotels (name, star_rating, province, city, district, address, longitude, latitude, phone, images, facilities, description, commission_rate, status) VALUES
    ('浪漫情侣主题酒店', 4, '广东省', '深圳市', '南山区', '科技园南路88号', 113.9447, 22.5405, '0755-88888888',
     '["https://img.example.com/hotel/1/1.jpg", "https://img.example.com/hotel/1/2.jpg"]',
     '["WiFi", "空调", "浴缸", "情趣设施", "24小时热水"]',
     '深圳首家情趣主题酒店，提供浪漫私密空间', 0.1500, 1),
    ('都市情缘酒店', 3, '广东省', '广州市', '天河区', '天河路199号', 113.3291, 23.1377, '020-88888888',
     '["https://img.example.com/hotel/2/1.jpg", "https://img.example.com/hotel/2/2.jpg"]',
     '["WiFi", "空调", "情趣设施", "24小时热水"]',
     '广州市中心便捷情侣酒店', 0.1200, 1),
    ('月光宝盒主题酒店', 5, '北京市', '朝阳区', '朝阳区', '建国路66号', 116.4551, 39.9084, '010-88888888',
     '["https://img.example.com/hotel/3/1.jpg", "https://img.example.com/hotel/3/2.jpg", "https://img.example.com/hotel/3/3.jpg"]',
     '["WiFi", "空调", "浴缸", "情趣设施", "私人影院", "24小时热水", "迷你吧"]',
     '北京高端情趣主题酒店，私密奢华体验', 0.1800, 1),
    ('爱琴海情侣酒店', 4, '上海市', '浦东新区', '浦东新区', '陆家嘴环路200号', 121.5076, 31.2396, '021-88888888',
     '["https://img.example.com/hotel/4/1.jpg", "https://img.example.com/hotel/4/2.jpg"]',
     '["WiFi", "空调", "浴缸", "情趣设施", "江景房", "24小时热水"]',
     '上海浦东江景情趣酒店', 0.1600, 1)
ON CONFLICT DO NOTHING;

-- 房间数据
INSERT INTO rooms (hotel_id, room_no, room_type, device_id, images, facilities, area, bed_type, max_guests, hourly_price, daily_price, status)
SELECT h.id, '101', '浪漫大床房', d.id,
    '["https://img.example.com/room/1/1.jpg", "https://img.example.com/room/1/2.jpg"]',
    '["WiFi", "空调", "浴缸", "情趣套装", "红酒"]',
    35, '1.8米大床', 2, 128.00, 398.00, 1
FROM hotels h
LEFT JOIN devices d ON d.device_no = 'DEV-SZ-NAS-001'
WHERE h.name = '浪漫情侣主题酒店'
ON CONFLICT DO NOTHING;

INSERT INTO rooms (hotel_id, room_no, room_type, images, facilities, area, bed_type, max_guests, hourly_price, daily_price, status)
SELECT id, '102', '温馨情侣房',
    '["https://img.example.com/room/2/1.jpg"]',
    '["WiFi", "空调", "情趣套装"]',
    28, '1.5米大床', 2, 98.00, 298.00, 1
FROM hotels WHERE name = '浪漫情侣主题酒店'
ON CONFLICT DO NOTHING;

INSERT INTO rooms (hotel_id, room_no, room_type, images, facilities, area, bed_type, max_guests, hourly_price, daily_price, status)
SELECT id, '201', '豪华主题房',
    '["https://img.example.com/room/3/1.jpg", "https://img.example.com/room/3/2.jpg"]',
    '["WiFi", "空调", "浴缸", "情趣套装", "红酒", "投影仪"]',
    45, '2.0米圆床', 2, 198.00, 598.00, 1
FROM hotels WHERE name = '浪漫情侣主题酒店'
ON CONFLICT DO NOTHING;

INSERT INTO rooms (hotel_id, room_no, room_type, images, facilities, area, bed_type, max_guests, hourly_price, daily_price, status)
SELECT id, '101', '标准情侣房',
    '["https://img.example.com/room/4/1.jpg"]',
    '["WiFi", "空调", "情趣套装"]',
    25, '1.5米大床', 2, 78.00, 228.00, 1
FROM hotels WHERE name = '都市情缘酒店'
ON CONFLICT DO NOTHING;

INSERT INTO rooms (hotel_id, room_no, room_type, images, facilities, area, bed_type, max_guests, hourly_price, daily_price, status)
SELECT id, '888', '总统套房',
    '["https://img.example.com/room/5/1.jpg", "https://img.example.com/room/5/2.jpg", "https://img.example.com/room/5/3.jpg"]',
    '["WiFi", "空调", "浴缸", "情趣套装", "红酒", "私人影院", "按摩浴缸", "迷你吧"]',
    80, '2.0米圆床', 2, 388.00, 1288.00, 1
FROM hotels WHERE name = '月光宝盒主题酒店'
ON CONFLICT DO NOTHING;

-- 房间时段价格
INSERT INTO room_time_slots (room_id, duration_hours, price, start_time, end_time, is_active, sort)
SELECT r.id, 2, 128.00, '10:00', '22:00', TRUE, 1
FROM rooms r
JOIN hotels h ON r.hotel_id = h.id
WHERE h.name = '浪漫情侣主题酒店' AND r.room_no = '101'
ON CONFLICT DO NOTHING;

INSERT INTO room_time_slots (room_id, duration_hours, price, start_time, end_time, is_active, sort)
SELECT r.id, 3, 168.00, '10:00', '22:00', TRUE, 2
FROM rooms r
JOIN hotels h ON r.hotel_id = h.id
WHERE h.name = '浪漫情侣主题酒店' AND r.room_no = '101'
ON CONFLICT DO NOTHING;

INSERT INTO room_time_slots (room_id, duration_hours, price, start_time, end_time, is_active, sort)
SELECT r.id, 6, 268.00, '10:00', '22:00', TRUE, 3
FROM rooms r
JOIN hotels h ON r.hotel_id = h.id
WHERE h.name = '浪漫情侣主题酒店' AND r.room_no = '101'
ON CONFLICT DO NOTHING;

INSERT INTO room_time_slots (room_id, duration_hours, price, start_time, end_time, is_active, sort)
SELECT r.id, 12, 358.00, NULL, NULL, TRUE, 4
FROM rooms r
JOIN hotels h ON r.hotel_id = h.id
WHERE h.name = '浪漫情侣主题酒店' AND r.room_no = '101'
ON CONFLICT DO NOTHING;
