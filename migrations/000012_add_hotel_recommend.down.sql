-- 移除酒店推荐相关字段
DROP INDEX IF EXISTS idx_hotel_recommended;

ALTER TABLE hotels DROP COLUMN IF EXISTS recommended_at;
ALTER TABLE hotels DROP COLUMN IF EXISTS recommend_score;
ALTER TABLE hotels DROP COLUMN IF EXISTS is_recommended;
