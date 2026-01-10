// Package distribution 分销服务
package distribution

import (
	"context"
	"time"

	"gorm.io/gorm"

	"github.com/dumeirei/smart-locker-backend/internal/models"
)

// DashboardService 分销商仪表盘服务
type DashboardService struct {
	db *gorm.DB
}

// NewDashboardService 创建分销商仪表盘服务
func NewDashboardService(db *gorm.DB) *DashboardService {
	return &DashboardService{db: db}
}

// DistributorOverview 分销商概览数据
type DistributorOverview struct {
	// 佣金统计
	TotalCommission      float64 `json:"total_commission"`       // 累计佣金
	AvailableCommission  float64 `json:"available_commission"`   // 可提现佣金
	FrozenCommission     float64 `json:"frozen_commission"`      // 冻结佣金
	WithdrawnCommission  float64 `json:"withdrawn_commission"`   // 已提现佣金
	TodayCommission      float64 `json:"today_commission"`       // 今日佣金
	MonthCommission      float64 `json:"month_commission"`       // 本月佣金

	// 团队统计
	TeamCount            int64   `json:"team_count"`             // 团队人数
	DirectCount          int64   `json:"direct_count"`           // 直推人数
	TodayNewMembers      int64   `json:"today_new_members"`      // 今日新增团队成员
	MonthNewMembers      int64   `json:"month_new_members"`      // 本月新增团队成员

	// 订单统计
	TotalOrders          int64   `json:"total_orders"`           // 累计推广订单
	TodayOrders          int64   `json:"today_orders"`           // 今日推广订单
	MonthOrders          int64   `json:"month_orders"`           // 本月推广订单
	TotalOrderAmount     float64 `json:"total_order_amount"`     // 累计订单金额
}

// GetDistributorOverview 获取分销商概览数据
func (s *DashboardService) GetDistributorOverview(ctx context.Context, distributorID int64) (*DistributorOverview, error) {
	overview := &DistributorOverview{}

	now := time.Now()
	today := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())
	tomorrow := today.Add(24 * time.Hour)
	monthStart := time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, now.Location())

	// 获取分销商信息
	var distributor models.Distributor
	err := s.db.WithContext(ctx).First(&distributor, distributorID).Error
	if err != nil {
		return nil, err
	}

	overview.TotalCommission = distributor.TotalCommission
	overview.AvailableCommission = distributor.AvailableCommission
	overview.FrozenCommission = distributor.FrozenCommission
	overview.WithdrawnCommission = distributor.WithdrawnCommission
	overview.TeamCount = int64(distributor.TeamCount)
	overview.DirectCount = int64(distributor.DirectCount)

	// 今日佣金
	s.db.WithContext(ctx).Model(&models.Commission{}).
		Where("distributor_id = ? AND created_at >= ? AND created_at < ?",
			distributorID, today, tomorrow).
		Select("COALESCE(SUM(amount), 0)").
		Row().Scan(&overview.TodayCommission)

	// 本月佣金
	s.db.WithContext(ctx).Model(&models.Commission{}).
		Where("distributor_id = ? AND created_at >= ?", distributorID, monthStart).
		Select("COALESCE(SUM(amount), 0)").
		Row().Scan(&overview.MonthCommission)

	// 今日新增团队成员
	s.db.WithContext(ctx).Model(&models.Distributor{}).
		Where("parent_id = ? AND created_at >= ? AND created_at < ?",
			distributorID, today, tomorrow).
		Count(&overview.TodayNewMembers)

	// 本月新增团队成员
	s.db.WithContext(ctx).Model(&models.Distributor{}).
		Where("parent_id = ? AND created_at >= ?", distributorID, monthStart).
		Count(&overview.MonthNewMembers)

	// 累计推广订单
	s.db.WithContext(ctx).Model(&models.Commission{}).
		Where("distributor_id = ?", distributorID).
		Count(&overview.TotalOrders)

	// 今日推广订单
	s.db.WithContext(ctx).Model(&models.Commission{}).
		Where("distributor_id = ? AND created_at >= ? AND created_at < ?",
			distributorID, today, tomorrow).
		Count(&overview.TodayOrders)

	// 本月推广订单
	s.db.WithContext(ctx).Model(&models.Commission{}).
		Where("distributor_id = ? AND created_at >= ?", distributorID, monthStart).
		Count(&overview.MonthOrders)

	// 累计订单金额
	s.db.WithContext(ctx).Model(&models.Commission{}).
		Where("distributor_id = ?", distributorID).
		Select("COALESCE(SUM(order_amount), 0)").
		Row().Scan(&overview.TotalOrderAmount)

	return overview, nil
}

