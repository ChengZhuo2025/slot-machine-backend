// Package admin 管理端 HTTP Handler
package admin

import (
	"strconv"
	"time"

	"github.com/gin-gonic/gin"

	"github.com/dumeirei/smart-locker-backend/internal/common/response"
	adminService "github.com/dumeirei/smart-locker-backend/internal/service/admin"
	distributionService "github.com/dumeirei/smart-locker-backend/internal/service/distribution"
	financeService "github.com/dumeirei/smart-locker-backend/internal/service/finance"
)

// DashboardHandler 仪表盘处理器
type DashboardHandler struct {
	dashboardService     *adminService.DashboardService
	operationService     *adminService.OperationDashboardService
	distributionDashboard *distributionService.DashboardService
	financeDashboard     *financeService.FinanceDashboardService
}

// NewDashboardHandler 创建仪表盘处理器
func NewDashboardHandler(
	dashboardSvc *adminService.DashboardService,
	operationSvc *adminService.OperationDashboardService,
	distributionDashboardSvc *distributionService.DashboardService,
	financeDashboardSvc *financeService.FinanceDashboardService,
) *DashboardHandler {
	return &DashboardHandler{
		dashboardService:      dashboardSvc,
		operationService:      operationSvc,
		distributionDashboard: distributionDashboardSvc,
		financeDashboard:      financeDashboardSvc,
	}
}

// ==================== 平台管理员仪表盘 ====================

// GetPlatformOverview 获取平台概览
// @Summary 获取平台概览
// @Tags 管理-仪表盘
// @Produce json
// @Security Bearer
// @Success 200 {object} response.Response{data=adminService.PlatformOverview}
// @Router /api/v1/admin/dashboard/platform/overview [get]
func (h *DashboardHandler) GetPlatformOverview(c *gin.Context) {
	overview, err := h.dashboardService.GetPlatformOverview(c.Request.Context())
	if err != nil {
		response.InternalError(c, err.Error())
		return
	}
	response.Success(c, overview)
}

// GetOrderTrend 获取订单趋势
// @Summary 获取订单趋势
// @Tags 管理-仪表盘
// @Produce json
// @Security Bearer
// @Param days query int false "天数" default(7)
// @Success 200 {object} response.Response{data=[]adminService.OrderTrend}
// @Router /api/v1/admin/dashboard/platform/order-trend [get]
func (h *DashboardHandler) GetOrderTrend(c *gin.Context) {
	days, _ := strconv.Atoi(c.DefaultQuery("days", "7"))

	trends, err := h.dashboardService.GetOrderTrend(c.Request.Context(), days)
	if err != nil {
		response.InternalError(c, err.Error())
		return
	}
	response.Success(c, trends)
}

// GetDeviceStatusSummary 获取设备状态汇总
// @Summary 获取设备状态汇总
// @Tags 管理-仪表盘
// @Produce json
// @Security Bearer
// @Success 200 {object} response.Response{data=[]adminService.DeviceStatusSummary}
// @Router /api/v1/admin/dashboard/platform/device-status [get]
func (h *DashboardHandler) GetDeviceStatusSummary(c *gin.Context) {
	summary, err := h.dashboardService.GetDeviceStatusSummary(c.Request.Context())
	if err != nil {
		response.InternalError(c, err.Error())
		return
	}
	response.Success(c, summary)
}

// GetOrderTypeSummary 获取订单类型汇总
// @Summary 获取订单类型汇总
// @Tags 管理-仪表盘
// @Produce json
// @Security Bearer
// @Param start_date query string false "开始日期 YYYY-MM-DD"
// @Param end_date query string false "结束日期 YYYY-MM-DD"
// @Success 200 {object} response.Response{data=[]adminService.OrderTypeSummary}
// @Router /api/v1/admin/dashboard/platform/order-type [get]
func (h *DashboardHandler) GetOrderTypeSummary(c *gin.Context) {
	var startDate, endDate *time.Time
	if s := c.Query("start_date"); s != "" {
		t, _ := time.Parse("2006-01-02", s)
		startDate = &t
	}
	if s := c.Query("end_date"); s != "" {
		t, _ := time.Parse("2006-01-02", s)
		endOfDay := t.Add(24*time.Hour - time.Second)
		endDate = &endOfDay
	}

	summary, err := h.dashboardService.GetOrderTypeSummary(c.Request.Context(), startDate, endDate)
	if err != nil {
		response.InternalError(c, err.Error())
		return
	}
	response.Success(c, summary)
}

