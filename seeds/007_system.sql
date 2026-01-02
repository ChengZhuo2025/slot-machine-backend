-- 007_system.sql
-- 租借定价、Banner、系统配置种子数据

-- 系统配置
INSERT INTO system_configs ("group", key, value, type, description, is_public) VALUES
    ('general', 'site_name', '爱上杜美人', 'string', '网站名称', TRUE),
    ('general', 'site_logo', 'https://img.example.com/logo.png', 'string', '网站Logo', TRUE),
    ('general', 'contact_phone', '400-888-8888', 'string', '客服电话', TRUE),
    ('general', 'contact_email', 'service@example.com', 'string', '客服邮箱', TRUE),
    ('general', 'about_us', '爱上杜美人是一家专注于成人情趣用品的智能零售平台...', 'string', '关于我们', TRUE),

    ('rental', 'default_deposit', '99', 'number', '默认押金金额', FALSE),
    ('rental', 'auto_purchase_hours', '24', 'number', '超时自动购买时间(小时)', FALSE),
    ('rental', 'overtime_rate', '10', 'number', '超时费率(元/小时)', FALSE),

    ('distribution', 'level1_rate', '0.10', 'number', '一级分销比例', FALSE),
    ('distribution', 'level2_rate', '0.05', 'number', '二级分销比例', FALSE),
    ('distribution', 'min_withdraw', '100', 'number', '最低提现金额', FALSE),
    ('distribution', 'withdraw_fee_rate', '0.01', 'number', '提现手续费比例', FALSE),

    ('member', 'points_rate', '1', 'number', '消费积分比例(每消费1元获得积分)', FALSE),
    ('member', 'points_to_money', '100', 'number', '积分抵扣比例(多少积分抵扣1元)', FALSE),

    ('payment', 'wechat_enabled', 'true', 'boolean', '微信支付开关', FALSE),
    ('payment', 'alipay_enabled', 'true', 'boolean', '支付宝开关', FALSE),
    ('payment', 'wallet_enabled', 'true', 'boolean', '余额支付开关', FALSE),

    ('sms', 'code_expire', '5', 'number', '验证码有效期(分钟)', FALSE),
    ('sms', 'send_interval', '60', 'number', '发送间隔(秒)', FALSE),
    ('sms', 'daily_limit', '10', 'number', '每日发送限制', FALSE)
ON CONFLICT DO NOTHING;

-- Banner轮播图
INSERT INTO banners (title, image, link_type, link_value, position, sort, start_time, end_time, is_active) VALUES
    ('新年大促', 'https://img.example.com/banner/newyear.jpg', 'activity', '1', 'home', 100, '2026-01-01 00:00:00+08', '2026-02-28 23:59:59+08', TRUE),
    ('热销爆款', 'https://img.example.com/banner/hot.jpg', 'product', '1', 'home', 90, NULL, NULL, TRUE),
    ('会员专享', 'https://img.example.com/banner/vip.jpg', 'url', '/member', 'home', 80, NULL, NULL, TRUE),
    ('酒店特惠', 'https://img.example.com/banner/hotel.jpg', 'hotel', '1', 'hotel', 100, NULL, NULL, TRUE),
    ('情人节专场', 'https://img.example.com/banner/valentine.jpg', 'activity', '2', 'mall', 100, '2026-02-01 00:00:00+08', '2026-02-14 23:59:59+08', TRUE),
    ('新品上市', 'https://img.example.com/banner/new.jpg', 'url', '/products?is_new=1', 'mall', 90, NULL, NULL, TRUE)
ON CONFLICT DO NOTHING;

-- 文章/帮助内容
INSERT INTO articles (category, title, content, cover_image, sort, is_published, published_at) VALUES
    ('help', '如何租借智能柜产品？', '1. 扫描设备二维码\n2. 选择租借时长\n3. 完成支付\n4. 设备自动开锁\n5. 使用完毕后归还', NULL, 1, TRUE, NOW()),
    ('help', '押金如何退还？', '产品归还后，押金将在24小时内自动退还到您的原支付账户。', NULL, 2, TRUE, NOW()),
    ('help', '超时未归还怎么办？', '超过租借时长后，系统将按照超时费率自动计费。超过24小时未归还，将按购买价格扣除押金。', NULL, 3, TRUE, NOW()),
    ('faq', '产品安全吗？', '所有产品均经过严格消毒，一次性密封包装，请放心使用。', NULL, 1, TRUE, NOW()),
    ('faq', '可以退款吗？', '未使用的订单可以在24小时内申请退款。已开锁使用的订单不支持退款。', NULL, 2, TRUE, NOW()),
    ('notice', '春节营业公告', '春节期间（2026年1月28日-2月4日）正常营业，客服在线时间调整为9:00-18:00。', 'https://img.example.com/notice/spring.jpg', 1, TRUE, NOW()),
    ('about', '关于我们', '爱上杜美人是一家专注于成人情趣用品的智能零售平台，致力于为用户提供私密、便捷、安全的购物体验。', NULL, 1, TRUE, NOW())
ON CONFLICT DO NOTHING;

-- 消息模板
INSERT INTO message_templates (code, name, type, content, variables, is_active) VALUES
    ('sms_verify_code', '短信验证码', 'sms', '您的验证码是${code}，${expire}分钟内有效。如非本人操作，请忽略本短信。', '["code", "expire"]', TRUE),
    ('order_paid', '订单支付成功', 'push', '您的订单${order_no}已支付成功，金额${amount}元。', '["order_no", "amount"]', TRUE),
    ('rental_unlock', '租借开锁成功', 'push', '您已成功开锁，请在${return_time}前归还。超时将按${rate}元/小时计费。', '["return_time", "rate"]', TRUE),
    ('rental_timeout', '租借超时提醒', 'push', '您的租借即将超时，请尽快归还。超过24小时将自动按购买处理。', '[]', TRUE),
    ('booking_verified', '预订核销成功', 'push', '您的预订${booking_no}已核销成功，开锁码：${unlock_code}，请在${expire_time}前使用。', '["booking_no", "unlock_code", "expire_time"]', TRUE),
    ('withdraw_success', '提现成功', 'push', '您的提现申请已处理成功，${amount}元已转入您的账户。', '["amount"]', TRUE),
    ('commission_earned', '佣金到账', 'push', '恭喜！您获得${amount}元佣金，来自用户${user}的消费。', '["amount", "user"]', TRUE)
ON CONFLICT DO NOTHING;
