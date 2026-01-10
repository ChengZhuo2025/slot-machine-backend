// Package finance 财务服务
package finance

import (
	"context"
	"time"

	"gorm.io/gorm"

	"github.com/dumeirei/smart-locker-backend/internal/models"
)

// FinanceDashboardService 财务仪表盘服务
type FinanceDashboardService struct {
	db *gorm.DB
}

// NewFinanceDashboardService 创建财务仪表盘服务
func NewFinanceDashboardService(db *gorm.DB) *FinanceDashboardService {
	return &FinanceDashboardService{db: db}
}

// FinanceOverviewData 财务概览数据
type FinanceOverviewData struct {
	// 收入统计
	TotalRevenue       float64 `json:"total_revenue"`        // 总收入
	TodayRevenue       float64 `json:"today_revenue"`        // 今日收入
	YesterdayRevenue   float64 `json:"yesterday_revenue"`    // 昨日收入
	MonthRevenue       float64 `json:"month_revenue"`        // 本月收入
	LastMonthRevenue   float64 `json:"last_month_revenue"`   // 上月收入
	RevenueGrowthRate  float64 `json:"revenue_growth_rate"`  // 收入增长率

	// 支出统计
	TotalRefund        float64 `json:"total_refund"`         // 总退款
	TodayRefund        float64 `json:"today_refund"`         // 今日退款
	MonthRefund        float64 `json:"month_refund"`         // 本月退款
	TotalCommission    float64 `json:"total_commission"`     // 总佣金支出
	MonthCommission    float64 `json:"month_commission"`     // 本月佣金支出

	// 结算统计
	PendingSettlement  float64 `json:"pending_settlement"`   // 待结算金额
	MonthSettled       float64 `json:"month_settled"`        // 本月已结算
	TotalSettled       float64 `json:"total_settled"`        // 累计结算

	// 提现统计
	PendingWithdrawal  float64 `json:"pending_withdrawal"`   // 待审核提现金额
	PendingCount       int64   `json:"pending_count"`        // 待审核提现数量
	MonthWithdrawal    float64 `json:"month_withdrawal"`     // 本月提现金额
	TotalWithdrawal    float64 `json:"total_withdrawal"`     // 累计提现金额

	// 净收益
	TotalNetProfit     float64 `json:"total_net_profit"`     // 总净收益
	MonthNetProfit     float64 `json:"month_net_profit"`     // 本月净收益
}