// GetTopVenues 获取热门场地
// @Summary 获取热门场地
// @Tags 管理-仪表盘
// @Produce json
// @Security Bearer
// @Param limit query int false "数量" default(10)
// @Param start_date query string false "开始日期 YYYY-MM-DD"
// @Param end_date query string false "结束日期 YYYY-MM-DD"
// @Success 200 {object} response.Response{data=[]adminService.TopVenue}
// @Router /api/v1/admin/dashboard/platform/top-venues [get]
func (h *DashboardHandler) GetTopVenues(c *gin.Context) {
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "10"))

	var startDate, endDate *time.Time
	if s := c.Query("start_date"); s != "" {
		t, _ := time.Parse("2006-01-02", s)
		startDate = &t
	}
	if s := c.Query("end_date"); s != "" {
		t, _ := time.Parse("2006-01-02", s)
		endOfDay := t.Add(24*time.Hour - time.Second)
		endDate = &endOfDay
	}

	venues, err := h.dashboardService.GetTopVenues(c.Request.Context(), limit, startDate, endDate)
	if err != nil {
		response.InternalError(c, err.Error())
		return
	}
	response.Success(c, venues)
}

// GetRecentOrders 获取最近订单
// @Summary 获取最近订单
// @Tags 管理-仪表盘
// @Produce json
// @Security Bearer
// @Param limit query int false "数量" default(10)
// @Success 200 {object} response.Response{data=[]adminService.RecentOrder}
// @Router /api/v1/admin/dashboard/platform/recent-orders [get]
func (h *DashboardHandler) GetRecentOrders(c *gin.Context) {
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "10"))

	orders, err := h.dashboardService.GetRecentOrders(c.Request.Context(), limit)
	if err != nil {
		response.InternalError(c, err.Error())
		return
	}
	response.Success(c, orders)
}

// GetAlerts 获取告警信息
// @Summary 获取告警信息
// @Tags 管理-仪表盘
// @Produce json
// @Security Bearer
// @Param limit query int false "数量" default(10)
// @Success 200 {object} response.Response{data=[]adminService.AlertInfo}
// @Router /api/v1/admin/dashboard/platform/alerts [get]
func (h *DashboardHandler) GetAlerts(c *gin.Context) {
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "10"))

	alerts, err := h.dashboardService.GetAlerts(c.Request.Context(), limit)
	if err != nil {
		response.InternalError(c, err.Error())
		return
	}
	response.Success(c, alerts)
}

// ==================== 运营仪表盘 ====================

// GetOperationOverview 获取运营概览
// @Summary 获取运营概览
// @Tags 管理-仪表盘
// @Produce json
// @Security Bearer
// @Success 200 {object} response.Response{data=adminService.OperationOverview}
// @Router /api/v1/admin/dashboard/operation/overview [get]
func (h *DashboardHandler) GetOperationOverview(c *gin.Context) {
	overview, err := h.operationService.GetOperationOverview(c.Request.Context())
	if err != nil {
		response.InternalError(c, err.Error())
		return
	}
	response.Success(c, overview)
}

// GetUserGrowthTrend 获取用户增长趋势
// @Summary 获取用户增长趋势
// @Tags 管理-仪表盘
// @Produce json
// @Security Bearer
// @Param days query int false "天数" default(7)
// @Success 200 {object} response.Response{data=[]adminService.UserGrowthTrend}
// @Router /api/v1/admin/dashboard/operation/user-growth [get]
func (h *DashboardHandler) GetUserGrowthTrend(c *gin.Context) {
	days, _ := strconv.Atoi(c.DefaultQuery("days", "7"))

	trends, err := h.operationService.GetUserGrowthTrend(c.Request.Context(), days)
	if err != nil {
		response.InternalError(c, err.Error())
		return
	}
	response.Success(c, trends)
}

// GetCouponUsageStats 获取优惠券使用统计
// @Summary 获取优惠券使用统计
// @Tags 管理-仪表盘
// @Produce json
// @Security Bearer
// @Param limit query int false "数量" default(10)
// @Success 200 {object} response.Response{data=[]adminService.CouponUsageStat}
// @Router /api/v1/admin/dashboard/operation/coupon-usage [get]
func (h *DashboardHandler) GetCouponUsageStats(c *gin.Context) {
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "10"))

	stats, err := h.operationService.GetCouponUsageStats(c.Request.Context(), limit)
	if err != nil {
		response.InternalError(c, err.Error())
		return
	}
	response.Success(c, stats)
}

