-- 000003_create_devices.up.sql
-- 商户、场地、设备表

-- 商户表
CREATE TABLE IF NOT EXISTS merchants (
    id BIGSERIAL PRIMARY KEY,
    name VARCHAR(100) NOT NULL,
    contact_name VARCHAR(50) NOT NULL,
    contact_phone VARCHAR(20) NOT NULL,
    address VARCHAR(255),
    business_license VARCHAR(255),
    commission_rate DECIMAL(5,4) NOT NULL DEFAULT 0.2000,
    settlement_type VARCHAR(20) NOT NULL DEFAULT 'monthly',
    bank_name VARCHAR(100),
    bank_account_encrypted TEXT,
    bank_holder_encrypted TEXT,
    status SMALLINT NOT NULL DEFAULT 1,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_merchant_status ON merchants(status);

-- 场地表
CREATE TABLE IF NOT EXISTS venues (
    id BIGSERIAL PRIMARY KEY,
    merchant_id BIGINT NOT NULL REFERENCES merchants(id),
    name VARCHAR(100) NOT NULL,
    type VARCHAR(20) NOT NULL,
    province VARCHAR(50) NOT NULL,
    city VARCHAR(50) NOT NULL,
    district VARCHAR(50) NOT NULL,
    address VARCHAR(255) NOT NULL,
    longitude DECIMAL(10,7),
    latitude DECIMAL(10,7),
    contact_name VARCHAR(50),
    contact_phone VARCHAR(20),
    status SMALLINT NOT NULL DEFAULT 1,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_venue_merchant ON venues(merchant_id);
CREATE INDEX IF NOT EXISTS idx_venue_location ON venues(province, city, district);
CREATE INDEX IF NOT EXISTS idx_venue_status ON venues(status);

-- 设备表
CREATE TABLE IF NOT EXISTS devices (
    id BIGSERIAL PRIMARY KEY,
    device_no VARCHAR(64) UNIQUE NOT NULL,
    name VARCHAR(100) NOT NULL,
    type VARCHAR(20) NOT NULL,
    model VARCHAR(50),
    venue_id BIGINT NOT NULL REFERENCES venues(id),
    qr_code VARCHAR(255) NOT NULL,
    product_name VARCHAR(100) NOT NULL,
    product_image VARCHAR(255),
    slot_count INT NOT NULL DEFAULT 1,
    available_slots INT NOT NULL DEFAULT 1,
    online_status SMALLINT NOT NULL DEFAULT 0,
    lock_status SMALLINT NOT NULL DEFAULT 0,
    rental_status SMALLINT NOT NULL DEFAULT 0,
    current_rental_id BIGINT,
    firmware_version VARCHAR(20),
    network_type VARCHAR(20) DEFAULT 'WiFi',
    signal_strength INT,
    battery_level INT,
    temperature DECIMAL(5,2),
    humidity DECIMAL(5,2),
    last_heartbeat_at TIMESTAMP WITH TIME ZONE,
    last_online_at TIMESTAMP WITH TIME ZONE,
    last_offline_at TIMESTAMP WITH TIME ZONE,
    install_time TIMESTAMP WITH TIME ZONE,
    status SMALLINT NOT NULL DEFAULT 1,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_device_venue ON devices(venue_id);
CREATE INDEX IF NOT EXISTS idx_device_status ON devices(online_status, rental_status);
CREATE INDEX IF NOT EXISTS idx_device_type ON devices(type);
CREATE INDEX IF NOT EXISTS idx_device_no ON devices(device_no);

-- 设备日志表
CREATE TABLE IF NOT EXISTS device_logs (
    id BIGSERIAL PRIMARY KEY,
    device_id BIGINT NOT NULL REFERENCES devices(id),
    type VARCHAR(20) NOT NULL,
    content TEXT,
    operator_id BIGINT,
    operator_type VARCHAR(10),
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_device_log_device_time ON device_logs(device_id, created_at DESC);

-- 设备维护记录表
CREATE TABLE IF NOT EXISTS device_maintenances (
    id BIGSERIAL PRIMARY KEY,
    device_id BIGINT NOT NULL REFERENCES devices(id),
    type VARCHAR(20) NOT NULL,
    description TEXT NOT NULL,
    before_images JSONB,
    after_images JSONB,
    cost DECIMAL(10,2) NOT NULL DEFAULT 0,
    operator_id BIGINT NOT NULL,
    status SMALLINT NOT NULL DEFAULT 0,
    started_at TIMESTAMP WITH TIME ZONE NOT NULL,
    completed_at TIMESTAMP WITH TIME ZONE,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_maintenance_device ON device_maintenances(device_id);
CREATE INDEX IF NOT EXISTS idx_maintenance_status ON device_maintenances(status);

-- 添加更新触发器
CREATE TRIGGER update_merchants_updated_at
    BEFORE UPDATE ON merchants
    FOR EACH ROW
    EXECUTE FUNCTION update_updated_at_column();

CREATE TRIGGER update_venues_updated_at
    BEFORE UPDATE ON venues
    FOR EACH ROW
    EXECUTE FUNCTION update_updated_at_column();

CREATE TRIGGER update_devices_updated_at
    BEFORE UPDATE ON devices
    FOR EACH ROW
    EXECUTE FUNCTION update_updated_at_column();

-- 更新 admins 表添加商户外键约束
ALTER TABLE admins ADD CONSTRAINT fk_admin_merchant
    FOREIGN KEY (merchant_id) REFERENCES merchants(id);

COMMENT ON TABLE merchants IS '商户表';
COMMENT ON TABLE venues IS '场地表';
COMMENT ON TABLE devices IS '智能柜设备表';
COMMENT ON TABLE device_logs IS '设备日志表';
COMMENT ON TABLE device_maintenances IS '设备维护记录表';
