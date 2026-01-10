// Package admin 管理端服务
package admin

import (
	"context"
	"time"

	"gorm.io/gorm"

	"github.com/dumeirei/smart-locker-backend/internal/models"
)

// OperationDashboardService 运营仪表盘服务
type OperationDashboardService struct {
	db *gorm.DB
}

// NewOperationDashboardService 创建运营仪表盘服务
func NewOperationDashboardService(db *gorm.DB) *OperationDashboardService {
	return &OperationDashboardService{db: db}
}

// OperationOverview 运营概览数据
type OperationOverview struct {
	// 用户统计
	TotalUsers       int64   `json:"total_users"`
	TodayNewUsers    int64   `json:"today_new_users"`
	MonthNewUsers    int64   `json:"month_new_users"`
	TodayActiveUsers int64   `json:"today_active_users"`
	WeekActiveUsers  int64   `json:"week_active_users"`

	// 会员统计
	TotalMembers     int64   `json:"total_members"`      // 付费会员数
	MonthNewMembers  int64   `json:"month_new_members"`  // 本月新增会员
	MemberRevenue    float64 `json:"member_revenue"`     // 会员套餐收入

	// 营销统计
	ActiveCoupons    int64   `json:"active_coupons"`     // 有效优惠券
	UsedCoupons      int64   `json:"used_coupons"`       // 已使用优惠券
	TodayUsedCoupons int64   `json:"today_used_coupons"` // 今日使用优惠券
	ActiveCampaigns  int64   `json:"active_campaigns"`   // 进行中活动

	// 分销统计
	TotalDistributors int64  `json:"total_distributors"`    // 分销商总数
	PendingDistributors int64 `json:"pending_distributors"` // 待审核分销商
	MonthCommission   float64 `json:"month_commission"`     // 本月佣金支出

	// 内容统计
	TotalArticles    int64   `json:"total_articles"`      // 文章总数
	PublishedArticles int64  `json:"published_articles"`  // 已发布文章
	TotalBanners     int64   `json:"total_banners"`       // 轮播图总数
	ActiveBanners    int64   `json:"active_banners"`      // 启用的轮播图
}

