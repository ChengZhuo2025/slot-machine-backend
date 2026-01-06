// Package models 定义数据模型
package models

import "time"

// SettlementType 结算类型
const (
	SettlementTypeMerchant    = "merchant"    // 商户结算
	SettlementTypeDistributor = "distributor" // 分销商结算
)

// SettlementStatus 结算状态（字符串版本）
const (
	SettlementStatusStrPending    = "pending"    // 待结算
	SettlementStatusStrProcessing = "processing" // 结算中
	SettlementStatusStrCompleted  = "completed"  // 已完成
	SettlementStatusStrFailed     = "failed"     // 结算失败
)

// SettlementSummary 结算汇总统计
type SettlementSummary struct {
	TotalSettlements int     `json:"total_settlements"`
	TotalAmount      float64 `json:"total_amount"`
	TotalFee         float64 `json:"total_fee"`
	TotalActual      float64 `json:"total_actual"`
	PendingCount     int     `json:"pending_count"`
	CompletedCount   int     `json:"completed_count"`
}

// FinanceOverview 财务概览
type FinanceOverview struct {
	TotalRevenue       float64 `json:"total_revenue"`        // 总收入
	TotalRefund        float64 `json:"total_refund"`         // 总退款
	TotalCommission    float64 `json:"total_commission"`     // 总佣金支出
	TotalSettlement    float64 `json:"total_settlement"`     // 总结算金额
	TodayRevenue       float64 `json:"today_revenue"`        // 今日收入
	TodayOrders        int     `json:"today_orders"`         // 今日订单数
	PendingWithdrawals int     `json:"pending_withdrawals"`  // 待审核提现数
	PendingSettlements int     `json:"pending_settlements"`  // 待结算数
}

// RevenueStatistics 收入统计
type RevenueStatistics struct {
	Date    string  `json:"date"`
	Revenue float64 `json:"revenue"`
	Orders  int     `json:"orders"`
	Refund  float64 `json:"refund"`
}

// SettlementDetail 结算明细（用于导出）
type SettlementDetail struct {
	SettlementNo string    `json:"settlement_no"`
	Type         string    `json:"type"`
	TargetName   string    `json:"target_name"`
	PeriodStart  time.Time `json:"period_start"`
	PeriodEnd    time.Time `json:"period_end"`
	TotalAmount  float64   `json:"total_amount"`
	Fee          float64   `json:"fee"`
	ActualAmount float64   `json:"actual_amount"`
	OrderCount   int       `json:"order_count"`
	Status       string    `json:"status"`
	SettledAt    string    `json:"settled_at"`
	CreatedAt    time.Time `json:"created_at"`
}

// TransactionStatistics 交易统计
type TransactionStatistics struct {
	TotalRecharge      float64 `json:"total_recharge"`       // 总充值
	TotalConsume       float64 `json:"total_consume"`        // 总消费
	TotalRefund        float64 `json:"total_refund"`         // 总退款
	TotalWithdraw      float64 `json:"total_withdraw"`       // 总提现
	TotalDeposit       float64 `json:"total_deposit"`        // 总押金
	TotalReturnDeposit float64 `json:"total_return_deposit"` // 总退还押金
}

// WithdrawalSummary 提现汇总
type WithdrawalSummary struct {
	TotalWithdrawals int     `json:"total_withdrawals"`
	TotalAmount      float64 `json:"total_amount"`
	PendingCount     int     `json:"pending_count"`
	PendingAmount    float64 `json:"pending_amount"`
	ApprovedCount    int     `json:"approved_count"`
	ApprovedAmount   float64 `json:"approved_amount"`
	RejectedCount    int     `json:"rejected_count"`
}

// OrderRevenue 订单收入
type OrderRevenue struct {
	OrderType    string  `json:"order_type"`
	TotalRevenue float64 `json:"total_revenue"`
	OrderCount   int     `json:"order_count"`
}

// MerchantSettlementReport 商户结算报表
type MerchantSettlementReport struct {
	MerchantID     int64   `json:"merchant_id"`
	MerchantName   string  `json:"merchant_name"`
	TotalRevenue   float64 `json:"total_revenue"`
	TotalOrders    int     `json:"total_orders"`
	CommissionRate float64 `json:"commission_rate"`
	SettledAmount  float64 `json:"settled_amount"`
	PendingAmount  float64 `json:"pending_amount"`
}

// DistributorSettlementReport 分销商结算报表
type DistributorSettlementReport struct {
	DistributorID int64   `json:"distributor_id"`
	UserName      string  `json:"user_name"`
	TotalOrders   int     `json:"total_orders"`
	TotalAmount   float64 `json:"total_amount"`
	SettledAmount float64 `json:"settled_amount"`
	PendingAmount float64 `json:"pending_amount"`
}

// DailyRevenueReport 每日收入报表
type DailyRevenueReport struct {
	Date           string  `json:"date"`
	RentalRevenue  float64 `json:"rental_revenue"`
	RentalOrders   int     `json:"rental_orders"`
	HotelRevenue   float64 `json:"hotel_revenue"`
	HotelOrders    int     `json:"hotel_orders"`
	MallRevenue    float64 `json:"mall_revenue"`
	MallOrders     int     `json:"mall_orders"`
	TotalRevenue   float64 `json:"total_revenue"`
	TotalOrders    int     `json:"total_orders"`
	RefundAmount   float64 `json:"refund_amount"`
	RefundCount    int     `json:"refund_count"`
	NetRevenue     float64 `json:"net_revenue"`
}
