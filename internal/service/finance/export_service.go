// Package finance 提供财务管理服务
package finance

import (
	"bytes"
	"context"
	"encoding/csv"
	"fmt"
	"time"

	"gorm.io/gorm"

	"github.com/dumeirei/smart-locker-backend/internal/common/errors"
	"github.com/dumeirei/smart-locker-backend/internal/models"
	"github.com/dumeirei/smart-locker-backend/internal/repository"
)

// ExportService 报表导出服务
type ExportService struct {
	db              *gorm.DB
	settlementRepo  *repository.SettlementRepository
	transactionRepo *repository.TransactionRepository
	orderRepo       *repository.OrderRepository
	withdrawalRepo  *repository.WithdrawalRepository
}

// NewExportService 创建报表导出服务
func NewExportService(
	db *gorm.DB,
	settlementRepo *repository.SettlementRepository,
	transactionRepo *repository.TransactionRepository,
	orderRepo *repository.OrderRepository,
	withdrawalRepo *repository.WithdrawalRepository,
) *ExportService {
	return &ExportService{
		db:              db,
		settlementRepo:  settlementRepo,
		transactionRepo: transactionRepo,
		orderRepo:       orderRepo,
		withdrawalRepo:  withdrawalRepo,
	}
}

// ExportSettlementsRequest 导出结算记录请求
type ExportSettlementsRequest struct {
	Type        string     `form:"type"`
	TargetID    *int64     `form:"target_id"`
	Status      string     `form:"status"`
	PeriodStart *time.Time `form:"period_start"`
	PeriodEnd   *time.Time `form:"period_end"`
}

// ExportSettlements 导出结算记录为 CSV
func (s *ExportService) ExportSettlements(ctx context.Context, req *ExportSettlementsRequest) ([]byte, string, error) {
	// 查询数据
	filter := &repository.SettlementFilter{
		Type:        req.Type,
		TargetID:    req.TargetID,
		Status:      req.Status,
		PeriodStart: req.PeriodStart,
		PeriodEnd:   req.PeriodEnd,
	}

	settlements, _, err := s.settlementRepo.List(ctx, filter, 0, 10000)
	if err != nil {
		return nil, "", errors.ErrExportFailed.WithError(err)
	}

	// 生成 CSV
	buf := new(bytes.Buffer)
	// 添加 BOM 以支持 Excel 中文显示
	buf.Write([]byte{0xEF, 0xBB, 0xBF})

	writer := csv.NewWriter(buf)

	// 写入表头
	headers := []string{
		"结算单号", "类型", "目标ID", "结算周期开始", "结算周期结束",
		"总金额", "手续费", "实际金额", "订单数", "状态", "结算时间", "创建时间",
	}
	if err := writer.Write(headers); err != nil {
		return nil, "", errors.ErrExportFailed.WithError(err)
	}

	// 写入数据
	for _, settlement := range settlements {
		settledAt := ""
		if settlement.SettledAt != nil {
			settledAt = settlement.SettledAt.Format("2006-01-02 15:04:05")
		}

		row := []string{
			settlement.SettlementNo,
			getSettlementTypeName(settlement.Type),
			fmt.Sprintf("%d", settlement.TargetID),
			settlement.PeriodStart.Format("2006-01-02"),
			settlement.PeriodEnd.Format("2006-01-02"),
			fmt.Sprintf("%.2f", settlement.TotalAmount),
			fmt.Sprintf("%.2f", settlement.Fee),
			fmt.Sprintf("%.2f", settlement.ActualAmount),
			fmt.Sprintf("%d", settlement.OrderCount),
			getSettlementStatusName(settlement.Status),
			settledAt,
			settlement.CreatedAt.Format("2006-01-02 15:04:05"),
		}
		if err := writer.Write(row); err != nil {
			return nil, "", errors.ErrExportFailed.WithError(err)
		}
	}

	writer.Flush()
	if err := writer.Error(); err != nil {
		return nil, "", errors.ErrExportFailed.WithError(err)
	}

	filename := fmt.Sprintf("settlements_%s.csv", time.Now().Format("20060102150405"))
	return buf.Bytes(), filename, nil
}

// ExportTransactionsRequest 导出交易记录请求
type ExportTransactionsRequest struct {
	UserID    *int64     `form:"user_id"`
	Type      string     `form:"type"`
	StartTime *time.Time `form:"start_time"`
	EndTime   *time.Time `form:"end_time"`
}

