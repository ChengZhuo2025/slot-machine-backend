-- 000009_create_marketing.up.sql
-- 优惠券、用户优惠券、营销活动、会员套餐表

-- 优惠券模板表
CREATE TABLE IF NOT EXISTS coupons (
    id BIGSERIAL PRIMARY KEY,
    name VARCHAR(100) NOT NULL,
    type VARCHAR(20) NOT NULL,
    value DECIMAL(10,2) NOT NULL,
    min_amount DECIMAL(10,2) NOT NULL DEFAULT 0,
    max_discount DECIMAL(10,2),
    applicable_type VARCHAR(20) NOT NULL DEFAULT 'all',
    total_count INT NOT NULL,
    issued_count INT NOT NULL DEFAULT 0,
    used_count INT NOT NULL DEFAULT 0,
    per_user_limit INT NOT NULL DEFAULT 1,
    start_time TIMESTAMP WITH TIME ZONE NOT NULL,
    end_time TIMESTAMP WITH TIME ZONE NOT NULL,
    validity_days INT,
    status SMALLINT NOT NULL DEFAULT 1,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_coupon_status ON coupons(status);
CREATE INDEX IF NOT EXISTS idx_coupon_time ON coupons(start_time, end_time);

-- 用户优惠券表
CREATE TABLE IF NOT EXISTS user_coupons (
    id BIGSERIAL PRIMARY KEY,
    user_id BIGINT NOT NULL REFERENCES users(id),
    coupon_id BIGINT NOT NULL REFERENCES coupons(id),
    status SMALLINT NOT NULL DEFAULT 0,
    order_id BIGINT REFERENCES orders(id),
    received_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    expire_at TIMESTAMP WITH TIME ZONE NOT NULL,
    used_at TIMESTAMP WITH TIME ZONE
);

CREATE INDEX IF NOT EXISTS idx_user_coupon_user ON user_coupons(user_id);
CREATE INDEX IF NOT EXISTS idx_user_coupon_coupon ON user_coupons(coupon_id);
CREATE INDEX IF NOT EXISTS idx_user_coupon_status ON user_coupons(status);

-- 更新订单表添加优惠券外键
ALTER TABLE orders ADD CONSTRAINT fk_order_coupon
    FOREIGN KEY (coupon_id) REFERENCES user_coupons(id);

-- 营销活动表
CREATE TABLE IF NOT EXISTS campaigns (
    id BIGSERIAL PRIMARY KEY,
    name VARCHAR(100) NOT NULL,
    type VARCHAR(20) NOT NULL,
    description TEXT,
    rules JSONB NOT NULL,
    start_time TIMESTAMP WITH TIME ZONE NOT NULL,
    end_time TIMESTAMP WITH TIME ZONE NOT NULL,
    status SMALLINT NOT NULL DEFAULT 1,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_campaign_status ON campaigns(status);
CREATE INDEX IF NOT EXISTS idx_campaign_time ON campaigns(start_time, end_time);

-- 会员套餐表
CREATE TABLE IF NOT EXISTS member_packages (
    id BIGSERIAL PRIMARY KEY,
    name VARCHAR(100) NOT NULL,
    target_level_id BIGINT NOT NULL REFERENCES member_levels(id),
    price DECIMAL(10,2) NOT NULL,
    duration_days INT NOT NULL,
    bonus_points INT NOT NULL DEFAULT 0,
    description TEXT,
    is_active BOOLEAN NOT NULL DEFAULT TRUE,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_member_package_level ON member_packages(target_level_id);
CREATE INDEX IF NOT EXISTS idx_member_package_active ON member_packages(is_active);

COMMENT ON TABLE coupons IS '优惠券模板表';
COMMENT ON TABLE user_coupons IS '用户优惠券表';
COMMENT ON TABLE campaigns IS '营销活动表';
COMMENT ON TABLE member_packages IS '会员套餐表';