// GetMemberLevelDistribution 获取会员等级分布
// @Summary 获取会员等级分布
// @Tags 管理-仪表盘
// @Produce json
// @Security Bearer
// @Success 200 {object} response.Response{data=[]adminService.MemberLevelDistribution}
// @Router /api/v1/admin/dashboard/operation/member-distribution [get]
func (h *DashboardHandler) GetMemberLevelDistribution(c *gin.Context) {
	distribution, err := h.operationService.GetMemberLevelDistribution(c.Request.Context())
	if err != nil {
		response.InternalError(c, err.Error())
		return
	}
	response.Success(c, distribution)
}

// GetDistributorRank 获取分销商排行
// @Summary 获取分销商排行
// @Tags 管理-仪表盘
// @Produce json
// @Security Bearer
// @Param limit query int false "数量" default(10)
// @Success 200 {object} response.Response{data=[]adminService.DistributorRank}
// @Router /api/v1/admin/dashboard/operation/distributor-rank [get]
func (h *DashboardHandler) GetDistributorRank(c *gin.Context) {
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "10"))

	rank, err := h.operationService.GetDistributorRank(c.Request.Context(), limit)
	if err != nil {
		response.InternalError(c, err.Error())
		return
	}
	response.Success(c, rank)
}

// GetActiveCampaigns 获取进行中的活动
// @Summary 获取进行中的活动
// @Tags 管理-仪表盘
// @Produce json
// @Security Bearer
// @Param limit query int false "数量" default(10)
// @Success 200 {object} response.Response{data=[]adminService.CampaignStat}
// @Router /api/v1/admin/dashboard/operation/active-campaigns [get]
func (h *DashboardHandler) GetActiveCampaigns(c *gin.Context) {
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "10"))

	campaigns, err := h.operationService.GetActiveCampaigns(c.Request.Context(), limit)
	if err != nil {
		response.InternalError(c, err.Error())
		return
	}
	response.Success(c, campaigns)
}

// GetUserFeedbackStats 获取用户反馈统计
// @Summary 获取用户反馈统计
// @Tags 管理-仪表盘
// @Produce json
// @Security Bearer
// @Success 200 {object} response.Response{data=adminService.UserFeedbackStat}
// @Router /api/v1/admin/dashboard/operation/feedback-stats [get]
func (h *DashboardHandler) GetUserFeedbackStats(c *gin.Context) {
	stats, err := h.operationService.GetUserFeedbackStats(c.Request.Context())
	if err != nil {
		response.InternalError(c, err.Error())
		return
	}
	response.Success(c, stats)
}

// ==================== 财务仪表盘 ====================

// GetFinanceDashboardOverview 获取财务仪表盘概览
// @Summary 获取财务仪表盘概览
// @Tags 管理-仪表盘
// @Produce json
// @Security Bearer
// @Success 200 {object} response.Response{data=financeService.FinanceOverviewData}
// @Router /api/v1/admin/dashboard/finance/overview [get]
func (h *DashboardHandler) GetFinanceDashboardOverview(c *gin.Context) {
	overview, err := h.financeDashboard.GetFinanceOverviewData(c.Request.Context())
	if err != nil {
		response.InternalError(c, err.Error())
		return
	}
	response.Success(c, overview)
}

// GetRevenueTrend 获取收入趋势
// @Summary 获取收入趋势
// @Tags 管理-仪表盘
// @Produce json
// @Security Bearer
// @Param days query int false "天数" default(7)
// @Success 200 {object} response.Response{data=[]financeService.RevenueTrend}
// @Router /api/v1/admin/dashboard/finance/revenue-trend [get]
func (h *DashboardHandler) GetRevenueTrend(c *gin.Context) {
	days, _ := strconv.Atoi(c.DefaultQuery("days", "7"))

	trends, err := h.financeDashboard.GetRevenueTrend(c.Request.Context(), days)
	if err != nil {
		response.InternalError(c, err.Error())
		return
	}
	response.Success(c, trends)
}

