// Package admin 管理端 HTTP Handler
package admin

import (
	"strconv"
	"time"

	"github.com/gin-gonic/gin"

	"github.com/dumeirei/smart-locker-backend/internal/common/handler"
	"github.com/dumeirei/smart-locker-backend/internal/common/response"
	financeService "github.com/dumeirei/smart-locker-backend/internal/service/finance"
)

// FinanceHandler 财务管理处理器
type FinanceHandler struct {
	settlementService *financeService.SettlementService
	statisticsService *financeService.StatisticsService
	withdrawalService *financeService.WithdrawalAuditService
	exportService     *financeService.ExportService
}

// NewFinanceHandler 创建财务管理处理器
func NewFinanceHandler(
	settlementSvc *financeService.SettlementService,
	statisticsSvc *financeService.StatisticsService,
	withdrawalSvc *financeService.WithdrawalAuditService,
	exportSvc *financeService.ExportService,
) *FinanceHandler {
	return &FinanceHandler{
		settlementService: settlementSvc,
		statisticsService: statisticsSvc,
		withdrawalService: withdrawalSvc,
		exportService:     exportSvc,
	}
}

// GetOverview 获取财务概览
// @Summary 获取财务概览
// @Tags 管理-财务
// @Produce json
// @Security Bearer
// @Success 200 {object} response.Response{data=models.FinanceOverview}
// @Router /api/v1/admin/finance/overview [get]
func (h *FinanceHandler) GetOverview(c *gin.Context) {
	if _, ok := handler.RequireAdminID(c); !ok {
		return
	}

	overview, err := h.statisticsService.GetFinanceOverview(c.Request.Context())
	handler.MustSucceed(c, err, overview)
}

// GetRevenueStatistics 获取收入统计
// @Summary 获取收入统计
// @Tags 管理-财务
// @Produce json
// @Security Bearer
// @Param start_date query string true "开始日期 YYYY-MM-DD"
// @Param end_date query string true "结束日期 YYYY-MM-DD"
// @Success 200 {object} response.Response{data=[]models.RevenueStatistics}
// @Router /api/v1/admin/finance/revenue/statistics [get]
func (h *FinanceHandler) GetRevenueStatistics(c *gin.Context) {
	if _, ok := handler.RequireAdminID(c); !ok {
		return
	}

	startDateStr := c.Query("start_date")
	endDateStr := c.Query("end_date")

	if startDateStr == "" || endDateStr == "" {
		response.BadRequest(c, "请指定开始和结束日期")
		return
	}

	startDate, err := time.Parse("2006-01-02", startDateStr)
	if err != nil {
		response.BadRequest(c, "无效的开始日期格式")
		return
	}
	endDate, err := time.Parse("2006-01-02", endDateStr)
	if err != nil {
		response.BadRequest(c, "无效的结束日期格式")
		return
	}
	endDate = endDate.Add(24*time.Hour - time.Second)

	stats, err := h.statisticsService.GetRevenueStatistics(c.Request.Context(), startDate, endDate)
	handler.MustSucceed(c, err, stats)
}

// GetDailyRevenueReport 获取每日收入报表
// @Summary 获取每日收入报表
// @Tags 管理-财务
// @Produce json
// @Security Bearer
// @Param start_date query string true "开始日期 YYYY-MM-DD"
// @Param end_date query string true "结束日期 YYYY-MM-DD"
// @Success 200 {object} response.Response{data=[]models.DailyRevenueReport}
// @Router /api/v1/admin/finance/revenue/daily [get]
func (h *FinanceHandler) GetDailyRevenueReport(c *gin.Context) {
	if _, ok := handler.RequireAdminID(c); !ok {
		return
	}

	startDateStr := c.Query("start_date")
	endDateStr := c.Query("end_date")

	if startDateStr == "" || endDateStr == "" {
		response.BadRequest(c, "请指定开始和结束日期")
		return
	}

	startDate, err := time.Parse("2006-01-02", startDateStr)
	if err != nil {
		response.BadRequest(c, "无效的开始日期格式")
		return
	}
	endDate, err := time.Parse("2006-01-02", endDateStr)
	if err != nil {
		response.BadRequest(c, "无效的结束日期格式")
		return
	}
	endDate = endDate.Add(24*time.Hour - time.Second)

	report, err := h.statisticsService.GetDailyRevenueReport(c.Request.Context(), startDate, endDate)
	handler.MustSucceed(c, err, report)
}

