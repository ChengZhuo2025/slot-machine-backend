// Package finance 提供财务管理服务
package finance

import (
	"context"
	"time"

	"gorm.io/gorm"

	"github.com/dumeirei/smart-locker-backend/internal/models"
	"github.com/dumeirei/smart-locker-backend/internal/repository"
)

// StatisticsService 财务统计服务
type StatisticsService struct {
	db              *gorm.DB
	settlementRepo  *repository.SettlementRepository
	transactionRepo *repository.TransactionRepository
	orderRepo       *repository.OrderRepository
	paymentRepo     *repository.PaymentRepository
	commissionRepo  *repository.CommissionRepository
	withdrawalRepo  *repository.WithdrawalRepository
}

// NewStatisticsService 创建财务统计服务
func NewStatisticsService(
	db *gorm.DB,
	settlementRepo *repository.SettlementRepository,
	transactionRepo *repository.TransactionRepository,
	orderRepo *repository.OrderRepository,
	paymentRepo *repository.PaymentRepository,
	commissionRepo *repository.CommissionRepository,
	withdrawalRepo *repository.WithdrawalRepository,
) *StatisticsService {
	return &StatisticsService{
		db:              db,
		settlementRepo:  settlementRepo,
		transactionRepo: transactionRepo,
		orderRepo:       orderRepo,
		paymentRepo:     paymentRepo,
		commissionRepo:  commissionRepo,
		withdrawalRepo:  withdrawalRepo,
	}
}

// GetFinanceOverview 获取财务概览
func (s *StatisticsService) GetFinanceOverview(ctx context.Context) (*models.FinanceOverview, error) {
	overview := &models.FinanceOverview{}

	// 总收入 - 从成功支付记录汇总
	err := s.db.WithContext(ctx).Model(&models.Payment{}).
		Where("status = ?", models.PaymentStatusSuccess).
		Select("COALESCE(SUM(amount), 0)").
		Row().Scan(&overview.TotalRevenue)
	if err != nil {
		return nil, err
	}

	// 总退款
	err = s.db.WithContext(ctx).Model(&models.Refund{}).
		Where("status = ?", models.RefundStatusSuccess).
		Select("COALESCE(SUM(amount), 0)").
		Row().Scan(&overview.TotalRefund)
	if err != nil {
		return nil, err
	}

	// 总佣金支出 - 已结算的佣金
	err = s.db.WithContext(ctx).Model(&models.Commission{}).
		Where("status = ?", models.CommissionStatusSettled).
		Select("COALESCE(SUM(amount), 0)").
		Row().Scan(&overview.TotalCommission)
	if err != nil {
		return nil, err
	}

	// 总结算金额
	err = s.db.WithContext(ctx).Model(&models.Settlement{}).
		Where("status = ?", models.SettlementStatusCompleted).
		Select("COALESCE(SUM(actual_amount), 0)").
		Row().Scan(&overview.TotalSettlement)
	if err != nil {
		return nil, err
	}

	// 今日收入
	today := time.Now().Truncate(24 * time.Hour)
	tomorrow := today.Add(24 * time.Hour)
	err = s.db.WithContext(ctx).Model(&models.Payment{}).
		Where("status = ? AND pay_time >= ? AND pay_time < ?", models.PaymentStatusSuccess, today, tomorrow).
		Select("COALESCE(SUM(amount), 0)").
		Row().Scan(&overview.TodayRevenue)
	if err != nil {
		return nil, err
	}

	// 今日订单数
	var todayOrders int64
	err = s.db.WithContext(ctx).Model(&models.Order{}).
		Where("created_at >= ? AND created_at < ?", today, tomorrow).
		Count(&todayOrders).Error
	if err != nil {
		return nil, err
	}
	overview.TodayOrders = int(todayOrders)

	// 待审核提现数
	var pendingWithdrawals int64
	err = s.db.WithContext(ctx).Model(&models.Withdrawal{}).
		Where("status = ?", models.WithdrawalStatusPending).
		Count(&pendingWithdrawals).Error
	if err != nil {
		return nil, err
	}
	overview.PendingWithdrawals = int(pendingWithdrawals)

	// 待结算数
	var pendingSettlements int64
	err = s.db.WithContext(ctx).Model(&models.Settlement{}).
		Where("status = ?", models.SettlementStatusPending).
		Count(&pendingSettlements).Error
	if err != nil {
		return nil, err
	}
	overview.PendingSettlements = int(pendingSettlements)

	return overview, nil
}

