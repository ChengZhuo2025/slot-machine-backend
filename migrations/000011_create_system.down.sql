-- 000011_create_system.down.sql
-- 回滚系统相关表

ALTER TABLE user_feedbacks DROP CONSTRAINT IF EXISTS fk_feedback_replied_by;

DROP TRIGGER IF EXISTS update_banners_updated_at ON banners;
DROP TRIGGER IF EXISTS update_system_configs_updated_at ON system_configs;
DROP TRIGGER IF EXISTS update_articles_updated_at ON articles;

DROP TABLE IF EXISTS banners;
DROP TABLE IF EXISTS sms_codes;
DROP TABLE IF EXISTS operation_logs;
DROP TABLE IF EXISTS system_configs;
DROP TABLE IF EXISTS message_templates;
DROP TABLE IF EXISTS notifications;
DROP TABLE IF EXISTS articles;