// GetOrderRevenueByType 按订单类型获取收入统计
// @Summary 按订单类型获取收入统计
// @Tags 管理-财务
// @Produce json
// @Security Bearer
// @Param start_date query string false "开始日期 YYYY-MM-DD"
// @Param end_date query string false "结束日期 YYYY-MM-DD"
// @Success 200 {object} response.Response{data=[]models.OrderRevenue}
// @Router /api/v1/admin/finance/revenue/by-type [get]
func (h *FinanceHandler) GetOrderRevenueByType(c *gin.Context) {
	if _, ok := handler.RequireAdminID(c); !ok {
		return
	}

	var startDate, endDate *time.Time

	if s := c.Query("start_date"); s != "" {
		t, err := time.Parse("2006-01-02", s)
		if err != nil {
			response.BadRequest(c, "无效的开始日期格式")
			return
		}
		startDate = &t
	}
	if s := c.Query("end_date"); s != "" {
		t, err := time.Parse("2006-01-02", s)
		if err != nil {
			response.BadRequest(c, "无效的结束日期格式")
			return
		}
		endOfDay := t.Add(24*time.Hour - time.Second)
		endDate = &endOfDay
	}

	result, err := h.statisticsService.GetOrderRevenueByType(c.Request.Context(), startDate, endDate)
	handler.MustSucceed(c, err, result)
}

// ListSettlements 获取结算列表
// @Summary 获取结算列表
// @Tags 管理-财务
// @Produce json
// @Security Bearer
// @Param page query int false "页码" default(1)
// @Param page_size query int false "每页数量" default(20)
// @Param type query string false "类型: merchant/distributor"
// @Param target_id query int false "目标ID"
// @Param status query string false "状态: pending/processing/completed/failed"
// @Param period_start query string false "周期开始日期 YYYY-MM-DD"
// @Param period_end query string false "周期结束日期 YYYY-MM-DD"
// @Success 200 {object} response.Response{data=response.PageData}
// @Router /api/v1/admin/finance/settlements [get]
func (h *FinanceHandler) ListSettlements(c *gin.Context) {
	if _, ok := handler.RequireAdminID(c); !ok {
		return
	}

	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "20"))

	if page < 1 {
		page = 1
	}
	if pageSize < 1 || pageSize > 100 {
		pageSize = 20
	}

	req := &financeService.SettlementListRequest{
		Type:        c.Query("type"),
		Status:      c.Query("status"),
		PeriodStart: c.Query("period_start"),
		PeriodEnd:   c.Query("period_end"),
		Page:        page,
		PageSize:    pageSize,
	}

	if targetIDStr := c.Query("target_id"); targetIDStr != "" {
		targetID, _ := strconv.ParseInt(targetIDStr, 10, 64)
		req.TargetID = &targetID
	}

	settlements, total, err := h.settlementService.ListSettlements(c.Request.Context(), req)
	handler.MustSucceedPage(c, err, settlements, total, page, pageSize)
}

// GetSettlement 获取结算详情
// @Summary 获取结算详情
// @Tags 管理-财务
// @Produce json
// @Security Bearer
// @Param id path int true "结算ID"
// @Success 200 {object} response.Response{data=models.SettlementDetail}
// @Router /api/v1/admin/finance/settlements/{id} [get]
func (h *FinanceHandler) GetSettlement(c *gin.Context) {
	if _, ok := handler.RequireAdminID(c); !ok {
		return
	}

	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		response.BadRequest(c, "无效的ID")
		return
	}

	detail, err := h.settlementService.GetSettlementDetail(c.Request.Context(), id)
	handler.MustSucceed(c, err, detail)
}

