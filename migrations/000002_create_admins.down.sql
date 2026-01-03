-- 000002_create_admins.down.sql
-- 回滚管理员相关表

DROP TRIGGER IF EXISTS update_admins_updated_at ON admins;

DROP TABLE IF EXISTS admins;
DROP TABLE IF EXISTS role_permissions;
DROP TABLE IF EXISTS permissions;
DROP TABLE IF EXISTS roles;
