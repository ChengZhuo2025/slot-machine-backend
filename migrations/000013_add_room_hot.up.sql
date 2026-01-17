-- 添加房间热门相关字段
ALTER TABLE rooms ADD COLUMN booking_count INT NOT NULL DEFAULT 0;
ALTER TABLE rooms ADD COLUMN average_rating DECIMAL(2,1) NOT NULL DEFAULT 0.0;
ALTER TABLE rooms ADD COLUMN review_count INT NOT NULL DEFAULT 0;
ALTER TABLE rooms ADD COLUMN is_hot BOOLEAN NOT NULL DEFAULT FALSE;
ALTER TABLE rooms ADD COLUMN hot_rank INT NOT NULL DEFAULT 0;
ALTER TABLE rooms ADD COLUMN hot_score DECIMAL(10,2) NOT NULL DEFAULT 0.00;

-- 创建部分索引优化热门房型查询
CREATE INDEX idx_room_hot ON rooms(is_hot, hot_score DESC)
WHERE is_hot = TRUE AND status = 1;

-- 创建酒店内热门房型索引
CREATE INDEX idx_room_hotel_hot ON rooms(hotel_id, is_hot, hot_score DESC)
WHERE status = 1;

-- 添加注释
COMMENT ON COLUMN rooms.booking_count IS '预订次数';
COMMENT ON COLUMN rooms.average_rating IS '平均评分';
COMMENT ON COLUMN rooms.review_count IS '评论数量';
COMMENT ON COLUMN rooms.is_hot IS '是否为热门房型';
COMMENT ON COLUMN rooms.hot_rank IS '热门排名';
COMMENT ON COLUMN rooms.hot_score IS '热门分数，用于排序，综合预订量、评分等计算';
