-- 000006_create_hotels.down.sql
-- 回滚酒店相关表

DROP TRIGGER IF EXISTS update_bookings_updated_at ON bookings;
DROP TRIGGER IF EXISTS update_room_time_slots_updated_at ON room_time_slots;
DROP TRIGGER IF EXISTS update_rooms_updated_at ON rooms;
DROP TRIGGER IF EXISTS update_hotels_updated_at ON hotels;

DROP TABLE IF EXISTS bookings;
DROP TABLE IF EXISTS room_time_slots;
DROP TABLE IF EXISTS rooms;
DROP TABLE IF EXISTS hotels;
