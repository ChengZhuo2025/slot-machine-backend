-- 添加酒店推荐相关字段
ALTER TABLE hotels ADD COLUMN is_recommended BOOLEAN NOT NULL DEFAULT FALSE;
ALTER TABLE hotels ADD COLUMN recommend_score INT NOT NULL DEFAULT 0;
ALTER TABLE hotels ADD COLUMN recommended_at TIMESTAMP WITH TIME ZONE;

-- 创建部分索引优化推荐酒店查询
CREATE INDEX idx_hotel_recommended ON hotels(is_recommended, recommend_score DESC)
WHERE is_recommended = TRUE AND status = 1;

-- 添加注释
COMMENT ON COLUMN hotels.is_recommended IS '是否推荐';
COMMENT ON COLUMN hotels.recommend_score IS '推荐分数，用于排序';
COMMENT ON COLUMN hotels.recommended_at IS '设为推荐的时间';
