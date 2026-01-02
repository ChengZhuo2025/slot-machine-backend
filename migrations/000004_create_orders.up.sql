-- 000004_create_orders.up.sql
-- 订单、支付、退款表

-- 订单表
CREATE TABLE IF NOT EXISTS orders (
    id BIGSERIAL PRIMARY KEY,
    order_no VARCHAR(64) UNIQUE NOT NULL,
    user_id BIGINT NOT NULL REFERENCES users(id),
    type VARCHAR(20) NOT NULL,
    original_amount DECIMAL(12,2) NOT NULL,
    discount_amount DECIMAL(12,2) NOT NULL DEFAULT 0,
    actual_amount DECIMAL(12,2) NOT NULL,
    deposit_amount DECIMAL(12,2) NOT NULL DEFAULT 0,
    status VARCHAR(20) NOT NULL,
    coupon_id BIGINT,
    remark VARCHAR(255),
    address_id BIGINT REFERENCES addresses(id),
    address_snapshot JSONB,
    express_company VARCHAR(50),
    express_no VARCHAR(64),
    shipped_at TIMESTAMP WITH TIME ZONE,
    received_at TIMESTAMP WITH TIME ZONE,
    paid_at TIMESTAMP WITH TIME ZONE,
    completed_at TIMESTAMP WITH TIME ZONE,
    cancelled_at TIMESTAMP WITH TIME ZONE,
    cancel_reason VARCHAR(255),
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_order_user ON orders(user_id);
CREATE INDEX IF NOT EXISTS idx_order_type_status ON orders(type, status);
CREATE INDEX IF NOT EXISTS idx_order_created ON orders(created_at DESC);
CREATE INDEX IF NOT EXISTS idx_order_express ON orders(express_no);
CREATE INDEX IF NOT EXISTS idx_order_no ON orders(order_no);

-- 订单项表
CREATE TABLE IF NOT EXISTS order_items (
    id BIGSERIAL PRIMARY KEY,
    order_id BIGINT NOT NULL REFERENCES orders(id) ON DELETE CASCADE,
    product_id BIGINT,
    product_name VARCHAR(100) NOT NULL,
    product_image VARCHAR(255),
    sku_info VARCHAR(255),
    price DECIMAL(12,2) NOT NULL,
    quantity INT NOT NULL,
    subtotal DECIMAL(12,2) NOT NULL,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_order_item_order ON order_items(order_id);

-- 支付记录表
CREATE TABLE IF NOT EXISTS payments (
    id BIGSERIAL PRIMARY KEY,
    payment_no VARCHAR(64) UNIQUE NOT NULL,
    order_id BIGINT NOT NULL REFERENCES orders(id),
    order_no VARCHAR(64) NOT NULL,
    user_id BIGINT NOT NULL REFERENCES users(id),
    channel VARCHAR(20) NOT NULL,
    amount DECIMAL(12,2) NOT NULL,
    status VARCHAR(20) NOT NULL,
    trade_no VARCHAR(64),
    pay_time TIMESTAMP WITH TIME ZONE,
    callback_data JSONB,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_payment_order ON payments(order_id);
CREATE INDEX IF NOT EXISTS idx_payment_order_no ON payments(order_no);
CREATE INDEX IF NOT EXISTS idx_payment_user ON payments(user_id);
CREATE INDEX IF NOT EXISTS idx_payment_status ON payments(status);

-- 退款记录表
CREATE TABLE IF NOT EXISTS refunds (
    id BIGSERIAL PRIMARY KEY,
    refund_no VARCHAR(64) UNIQUE NOT NULL,
    order_id BIGINT NOT NULL REFERENCES orders(id),
    payment_id BIGINT NOT NULL REFERENCES payments(id),
    user_id BIGINT NOT NULL REFERENCES users(id),
    amount DECIMAL(12,2) NOT NULL,
    reason VARCHAR(255) NOT NULL,
    status VARCHAR(20) NOT NULL,
    trade_refund_no VARCHAR(64),
    operator_id BIGINT,
    processed_at TIMESTAMP WITH TIME ZONE,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_refund_order ON refunds(order_id);
CREATE INDEX IF NOT EXISTS idx_refund_user ON refunds(user_id);
CREATE INDEX IF NOT EXISTS idx_refund_status ON refunds(status);

-- 添加更新触发器
CREATE TRIGGER update_orders_updated_at
    BEFORE UPDATE ON orders
    FOR EACH ROW
    EXECUTE FUNCTION update_updated_at_column();

CREATE TRIGGER update_payments_updated_at
    BEFORE UPDATE ON payments
    FOR EACH ROW
    EXECUTE FUNCTION update_updated_at_column();

CREATE TRIGGER update_refunds_updated_at
    BEFORE UPDATE ON refunds
    FOR EACH ROW
    EXECUTE FUNCTION update_updated_at_column();

COMMENT ON TABLE orders IS '订单表';
COMMENT ON TABLE order_items IS '订单项表';
COMMENT ON TABLE payments IS '支付记录表';
COMMENT ON TABLE refunds IS '退款记录表';