// GetFinanceOverviewData 获取财务概览数据
func (s *FinanceDashboardService) GetFinanceOverviewData(ctx context.Context) (*FinanceOverviewData, error) {
	overview := &FinanceOverviewData{}

	now := time.Now()
	today := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())
	tomorrow := today.Add(24 * time.Hour)
	yesterday := today.Add(-24 * time.Hour)
	monthStart := time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, now.Location())
	lastMonthStart := monthStart.AddDate(0, -1, 0)
	lastMonthEnd := monthStart.Add(-time.Second)

	// 总收入
	s.db.WithContext(ctx).Model(&models.Payment{}).
		Where("status = ?", models.PaymentStatusSuccess).
		Select("COALESCE(SUM(amount), 0)").
		Row().Scan(&overview.TotalRevenue)

	// 今日收入
	s.db.WithContext(ctx).Model(&models.Payment{}).
		Where("status = ? AND pay_time >= ? AND pay_time < ?",
			models.PaymentStatusSuccess, today, tomorrow).
		Select("COALESCE(SUM(amount), 0)").
		Row().Scan(&overview.TodayRevenue)

	// 昨日收入
	s.db.WithContext(ctx).Model(&models.Payment{}).
		Where("status = ? AND pay_time >= ? AND pay_time < ?",
			models.PaymentStatusSuccess, yesterday, today).
		Select("COALESCE(SUM(amount), 0)").
		Row().Scan(&overview.YesterdayRevenue)

	// 本月收入
	s.db.WithContext(ctx).Model(&models.Payment{}).
		Where("status = ? AND pay_time >= ?", models.PaymentStatusSuccess, monthStart).
		Select("COALESCE(SUM(amount), 0)").
		Row().Scan(&overview.MonthRevenue)

	// 上月收入
	s.db.WithContext(ctx).Model(&models.Payment{}).
		Where("status = ? AND pay_time >= ? AND pay_time <= ?",
			models.PaymentStatusSuccess, lastMonthStart, lastMonthEnd).
		Select("COALESCE(SUM(amount), 0)").
		Row().Scan(&overview.LastMonthRevenue)

	// 计算收入增长率
	if overview.LastMonthRevenue > 0 {
		overview.RevenueGrowthRate = (overview.MonthRevenue - overview.LastMonthRevenue) / overview.LastMonthRevenue * 100
	}

	// 总退款
	s.db.WithContext(ctx).Model(&models.Refund{}).
		Where("status = ?", models.RefundStatusSuccess).
		Select("COALESCE(SUM(amount), 0)").
		Row().Scan(&overview.TotalRefund)

	// 今日退款
	s.db.WithContext(ctx).Model(&models.Refund{}).
		Where("status = ? AND created_at >= ? AND created_at < ?",
			models.RefundStatusSuccess, today, tomorrow).
		Select("COALESCE(SUM(amount), 0)").
		Row().Scan(&overview.TodayRefund)

	// 本月退款
	s.db.WithContext(ctx).Model(&models.Refund{}).
		Where("status = ? AND created_at >= ?", models.RefundStatusSuccess, monthStart).
		Select("COALESCE(SUM(amount), 0)").
		Row().Scan(&overview.MonthRefund)

	// 总佣金支出
	s.db.WithContext(ctx).Model(&models.Commission{}).
		Where("status = ?", models.CommissionStatusSettled).
		Select("COALESCE(SUM(amount), 0)").
		Row().Scan(&overview.TotalCommission)

	// 本月佣金支出
	s.db.WithContext(ctx).Model(&models.Commission{}).
		Where("status = ? AND settled_at >= ?", models.CommissionStatusSettled, monthStart).
		Select("COALESCE(SUM(amount), 0)").
		Row().Scan(&overview.MonthCommission)

	// 待结算金额
	s.db.WithContext(ctx).Model(&models.Settlement{}).
		Where("status = ?", models.SettlementStatusPending).
		Select("COALESCE(SUM(total_amount), 0)").
		Row().Scan(&overview.PendingSettlement)

	// 本月已结算
	s.db.WithContext(ctx).Model(&models.Settlement{}).
		Where("status = ? AND settled_at >= ?", models.SettlementStatusCompleted, monthStart).
		Select("COALESCE(SUM(actual_amount), 0)").
		Row().Scan(&overview.MonthSettled)

	// 累计结算
	s.db.WithContext(ctx).Model(&models.Settlement{}).
		Where("status = ?", models.SettlementStatusCompleted).
		Select("COALESCE(SUM(actual_amount), 0)").
		Row().Scan(&overview.TotalSettled)

	// 待审核提现金额和数量
	s.db.WithContext(ctx).Model(&models.Withdrawal{}).
		Where("status = ?", models.WithdrawalStatusPending).
		Select("COALESCE(SUM(amount), 0)").
		Row().Scan(&overview.PendingWithdrawal)
	s.db.WithContext(ctx).Model(&models.Withdrawal{}).
		Where("status = ?", models.WithdrawalStatusPending).
		Count(&overview.PendingCount)

	// 本月提现金额
	s.db.WithContext(ctx).Model(&models.Withdrawal{}).
		Where("status = ? AND processed_at >= ?", models.WithdrawalStatusSuccess, monthStart).
		Select("COALESCE(SUM(actual_amount), 0)").
		Row().Scan(&overview.MonthWithdrawal)

	// 累计提现金额
	s.db.WithContext(ctx).Model(&models.Withdrawal{}).
		Where("status = ?", models.WithdrawalStatusSuccess).
		Select("COALESCE(SUM(actual_amount), 0)").
		Row().Scan(&overview.TotalWithdrawal)

	// 计算净收益
	overview.TotalNetProfit = overview.TotalRevenue - overview.TotalRefund - overview.TotalCommission - overview.TotalSettled
	overview.MonthNetProfit = overview.MonthRevenue - overview.MonthRefund - overview.MonthCommission - overview.MonthSettled

	return overview, nil
}

// RevenueTrend 收入趋势数据
type RevenueTrend struct {
	Date         string  `json:"date"`
	Revenue      float64 `json:"revenue"`
	Refund       float64 `json:"refund"`
	Commission   float64 `json:"commission"`
	NetRevenue   float64 `json:"net_revenue"`
	OrderCount   int64   `json:"order_count"`
}