// CreateSettlementRequest 创建结算请求
type CreateSettlementRequest struct {
	Type        string `json:"type" binding:"required,oneof=merchant distributor"`
	TargetID    int64  `json:"target_id" binding:"required"`
	PeriodStart string `json:"period_start" binding:"required"`
	PeriodEnd   string `json:"period_end" binding:"required"`
}

// CreateSettlement 创建结算记录
// @Summary 创建结算记录
// @Tags 管理-财务
// @Accept json
// @Produce json
// @Security Bearer
// @Param request body CreateSettlementRequest true "请求参数"
// @Success 200 {object} response.Response{data=models.Settlement}
// @Router /api/v1/admin/finance/settlements [post]
func (h *FinanceHandler) CreateSettlement(c *gin.Context) {
	operatorID, ok := handler.RequireAdminID(c)
	if !ok {
		return
	}

	var req CreateSettlementRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "参数错误: "+err.Error())
		return
	}

	periodStart, err := time.Parse("2006-01-02", req.PeriodStart)
	if err != nil {
		response.BadRequest(c, "无效的周期开始日期")
		return
	}
	periodEnd, err := time.Parse("2006-01-02", req.PeriodEnd)
	if err != nil {
		response.BadRequest(c, "无效的周期结束日期")
		return
	}

	serviceReq := &financeService.CreateSettlementRequest{
		Type:        req.Type,
		TargetID:    req.TargetID,
		PeriodStart: periodStart,
		PeriodEnd:   periodEnd,
	}

	settlement, err := h.settlementService.CreateSettlement(c.Request.Context(), serviceReq, operatorID)
	if err != nil {
		response.InternalError(c, err.Error())
		return
	}

	response.Success(c, settlement)
}

// ProcessSettlement 处理结算
// @Summary 处理结算
// @Tags 管理-财务
// @Produce json
// @Security Bearer
// @Param id path int true "结算ID"
// @Success 200 {object} response.Response
// @Router /api/v1/admin/finance/settlements/{id}/process [post]
func (h *FinanceHandler) ProcessSettlement(c *gin.Context) {
	operatorID, ok := handler.RequireAdminID(c)
	if !ok {
		return
	}

	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		response.BadRequest(c, "无效的ID")
		return
	}

	if err := h.settlementService.ProcessSettlement(c.Request.Context(), id, operatorID); err != nil {
		response.InternalError(c, err.Error())
		return
	}

	response.Success(c, nil)
}

// GenerateSettlementsRequest 生成结算请求
type GenerateSettlementsRequest struct {
	Type        string `json:"type" binding:"required,oneof=merchant distributor"`
	PeriodStart string `json:"period_start" binding:"required"`
	PeriodEnd   string `json:"period_end" binding:"required"`
}

// GenerateSettlements 批量生成结算记录
// @Summary 批量生成结算记录
// @Tags 管理-财务
// @Accept json
// @Produce json
// @Security Bearer
// @Param request body GenerateSettlementsRequest true "请求参数"
// @Success 200 {object} response.Response{data=[]models.Settlement}
// @Router /api/v1/admin/finance/settlements/generate [post]
func (h *FinanceHandler) GenerateSettlements(c *gin.Context) {
	operatorID, ok := handler.RequireAdminID(c)
	if !ok {
		return
	}

	var req GenerateSettlementsRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "参数错误: "+err.Error())
		return
	}

	periodStart, err := time.Parse("2006-01-02", req.PeriodStart)
	if err != nil {
		response.BadRequest(c, "无效的周期开始日期")
		return
	}
	periodEnd, err := time.Parse("2006-01-02", req.PeriodEnd)
	if err != nil {
		response.BadRequest(c, "无效的周期结束日期")
		return
	}

	var settlements interface{}
	if req.Type == "merchant" {
		settlements, err = h.settlementService.GenerateMerchantSettlements(c.Request.Context(), periodStart, periodEnd, operatorID)
	} else {
		settlements, err = h.settlementService.GenerateDistributorSettlements(c.Request.Context(), periodStart, periodEnd, operatorID)
	}

	handler.MustSucceed(c, err, settlements)
}

