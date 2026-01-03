-- 000006_create_hotels.up.sql
-- 酒店、房间、预订表

-- 酒店表
CREATE TABLE IF NOT EXISTS hotels (
    id BIGSERIAL PRIMARY KEY,
    name VARCHAR(100) NOT NULL,
    star_rating SMALLINT,
    province VARCHAR(50) NOT NULL,
    city VARCHAR(50) NOT NULL,
    district VARCHAR(50) NOT NULL,
    address VARCHAR(255) NOT NULL,
    longitude DECIMAL(10,7),
    latitude DECIMAL(10,7),
    phone VARCHAR(20) NOT NULL,
    images JSONB,
    facilities JSONB,
    description TEXT,
    check_in_time TIME NOT NULL DEFAULT '14:00',
    check_out_time TIME NOT NULL DEFAULT '12:00',
    commission_rate DECIMAL(5,4) NOT NULL DEFAULT 0.1500,
    status SMALLINT NOT NULL DEFAULT 1,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_hotel_location ON hotels(province, city, district);
CREATE INDEX IF NOT EXISTS idx_hotel_status ON hotels(status);

-- 房间表
CREATE TABLE IF NOT EXISTS rooms (
    id BIGSERIAL PRIMARY KEY,
    hotel_id BIGINT NOT NULL REFERENCES hotels(id),
    room_no VARCHAR(20) NOT NULL,
    room_type VARCHAR(50) NOT NULL,
    device_id BIGINT REFERENCES devices(id),
    images JSONB,
    facilities JSONB,
    area INT,
    bed_type VARCHAR(50),
    max_guests INT NOT NULL DEFAULT 2,
    hourly_price DECIMAL(10,2) NOT NULL,
    daily_price DECIMAL(10,2) NOT NULL,
    status SMALLINT NOT NULL DEFAULT 1,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_room_hotel ON rooms(hotel_id);
CREATE INDEX IF NOT EXISTS idx_room_device ON rooms(device_id);
CREATE INDEX IF NOT EXISTS idx_room_status ON rooms(status);

-- 房间时段价格表
CREATE TABLE IF NOT EXISTS room_time_slots (
    id BIGSERIAL PRIMARY KEY,
    room_id BIGINT NOT NULL REFERENCES rooms(id) ON DELETE CASCADE,
    duration_hours INT NOT NULL,
    price DECIMAL(10,2) NOT NULL,
    start_time TIME,
    end_time TIME,
    is_active BOOLEAN NOT NULL DEFAULT TRUE,
    sort INT NOT NULL DEFAULT 0,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    UNIQUE(room_id, duration_hours)
);

CREATE INDEX IF NOT EXISTS idx_timeslot_room ON room_time_slots(room_id);

-- 预订记录表
CREATE TABLE IF NOT EXISTS bookings (
    id BIGSERIAL PRIMARY KEY,
    booking_no VARCHAR(64) UNIQUE NOT NULL,
    order_id BIGINT NOT NULL UNIQUE REFERENCES orders(id),
    user_id BIGINT NOT NULL REFERENCES users(id),
    hotel_id BIGINT NOT NULL REFERENCES hotels(id),
    room_id BIGINT NOT NULL REFERENCES rooms(id),
    device_id BIGINT REFERENCES devices(id),
    check_in_time TIMESTAMP WITH TIME ZONE NOT NULL,
    check_out_time TIMESTAMP WITH TIME ZONE NOT NULL,
    duration_hours INT NOT NULL,
    amount DECIMAL(10,2) NOT NULL,
    verification_code VARCHAR(20) NOT NULL,
    unlock_code VARCHAR(10) NOT NULL,
    qr_code VARCHAR(255) NOT NULL,
    status VARCHAR(20) NOT NULL,
    verified_at TIMESTAMP WITH TIME ZONE,
    verified_by BIGINT,
    unlocked_at TIMESTAMP WITH TIME ZONE,
    completed_at TIMESTAMP WITH TIME ZONE,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_booking_user ON bookings(user_id);
CREATE INDEX IF NOT EXISTS idx_booking_hotel ON bookings(hotel_id);
CREATE INDEX IF NOT EXISTS idx_booking_room ON bookings(room_id);
CREATE INDEX IF NOT EXISTS idx_booking_status ON bookings(status);
CREATE INDEX IF NOT EXISTS idx_booking_no ON bookings(booking_no);

-- 添加更新触发器
CREATE TRIGGER update_hotels_updated_at
    BEFORE UPDATE ON hotels
    FOR EACH ROW
    EXECUTE FUNCTION update_updated_at_column();

CREATE TRIGGER update_rooms_updated_at
    BEFORE UPDATE ON rooms
    FOR EACH ROW
    EXECUTE FUNCTION update_updated_at_column();

CREATE TRIGGER update_room_time_slots_updated_at
    BEFORE UPDATE ON room_time_slots
    FOR EACH ROW
    EXECUTE FUNCTION update_updated_at_column();

CREATE TRIGGER update_bookings_updated_at
    BEFORE UPDATE ON bookings
    FOR EACH ROW
    EXECUTE FUNCTION update_updated_at_column();

COMMENT ON TABLE hotels IS '酒店表';
COMMENT ON TABLE rooms IS '房间表';
COMMENT ON TABLE room_time_slots IS '房间时段价格表';
COMMENT ON TABLE bookings IS '预订记录表';
