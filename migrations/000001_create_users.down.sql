-- 000001_create_users.down.sql
-- 回滚用户相关表

DROP TRIGGER IF EXISTS update_addresses_updated_at ON addresses;
DROP TRIGGER IF EXISTS update_user_wallets_updated_at ON user_wallets;
DROP TRIGGER IF EXISTS update_users_updated_at ON users;

DROP TABLE IF EXISTS addresses;
DROP TABLE IF EXISTS user_feedbacks;
DROP TABLE IF EXISTS user_wallets;
DROP TABLE IF EXISTS users;
DROP TABLE IF EXISTS member_levels;

DROP FUNCTION IF EXISTS update_updated_at_column();