// GetSettlementSummary 获取结算汇总
// @Summary 获取结算汇总
// @Tags 管理-财务
// @Produce json
// @Security Bearer
// @Param type query string false "类型: merchant/distributor"
// @Param start_date query string false "开始日期 YYYY-MM-DD"
// @Param end_date query string false "结束日期 YYYY-MM-DD"
// @Success 200 {object} response.Response{data=models.SettlementSummary}
// @Router /api/v1/admin/finance/settlements/summary [get]
func (h *FinanceHandler) GetSettlementSummary(c *gin.Context) {
	if _, ok := handler.RequireAdminID(c); !ok {
		return
	}

	settlementType := c.Query("type")

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

	summary, err := h.statisticsService.GetSettlementSummary(c.Request.Context(), settlementType, startDate, endDate)
	handler.MustSucceed(c, err, summary)
}

// ListWithdrawals 获取提现列表
// @Summary 获取提现列表
// @Tags 管理-财务
// @Produce json
// @Security Bearer
// @Param page query int false "页码" default(1)
// @Param page_size query int false "每页数量" default(20)
// @Param user_id query int false "用户ID"
// @Param type query string false "类型: wallet/commission"
// @Param status query string false "状态: pending/approved/processing/success/rejected"
// @Param start_date query string false "开始日期 YYYY-MM-DD"
// @Param end_date query string false "结束日期 YYYY-MM-DD"
// @Success 200 {object} response.Response{data=response.PageData}
// @Router /api/v1/admin/finance/withdrawals [get]
func (h *FinanceHandler) ListWithdrawals(c *gin.Context) {
	if _, ok := handler.RequireAdminID(c); !ok {
		return
	}

	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "20"))

	if page < 1 {
		page = 1
	}
	if pageSize < 1 || pageSize > 100 {
		pageSize = 20
	}

	req := &financeService.WithdrawalListRequest{
		Type:      c.Query("type"),
		Status:    c.Query("status"),
		StartDate: c.Query("start_date"),
		EndDate:   c.Query("end_date"),
		Page:      page,
		PageSize:  pageSize,
	}

	if userIDStr := c.Query("user_id"); userIDStr != "" {
		userID, _ := strconv.ParseInt(userIDStr, 10, 64)
		req.UserID = &userID
	}

	withdrawals, total, err := h.withdrawalService.ListWithdrawals(c.Request.Context(), req)
	handler.MustSucceedPage(c, err, withdrawals, total, page, pageSize)
}

// GetWithdrawal 获取提现详情
// @Summary 获取提现详情
// @Tags 管理-财务
// @Produce json
// @Security Bearer
// @Param id path int true "提现ID"
// @Success 200 {object} response.Response{data=models.Withdrawal}
// @Router /api/v1/admin/finance/withdrawals/{id} [get]
func (h *FinanceHandler) GetWithdrawal(c *gin.Context) {
	if _, ok := handler.RequireAdminID(c); !ok {
		return
	}

	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		response.BadRequest(c, "无效的ID")
		return
	}

	withdrawal, err := h.withdrawalService.GetWithdrawal(c.Request.Context(), id)
	handler.MustSucceed(c, err, withdrawal)
}

// WithdrawalActionRequest 提现操作请求
type WithdrawalActionRequest struct {
	Action string `json:"action" binding:"required,oneof=approve reject process complete"`
	Reason string `json:"reason"`
}

