-- 000008_create_distribution.down.sql
-- 回滚分销相关表

DROP TRIGGER IF EXISTS update_withdrawals_updated_at ON withdrawals;
DROP TRIGGER IF EXISTS update_distributors_updated_at ON distributors;

DROP TABLE IF EXISTS withdrawals;
DROP TABLE IF EXISTS commissions;
DROP TABLE IF EXISTS distributors;