// ExportTransactions 导出交易记录为 CSV
func (s *ExportService) ExportTransactions(ctx context.Context, req *ExportTransactionsRequest) ([]byte, string, error) {
	// 查询数据
	filter := &repository.TransactionFilter{
		UserID:    req.UserID,
		Type:      req.Type,
		StartDate: req.StartTime,
		EndDate:   req.EndTime,
	}

	transactions, _, err := s.transactionRepo.List(ctx, filter, 0, 50000)
	if err != nil {
		return nil, "", errors.ErrExportFailed.WithError(err)
	}

	// 生成 CSV
	buf := new(bytes.Buffer)
	buf.Write([]byte{0xEF, 0xBB, 0xBF})

	writer := csv.NewWriter(buf)

	// 写入表头
	headers := []string{
		"用户ID", "交易类型", "金额", "交易前余额", "交易后余额", "关联订单号", "备注", "创建时间",
	}
	if err := writer.Write(headers); err != nil {
		return nil, "", errors.ErrExportFailed.WithError(err)
	}

	// 写入数据
	for _, tx := range transactions {
		orderNo := ""
		if tx.OrderNo != nil {
			orderNo = *tx.OrderNo
		}
		remark := ""
		if tx.Remark != nil {
			remark = *tx.Remark
		}

		row := []string{
			fmt.Sprintf("%d", tx.UserID),
			getTransactionTypeName(tx.Type),
			fmt.Sprintf("%.2f", tx.Amount),
			fmt.Sprintf("%.2f", tx.BalanceBefore),
			fmt.Sprintf("%.2f", tx.BalanceAfter),
			orderNo,
			remark,
			tx.CreatedAt.Format("2006-01-02 15:04:05"),
		}
		if err := writer.Write(row); err != nil {
			return nil, "", errors.ErrExportFailed.WithError(err)
		}
	}

	writer.Flush()
	if err := writer.Error(); err != nil {
		return nil, "", errors.ErrExportFailed.WithError(err)
	}

	filename := fmt.Sprintf("transactions_%s.csv", time.Now().Format("20060102150405"))
	return buf.Bytes(), filename, nil
}

// ExportWithdrawalsRequest 导出提现记录请求
type ExportWithdrawalsRequest struct {
	UserID    *int64 `form:"user_id"`
	Type      string `form:"type"`
	Status    string `form:"status"`
	StartDate string `form:"start_date"`
	EndDate   string `form:"end_date"`
}

// ExportWithdrawals 导出提现记录为 CSV
func (s *ExportService) ExportWithdrawals(ctx context.Context, req *ExportWithdrawalsRequest) ([]byte, string, error) {
	// 构建查询条件
	filters := make(map[string]interface{})
	if req.UserID != nil {
		filters["user_id"] = *req.UserID
	}
	if req.Type != "" {
		filters["type"] = req.Type
	}
	if req.Status != "" {
		filters["status"] = req.Status
	}
	if req.StartDate != "" {
		if t, err := time.Parse("2006-01-02", req.StartDate); err == nil {
			filters["start_time"] = t
		}
	}
	if req.EndDate != "" {
		if t, err := time.Parse("2006-01-02", req.EndDate); err == nil {
			endOfDay := t.Add(24*time.Hour - time.Second)
			filters["end_time"] = endOfDay
		}
	}

	withdrawals, _, err := s.withdrawalRepo.List(ctx, 0, 50000, filters)
	if err != nil {
		return nil, "", errors.ErrExportFailed.WithError(err)
	}

	// 生成 CSV
	buf := new(bytes.Buffer)
	buf.Write([]byte{0xEF, 0xBB, 0xBF})

	writer := csv.NewWriter(buf)

	// 写入表头
	headers := []string{
		"提现单号", "用户ID", "提现类型", "申请金额", "手续费", "实际到账", "状态", "提现方式", "拒绝原因", "申请时间", "处理时间",
	}
	if err := writer.Write(headers); err != nil {
		return nil, "", errors.ErrExportFailed.WithError(err)
	}

	// 写入数据
	for _, w := range withdrawals {
		processedAt := ""
		if w.ProcessedAt != nil {
			processedAt = w.ProcessedAt.Format("2006-01-02 15:04:05")
		}
		rejectReason := ""
		if w.RejectReason != nil {
			rejectReason = *w.RejectReason
		}

		row := []string{
			w.WithdrawalNo,
			fmt.Sprintf("%d", w.UserID),
			getWithdrawalTypeName(w.Type),
			fmt.Sprintf("%.2f", w.Amount),
			fmt.Sprintf("%.2f", w.Fee),
			fmt.Sprintf("%.2f", w.ActualAmount),
			getWithdrawalStatusName(w.Status),
			getWithdrawToName(w.WithdrawTo),
			rejectReason,
			w.CreatedAt.Format("2006-01-02 15:04:05"),
			processedAt,
		}
		if err := writer.Write(row); err != nil {
			return nil, "", errors.ErrExportFailed.WithError(err)
		}
	}

	writer.Flush()
	if err := writer.Error(); err != nil {
		return nil, "", errors.ErrExportFailed.WithError(err)
	}

	filename := fmt.Sprintf("withdrawals_%s.csv", time.Now().Format("20060102150405"))
	return buf.Bytes(), filename, nil
}