// HandleWithdrawal 处理提现
// @Summary 处理提现
// @Tags 管理-财务
// @Accept json
// @Produce json
// @Security Bearer
// @Param id path int true "提现ID"
// @Param request body WithdrawalActionRequest true "请求参数"
// @Success 200 {object} response.Response
// @Router /api/v1/admin/finance/withdrawals/{id}/handle [post]
func (h *FinanceHandler) HandleWithdrawal(c *gin.Context) {
	operatorID, ok := handler.RequireAdminID(c)
	if !ok {
		return
	}

	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		response.BadRequest(c, "无效的ID")
		return
	}

	var req WithdrawalActionRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "参数错误")
		return
	}

	switch req.Action {
	case "approve":
		err = h.withdrawalService.ApproveWithdrawal(c.Request.Context(), id, operatorID)
	case "reject":
		if req.Reason == "" {
			response.BadRequest(c, "请填写拒绝原因")
			return
		}
		err = h.withdrawalService.RejectWithdrawal(c.Request.Context(), id, operatorID, req.Reason)
	case "process":
		err = h.withdrawalService.ProcessWithdrawal(c.Request.Context(), id, operatorID)
	case "complete":
		err = h.withdrawalService.CompleteWithdrawal(c.Request.Context(), id, operatorID)
	}

	handler.MustSucceed(c, err, nil)
}

// BatchWithdrawalRequest 批量提现操作请求
type BatchWithdrawalRequest struct {
	IDs    []int64 `json:"ids" binding:"required,min=1"`
	Action string  `json:"action" binding:"required,oneof=approve reject complete"`
	Reason string  `json:"reason"`
}

// BatchHandleWithdrawals 批量处理提现
// @Summary 批量处理提现
// @Tags 管理-财务
// @Accept json
// @Produce json
// @Security Bearer
// @Param request body BatchWithdrawalRequest true "请求参数"
// @Success 200 {object} response.Response
// @Router /api/v1/admin/finance/withdrawals/batch [post]
func (h *FinanceHandler) BatchHandleWithdrawals(c *gin.Context) {
	operatorID, ok := handler.RequireAdminID(c)
	if !ok {
		return
	}

	var req BatchWithdrawalRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "参数错误")
		return
	}

	var err error
	switch req.Action {
	case "approve":
		err = h.withdrawalService.BatchApprove(c.Request.Context(), req.IDs, operatorID)
	case "reject":
		if req.Reason == "" {
			response.BadRequest(c, "请填写拒绝原因")
			return
		}
		err = h.withdrawalService.BatchReject(c.Request.Context(), req.IDs, operatorID, req.Reason)
	case "complete":
		err = h.withdrawalService.BatchComplete(c.Request.Context(), req.IDs, operatorID)
	}

	handler.MustSucceed(c, err, nil)
}