// CommissionTrend 佣金趋势数据
type CommissionTrend struct {
	Date             string  `json:"date"`
	Commission       float64 `json:"commission"`
	Orders           int64   `json:"orders"`
	DirectCommission float64 `json:"direct_commission"`
	IndirectCommission float64 `json:"indirect_commission"`
}

// GetCommissionTrend 获取佣金趋势（最近N天）
func (s *DashboardService) GetCommissionTrend(ctx context.Context, distributorID int64, days int) ([]CommissionTrend, error) {
	if days <= 0 {
		days = 7
	}
	if days > 30 {
		days = 30
	}

	trends := make([]CommissionTrend, days)
	now := time.Now()

	for i := days - 1; i >= 0; i-- {
		date := now.Add(-time.Duration(i) * 24 * time.Hour)
		startOfDay := time.Date(date.Year(), date.Month(), date.Day(), 0, 0, 0, 0, date.Location())
		endOfDay := startOfDay.Add(24 * time.Hour)

		trend := CommissionTrend{
			Date: startOfDay.Format("2006-01-02"),
		}

		// 当日总佣金
		s.db.WithContext(ctx).Model(&models.Commission{}).
			Where("distributor_id = ? AND created_at >= ? AND created_at < ?",
				distributorID, startOfDay, endOfDay).
			Select("COALESCE(SUM(amount), 0)").
			Row().Scan(&trend.Commission)

		// 当日订单数
		s.db.WithContext(ctx).Model(&models.Commission{}).
			Where("distributor_id = ? AND created_at >= ? AND created_at < ?",
				distributorID, startOfDay, endOfDay).
			Count(&trend.Orders)

		// 直推佣金
		s.db.WithContext(ctx).Model(&models.Commission{}).
			Where("distributor_id = ? AND type = ? AND created_at >= ? AND created_at < ?",
				distributorID, models.CommissionTypeDirect, startOfDay, endOfDay).
			Select("COALESCE(SUM(amount), 0)").
			Row().Scan(&trend.DirectCommission)

		// 间推佣金
		s.db.WithContext(ctx).Model(&models.Commission{}).
			Where("distributor_id = ? AND type = ? AND created_at >= ? AND created_at < ?",
				distributorID, models.CommissionTypeIndirect, startOfDay, endOfDay).
			Select("COALESCE(SUM(amount), 0)").
			Row().Scan(&trend.IndirectCommission)

		trends[days-1-i] = trend
	}

	return trends, nil
}

// TeamMemberRank 团队成员业绩排行
type TeamMemberRank struct {
	UserID      int64   `json:"user_id"`
	UserPhone   string  `json:"user_phone"`
	UserNickname string `json:"user_nickname"`
	Level       int     `json:"level"`
	OrderCount  int64   `json:"order_count"`
	OrderAmount float64 `json:"order_amount"`
	Commission  float64 `json:"commission"`
}

// GetTeamRank 获取团队成员排行
func (s *DashboardService) GetTeamRank(ctx context.Context, distributorID int64, limit int, startDate, endDate *time.Time) ([]TeamMemberRank, error) {
	if limit <= 0 {
		limit = 10
	}

	var results []TeamMemberRank

	// 获取下级分销商
	query := s.db.WithContext(ctx).Table("distributors d").
		Select(`
			d.user_id,
			u.phone as user_phone,
			u.nickname as user_nickname,
			d.level,
			COUNT(DISTINCT c.order_id) as order_count,
			COALESCE(SUM(c.order_amount), 0) as order_amount,
			COALESCE(SUM(c.amount), 0) as commission
		`).
		Joins("JOIN users u ON d.user_id = u.id").
		Joins("LEFT JOIN commissions c ON c.from_user_id = d.user_id").
		Where("d.parent_id = ?", distributorID).
		Group("d.user_id, u.phone, u.nickname, d.level").
		Order("commission DESC").
		Limit(limit)

	if startDate != nil {
		query = query.Where("c.created_at >= ?", *startDate)
	}
	if endDate != nil {
		query = query.Where("c.created_at <= ?", *endDate)
	}

	err := query.Find(&results).Error
	return results, err
}

