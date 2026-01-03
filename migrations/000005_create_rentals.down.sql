-- 000005_create_rentals.down.sql
-- 回滚租借相关表

ALTER TABLE devices DROP CONSTRAINT IF EXISTS fk_device_rental;

DROP TRIGGER IF EXISTS update_rentals_updated_at ON rentals;
DROP TRIGGER IF EXISTS update_rental_pricings_updated_at ON rental_pricings;

DROP TABLE IF EXISTS rentals;
DROP TABLE IF EXISTS rental_pricings;