// GetRevenueStatistics 获取收入统计
func (s *StatisticsService) GetRevenueStatistics(ctx context.Context, startDate, endDate time.Time) ([]models.RevenueStatistics, error) {
	var results []models.RevenueStatistics

	// 按天统计收入
	rows, err := s.db.WithContext(ctx).Model(&models.Payment{}).
		Select(
			"DATE(pay_time) as date",
			"COALESCE(SUM(amount), 0) as revenue",
		).
		Where("status = ? AND pay_time >= ? AND pay_time <= ?", models.PaymentStatusSuccess, startDate, endDate).
		Group("DATE(pay_time)").
		Order("date ASC").
		Rows()
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	dateMap := make(map[string]*models.RevenueStatistics)
	for rows.Next() {
		var stat models.RevenueStatistics
		if err := rows.Scan(&stat.Date, &stat.Revenue); err != nil {
			return nil, err
		}
		dateMap[stat.Date] = &stat
	}

	// 按天统计订单数
	rows, err = s.db.WithContext(ctx).Model(&models.Order{}).
		Select(
			"DATE(paid_at) as date",
			"COUNT(*) as orders",
		).
		Where("status NOT IN (?, ?) AND paid_at >= ? AND paid_at <= ?", models.OrderStatusPending, models.OrderStatusCancelled, startDate, endDate).
		Group("DATE(paid_at)").
		Rows()
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	for rows.Next() {
		var date string
		var orders int
		if err := rows.Scan(&date, &orders); err != nil {
			return nil, err
		}
		if stat, exists := dateMap[date]; exists {
			stat.Orders = orders
		}
	}

	// 按天统计退款
	rows, err = s.db.WithContext(ctx).Model(&models.Refund{}).
		Select(
			"DATE(refunded_at) as date",
			"COALESCE(SUM(amount), 0) as refund",
		).
		Where("status = ? AND refunded_at >= ? AND refunded_at <= ?", models.RefundStatusSuccess, startDate, endDate).
		Group("DATE(refunded_at)").
		Rows()
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	for rows.Next() {
		var date string
		var refund float64
		if err := rows.Scan(&date, &refund); err != nil {
			return nil, err
		}
		if stat, exists := dateMap[date]; exists {
			stat.Refund = refund
		}
	}

	// 填充日期范围内所有日期
	current := startDate
	for current.Before(endDate) || current.Equal(endDate) {
		dateStr := current.Format("2006-01-02")
		if stat, exists := dateMap[dateStr]; exists {
			results = append(results, *stat)
		} else {
			results = append(results, models.RevenueStatistics{
				Date:    dateStr,
				Revenue: 0,
				Orders:  0,
				Refund:  0,
			})
		}
		current = current.Add(24 * time.Hour)
	}

	return results, nil
}

// GetOrderRevenueByType 按订单类型获取收入统计
func (s *StatisticsService) GetOrderRevenueByType(ctx context.Context, startDate, endDate *time.Time) ([]models.OrderRevenue, error) {
	var results []models.OrderRevenue

	query := s.db.WithContext(ctx).Model(&models.Order{}).
		Select(
			"type as order_type",
			"COALESCE(SUM(actual_amount), 0) as total_revenue",
			"COUNT(*) as order_count",
		).
		Where("status NOT IN (?, ?)", models.OrderStatusPending, models.OrderStatusCancelled).
		Group("type")

	if startDate != nil {
		query = query.Where("paid_at >= ?", *startDate)
	}
	if endDate != nil {
		query = query.Where("paid_at <= ?", *endDate)
	}

	err := query.Find(&results).Error
	return results, err
}