// GetRevenueTrend 获取收入趋势（最近N天）
func (s *FinanceDashboardService) GetRevenueTrend(ctx context.Context, days int) ([]RevenueTrend, error) {
	if days <= 0 {
		days = 7
	}
	if days > 30 {
		days = 30
	}

	trends := make([]RevenueTrend, days)
	now := time.Now()

	for i := days - 1; i >= 0; i-- {
		date := now.Add(-time.Duration(i) * 24 * time.Hour)
		startOfDay := time.Date(date.Year(), date.Month(), date.Day(), 0, 0, 0, 0, date.Location())
		endOfDay := startOfDay.Add(24 * time.Hour)

		trend := RevenueTrend{
			Date: startOfDay.Format("2006-01-02"),
		}

		// 当日收入
		s.db.WithContext(ctx).Model(&models.Payment{}).
			Where("status = ? AND pay_time >= ? AND pay_time < ?",
				models.PaymentStatusSuccess, startOfDay, endOfDay).
			Select("COALESCE(SUM(amount), 0)").
			Row().Scan(&trend.Revenue)

		// 当日退款
		s.db.WithContext(ctx).Model(&models.Refund{}).
			Where("status = ? AND created_at >= ? AND created_at < ?",
				models.RefundStatusSuccess, startOfDay, endOfDay).
			Select("COALESCE(SUM(amount), 0)").
			Row().Scan(&trend.Refund)

		// 当日佣金支出
		s.db.WithContext(ctx).Model(&models.Commission{}).
			Where("status = ? AND settled_at >= ? AND settled_at < ?",
				models.CommissionStatusSettled, startOfDay, endOfDay).
			Select("COALESCE(SUM(amount), 0)").
			Row().Scan(&trend.Commission)

		// 当日订单数
		s.db.WithContext(ctx).Model(&models.Order{}).
			Where("status NOT IN ? AND created_at >= ? AND created_at < ?",
				[]string{models.OrderStatusPending, models.OrderStatusCancelled}, startOfDay, endOfDay).
			Count(&trend.OrderCount)

		// 净收入
		trend.NetRevenue = trend.Revenue - trend.Refund - trend.Commission

		trends[days-1-i] = trend
	}

	return trends, nil
}

// PaymentChannelSummary 支付渠道汇总
type PaymentChannelSummary struct {
	Channel    string  `json:"channel"`
	Count      int64   `json:"count"`
	Amount     float64 `json:"amount"`
	Percentage float64 `json:"percentage"`
}

// GetPaymentChannelSummary 获取支付渠道汇总
func (s *FinanceDashboardService) GetPaymentChannelSummary(ctx context.Context, startDate, endDate *time.Time) ([]PaymentChannelSummary, error) {
	var results []PaymentChannelSummary

	query := s.db.WithContext(ctx).Model(&models.Payment{}).
		Select("channel, COUNT(*) as count, COALESCE(SUM(amount), 0) as amount").
		Where("status = ?", models.PaymentStatusSuccess).
		Group("channel")

	if startDate != nil {
		query = query.Where("pay_time >= ?", *startDate)
	}
	if endDate != nil {
		query = query.Where("pay_time <= ?", *endDate)
	}

	err := query.Find(&results).Error
	if err != nil {
		return nil, err
	}

	// 计算百分比
	var totalAmount float64
	for _, r := range results {
		totalAmount += r.Amount
	}
	if totalAmount > 0 {
		for i := range results {
			results[i].Percentage = results[i].Amount / totalAmount * 100
		}
	}

	return results, nil
}

// SettlementStat 结算统计
type SettlementStat struct {
	Type           string  `json:"type"`
	PendingCount   int64   `json:"pending_count"`
	PendingAmount  float64 `json:"pending_amount"`
	CompletedCount int64   `json:"completed_count"`
	CompletedAmount float64 `json:"completed_amount"`
}