// ExportDailyRevenueRequest 导出每日收入报表请求
type ExportDailyRevenueRequest struct {
	StartDate time.Time `form:"start_date" binding:"required"`
	EndDate   time.Time `form:"end_date" binding:"required"`
}

// ExportDailyRevenue 导出每日收入报表为 CSV
func (s *ExportService) ExportDailyRevenue(ctx context.Context, startDate, endDate time.Time) ([]byte, string, error) {
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
		return nil, "", errors.ErrExportFailed.WithError(err)
	}
	defer rows.Close()

	dateMap := make(map[string]*models.DailyRevenueReport)
	for rows.Next() {
		var date string
		var orderType string
		var revenue float64
		var orders int
		if err := rows.Scan(&date, &orderType, &revenue, &orders); err != nil {
			return nil, "", errors.ErrExportFailed.WithError(err)
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
		return nil, "", errors.ErrExportFailed.WithError(err)
	}
	defer refundRows.Close()

	for refundRows.Next() {
		var date string
		var refund float64
		var count int
		if err := refundRows.Scan(&date, &refund, &count); err != nil {
			return nil, "", errors.ErrExportFailed.WithError(err)
		}

		if report, exists := dateMap[date]; exists {
			report.RefundAmount = refund
			report.RefundCount = count
		}
	}

	// 填充日期范围
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

	// 生成 CSV
	buf := new(bytes.Buffer)
	buf.Write([]byte{0xEF, 0xBB, 0xBF})

	writer := csv.NewWriter(buf)

	// 写入表头
	headers := []string{
		"日期", "租借收入", "租借订单", "酒店收入", "酒店订单", "商城收入", "商城订单",
		"总收入", "总订单", "退款金额", "退款笔数", "净收入",
	}
	if err := writer.Write(headers); err != nil {
		return nil, "", errors.ErrExportFailed.WithError(err)
	}

	// 写入数据
	for _, r := range reports {
		row := []string{
			r.Date,
			fmt.Sprintf("%.2f", r.RentalRevenue),
			fmt.Sprintf("%d", r.RentalOrders),
			fmt.Sprintf("%.2f", r.HotelRevenue),
			fmt.Sprintf("%d", r.HotelOrders),
			fmt.Sprintf("%.2f", r.MallRevenue),
			fmt.Sprintf("%d", r.MallOrders),
			fmt.Sprintf("%.2f", r.TotalRevenue),
			fmt.Sprintf("%d", r.TotalOrders),
			fmt.Sprintf("%.2f", r.RefundAmount),
			fmt.Sprintf("%d", r.RefundCount),
			fmt.Sprintf("%.2f", r.NetRevenue),
		}
		if err := writer.Write(row); err != nil {
			return nil, "", errors.ErrExportFailed.WithError(err)
		}
	}

	writer.Flush()
	if err := writer.Error(); err != nil {
		return nil, "", errors.ErrExportFailed.WithError(err)
	}

	filename := fmt.Sprintf("daily_revenue_%s_%s.csv",
		startDate.Format("20060102"),
		endDate.Format("20060102"))
	return buf.Bytes(), filename, nil
}