// GetDailyRevenueReport 获取每日收入报表
func (s *StatisticsService) GetDailyRevenueReport(ctx context.Context, startDate, endDate time.Time) ([]models.DailyRevenueReport, error) {
	var reports []models.DailyRevenueReport

	// 按日期和订单类型统计
	rows, err := s.db.WithContext(ctx).Model(&models.Order{}).
		Select(
			"DATE(paid_at) as date",
			"type",
			"COALESCE(SUM(actual_amount), 0) as revenue",
			"COUNT(*) as orders",
		).
		Where("status NOT IN (?, ?) AND paid_at >= ? AND paid_at <= ?",
			models.OrderStatusPending, models.OrderStatusCancelled, startDate, endDate).
		Group("DATE(paid_at), type").
		Order("date ASC").
		Rows()
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	// 按日期聚合数据
	dateMap := make(map[string]*models.DailyRevenueReport)
	for rows.Next() {
		var date string
		var orderType string
		var revenue float64
		var orders int
		if err := rows.Scan(&date, &orderType, &revenue, &orders); err != nil {
			return nil, err
		}

		report, exists := dateMap[date]
		if !exists {
			report = &models.DailyRevenueReport{Date: date}
			dateMap[date] = report
		}

		switch orderType {
		case models.OrderTypeRental:
			report.RentalRevenue = revenue
			report.RentalOrders = orders
		case models.OrderTypeHotel:
			report.HotelRevenue = revenue
			report.HotelOrders = orders
		case models.OrderTypeMall:
			report.MallRevenue = revenue
			report.MallOrders = orders
		}
		report.TotalRevenue += revenue
		report.TotalOrders += orders
	}

	// 统计退款
	refundRows, err := s.db.WithContext(ctx).Model(&models.Refund{}).
		Select(
			"DATE(refunded_at) as date",
			"COALESCE(SUM(amount), 0) as refund",
			"COUNT(*) as count",
		).
		Where("status = ? AND refunded_at >= ? AND refunded_at <= ?",
			models.RefundStatusSuccess, startDate, endDate).
		Group("DATE(refunded_at)").
		Rows()
	if err != nil {
		return nil, err
	}
	defer refundRows.Close()

	for refundRows.Next() {
		var date string
		var refund float64
		var count int
		if err := refundRows.Scan(&date, &refund, &count); err != nil {
			return nil, err
		}

		if report, exists := dateMap[date]; exists {
			report.RefundAmount = refund
			report.RefundCount = count
			report.NetRevenue = report.TotalRevenue - refund
		}
	}

	// 填充日期范围并转换为切片
	current := startDate
	for current.Before(endDate) || current.Equal(endDate) {
		dateStr := current.Format("2006-01-02")
		if report, exists := dateMap[dateStr]; exists {
			report.NetRevenue = report.TotalRevenue - report.RefundAmount
			reports = append(reports, *report)
		} else {
			reports = append(reports, models.DailyRevenueReport{Date: dateStr})
		}
		current = current.Add(24 * time.Hour)
	}

	return reports, nil
}

// GetTransactionStatistics 获取交易统计
func (s *StatisticsService) GetTransactionStatistics(ctx context.Context, startDate, endDate *time.Time) (*models.TransactionStatistics, error) {
	return s.transactionRepo.GetStatistics(ctx, startDate, endDate)
}

// GetSettlementSummary 获取结算汇总
func (s *StatisticsService) GetSettlementSummary(ctx context.Context, settlementType string, startDate, endDate *time.Time) (*models.SettlementSummary, error) {
	return s.settlementRepo.GetSummary(ctx, settlementType, startDate, endDate)
}

