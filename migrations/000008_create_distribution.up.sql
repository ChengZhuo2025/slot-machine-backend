-- 000008_create_distribution.up.sql
-- 分销商、佣金、提现表

-- 分销商表
CREATE TABLE IF NOT EXISTS distributors (
    id BIGSERIAL PRIMARY KEY,
    user_id BIGINT NOT NULL UNIQUE REFERENCES users(id),
    parent_id BIGINT REFERENCES distributors(id),
    level SMALLINT NOT NULL DEFAULT 1,
    invite_code VARCHAR(20) UNIQUE NOT NULL,
    total_commission DECIMAL(12,2) NOT NULL DEFAULT 0,
    available_commission DECIMAL(12,2) NOT NULL DEFAULT 0,
    frozen_commission DECIMAL(12,2) NOT NULL DEFAULT 0,
    withdrawn_commission DECIMAL(12,2) NOT NULL DEFAULT 0,
    team_count INT NOT NULL DEFAULT 0,
    direct_count INT NOT NULL DEFAULT 0,
    status SMALLINT NOT NULL DEFAULT 0,
    approved_at TIMESTAMP WITH TIME ZONE,
    approved_by BIGINT,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_distributor_user ON distributors(user_id);
CREATE INDEX IF NOT EXISTS idx_distributor_parent ON distributors(parent_id);
CREATE INDEX IF NOT EXISTS idx_distributor_status ON distributors(status);
CREATE INDEX IF NOT EXISTS idx_distributor_invite ON distributors(invite_code);

-- 佣金记录表
CREATE TABLE IF NOT EXISTS commissions (
    id BIGSERIAL PRIMARY KEY,
    distributor_id BIGINT NOT NULL REFERENCES distributors(id),
    order_id BIGINT NOT NULL REFERENCES orders(id),
    from_user_id BIGINT NOT NULL REFERENCES users(id),
    type VARCHAR(20) NOT NULL,
    order_amount DECIMAL(12,2) NOT NULL,
    rate DECIMAL(5,4) NOT NULL,
    amount DECIMAL(12,2) NOT NULL,
    status SMALLINT NOT NULL DEFAULT 0,
    settled_at TIMESTAMP WITH TIME ZONE,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_commission_distributor ON commissions(distributor_id);
CREATE INDEX IF NOT EXISTS idx_commission_order ON commissions(order_id);
CREATE INDEX IF NOT EXISTS idx_commission_status ON commissions(status);

-- 提现申请表
CREATE TABLE IF NOT EXISTS withdrawals (
    id BIGSERIAL PRIMARY KEY,
    withdrawal_no VARCHAR(64) UNIQUE NOT NULL,
    user_id BIGINT NOT NULL REFERENCES users(id),
    type VARCHAR(20) NOT NULL,
    amount DECIMAL(12,2) NOT NULL,
    fee DECIMAL(10,2) NOT NULL DEFAULT 0,
    actual_amount DECIMAL(12,2) NOT NULL,
    withdraw_to VARCHAR(20) NOT NULL,
    account_info_encrypted TEXT NOT NULL,
    status VARCHAR(20) NOT NULL,
    operator_id BIGINT,
    processed_at TIMESTAMP WITH TIME ZONE,
    reject_reason VARCHAR(255),
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_withdrawal_user ON withdrawals(user_id);
CREATE INDEX IF NOT EXISTS idx_withdrawal_status ON withdrawals(status);

-- 添加更新触发器
CREATE TRIGGER update_distributors_updated_at
    BEFORE UPDATE ON distributors
    FOR EACH ROW
    EXECUTE FUNCTION update_updated_at_column();

CREATE TRIGGER update_withdrawals_updated_at
    BEFORE UPDATE ON withdrawals
    FOR EACH ROW
    EXECUTE FUNCTION update_updated_at_column();

COMMENT ON TABLE distributors IS '分销商表';
COMMENT ON TABLE commissions IS '佣金记录表';
COMMENT ON TABLE withdrawals IS '提现申请表';
