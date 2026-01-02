-- 000010_create_finance.up.sql
-- 结算、钱包交易记录表

-- 结算记录表
CREATE TABLE IF NOT EXISTS settlements (
    id BIGSERIAL PRIMARY KEY,
    settlement_no VARCHAR(64) UNIQUE NOT NULL,
    type VARCHAR(20) NOT NULL,
    target_id BIGINT NOT NULL,
    period_start DATE NOT NULL,
    period_end DATE NOT NULL,
    total_amount DECIMAL(12,2) NOT NULL,
    fee DECIMAL(10,2) NOT NULL DEFAULT 0,
    actual_amount DECIMAL(12,2) NOT NULL,
    order_count INT NOT NULL,
    status VARCHAR(20) NOT NULL,
    operator_id BIGINT,
    settled_at TIMESTAMP WITH TIME ZONE,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_settlement_type_target ON settlements(type, target_id);
CREATE INDEX IF NOT EXISTS idx_settlement_status ON settlements(status);
CREATE INDEX IF NOT EXISTS idx_settlement_period ON settlements(period_start, period_end);

-- 钱包交易记录表
CREATE TABLE IF NOT EXISTS wallet_transactions (
    id BIGSERIAL PRIMARY KEY,
    user_id BIGINT NOT NULL REFERENCES users(id),
    type VARCHAR(20) NOT NULL,
    amount DECIMAL(12,2) NOT NULL,
    balance_before DECIMAL(12,2) NOT NULL,
    balance_after DECIMAL(12,2) NOT NULL,
    order_no VARCHAR(64),
    remark VARCHAR(255),
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_wallet_tx_user ON wallet_transactions(user_id);
CREATE INDEX IF NOT EXISTS idx_wallet_tx_order ON wallet_transactions(order_no);
CREATE INDEX IF NOT EXISTS idx_wallet_tx_type ON wallet_transactions(type);
CREATE INDEX IF NOT EXISTS idx_wallet_tx_created ON wallet_transactions(created_at);

COMMENT ON TABLE settlements IS '结算记录表';
COMMENT ON TABLE wallet_transactions IS '钱包交易记录表';