// GetOperationOverview 获取运营概览数据
func (s *OperationDashboardService) GetOperationOverview(ctx context.Context) (*OperationOverview, error) {
	overview := &OperationOverview{}

	now := time.Now()
	today := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())
	tomorrow := today.Add(24 * time.Hour)
	monthStart := time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, now.Location())
	weekAgo := now.Add(-7 * 24 * time.Hour)

	// 用户统计
	s.db.WithContext(ctx).Model(&models.User{}).Count(&overview.TotalUsers)
	s.db.WithContext(ctx).Model(&models.User{}).
		Where("created_at >= ? AND created_at < ?", today, tomorrow).
		Count(&overview.TodayNewUsers)
	s.db.WithContext(ctx).Model(&models.User{}).
		Where("created_at >= ?", monthStart).
		Count(&overview.MonthNewUsers)
	s.db.WithContext(ctx).Model(&models.User{}).
		Where("updated_at >= ? AND updated_at < ?", today, tomorrow).
		Count(&overview.TodayActiveUsers)
	s.db.WithContext(ctx).Model(&models.User{}).
		Where("updated_at >= ?", weekAgo).
		Count(&overview.WeekActiveUsers)

	// 会员统计 - 统计购买过会员套餐的用户
	s.db.WithContext(ctx).Model(&models.User{}).
		Where("member_level_id > 1").
		Count(&overview.TotalMembers)

	// 本月新增会员（通过订单统计）
	s.db.WithContext(ctx).Model(&models.Order{}).
		Where("type = ? AND status = ? AND created_at >= ?",
			"member", models.OrderStatusCompleted, monthStart).
		Count(&overview.MonthNewMembers)

	// 会员套餐收入
	s.db.WithContext(ctx).Model(&models.Order{}).
		Where("type = ? AND status = ?", "member", models.OrderStatusCompleted).
		Select("COALESCE(SUM(actual_amount), 0)").
		Row().Scan(&overview.MemberRevenue)

	// 营销统计 - 有效优惠券
	s.db.WithContext(ctx).Model(&models.Coupon{}).
		Where("status = ? AND end_time > ?", 1, now).
		Count(&overview.ActiveCoupons)

	// 已使用优惠券
	s.db.WithContext(ctx).Model(&models.UserCoupon{}).
		Where("status = ?", models.UserCouponStatusUsed).
		Count(&overview.UsedCoupons)

	// 今日使用优惠券
	s.db.WithContext(ctx).Model(&models.UserCoupon{}).
		Where("status = ? AND used_at >= ? AND used_at < ?",
			models.UserCouponStatusUsed, today, tomorrow).
		Count(&overview.TodayUsedCoupons)

	// 进行中活动
	s.db.WithContext(ctx).Model(&models.Campaign{}).
		Where("status = ? AND start_time <= ? AND end_time >= ?", 1, now, now).
		Count(&overview.ActiveCampaigns)

	// 分销统计
	s.db.WithContext(ctx).Model(&models.Distributor{}).
		Where("status = ?", models.DistributorStatusApproved).
		Count(&overview.TotalDistributors)

	s.db.WithContext(ctx).Model(&models.Distributor{}).
		Where("status = ?", models.DistributorStatusPending).
		Count(&overview.PendingDistributors)

	// 本月佣金支出
	s.db.WithContext(ctx).Model(&models.Commission{}).
		Where("status = ? AND settled_at >= ?", models.CommissionStatusSettled, monthStart).
		Select("COALESCE(SUM(amount), 0)").
		Row().Scan(&overview.MonthCommission)

	// 内容统计
	s.db.WithContext(ctx).Model(&models.Article{}).Count(&overview.TotalArticles)
	s.db.WithContext(ctx).Model(&models.Article{}).
		Where("status = ?", models.ArticleStatusPublished).
		Count(&overview.PublishedArticles)

	s.db.WithContext(ctx).Model(&models.Banner{}).Count(&overview.TotalBanners)
	s.db.WithContext(ctx).Model(&models.Banner{}).
		Where("status = ?", models.BannerStatusActive).
		Count(&overview.ActiveBanners)

	return overview, nil
}

// UserGrowthTrend 用户增长趋势
type UserGrowthTrend struct {
	Date        string `json:"date"`
	NewUsers    int64  `json:"new_users"`
	ActiveUsers int64  `json:"active_users"`
	TotalUsers  int64  `json:"total_users"`
}

// GetUserGrowthTrend 获取用户增长趋势（最近N天）
func (s *OperationDashboardService) GetUserGrowthTrend(ctx context.Context, days int) ([]UserGrowthTrend, error) {
	if days <= 0 {
		days = 7
	}
	if days > 30 {
		days = 30
	}

	trends := make([]UserGrowthTrend, days)
	now := time.Now()

	// 获取起始日期前的累计用户数
	startDate := now.Add(-time.Duration(days) * 24 * time.Hour)
	startOfStart := time.Date(startDate.Year(), startDate.Month(), startDate.Day(), 0, 0, 0, 0, startDate.Location())
	var baseTotal int64
	s.db.WithContext(ctx).Model(&models.User{}).
		Where("created_at < ?", startOfStart).
		Count(&baseTotal)

	runningTotal := baseTotal

	for i := days - 1; i >= 0; i-- {
		date := now.Add(-time.Duration(i) * 24 * time.Hour)
		startOfDay := time.Date(date.Year(), date.Month(), date.Day(), 0, 0, 0, 0, date.Location())
		endOfDay := startOfDay.Add(24 * time.Hour)

		trend := UserGrowthTrend{
			Date: startOfDay.Format("2006-01-02"),
		}

		// 当日新增用户
		s.db.WithContext(ctx).Model(&models.User{}).
			Where("created_at >= ? AND created_at < ?", startOfDay, endOfDay).
			Count(&trend.NewUsers)

		// 当日活跃用户
		s.db.WithContext(ctx).Model(&models.User{}).
			Where("updated_at >= ? AND updated_at < ?", startOfDay, endOfDay).
			Count(&trend.ActiveUsers)

		// 累计用户
		runningTotal += trend.NewUsers
		trend.TotalUsers = runningTotal

		trends[days-1-i] = trend
	}

	return trends, nil
}