// CommissionRecord 佣金记录
type CommissionRecord struct {
	ID           int64     `json:"id"`
	OrderID      int64     `json:"order_id"`
	OrderNo      string    `json:"order_no"`
	FromUserID   int64     `json:"from_user_id"`
	FromUserPhone string   `json:"from_user_phone"`
	Type         string    `json:"type"`
	OrderAmount  float64   `json:"order_amount"`
	Rate         float64   `json:"rate"`
	Amount       float64   `json:"amount"`
	Status       int8      `json:"status"`
	CreatedAt    time.Time `json:"created_at"`
}

// GetRecentCommissions 获取最近佣金记录
func (s *DashboardService) GetRecentCommissions(ctx context.Context, distributorID int64, limit int) ([]CommissionRecord, error) {
	if limit <= 0 {
		limit = 10
	}

	var commissions []models.Commission
	err := s.db.WithContext(ctx).Model(&models.Commission{}).
		Preload("Order").
		Preload("FromUser").
		Where("distributor_id = ?", distributorID).
		Order("created_at DESC").
		Limit(limit).
		Find(&commissions).Error

	if err != nil {
		return nil, err
	}

	results := make([]CommissionRecord, len(commissions))
	for i, commission := range commissions {
		results[i] = CommissionRecord{
			ID:          commission.ID,
			OrderID:     commission.OrderID,
			FromUserID:  commission.FromUserID,
			Type:        commission.Type,
			OrderAmount: commission.OrderAmount,
			Rate:        commission.Rate,
			Amount:      commission.Amount,
			Status:      commission.Status,
			CreatedAt:   commission.CreatedAt,
		}
		if commission.Order != nil {
			results[i].OrderNo = commission.Order.OrderNo
		}
		if commission.FromUser != nil {
			results[i].FromUserPhone = commission.FromUser.Phone
		}
	}

	return results, nil
}

// WithdrawalRecord 提现记录
type WithdrawalRecord struct {
	ID           int64      `json:"id"`
	WithdrawalNo string     `json:"withdrawal_no"`
	Amount       float64    `json:"amount"`
	Fee          float64    `json:"fee"`
	ActualAmount float64    `json:"actual_amount"`
	WithdrawTo   string     `json:"withdraw_to"`
	Status       string     `json:"status"`
	CreatedAt    time.Time  `json:"created_at"`
	ProcessedAt  *time.Time `json:"processed_at,omitempty"`
}

// GetRecentWithdrawals 获取最近提现记录
func (s *DashboardService) GetRecentWithdrawals(ctx context.Context, userID int64, limit int) ([]WithdrawalRecord, error) {
	if limit <= 0 {
		limit = 10
	}

	var withdrawals []models.Withdrawal
	err := s.db.WithContext(ctx).Model(&models.Withdrawal{}).
		Where("user_id = ? AND type = ?", userID, models.WithdrawalTypeCommission).
		Order("created_at DESC").
		Limit(limit).
		Find(&withdrawals).Error

	if err != nil {
		return nil, err
	}

	results := make([]WithdrawalRecord, len(withdrawals))
	for i, w := range withdrawals {
		results[i] = WithdrawalRecord{
			ID:           w.ID,
			WithdrawalNo: w.WithdrawalNo,
			Amount:       w.Amount,
			Fee:          w.Fee,
			ActualAmount: w.ActualAmount,
			WithdrawTo:   w.WithdrawTo,
			Status:       w.Status,
			CreatedAt:    w.CreatedAt,
			ProcessedAt:  w.ProcessedAt,
		}
	}

	return results, nil
}

// CommissionTypeSummary 佣金类型汇总
type CommissionTypeSummary struct {
	Type       string  `json:"type"`
	Count      int64   `json:"count"`
	TotalAmount float64 `json:"total_amount"`
}

// GetCommissionTypeSummary 获取佣金类型汇总
func (s *DashboardService) GetCommissionTypeSummary(ctx context.Context, distributorID int64, startDate, endDate *time.Time) ([]CommissionTypeSummary, error) {
	var results []CommissionTypeSummary

	query := s.db.WithContext(ctx).Model(&models.Commission{}).
		Select("type, COUNT(*) as count, COALESCE(SUM(amount), 0) as total_amount").
		Where("distributor_id = ?", distributorID).
		Group("type")

	if startDate != nil {
		query = query.Where("created_at >= ?", *startDate)
	}
	if endDate != nil {
		query = query.Where("created_at <= ?", *endDate)
	}

	err := query.Find(&results).Error
	return results, err
}
