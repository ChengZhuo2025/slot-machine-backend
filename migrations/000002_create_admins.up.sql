-- 000002_create_admins.up.sql
-- 管理员、角色、权限表

-- 角色表
CREATE TABLE IF NOT EXISTS roles (
    id BIGSERIAL PRIMARY KEY,
    code VARCHAR(50) UNIQUE NOT NULL,
    name VARCHAR(50) NOT NULL,
    description VARCHAR(255),
    is_system BOOLEAN NOT NULL DEFAULT FALSE,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);

-- 预置角色
INSERT INTO roles (code, name, description, is_system) VALUES
    ('super_admin', '超级管理员', '拥有所有权限', TRUE),
    ('platform_admin', '平台管理员', '平台运营管理', TRUE),
    ('operation_admin', '运营管理员', '日常运营管理', TRUE),
    ('finance_admin', '财务管理员', '财务相关操作', TRUE),
    ('partner', '合作商', '合作商户管理', TRUE),
    ('customer_service', '客服', '客户服务支持', TRUE)
ON CONFLICT DO NOTHING;

-- 权限表
CREATE TABLE IF NOT EXISTS permissions (
    id BIGSERIAL PRIMARY KEY,
    code VARCHAR(100) UNIQUE NOT NULL,
    name VARCHAR(100) NOT NULL,
    type VARCHAR(20) NOT NULL,
    parent_id BIGINT REFERENCES permissions(id),
    path VARCHAR(255),
    method VARCHAR(10),
    sort INT NOT NULL DEFAULT 0,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_permission_parent ON permissions(parent_id);
CREATE INDEX IF NOT EXISTS idx_permission_type ON permissions(type);

-- 角色权限关联表
CREATE TABLE IF NOT EXISTS role_permissions (
    role_id BIGINT NOT NULL REFERENCES roles(id) ON DELETE CASCADE,
    permission_id BIGINT NOT NULL REFERENCES permissions(id) ON DELETE CASCADE,
    PRIMARY KEY (role_id, permission_id)
);

-- 管理员表
CREATE TABLE IF NOT EXISTS admins (
    id BIGSERIAL PRIMARY KEY,
    username VARCHAR(50) UNIQUE NOT NULL,
    password_hash VARCHAR(255) NOT NULL,
    name VARCHAR(50) NOT NULL,
    phone VARCHAR(20),
    email VARCHAR(100),
    role_id BIGINT NOT NULL REFERENCES roles(id),
    merchant_id BIGINT,
    status SMALLINT NOT NULL DEFAULT 1,
    last_login_at TIMESTAMP WITH TIME ZONE,
    last_login_ip VARCHAR(45),
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_admin_role ON admins(role_id);
CREATE INDEX IF NOT EXISTS idx_admin_merchant ON admins(merchant_id);
CREATE INDEX IF NOT EXISTS idx_admin_status ON admins(status);

-- 为 admins 表添加更新触发器
CREATE TRIGGER update_admins_updated_at
    BEFORE UPDATE ON admins
    FOR EACH ROW
    EXECUTE FUNCTION update_updated_at_column();

COMMENT ON TABLE roles IS '角色表';
COMMENT ON TABLE permissions IS '权限表';
COMMENT ON TABLE role_permissions IS '角色权限关联表';
COMMENT ON TABLE admins IS '管理员表';