// CouponUsageStat 优惠券使用统计
type CouponUsageStat struct {
	CouponID    int64   `json:"coupon_id"`
	CouponName  string  `json:"coupon_name"`
	Type        string  `json:"type"`
	TotalCount  int64   `json:"total_count"`
	IssuedCount int64   `json:"issued_count"`
	UsedCount   int64   `json:"used_count"`
	UsageRate   float64 `json:"usage_rate"`
}

// GetCouponUsageStats 获取优惠券使用统计
func (s *OperationDashboardService) GetCouponUsageStats(ctx context.Context, limit int) ([]CouponUsageStat, error) {
	if limit <= 0 {
		limit = 10
	}

	var coupons []models.Coupon
	err := s.db.WithContext(ctx).Model(&models.Coupon{}).
		Order("used_count DESC").
		Limit(limit).
		Find(&coupons).Error

	if err != nil {
		return nil, err
	}

	results := make([]CouponUsageStat, len(coupons))
	for i, coupon := range coupons {
		results[i] = CouponUsageStat{
			CouponID:    coupon.ID,
			CouponName:  coupon.Name,
			Type:        coupon.Type,
			TotalCount:  int64(coupon.TotalCount),
			IssuedCount: int64(coupon.IssuedCount),
			UsedCount:   int64(coupon.UsedCount),
		}
		if coupon.IssuedCount > 0 {
			results[i].UsageRate = float64(coupon.UsedCount) / float64(coupon.IssuedCount) * 100
		}
	}

	return results, nil
}

// MemberLevelDistribution 会员等级分布
type MemberLevelDistribution struct {
	LevelID     int64  `json:"level_id"`
	LevelName   string `json:"level_name"`
	UserCount   int64  `json:"user_count"`
	Percentage  float64 `json:"percentage"`
}

// GetMemberLevelDistribution 获取会员等级分布
func (s *OperationDashboardService) GetMemberLevelDistribution(ctx context.Context) ([]MemberLevelDistribution, error) {
	var results []MemberLevelDistribution

	// 获取所有会员等级
	var levels []models.MemberLevel
	err := s.db.WithContext(ctx).Model(&models.MemberLevel{}).
		Order("level ASC").
		Find(&levels).Error
	if err != nil {
		return nil, err
	}

	// 统计各等级用户数
	var totalUsers int64
	s.db.WithContext(ctx).Model(&models.User{}).Count(&totalUsers)

	for _, level := range levels {
		var count int64
		s.db.WithContext(ctx).Model(&models.User{}).
			Where("member_level_id = ?", level.ID).
			Count(&count)

		result := MemberLevelDistribution{
			LevelID:   level.ID,
			LevelName: level.Name,
			UserCount: count,
		}
		if totalUsers > 0 {
			result.Percentage = float64(count) / float64(totalUsers) * 100
		}
		results = append(results, result)
	}

	return results, nil
}

// DistributorRank 分销商排行
type DistributorRank struct {
	DistributorID  int64   `json:"distributor_id"`
	UserID         int64   `json:"user_id"`
	UserPhone      string  `json:"user_phone"`
	TeamCount      int     `json:"team_count"`
	TotalCommission float64 `json:"total_commission"`
	MonthCommission float64 `json:"month_commission"`
}