// ExportMerchantSettlementReport 导出商户结算报表
func (s *ExportService) ExportMerchantSettlementReport(ctx context.Context, startDate, endDate *time.Time) ([]byte, string, error) {
	// 获取结算数据
	settlementData, err := s.settlementRepo.GetMerchantSettlements(ctx, startDate, endDate)
	if err != nil {
		return nil, "", errors.ErrExportFailed.WithError(err)
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
			return nil, "", errors.ErrExportFailed.WithError(err)
		}
	}

	merchantMap := make(map[int64]*models.Merchant)
	for i := range merchants {
		merchantMap[merchants[i].ID] = &merchants[i]
	}

	// 生成 CSV
	buf := new(bytes.Buffer)
	buf.Write([]byte{0xEF, 0xBB, 0xBF})

	writer := csv.NewWriter(buf)

	// 写入表头
	headers := []string{
		"商户ID", "商户名称", "分成比例", "总收入", "手续费", "已结算金额", "订单数",
	}
	if err := writer.Write(headers); err != nil {
		return nil, "", errors.ErrExportFailed.WithError(err)
	}

	// 写入数据
	for _, data := range settlementData {
		targetID, _ := data["target_id"].(int64)
		merchant := merchantMap[targetID]

		merchantName := ""
		commissionRate := 0.0
		if merchant != nil {
			merchantName = merchant.Name
			commissionRate = merchant.CommissionRate
		}

		totalAmount := 0.0
		if v, ok := data["total_amount"].(float64); ok {
			totalAmount = v
		}
		totalFee := 0.0
		if v, ok := data["total_fee"].(float64); ok {
			totalFee = v
		}
		actualAmount := 0.0
		if v, ok := data["actual_amount"].(float64); ok {
			actualAmount = v
		}
		orderCount := int64(0)
		if v, ok := data["order_count"].(int64); ok {
			orderCount = v
		}

		row := []string{
			fmt.Sprintf("%d", targetID),
			merchantName,
			fmt.Sprintf("%.2f%%", commissionRate*100),
			fmt.Sprintf("%.2f", totalAmount),
			fmt.Sprintf("%.2f", totalFee),
			fmt.Sprintf("%.2f", actualAmount),
			fmt.Sprintf("%d", orderCount),
		}
		if err := writer.Write(row); err != nil {
			return nil, "", errors.ErrExportFailed.WithError(err)
		}
	}

	writer.Flush()
	if err := writer.Error(); err != nil {
		return nil, "", errors.ErrExportFailed.WithError(err)
	}

	filename := fmt.Sprintf("merchant_settlement_%s.csv", time.Now().Format("20060102150405"))
	return buf.Bytes(), filename, nil
}

// 辅助函数：获取结算类型名称
func getSettlementTypeName(t string) string {
	switch t {
	case models.SettlementTypeMerchant:
		return "商户结算"
	case models.SettlementTypeDistributor:
		return "分销商结算"
	default:
		return t
	}
}

// 辅助函数：获取结算状态名称
func getSettlementStatusName(status string) string {
	switch status {
	case models.SettlementStatusPending:
		return "待结算"
	case models.SettlementStatusProcessing:
		return "处理中"
	case models.SettlementStatusCompleted:
		return "已完成"
	case models.SettlementStatusFailed:
		return "已失败"
	default:
		return status
	}
}

// 辅助函数：获取交易类型名称
func getTransactionTypeName(t string) string {
	switch t {
	case models.WalletTxTypeRecharge:
		return "充值"
	case models.WalletTxTypeConsume:
		return "消费"
	case models.WalletTxTypeRefund:
		return "退款"
	case models.WalletTxTypeWithdraw:
		return "提现"
	case models.WalletTxTypeDeposit:
		return "押金"
	case models.WalletTxTypeReturnDeposit:
		return "退还押金"
	default:
		return t
	}
}

// 辅助函数：获取提现类型名称
func getWithdrawalTypeName(t string) string {
	switch t {
	case models.WithdrawalTypeWallet:
		return "钱包提现"
	case models.WithdrawalTypeCommission:
		return "佣金提现"
	default:
		return t
	}
}

// 辅助函数：获取提现状态名称
func getWithdrawalStatusName(status string) string {
	switch status {
	case models.WithdrawalStatusPending:
		return "待审核"
	case models.WithdrawalStatusApproved:
		return "已审核"
	case models.WithdrawalStatusProcessing:
		return "打款中"
	case models.WithdrawalStatusSuccess:
		return "已完成"
	case models.WithdrawalStatusRejected:
		return "已拒绝"
	default:
		return status
	}
}

// 辅助函数：获取提现方式名称
func getWithdrawToName(withdrawTo string) string {
	switch withdrawTo {
	case models.WithdrawToWechat:
		return "微信"
	case models.WithdrawToAlipay:
		return "支付宝"
	case models.WithdrawToBank:
		return "银行卡"
	default:
		return withdrawTo
	}
}