// GetPaymentChannelSummary 获取支付渠道汇总
// @Summary 获取支付渠道汇总
// @Tags 管理-仪表盘
// @Produce json
// @Security Bearer
// @Param start_date query string false "开始日期 YYYY-MM-DD"
// @Param end_date query string false "结束日期 YYYY-MM-DD"
// @Success 200 {object} response.Response{data=[]financeService.PaymentChannelSummary}
// @Router /api/v1/admin/dashboard/finance/payment-channels [get]
func (h *DashboardHandler) GetPaymentChannelSummary(c *gin.Context) {
	var startDate, endDate *time.Time
	if s := c.Query("start_date"); s != "" {
		t, _ := time.Parse("2006-01-02", s)
		startDate = &t
	}
	if s := c.Query("end_date"); s != "" {
		t, _ := time.Parse("2006-01-02", s)
		endOfDay := t.Add(24*time.Hour - time.Second)
		endDate = &endOfDay
	}

	summary, err := h.financeDashboard.GetPaymentChannelSummary(c.Request.Context(), startDate, endDate)
	if err != nil {
		response.InternalError(c, err.Error())
		return
	}
	response.Success(c, summary)
}

// GetSettlementStats 获取结算统计
// @Summary 获取结算统计
// @Tags 管理-仪表盘
// @Produce json
// @Security Bearer
// @Success 200 {object} response.Response{data=[]financeService.SettlementStat}
// @Router /api/v1/admin/dashboard/finance/settlement-stats [get]
func (h *DashboardHandler) GetSettlementStats(c *gin.Context) {
	stats, err := h.financeDashboard.GetSettlementStats(c.Request.Context())
	if err != nil {
		response.InternalError(c, err.Error())
		return
	}
	response.Success(c, stats)
}

// GetPendingWithdrawals 获取待处理提现
// @Summary 获取待处理提现
// @Tags 管理-仪表盘
// @Produce json
// @Security Bearer
// @Param limit query int false "数量" default(10)
// @Success 200 {object} response.Response{data=[]financeService.PendingWithdrawalItem}
// @Router /api/v1/admin/dashboard/finance/pending-withdrawals [get]
func (h *DashboardHandler) GetPendingWithdrawals(c *gin.Context) {
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "10"))

	withdrawals, err := h.financeDashboard.GetPendingWithdrawals(c.Request.Context(), limit)
	if err != nil {
		response.InternalError(c, err.Error())
		return
	}
	response.Success(c, withdrawals)
}

// GetRefundStats 获取退款统计
// @Summary 获取退款统计
// @Tags 管理-仪表盘
// @Produce json
// @Security Bearer
// @Param start_date query string false "开始日期 YYYY-MM-DD"
// @Param end_date query string false "结束日期 YYYY-MM-DD"
// @Success 200 {object} response.Response{data=[]financeService.RefundStat}
// @Router /api/v1/admin/dashboard/finance/refund-stats [get]
func (h *DashboardHandler) GetRefundStats(c *gin.Context) {
	var startDate, endDate *time.Time
	if s := c.Query("start_date"); s != "" {
		t, _ := time.Parse("2006-01-02", s)
		startDate = &t
	}
	if s := c.Query("end_date"); s != "" {
		t, _ := time.Parse("2006-01-02", s)
		endOfDay := t.Add(24*time.Hour - time.Second)
		endDate = &endOfDay
	}

	stats, err := h.financeDashboard.GetRefundStats(c.Request.Context(), startDate, endDate)
	if err != nil {
		response.InternalError(c, err.Error())
		return
	}
	response.Success(c, stats)
}

// ==================== 分销商仪表盘 ====================

// GetDistributorOverview 获取分销商概览
// @Summary 获取分销商概览
// @Tags 管理-仪表盘
// @Produce json
// @Security Bearer
// @Param distributor_id path int true "分销商ID"
// @Success 200 {object} response.Response{data=distributionService.DistributorOverview}
// @Router /api/v1/admin/dashboard/distributor/{distributor_id}/overview [get]
func (h *DashboardHandler) GetDistributorOverview(c *gin.Context) {
	distributorID, err := strconv.ParseInt(c.Param("distributor_id"), 10, 64)
	if err != nil {
		response.BadRequest(c, "无效的分销商ID")
		return
	}

	overview, err := h.distributionDashboard.GetDistributorOverview(c.Request.Context(), distributorID)
	if err != nil {
		response.InternalError(c, err.Error())
		return
	}
	response.Success(c, overview)
}