// GetDistributorRank 获取分销商排行
func (s *OperationDashboardService) GetDistributorRank(ctx context.Context, limit int) ([]DistributorRank, error) {
	if limit <= 0 {
		limit = 10
	}

	now := time.Now()
	monthStart := time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, now.Location())

	var distributors []models.Distributor
	err := s.db.WithContext(ctx).Model(&models.Distributor{}).
		Preload("User").
		Where("status = ?", models.DistributorStatusApproved).
		Order("total_commission DESC").
		Limit(limit).
		Find(&distributors).Error

	if err != nil {
		return nil, err
	}

	results := make([]DistributorRank, len(distributors))
	for i, d := range distributors {
		results[i] = DistributorRank{
			DistributorID:   d.ID,
			UserID:          d.UserID,
			TeamCount:       d.TeamCount,
			TotalCommission: d.TotalCommission,
		}
		if d.User != nil {
			results[i].UserPhone = d.User.Phone
		}

		// 获取本月佣金
		var monthCommission float64
		s.db.WithContext(ctx).Model(&models.Commission{}).
			Where("distributor_id = ? AND created_at >= ?", d.ID, monthStart).
			Select("COALESCE(SUM(amount), 0)").
			Row().Scan(&monthCommission)
		results[i].MonthCommission = monthCommission
	}

	return results, nil
}

// CampaignStat 活动统计
type CampaignStat struct {
	ID           int64     `json:"id"`
	Name         string    `json:"name"`
	Type         string    `json:"type"`
	Status       int8      `json:"status"`
	StartTime    time.Time `json:"start_time"`
	EndTime      time.Time `json:"end_time"`
	Participants int64     `json:"participants"`
}

// GetActiveCampaigns 获取进行中的活动
func (s *OperationDashboardService) GetActiveCampaigns(ctx context.Context, limit int) ([]CampaignStat, error) {
	if limit <= 0 {
		limit = 10
	}

	now := time.Now()

	var campaigns []models.Campaign
	err := s.db.WithContext(ctx).Model(&models.Campaign{}).
		Where("status = ? AND start_time <= ? AND end_time >= ?", 1, now, now).
		Order("start_time DESC").
		Limit(limit).
		Find(&campaigns).Error

	if err != nil {
		return nil, err
	}

	results := make([]CampaignStat, len(campaigns))
	for i, c := range campaigns {
		results[i] = CampaignStat{
			ID:        c.ID,
			Name:      c.Name,
			Type:      c.Type,
			Status:    c.Status,
			StartTime: c.StartTime,
			EndTime:   c.EndTime,
		}
		// 这里可以根据活动类型统计参与人数，暂时设为0
		results[i].Participants = 0
	}

	return results, nil
}

// UserFeedbackStat 用户反馈统计
type UserFeedbackStat struct {
	TotalCount      int64 `json:"total_count"`
	PendingCount    int64 `json:"pending_count"`
	ProcessingCount int64 `json:"processing_count"`
	ProcessedCount  int64 `json:"processed_count"`
	TodayCount      int64 `json:"today_count"`
}

// GetUserFeedbackStats 获取用户反馈统计
func (s *OperationDashboardService) GetUserFeedbackStats(ctx context.Context) (*UserFeedbackStat, error) {
	stat := &UserFeedbackStat{}

	now := time.Now()
	today := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())
	tomorrow := today.Add(24 * time.Hour)

	s.db.WithContext(ctx).Model(&models.UserFeedback{}).Count(&stat.TotalCount)
	s.db.WithContext(ctx).Model(&models.UserFeedback{}).
		Where("status = ?", 0).
		Count(&stat.PendingCount)
	s.db.WithContext(ctx).Model(&models.UserFeedback{}).
		Where("status = ?", 1).
		Count(&stat.ProcessingCount)
	s.db.WithContext(ctx).Model(&models.UserFeedback{}).
		Where("status = ?", 2).
		Count(&stat.ProcessedCount)
	s.db.WithContext(ctx).Model(&models.UserFeedback{}).
		Where("created_at >= ? AND created_at < ?", today, tomorrow).
		Count(&stat.TodayCount)

	return stat, nil
}
