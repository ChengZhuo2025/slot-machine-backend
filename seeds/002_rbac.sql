-- 002_rbac.sql
-- 角色权限和管理员种子数据

-- 菜单权限
INSERT INTO permissions (code, name, type, parent_id, path, sort) VALUES
    ('dashboard', '仪表盘', 'menu', NULL, '/dashboard', 1),
    ('user', '用户管理', 'menu', NULL, '/user', 2),
    ('device', '设备管理', 'menu', NULL, '/device', 3),
    ('order', '订单管理', 'menu', NULL, '/order', 4),
    ('hotel', '酒店管理', 'menu', NULL, '/hotel', 5),
    ('mall', '商城管理', 'menu', NULL, '/mall', 6),
    ('distribution', '分销管理', 'menu', NULL, '/distribution', 7),
    ('marketing', '营销管理', 'menu', NULL, '/marketing', 8),
    ('finance', '财务管理', 'menu', NULL, '/finance', 9),
    ('content', '内容管理', 'menu', NULL, '/content', 10),
    ('system', '系统管理', 'menu', NULL, '/system', 11)
ON CONFLICT DO NOTHING;

-- 子菜单权限
INSERT INTO permissions (code, name, type, parent_id, path, sort)
SELECT 'user:list', '用户列表', 'menu', id, '/user/list', 1 FROM permissions WHERE code = 'user'
ON CONFLICT DO NOTHING;

INSERT INTO permissions (code, name, type, parent_id, path, sort)
SELECT 'device:list', '设备列表', 'menu', id, '/device/list', 1 FROM permissions WHERE code = 'device'
ON CONFLICT DO NOTHING;

INSERT INTO permissions (code, name, type, parent_id, path, sort)
SELECT 'device:venue', '场地管理', 'menu', id, '/device/venue', 2 FROM permissions WHERE code = 'device'
ON CONFLICT DO NOTHING;

INSERT INTO permissions (code, name, type, parent_id, path, sort)
SELECT 'device:merchant', '商户管理', 'menu', id, '/device/merchant', 3 FROM permissions WHERE code = 'device'
ON CONFLICT DO NOTHING;

INSERT INTO permissions (code, name, type, parent_id, path, sort)
SELECT 'order:list', '订单列表', 'menu', id, '/order/list', 1 FROM permissions WHERE code = 'order'
ON CONFLICT DO NOTHING;

INSERT INTO permissions (code, name, type, parent_id, path, sort)
SELECT 'order:rental', '租借订单', 'menu', id, '/order/rental', 2 FROM permissions WHERE code = 'order'
ON CONFLICT DO NOTHING;

INSERT INTO permissions (code, name, type, parent_id, path, sort)
SELECT 'finance:settlement', '结算管理', 'menu', id, '/finance/settlement', 1 FROM permissions WHERE code = 'finance'
ON CONFLICT DO NOTHING;

INSERT INTO permissions (code, name, type, parent_id, path, sort)
SELECT 'finance:withdrawal', '提现审核', 'menu', id, '/finance/withdrawal', 2 FROM permissions WHERE code = 'finance'
ON CONFLICT DO NOTHING;

INSERT INTO permissions (code, name, type, parent_id, path, sort)
SELECT 'system:admin', '管理员管理', 'menu', id, '/system/admin', 1 FROM permissions WHERE code = 'system'
ON CONFLICT DO NOTHING;

INSERT INTO permissions (code, name, type, parent_id, path, sort)
SELECT 'system:role', '角色管理', 'menu', id, '/system/role', 2 FROM permissions WHERE code = 'system'
ON CONFLICT DO NOTHING;

INSERT INTO permissions (code, name, type, parent_id, path, sort)
SELECT 'system:config', '系统配置', 'menu', id, '/system/config', 3 FROM permissions WHERE code = 'system'
ON CONFLICT DO NOTHING;

-- 为超级管理员分配所有权限
INSERT INTO role_permissions (role_id, permission_id)
SELECT r.id, p.id FROM roles r, permissions p WHERE r.code = 'super_admin'
ON CONFLICT DO NOTHING;

-- 为平台管理员分配权限
INSERT INTO role_permissions (role_id, permission_id)
SELECT r.id, p.id FROM roles r, permissions p
WHERE r.code = 'platform_admin' AND p.code NOT IN ('system', 'system:admin', 'system:role')
ON CONFLICT DO NOTHING;

-- 为运营管理员分配权限
INSERT INTO role_permissions (role_id, permission_id)
SELECT r.id, p.id FROM roles r, permissions p
WHERE r.code = 'operation_admin' AND p.code IN ('dashboard', 'user', 'user:list', 'order', 'order:list', 'order:rental', 'marketing', 'content')
ON CONFLICT DO NOTHING;

-- 为财务管理员分配权限
INSERT INTO role_permissions (role_id, permission_id)
SELECT r.id, p.id FROM roles r, permissions p
WHERE r.code = 'finance_admin' AND p.code IN ('dashboard', 'finance', 'finance:settlement', 'finance:withdrawal', 'order', 'order:list')
ON CONFLICT DO NOTHING;

-- 测试管理员账号 (密码: admin123，使用 bcrypt 加密)
INSERT INTO admins (username, password_hash, name, phone, email, role_id, status) VALUES
    ('admin', '$2a$10$KdAXVq7l3.Xa0v7xS3qvXegaslCPPa45XVEWqekSLmRx1KCAeUNWe', '超级管理员', '13900000001', 'admin@example.com', (SELECT id FROM roles WHERE code = 'super_admin'), 1),
    ('platform', '$2a$10$KdAXVq7l3.Xa0v7xS3qvXegaslCPPa45XVEWqekSLmRx1KCAeUNWe', '平台管理员', '13900000002', 'platform@example.com', (SELECT id FROM roles WHERE code = 'platform_admin'), 1),
    ('operator', '$2a$10$KdAXVq7l3.Xa0v7xS3qvXegaslCPPa45XVEWqekSLmRx1KCAeUNWe', '运营管理员', '13900000003', 'operator@example.com', (SELECT id FROM roles WHERE code = 'operation_admin'), 1),
    ('finance', '$2a$10$KdAXVq7l3.Xa0v7xS3qvXegaslCPPa45XVEWqekSLmRx1KCAeUNWe', '财务管理员', '13900000004', 'finance@example.com', (SELECT id FROM roles WHERE code = 'finance_admin'), 1)
ON CONFLICT DO NOTHING;