// GetDistributorCommissionTrend 获取分销商佣金趋势
// @Summary 获取分销商佣金趋势
// @Tags 管理-仪表盘
// @Produce json
// @Security Bearer
// @Param distributor_id path int true "分销商ID"
// @Param days query int false "天数" default(7)
// @Success 200 {object} response.Response{data=[]distributionService.CommissionTrend}
// @Router /api/v1/admin/dashboard/distributor/{distributor_id}/commission-trend [get]
func (h *DashboardHandler) GetDistributorCommissionTrend(c *gin.Context) {
	distributorID, err := strconv.ParseInt(c.Param("distributor_id"), 10, 64)
	if err != nil {
		response.BadRequest(c, "无效的分销商ID")
		return
	}

	days, _ := strconv.Atoi(c.DefaultQuery("days", "7"))

	trends, err := h.distributionDashboard.GetCommissionTrend(c.Request.Context(), distributorID, days)
	if err != nil {
		response.InternalError(c, err.Error())
		return
	}
	response.Success(c, trends)
}

// GetDistributorTeamRank 获取分销商团队排行
// @Summary 获取分销商团队排行
// @Tags 管理-仪表盘
// @Produce json
// @Security Bearer
// @Param distributor_id path int true "分销商ID"
// @Param limit query int false "数量" default(10)
// @Param start_date query string false "开始日期 YYYY-MM-DD"
// @Param end_date query string false "结束日期 YYYY-MM-DD"
// @Success 200 {object} response.Response{data=[]distributionService.TeamMemberRank}
// @Router /api/v1/admin/dashboard/distributor/{distributor_id}/team-rank [get]
func (h *DashboardHandler) GetDistributorTeamRank(c *gin.Context) {
	distributorID, err := strconv.ParseInt(c.Param("distributor_id"), 10, 64)
	if err != nil {
		response.BadRequest(c, "无效的分销商ID")
		return
	}

	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "10"))

	var startDate, endDate *time.Time
	if s := c.Query("start_date"); s != "" {
		t, _ := time.Parse("2006-01-02", s)
		startDate = &t
	}
	if s := c.Query("end_date"); s != "" {
		t, _ := time.Parse("2006-01-02", s)
		endOfDay := t.Add(24*time.Hour - time.Second)
		endDate = &endOfDay
	}

	rank, err := h.distributionDashboard.GetTeamRank(c.Request.Context(), distributorID, limit, startDate, endDate)
	if err != nil {
		response.InternalError(c, err.Error())
		return
	}
	response.Success(c, rank)
}

// GetDistributorRecentCommissions 获取分销商最近佣金记录
// @Summary 获取分销商最近佣金记录
// @Tags 管理-仪表盘
// @Produce json
// @Security Bearer
// @Param distributor_id path int true "分销商ID"
// @Param limit query int false "数量" default(10)
// @Success 200 {object} response.Response{data=[]distributionService.CommissionRecord}
// @Router /api/v1/admin/dashboard/distributor/{distributor_id}/recent-commissions [get]
func (h *DashboardHandler) GetDistributorRecentCommissions(c *gin.Context) {
	distributorID, err := strconv.ParseInt(c.Param("distributor_id"), 10, 64)
	if err != nil {
		response.BadRequest(c, "无效的分销商ID")
		return
	}

	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "10"))

	records, err := h.distributionDashboard.GetRecentCommissions(c.Request.Context(), distributorID, limit)
	if err != nil {
		response.InternalError(c, err.Error())
		return
	}
	response.Success(c, records)
}

// GetDistributorCommissionTypeSummary 获取分销商佣金类型汇总
// @Summary 获取分销商佣金类型汇总
// @Tags 管理-仪表盘
// @Produce json
// @Security Bearer
// @Param distributor_id path int true "分销商ID"
// @Param start_date query string false "开始日期 YYYY-MM-DD"
// @Param end_date query string false "结束日期 YYYY-MM-DD"
// @Success 200 {object} response.Response{data=[]distributionService.CommissionTypeSummary}
// @Router /api/v1/admin/dashboard/distributor/{distributor_id}/commission-type [get]
func (h *DashboardHandler) GetDistributorCommissionTypeSummary(c *gin.Context) {
	distributorID, err := strconv.ParseInt(c.Param("distributor_id"), 10, 64)
	if err != nil {
		response.BadRequest(c, "无效的分销商ID")
		return
	}

	var startDate, endDate *time.Time
	if s := c.Query("start_date"); s != "" {
		t, _ := time.Parse("2006-01-02", s)
		startDate = &t
	}
	if s := c.Query("end_date"); s != "" {
		t, _ := time.Parse("2006-01-02", s)
		endOfDay := t.Add(24*time.Hour - time.Second)
		endDate = &endOfDay
	}

	summary, err := h.distributionDashboard.GetCommissionTypeSummary(c.Request.Context(), distributorID, startDate, endDate)
	if err != nil {
		response.InternalError(c, err.Error())
		return
	}
	response.Success(c, summary)
}