// GetSettlementStats 获取结算统计
func (s *FinanceDashboardService) GetSettlementStats(ctx context.Context) ([]SettlementStat, error) {
	var results []SettlementStat

	// 商户结算统计
	var merchantPendingCount, merchantCompletedCount int64
	var merchantPendingAmount, merchantCompletedAmount float64

	s.db.WithContext(ctx).Model(&models.Settlement{}).
		Where("type = ? AND status = ?", models.SettlementTypeMerchant, models.SettlementStatusPending).
		Count(&merchantPendingCount)
	s.db.WithContext(ctx).Model(&models.Settlement{}).
		Where("type = ? AND status = ?", models.SettlementTypeMerchant, models.SettlementStatusPending).
		Select("COALESCE(SUM(total_amount), 0)").
		Row().Scan(&merchantPendingAmount)
	s.db.WithContext(ctx).Model(&models.Settlement{}).
		Where("type = ? AND status = ?", models.SettlementTypeMerchant, models.SettlementStatusCompleted).
		Count(&merchantCompletedCount)
	s.db.WithContext(ctx).Model(&models.Settlement{}).
		Where("type = ? AND status = ?", models.SettlementTypeMerchant, models.SettlementStatusCompleted).
		Select("COALESCE(SUM(actual_amount), 0)").
		Row().Scan(&merchantCompletedAmount)

	results = append(results, SettlementStat{
		Type:            "merchant",
		PendingCount:    merchantPendingCount,
		PendingAmount:   merchantPendingAmount,
		CompletedCount:  merchantCompletedCount,
		CompletedAmount: merchantCompletedAmount,
	})

	// 分销商结算统计
	var distributorPendingCount, distributorCompletedCount int64
	var distributorPendingAmount, distributorCompletedAmount float64

	s.db.WithContext(ctx).Model(&models.Settlement{}).
		Where("type = ? AND status = ?", models.SettlementTypeDistributor, models.SettlementStatusPending).
		Count(&distributorPendingCount)
	s.db.WithContext(ctx).Model(&models.Settlement{}).
		Where("type = ? AND status = ?", models.SettlementTypeDistributor, models.SettlementStatusPending).
		Select("COALESCE(SUM(total_amount), 0)").
		Row().Scan(&distributorPendingAmount)
	s.db.WithContext(ctx).Model(&models.Settlement{}).
		Where("type = ? AND status = ?", models.SettlementTypeDistributor, models.SettlementStatusCompleted).
		Count(&distributorCompletedCount)
	s.db.WithContext(ctx).Model(&models.Settlement{}).
		Where("type = ? AND status = ?", models.SettlementTypeDistributor, models.SettlementStatusCompleted).
		Select("COALESCE(SUM(actual_amount), 0)").
		Row().Scan(&distributorCompletedAmount)

	results = append(results, SettlementStat{
		Type:            "distributor",
		PendingCount:    distributorPendingCount,
		PendingAmount:   distributorPendingAmount,
		CompletedCount:  distributorCompletedCount,
		CompletedAmount: distributorCompletedAmount,
	})

	return results, nil
}

// PendingWithdrawalItem 待处理提现项
type PendingWithdrawalItem struct {
	ID           int64     `json:"id"`
	WithdrawalNo string    `json:"withdrawal_no"`
	UserID       int64     `json:"user_id"`
	UserPhone    string    `json:"user_phone"`
	Type         string    `json:"type"`
	Amount       float64   `json:"amount"`
	WithdrawTo   string    `json:"withdraw_to"`
	CreatedAt    time.Time `json:"created_at"`
}

// GetPendingWithdrawals 获取待处理提现列表
func (s *FinanceDashboardService) GetPendingWithdrawals(ctx context.Context, limit int) ([]PendingWithdrawalItem, error) {
	if limit <= 0 {
		limit = 10
	}

	var withdrawals []models.Withdrawal
	err := s.db.WithContext(ctx).Model(&models.Withdrawal{}).
		Preload("User").
		Where("status = ?", models.WithdrawalStatusPending).
		Order("created_at ASC").
		Limit(limit).
		Find(&withdrawals).Error

	if err != nil {
		return nil, err
	}

	results := make([]PendingWithdrawalItem, len(withdrawals))
	for i, w := range withdrawals {
		results[i] = PendingWithdrawalItem{
			ID:           w.ID,
			WithdrawalNo: w.WithdrawalNo,
			UserID:       w.UserID,
			Type:         w.Type,
			Amount:       w.Amount,
			WithdrawTo:   w.WithdrawTo,
			CreatedAt:    w.CreatedAt,
		}
		if w.User != nil {
			results[i].UserPhone = w.User.Phone
		}
	}

	return results, nil
}

// RefundStat 退款统计
type RefundStat struct {
	Status string  `json:"status"`
	Count  int64   `json:"count"`
	Amount float64 `json:"amount"`
}

// GetRefundStats 获取退款统计
func (s *FinanceDashboardService) GetRefundStats(ctx context.Context, startDate, endDate *time.Time) ([]RefundStat, error) {
	var results []RefundStat

	query := s.db.WithContext(ctx).Model(&models.Refund{}).
		Select("status, COUNT(*) as count, COALESCE(SUM(amount), 0) as amount").
		Group("status")

	if startDate != nil {
		query = query.Where("created_at >= ?", *startDate)
	}
	if endDate != nil {
		query = query.Where("created_at <= ?", *endDate)
	}

	err := query.Find(&results).Error
	return results, err
}