// GetWithdrawalSummary 获取提现汇总
// @Summary 获取提现汇总
// @Tags 管理-财务
// @Produce json
// @Security Bearer
// @Param start_date query string false "开始日期 YYYY-MM-DD"
// @Param end_date query string false "结束日期 YYYY-MM-DD"
// @Success 200 {object} response.Response{data=models.WithdrawalSummary}
// @Router /api/v1/admin/finance/withdrawals/summary [get]
func (h *FinanceHandler) GetWithdrawalSummary(c *gin.Context) {
	if _, ok := handler.RequireAdminID(c); !ok {
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

	summary, err := h.statisticsService.GetWithdrawalSummary(c.Request.Context(), startDate, endDate)
	handler.MustSucceed(c, err, summary)
}

// ExportSettlements 导出结算记录
// @Summary 导出结算记录
// @Tags 管理-财务
// @Produce text/csv
// @Security Bearer
// @Param type query string false "类型: merchant/distributor"
// @Param target_id query int false "目标ID"
// @Param status query string false "状态"
// @Param period_start query string false "周期开始日期"
// @Param period_end query string false "周期结束日期"
// @Success 200 {file} file "CSV文件"
// @Router /api/v1/admin/finance/export/settlements [get]
func (h *FinanceHandler) ExportSettlements(c *gin.Context) {
	if _, ok := handler.RequireAdminID(c); !ok {
		return
	}

	req := &financeService.ExportSettlementsRequest{
		Type:   c.Query("type"),
		Status: c.Query("status"),
	}

	if targetIDStr := c.Query("target_id"); targetIDStr != "" {
		targetID, _ := strconv.ParseInt(targetIDStr, 10, 64)
		req.TargetID = &targetID
	}
	if s := c.Query("period_start"); s != "" {
		t, _ := time.Parse("2006-01-02", s)
		req.PeriodStart = &t
	}
	if s := c.Query("period_end"); s != "" {
		t, _ := time.Parse("2006-01-02", s)
		req.PeriodEnd = &t
	}

	data, filename, err := h.exportService.ExportSettlements(c.Request.Context(), req)
	if handler.HandleError(c, err) {
		return
	}

	c.Header("Content-Type", "text/csv; charset=utf-8")
	c.Header("Content-Disposition", "attachment; filename="+filename)
	c.Data(200, "text/csv", data)
}

// ExportWithdrawals 导出提现记录
// @Summary 导出提现记录
// @Tags 管理-财务
// @Produce text/csv
// @Security Bearer
// @Param user_id query int false "用户ID"
// @Param type query string false "类型"
// @Param status query string false "状态"
// @Param start_date query string false "开始日期"
// @Param end_date query string false "结束日期"
// @Success 200 {file} file "CSV文件"
// @Router /api/v1/admin/finance/export/withdrawals [get]
func (h *FinanceHandler) ExportWithdrawals(c *gin.Context) {
	if _, ok := handler.RequireAdminID(c); !ok {
		return
	}

	req := &financeService.ExportWithdrawalsRequest{
		Type:      c.Query("type"),
		Status:    c.Query("status"),
		StartDate: c.Query("start_date"),
		EndDate:   c.Query("end_date"),
	}

	if userIDStr := c.Query("user_id"); userIDStr != "" {
		userID, _ := strconv.ParseInt(userIDStr, 10, 64)
		req.UserID = &userID
	}

	data, filename, err := h.exportService.ExportWithdrawals(c.Request.Context(), req)
	if handler.HandleError(c, err) {
		return
	}

	c.Header("Content-Type", "text/csv; charset=utf-8")
	c.Header("Content-Disposition", "attachment; filename="+filename)
	c.Data(200, "text/csv", data)
}

// ExportDailyRevenue 导出每日收入报表
// @Summary 导出每日收入报表
// @Tags 管理-财务
// @Produce text/csv
// @Security Bearer
// @Param start_date query string true "开始日期 YYYY-MM-DD"
// @Param end_date query string true "结束日期 YYYY-MM-DD"
// @Success 200 {file} file "CSV文件"
// @Router /api/v1/admin/finance/export/daily-revenue [get]
func (h *FinanceHandler) ExportDailyRevenue(c *gin.Context) {
	if _, ok := handler.RequireAdminID(c); !ok {
		return
	}

	startDateStr := c.Query("start_date")
	endDateStr := c.Query("end_date")

	if startDateStr == "" || endDateStr == "" {
		response.BadRequest(c, "请指定开始和结束日期")
		return
	}

	startDate, err := time.Parse("2006-01-02", startDateStr)
	if err != nil {
		response.BadRequest(c, "无效的开始日期格式")
		return
	}
	endDate, err := time.Parse("2006-01-02", endDateStr)
	if err != nil {
		response.BadRequest(c, "无效的结束日期格式")
		return
	}

	data, filename, err := h.exportService.ExportDailyRevenue(c.Request.Context(), startDate, endDate)
	if handler.HandleError(c, err) {
		return
	}

	c.Header("Content-Type", "text/csv; charset=utf-8")
	c.Header("Content-Disposition", "attachment; filename="+filename)
	c.Data(200, "text/csv", data)
}

// ExportMerchantSettlement 导出商户结算报表
// @Summary 导出商户结算报表
// @Tags 管理-财务
// @Produce text/csv
// @Security Bearer
// @Param start_date query string false "开始日期 YYYY-MM-DD"
// @Param end_date query string false "结束日期 YYYY-MM-DD"
// @Success 200 {file} file "CSV文件"
// @Router /api/v1/admin/finance/export/merchant-settlement [get]
func (h *FinanceHandler) ExportMerchantSettlement(c *gin.Context) {
	if _, ok := handler.RequireAdminID(c); !ok {
		return
	}

	var startDate, endDate *time.Time
	if s := c.Query("start_date"); s != "" {
		t, _ := time.Parse("2006-01-02", s)
		startDate = &t
	}
	if s := c.Query("end_date"); s != "" {
		t, _ := time.Parse("2006-01-02", s)
		endDate = &t
	}

	data, filename, err := h.exportService.ExportMerchantSettlementReport(c.Request.Context(), startDate, endDate)
	if handler.HandleError(c, err) {
		return
	}

	c.Header("Content-Type", "text/csv; charset=utf-8")
	c.Header("Content-Disposition", "attachment; filename="+filename)
	c.Data(200, "text/csv", data)
}

// GetTransactionStatistics 获取交易统计
// @Summary 获取交易统计
// @Tags 管理-财务
// @Produce json
// @Security Bearer
// @Param start_date query string false "开始日期 YYYY-MM-DD"
// @Param end_date query string false "结束日期 YYYY-MM-DD"
// @Success 200 {object} response.Response{data=models.TransactionStatistics}
// @Router /api/v1/admin/finance/transactions/statistics [get]
func (h *FinanceHandler) GetTransactionStatistics(c *gin.Context) {
	if _, ok := handler.RequireAdminID(c); !ok {
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

	stats, err := h.statisticsService.GetTransactionStatistics(c.Request.Context(), startDate, endDate)
	handler.MustSucceed(c, err, stats)
}

// ExportTransactions 导出交易记录
// @Summary 导出交易记录
// @Tags 管理-财务
// @Produce text/csv
// @Security Bearer
// @Param user_id query int false "用户ID"
// @Param type query string false "类型"
// @Param start_date query string false "开始日期 YYYY-MM-DD"
// @Param end_date query string false "结束日期 YYYY-MM-DD"
// @Success 200 {file} file "CSV文件"
// @Router /api/v1/admin/finance/export/transactions [get]
func (h *FinanceHandler) ExportTransactions(c *gin.Context) {
	if _, ok := handler.RequireAdminID(c); !ok {
		return
	}

	req := &financeService.ExportTransactionsRequest{
		Type: c.Query("type"),
	}

	if userIDStr := c.Query("user_id"); userIDStr != "" {
		userID, _ := strconv.ParseInt(userIDStr, 10, 64)
		req.UserID = &userID
	}
	if s := c.Query("start_date"); s != "" {
		t, _ := time.Parse("2006-01-02", s)
		req.StartTime = &t
	}
	if s := c.Query("end_date"); s != "" {
		t, _ := time.Parse("2006-01-02", s)
		endOfDay := t.Add(24*time.Hour - time.Second)
		req.EndTime = &endOfDay
	}

	data, filename, err := h.exportService.ExportTransactions(c.Request.Context(), req)
	if handler.HandleError(c, err) {
		return
	}

	c.Header("Content-Type", "text/csv; charset=utf-8")
	c.Header("Content-Disposition", "attachment; filename="+filename)
	c.Data(200, "text/csv", data)
}

// GetMerchantSettlementReport 获取商户结算报表
// @Summary 获取商户结算报表
// @Tags 管理-财务
// @Produce json
// @Security Bearer
// @Param start_date query string false "开始日期 YYYY-MM-DD"
// @Param end_date query string false "结束日期 YYYY-MM-DD"
// @Success 200 {object} response.Response{data=[]models.MerchantSettlementReport}
// @Router /api/v1/admin/finance/reports/merchant-settlement [get]
func (h *FinanceHandler) GetMerchantSettlementReport(c *gin.Context) {
	if _, ok := handler.RequireAdminID(c); !ok {
		return
	}

	var startDate, endDate *time.Time
	if s := c.Query("start_date"); s != "" {
		t, _ := time.Parse("2006-01-02", s)
		startDate = &t
	}
	if s := c.Query("end_date"); s != "" {
		t, _ := time.Parse("2006-01-02", s)
		endDate = &t
	}

	report, err := h.statisticsService.GetMerchantSettlementReport(c.Request.Context(), startDate, endDate)
	handler.MustSucceed(c, err, report)
}
