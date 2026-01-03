-- 000001_create_users.up.sql
-- 用户表和用户钱包表

-- 会员等级表 (需要先创建，因为 users 表引用它)
CREATE TABLE IF NOT EXISTS member_levels (
    id BIGSERIAL PRIMARY KEY,
    name VARCHAR(50) NOT NULL,
    level INT UNIQUE NOT NULL,
    min_points INT NOT NULL DEFAULT 0,
    discount DECIMAL(3,2) NOT NULL DEFAULT 1.00,
    benefits JSONB,
    icon VARCHAR(255),
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);

-- 创建默认会员等级
INSERT INTO member_levels (name, level, min_points, discount, benefits, icon) VALUES
    ('普通会员', 1, 0, 1.00, '{"description": "基础会员权益"}', NULL),
    ('银卡会员', 2, 1000, 0.98, '{"description": "银卡会员权益", "discount": "98折"}', NULL),
    ('金卡会员', 3, 5000, 0.95, '{"description": "金卡会员权益", "discount": "95折"}', NULL),
    ('钻石会员', 4, 20000, 0.90, '{"description": "钻石会员权益", "discount": "9折"}', NULL)
ON CONFLICT DO NOTHING;

-- 用户表
CREATE TABLE IF NOT EXISTS users (
    id BIGSERIAL PRIMARY KEY,
    phone VARCHAR(20) UNIQUE,
    openid VARCHAR(64) UNIQUE,
    unionid VARCHAR(64),
    nickname VARCHAR(50) NOT NULL DEFAULT '',
    avatar VARCHAR(255),
    gender SMALLINT NOT NULL DEFAULT 0,
    birthday DATE,
    member_level_id BIGINT NOT NULL DEFAULT 1 REFERENCES member_levels(id),
    points INT NOT NULL DEFAULT 0,
    is_verified BOOLEAN NOT NULL DEFAULT FALSE,
    real_name_encrypted TEXT,
    id_card_encrypted TEXT,
    referrer_id BIGINT REFERENCES users(id),
    status SMALLINT NOT NULL DEFAULT 1,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);

-- 用户表索引
CREATE INDEX IF NOT EXISTS idx_user_phone ON users(phone);
CREATE INDEX IF NOT EXISTS idx_user_openid ON users(openid);
CREATE INDEX IF NOT EXISTS idx_user_referrer ON users(referrer_id);
CREATE INDEX IF NOT EXISTS idx_user_status ON users(status);
CREATE INDEX IF NOT EXISTS idx_user_created_at ON users(created_at);

-- 用户钱包表
CREATE TABLE IF NOT EXISTS user_wallets (
    id BIGSERIAL PRIMARY KEY,
    user_id BIGINT NOT NULL UNIQUE REFERENCES users(id) ON DELETE CASCADE,
    balance DECIMAL(12,2) NOT NULL DEFAULT 0.00,
    frozen_balance DECIMAL(12,2) NOT NULL DEFAULT 0.00,
    total_recharged DECIMAL(12,2) NOT NULL DEFAULT 0.00,
    total_consumed DECIMAL(12,2) NOT NULL DEFAULT 0.00,
    total_withdrawn DECIMAL(12,2) NOT NULL DEFAULT 0.00,
    version INT NOT NULL DEFAULT 0,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);

-- 用户钱包约束
ALTER TABLE user_wallets ADD CONSTRAINT chk_wallet_balance CHECK (balance >= 0);
ALTER TABLE user_wallets ADD CONSTRAINT chk_wallet_frozen CHECK (frozen_balance >= 0);

-- 用户反馈表
CREATE TABLE IF NOT EXISTS user_feedbacks (
    id BIGSERIAL PRIMARY KEY,
    user_id BIGINT NOT NULL REFERENCES users(id),
    type VARCHAR(20) NOT NULL,
    content TEXT NOT NULL,
    images JSONB,
    contact VARCHAR(100),
    status SMALLINT NOT NULL DEFAULT 0,
    reply TEXT,
    replied_by BIGINT,
    replied_at TIMESTAMP WITH TIME ZONE,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_feedback_user ON user_feedbacks(user_id);
CREATE INDEX IF NOT EXISTS idx_feedback_status ON user_feedbacks(status);

-- 用户收货地址表
CREATE TABLE IF NOT EXISTS addresses (
    id BIGSERIAL PRIMARY KEY,
    user_id BIGINT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    receiver_name VARCHAR(50) NOT NULL,
    receiver_phone VARCHAR(20) NOT NULL,
    province VARCHAR(50) NOT NULL,
    city VARCHAR(50) NOT NULL,
    district VARCHAR(50) NOT NULL,
    detail VARCHAR(255) NOT NULL,
    postal_code VARCHAR(10),
    is_default BOOLEAN NOT NULL DEFAULT FALSE,
    tag VARCHAR(20),
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_address_user ON addresses(user_id);
CREATE INDEX IF NOT EXISTS idx_address_user_default ON addresses(user_id, is_default);

-- 创建更新时间触发器函数
CREATE OR REPLACE FUNCTION update_updated_at_column()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = CURRENT_TIMESTAMP;
    RETURN NEW;
END;
$$ language 'plpgsql';

-- 为 users 表添加更新触发器
CREATE TRIGGER update_users_updated_at
    BEFORE UPDATE ON users
    FOR EACH ROW
    EXECUTE FUNCTION update_updated_at_column();

-- 为 user_wallets 表添加更新触发器
CREATE TRIGGER update_user_wallets_updated_at
    BEFORE UPDATE ON user_wallets
    FOR EACH ROW
    EXECUTE FUNCTION update_updated_at_column();

-- 为 addresses 表添加更新触发器
CREATE TRIGGER update_addresses_updated_at
    BEFORE UPDATE ON addresses
    FOR EACH ROW
    EXECUTE FUNCTION update_updated_at_column();

COMMENT ON TABLE users IS '用户表';
COMMENT ON TABLE user_wallets IS '用户钱包表';
COMMENT ON TABLE member_levels IS '会员等级表';
COMMENT ON TABLE user_feedbacks IS '用户反馈表';
COMMENT ON TABLE addresses IS '用户收货地址表';
