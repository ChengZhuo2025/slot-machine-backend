-- 000005_create_rentals.up.sql
-- 租借记录和定价表

-- 租借定价表
CREATE TABLE IF NOT EXISTS rental_pricings (
    id BIGSERIAL PRIMARY KEY,
    venue_id BIGINT REFERENCES venues(id),
    duration_hours INT NOT NULL,
    price DECIMAL(10,2) NOT NULL,
    deposit DECIMAL(10,2) NOT NULL,
    overtime_rate DECIMAL(10,2) NOT NULL,
    is_active BOOLEAN NOT NULL DEFAULT TRUE,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    UNIQUE(venue_id, duration_hours)
);

CREATE INDEX IF NOT EXISTS idx_pricing_venue ON rental_pricings(venue_id);

-- 租借记录表
CREATE TABLE IF NOT EXISTS rentals (
    id BIGSERIAL PRIMARY KEY,
    order_id BIGINT NOT NULL UNIQUE REFERENCES orders(id),
    user_id BIGINT NOT NULL REFERENCES users(id),
    device_id BIGINT NOT NULL REFERENCES devices(id),
    duration_hours INT NOT NULL,
    rental_fee DECIMAL(10,2) NOT NULL,
    deposit DECIMAL(10,2) NOT NULL,
    overtime_rate DECIMAL(10,2) NOT NULL,
    overtime_fee DECIMAL(10,2) NOT NULL DEFAULT 0,
    status VARCHAR(20) NOT NULL,
    unlocked_at TIMESTAMP WITH TIME ZONE,
    expected_return_at TIMESTAMP WITH TIME ZONE,
    returned_at TIMESTAMP WITH TIME ZONE,
    is_purchased BOOLEAN NOT NULL DEFAULT FALSE,
    purchased_at TIMESTAMP WITH TIME ZONE,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_rental_user ON rentals(user_id);
CREATE INDEX IF NOT EXISTS idx_rental_device ON rentals(device_id);
CREATE INDEX IF NOT EXISTS idx_rental_status ON rentals(status);
CREATE INDEX IF NOT EXISTS idx_rental_order ON rentals(order_id);

-- 添加设备表的租借记录外键
ALTER TABLE devices ADD CONSTRAINT fk_device_rental
    FOREIGN KEY (current_rental_id) REFERENCES rentals(id);

-- 添加更新触发器
CREATE TRIGGER update_rental_pricings_updated_at
    BEFORE UPDATE ON rental_pricings
    FOR EACH ROW
    EXECUTE FUNCTION update_updated_at_column();

CREATE TRIGGER update_rentals_updated_at
    BEFORE UPDATE ON rentals
    FOR EACH ROW
    EXECUTE FUNCTION update_updated_at_column();

-- 插入默认定价
INSERT INTO rental_pricings (venue_id, duration_hours, price, deposit, overtime_rate) VALUES
    (NULL, 1, 29.00, 99.00, 10.00),
    (NULL, 2, 49.00, 99.00, 10.00),
    (NULL, 3, 69.00, 99.00, 10.00),
    (NULL, 6, 99.00, 99.00, 10.00),
    (NULL, 12, 149.00, 99.00, 10.00),
    (NULL, 24, 199.00, 99.00, 10.00)
ON CONFLICT DO NOTHING;

COMMENT ON TABLE rental_pricings IS '租借定价表';
COMMENT ON TABLE rentals IS '租借记录表';
