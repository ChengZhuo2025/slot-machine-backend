-- 移除房间热门相关字段
DROP INDEX IF EXISTS idx_room_hotel_hot;
DROP INDEX IF EXISTS idx_room_hot;

ALTER TABLE rooms DROP COLUMN IF EXISTS hot_score;
ALTER TABLE rooms DROP COLUMN IF EXISTS hot_rank;
ALTER TABLE rooms DROP COLUMN IF EXISTS is_hot;
ALTER TABLE rooms DROP COLUMN IF EXISTS review_count;
ALTER TABLE rooms DROP COLUMN IF EXISTS average_rating;
ALTER TABLE rooms DROP COLUMN IF EXISTS booking_count;
