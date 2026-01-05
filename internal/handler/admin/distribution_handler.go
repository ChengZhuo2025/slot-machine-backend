// Package admin 管理端 HTTP Handler
package admin

import (
	"strconv"

	"github.com/gin-gonic/gin"

	"github.com/dumeirei/smart-locker-backend/internal/common/response"
	"github.com/dumeirei/smart-locker-backend/internal/middleware"
	adminService "github.com/dumeirei/smart-locker-backend/internal/service/admin"
)

// DistributionHandler 分销管理处理器
type DistributionHandler struct {
	distributionService *adminService.DistributionAdminService
}

// NewDistributionHandler 创建分销管理处理器
func NewDistributionHandler(distributionSvc *adminService.DistributionAdminService) *DistributionHandler {
	return &DistributionHandler{
		distributionService: distributionSvc,
	}
}

// ListDistributors 获取分销商列表
// @Summary 获取分销商列表
// @Tags 管理-分销
// @Produce json
// @Security Bearer
// @Param page query int false "页码" default(1)
// @Param page_size query int false "每页数量" default(10)
// @Param status query int false "状态: 0待审核 1已通过 2已拒绝"
// @Param level query int false "层级: 1直推 2间推"
// @Success 200 {object} response.Response{data=response.PageData}
// @Router /api/v1/admin/distribution/distributors [get]
func (h *DistributionHandler) ListDistributors(c *gin.Context) {
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "10"))

	if page < 1 {
		page = 1
	}
	if pageSize < 1 || pageSize > 100 {
		pageSize = 10
	}
	offset := (page - 1) * pageSize

	filter := &adminService.DistributorListFilter{}
	if statusStr := c.Query("status"); statusStr != "" {
		status, _ := strconv.Atoi(statusStr)
		filter.Status = &status
	}
	if levelStr := c.Query("level"); levelStr != "" {
		level, _ := strconv.Atoi(levelStr)
		filter.Level = &level
	}

	distributors, total, err := h.distributionService.ListDistributors(c.Request.Context(), offset, pageSize, filter)
	if err != nil {
		response.InternalError(c, err.Error())
		return
	}

	response.SuccessPage(c, distributors, total, page, pageSize)
}

// GetDistributor 获取分销商详情
// @Summary 获取分销商详情
// @Tags 管理-分销
// @Produce json
// @Security Bearer
// @Param id path int true "分销商ID"
// @Success 200 {object} response.Response{data=models.Distributor}
// @Router /api/v1/admin/distribution/distributors/{id} [get]
func (h *DistributionHandler) GetDistributor(c *gin.Context) {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		response.BadRequest(c, "无效的ID")
		return
	}

	distributor, err := h.distributionService.GetDistributor(c.Request.Context(), id)
	if err != nil {
		response.InternalError(c, err.Error())
		return
	}

	response.Success(c, distributor)
}

// GetPendingDistributors 获取待审核分销商列表
// @Summary 获取待审核分销商列表
// @Tags 管理-分销
// @Produce json
// @Security Bearer
// @Param page query int false "页码" default(1)
// @Param page_size query int false "每页数量" default(10)
// @Success 200 {object} response.Response{data=response.PageData}
// @Router /api/v1/admin/distribution/distributors/pending [get]
func (h *DistributionHandler) GetPendingDistributors(c *gin.Context) {
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "10"))

	if page < 1 {
		page = 1
	}
	if pageSize < 1 || pageSize > 100 {
		pageSize = 10
	}
	offset := (page - 1) * pageSize

	distributors, total, err := h.distributionService.GetPendingDistributors(c.Request.Context(), offset, pageSize)
	if err != nil {
		response.InternalError(c, err.Error())
		return
	}

	response.SuccessPage(c, distributors, total, page, pageSize)
}

// ApproveRequest 审核请求
type ApproveRequest struct {
	Approved bool   `json:"approved"` // 是否通过
	Reason   string `json:"reason"`   // 拒绝原因
}

// ApproveDistributor 审核分销商
// @Summary 审核分销商
// @Tags 管理-分销
// @Accept json
// @Produce json
// @Security Bearer
// @Param id path int true "分销商ID"
// @Param request body ApproveRequest true "请求参数"
// @Success 200 {object} response.Response
// @Router /api/v1/admin/distribution/distributors/{id}/approve [post]
func (h *DistributionHandler) ApproveDistributor(c *gin.Context) {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		response.BadRequest(c, "无效的ID")
		return
	}

	var req ApproveRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "参数错误")
		return
	}

	operatorID := middleware.GetAdminID(c)

	if req.Approved {
		err = h.distributionService.ApproveDistributor(c.Request.Context(), id, operatorID)
	} else {
		err = h.distributionService.RejectDistributor(c.Request.Context(), id, operatorID, req.Reason)
	}

	if err != nil {
		response.InternalError(c, err.Error())
		return
	}

	response.Success(c, nil)
}

// ListCommissions 获取佣金记录列表
// @Summary 获取佣金记录列表
// @Tags 管理-分销
// @Produce json
// @Security Bearer
// @Param page query int false "页码" default(1)
// @Param page_size query int false "每页数量" default(10)
// @Param distributor_id query int false "分销商ID"
// @Param status query int false "状态: 0待结算 1已结算 2已取消"
// @Param type query string false "类型: direct/indirect"
// @Success 200 {object} response.Response{data=response.PageData}
// @Router /api/v1/admin/distribution/commissions [get]
func (h *DistributionHandler) ListCommissions(c *gin.Context) {
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "10"))

	if page < 1 {
		page = 1
	}
	if pageSize < 1 || pageSize > 100 {
		pageSize = 10
	}
	offset := (page - 1) * pageSize

	filter := &adminService.CommissionListFilter{}
	if distributorIDStr := c.Query("distributor_id"); distributorIDStr != "" {
		distributorID, _ := strconv.ParseInt(distributorIDStr, 10, 64)
		filter.DistributorID = &distributorID
	}
	if statusStr := c.Query("status"); statusStr != "" {
		status, _ := strconv.Atoi(statusStr)
		filter.Status = &status
	}
	if typeStr := c.Query("type"); typeStr != "" {
		filter.Type = &typeStr
	}

	commissions, total, err := h.distributionService.ListCommissions(c.Request.Context(), offset, pageSize, filter)
	if err != nil {
		response.InternalError(c, err.Error())
		return
	}

	response.SuccessPage(c, commissions, total, page, pageSize)
}