// GetWithdrawalSummary 获取提现汇总
func (s *StatisticsService) GetWithdrawalSummary(ctx context.Context, startDate, endDate *time.Time) (*models.WithdrawalSummary, error) {
	var summary models.WithdrawalSummary

	query := s.db.WithContext(ctx).Model(&models.Withdrawal{})
	if startDate != nil {
		query = query.Where("created_at >= ?", *startDate)
	}
	if endDate != nil {
		query = query.Where("created_at <= ?", *endDate)
	}

	// 总数和总金额
	var totalCount int64
	err := query.Count(&totalCount).Error
	if err != nil {
		return nil, err
	}
	summary.TotalWithdrawals = int(totalCount)

	err = s.db.WithContext(ctx).Model(&models.Withdrawal{}).
		Select("COALESCE(SUM(amount), 0)").
		Row().Scan(&summary.TotalAmount)
	if err != nil {
		return nil, err
	}

	// 待审核
	var pendingCount int64
	err = s.db.WithContext(ctx).Model(&models.Withdrawal{}).
		Where("status = ?", models.WithdrawalStatusPending).
		Count(&pendingCount).Error
	if err != nil {
		return nil, err
	}
	summary.PendingCount = int(pendingCount)

	err = s.db.WithContext(ctx).Model(&models.Withdrawal{}).
		Where("status = ?", models.WithdrawalStatusPending).
		Select("COALESCE(SUM(amount), 0)").
		Row().Scan(&summary.PendingAmount)
	if err != nil {
		return nil, err
	}

	// 已通过
	var approvedCount int64
	err = s.db.WithContext(ctx).Model(&models.Withdrawal{}).
		Where("status = ?", models.WithdrawalStatusSuccess).
		Count(&approvedCount).Error
	if err != nil {
		return nil, err
	}
	summary.ApprovedCount = int(approvedCount)

	err = s.db.WithContext(ctx).Model(&models.Withdrawal{}).
		Where("status = ?", models.WithdrawalStatusSuccess).
		Select("COALESCE(SUM(actual_amount), 0)").
		Row().Scan(&summary.ApprovedAmount)
	if err != nil {
		return nil, err
	}

	// 已拒绝
	var rejectedCount int64
	err = s.db.WithContext(ctx).Model(&models.Withdrawal{}).
		Where("status = ?", models.WithdrawalStatusRejected).
		Count(&rejectedCount).Error
	if err != nil {
		return nil, err
	}
	summary.RejectedCount = int(rejectedCount)

	return &summary, nil
}

// GetMerchantSettlementReport 获取商户结算报表
func (s *StatisticsService) GetMerchantSettlementReport(ctx context.Context, startDate, endDate *time.Time) ([]models.MerchantSettlementReport, error) {
	var reports []models.MerchantSettlementReport

	// 获取结算数据
	settlementData, err := s.settlementRepo.GetMerchantSettlements(ctx, startDate, endDate)
	if err != nil {
		return nil, err
	}

	// 获取商户信息
	merchantIDs := make([]int64, 0, len(settlementData))
	for _, data := range settlementData {
		if id, ok := data["target_id"].(int64); ok {
			merchantIDs = append(merchantIDs, id)
		}
	}

	var merchants []models.Merchant
	if len(merchantIDs) > 0 {
		err = s.db.WithContext(ctx).Where("id IN ?", merchantIDs).Find(&merchants).Error
		if err != nil {
			return nil, err
		}
	}

	merchantMap := make(map[int64]*models.Merchant)
	for i := range merchants {
		merchantMap[merchants[i].ID] = &merchants[i]
	}

	for _, data := range settlementData {
		targetID, _ := data["target_id"].(int64)
		merchant := merchantMap[targetID]

		report := models.MerchantSettlementReport{
			MerchantID: targetID,
		}
		if merchant != nil {
			report.MerchantName = merchant.Name
			report.CommissionRate = merchant.CommissionRate
		}

		if total, ok := data["total_amount"].(float64); ok {
			report.TotalRevenue = total
		}
		if actual, ok := data["actual_amount"].(float64); ok {
			report.SettledAmount = actual
		}
		if count, ok := data["order_count"].(int64); ok {
			report.TotalOrders = int(count)
		}

		reports = append(reports, report)
	}

	return reports, nil
}
