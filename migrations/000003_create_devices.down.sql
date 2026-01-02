-- 000003_create_devices.down.sql
-- 回滚设备相关表

ALTER TABLE admins DROP CONSTRAINT IF EXISTS fk_admin_merchant;

DROP TRIGGER IF EXISTS update_devices_updated_at ON devices;
DROP TRIGGER IF EXISTS update_venues_updated_at ON venues;
DROP TRIGGER IF EXISTS update_merchants_updated_at ON merchants;

DROP TABLE IF EXISTS device_maintenances;
DROP TABLE IF EXISTS device_logs;
DROP TABLE IF EXISTS devices;
DROP TABLE IF EXISTS venues;
DROP TABLE IF EXISTS merchants;