// ListWithdrawals 获取提现记录列表
// @Summary 获取提现记录列表
// @Tags 管理-分销
// @Produce json
// @Security Bearer
// @Param page query int false "页码" default(1)
// @Param page_size query int false "每页数量" default(10)
// @Param user_id query int false "用户ID"
// @Param status query string false "状态: pending/approved/processing/success/rejected"
// @Success 200 {object} response.Response{data=response.PageData}
// @Router /api/v1/admin/distribution/withdrawals [get]
func (h *DistributionHandler) ListWithdrawals(c *gin.Context) {
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "10"))

	if page < 1 {
		page = 1
	}
	if pageSize < 1 || pageSize > 100 {
		pageSize = 10
	}
	offset := (page - 1) * pageSize

	filter := &adminService.WithdrawalListFilter{}
	if userIDStr := c.Query("user_id"); userIDStr != "" {
		userID, _ := strconv.ParseInt(userIDStr, 10, 64)
		filter.UserID = &userID
	}
	if status := c.Query("status"); status != "" {
		filter.Status = &status
	}

	withdrawals, total, err := h.distributionService.ListWithdrawals(c.Request.Context(), offset, pageSize, filter)
	if err != nil {
		response.InternalError(c, err.Error())
		return
	}

	response.SuccessPage(c, withdrawals, total, page, pageSize)
}

// GetWithdrawal 获取提现详情
// @Summary 获取提现详情
// @Tags 管理-分销
// @Produce json
// @Security Bearer
// @Param id path int true "提现ID"
// @Success 200 {object} response.Response{data=models.Withdrawal}
// @Router /api/v1/admin/distribution/withdrawals/{id} [get]
func (h *DistributionHandler) GetWithdrawal(c *gin.Context) {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		response.BadRequest(c, "无效的ID")
		return
	}

	withdrawal, err := h.distributionService.GetWithdrawal(c.Request.Context(), id)
	if err != nil {
		response.InternalError(c, err.Error())
		return
	}

	response.Success(c, withdrawal)
}

// GetPendingWithdrawals 获取待审核提现列表
// @Summary 获取待审核提现列表
// @Tags 管理-分销
// @Produce json
// @Security Bearer
// @Param page query int false "页码" default(1)
// @Param page_size query int false "每页数量" default(10)
// @Success 200 {object} response.Response{data=response.PageData}
// @Router /api/v1/admin/distribution/withdrawals/pending [get]
func (h *DistributionHandler) GetPendingWithdrawals(c *gin.Context) {
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "10"))

	if page < 1 {
		page = 1
	}
	if pageSize < 1 || pageSize > 100 {
		pageSize = 10
	}
	offset := (page - 1) * pageSize

	withdrawals, total, err := h.distributionService.GetPendingWithdrawals(c.Request.Context(), offset, pageSize)
	if err != nil {
		response.InternalError(c, err.Error())
		return
	}

	response.SuccessPage(c, withdrawals, total, page, pageSize)
}

// WithdrawalApproveRequest 提现审核请求
type WithdrawalApproveRequest struct {
	Action string `json:"action" binding:"required"` // approve/reject/process/complete
	Reason string `json:"reason"`                    // 拒绝原因
}

// HandleWithdrawal 处理提现
// @Summary 处理提现
// @Tags 管理-分销
// @Accept json
// @Produce json
// @Security Bearer
// @Param id path int true "提现ID"
// @Param request body WithdrawalApproveRequest true "请求参数"
// @Success 200 {object} response.Response
// @Router /api/v1/admin/distribution/withdrawals/{id}/handle [post]
func (h *DistributionHandler) HandleWithdrawal(c *gin.Context) {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		response.BadRequest(c, "无效的ID")
		return
	}

	var req WithdrawalApproveRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "参数错误")
		return
	}

	operatorID := middleware.GetAdminID(c)

	switch req.Action {
	case "approve":
		err = h.distributionService.ApproveWithdrawal(c.Request.Context(), id, operatorID)
	case "reject":
		if req.Reason == "" {
			response.BadRequest(c, "请填写拒绝原因")
			return
		}
		err = h.distributionService.RejectWithdrawal(c.Request.Context(), id, operatorID, req.Reason)
	case "process":
		err = h.distributionService.ProcessWithdrawal(c.Request.Context(), id)
	case "complete":
		err = h.distributionService.CompleteWithdrawal(c.Request.Context(), id)
	default:
		response.BadRequest(c, "无效的操作")
		return
	}

	if err != nil {
		response.InternalError(c, err.Error())
		return
	}

	response.Success(c, nil)
}

// GetStats 获取分销统计
// @Summary 获取分销统计
// @Tags 管理-分销
// @Produce json
// @Security Bearer
// @Success 200 {object} response.Response{data=adminService.DistributionStats}
// @Router /api/v1/admin/distribution/stats [get]
func (h *DistributionHandler) GetStats(c *gin.Context) {
	stats, err := h.distributionService.GetStats(c.Request.Context())
	if err != nil {
		response.InternalError(c, err.Error())
		return
	}

	response.Success(c, stats)
}
