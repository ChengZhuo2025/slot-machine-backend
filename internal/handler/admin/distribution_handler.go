// Package admin 管理端 HTTP Handler
package admin

import (
	"strconv"

	"github.com/gin-gonic/gin"

	"github.com/dumeirei/smart-locker-backend/internal/common/handler"
	"github.com/dumeirei/smart-locker-backend/internal/common/response"
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
// @Param page_size query int false "每页数量" default(20)
// @Param status query int false "状态: 0待审核 1已通过 2已拒绝"
// @Param level query int false "层级: 1直推 2间推"
// @Success 200 {object} response.Response{data=response.PageData}
// @Router /api/v1/admin/distribution/distributors [get]
func (h *DistributionHandler) ListDistributors(c *gin.Context) {
	if _, ok := handler.RequireAdminID(c); !ok {
		return
	}

	p := handler.BindPaginationWithDefaults(c, 1, 20)

	filter := &adminService.DistributorListFilter{}
	if statusStr := c.Query("status"); statusStr != "" {
		status, _ := strconv.Atoi(statusStr)
		filter.Status = &status
	}
	if levelStr := c.Query("level"); levelStr != "" {
		level, _ := strconv.Atoi(levelStr)
		filter.Level = &level
	}

	distributors, total, err := h.distributionService.ListDistributors(c.Request.Context(), p.GetOffset(), p.GetLimit(), filter)
	handler.MustSucceedPage(c, err, distributors, total, p.Page, p.PageSize)
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
	if _, ok := handler.RequireAdminID(c); !ok {
		return
	}

	id, ok := handler.ParseID(c, "分销商")
	if !ok {
		return
	}

	distributor, err := h.distributionService.GetDistributor(c.Request.Context(), id)
	handler.MustSucceed(c, err, distributor)
}

// GetPendingDistributors 获取待审核分销商列表
// @Summary 获取待审核分销商列表
// @Tags 管理-分销
// @Produce json
// @Security Bearer
// @Param page query int false "页码" default(1)
// @Param page_size query int false "每页数量" default(20)
// @Success 200 {object} response.Response{data=response.PageData}
// @Router /api/v1/admin/distribution/distributors/pending [get]
func (h *DistributionHandler) GetPendingDistributors(c *gin.Context) {
	if _, ok := handler.RequireAdminID(c); !ok {
		return
	}

	p := handler.BindPaginationWithDefaults(c, 1, 20)

	distributors, total, err := h.distributionService.GetPendingDistributors(c.Request.Context(), p.GetOffset(), p.GetLimit())
	handler.MustSucceedPage(c, err, distributors, total, p.Page, p.PageSize)
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
	operatorID, ok := handler.RequireAdminID(c)
	if !ok {
		return
	}

	id, ok := handler.ParseID(c, "分销商")
	if !ok {
		return
	}

	var req ApproveRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "参数错误")
		return
	}

	var err error
	if req.Approved {
		err = h.distributionService.ApproveDistributor(c.Request.Context(), id, operatorID)
	} else {
		err = h.distributionService.RejectDistributor(c.Request.Context(), id, operatorID, req.Reason)
	}

	handler.MustSucceed(c, err, nil)
}

// ListCommissions 获取佣金记录列表
// @Summary 获取佣金记录列表
// @Tags 管理-分销
// @Produce json
// @Security Bearer
// @Param page query int false "页码" default(1)
// @Param page_size query int false "每页数量" default(20)
// @Param distributor_id query int false "分销商ID"
// @Param status query int false "状态: 0待结算 1已结算 2已取消"
// @Param type query string false "类型: direct/indirect"
// @Success 200 {object} response.Response{data=response.PageData}
// @Router /api/v1/admin/distribution/commissions [get]
func (h *DistributionHandler) ListCommissions(c *gin.Context) {
	if _, ok := handler.RequireAdminID(c); !ok {
		return
	}

	p := handler.BindPaginationWithDefaults(c, 1, 20)

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

	commissions, total, err := h.distributionService.ListCommissions(c.Request.Context(), p.GetOffset(), p.GetLimit(), filter)
	handler.MustSucceedPage(c, err, commissions, total, p.Page, p.PageSize)
}

// ListWithdrawals 获取提现记录列表
// @Summary 获取提现记录列表
// @Tags 管理-分销
// @Produce json
// @Security Bearer
// @Param page query int false "页码" default(1)
// @Param page_size query int false "每页数量" default(20)
// @Param user_id query int false "用户ID"
// @Param status query string false "状态: pending/approved/processing/success/rejected"
// @Success 200 {object} response.Response{data=response.PageData}
// @Router /api/v1/admin/distribution/withdrawals [get]
func (h *DistributionHandler) ListWithdrawals(c *gin.Context) {
	if _, ok := handler.RequireAdminID(c); !ok {
		return
	}

	p := handler.BindPaginationWithDefaults(c, 1, 20)

	filter := &adminService.WithdrawalListFilter{}
	if userIDStr := c.Query("user_id"); userIDStr != "" {
		userID, _ := strconv.ParseInt(userIDStr, 10, 64)
		filter.UserID = &userID
	}
	if status := c.Query("status"); status != "" {
		filter.Status = &status
	}

	withdrawals, total, err := h.distributionService.ListWithdrawals(c.Request.Context(), p.GetOffset(), p.GetLimit(), filter)
	handler.MustSucceedPage(c, err, withdrawals, total, p.Page, p.PageSize)
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
	if _, ok := handler.RequireAdminID(c); !ok {
		return
	}

	id, ok := handler.ParseID(c, "提现")
	if !ok {
		return
	}

	withdrawal, err := h.distributionService.GetWithdrawal(c.Request.Context(), id)
	handler.MustSucceed(c, err, withdrawal)
}

// GetPendingWithdrawals 获取待审核提现列表
// @Summary 获取待审核提现列表
// @Tags 管理-分销
// @Produce json
// @Security Bearer
// @Param page query int false "页码" default(1)
// @Param page_size query int false "每页数量" default(20)
// @Success 200 {object} response.Response{data=response.PageData}
// @Router /api/v1/admin/distribution/withdrawals/pending [get]
func (h *DistributionHandler) GetPendingWithdrawals(c *gin.Context) {
	if _, ok := handler.RequireAdminID(c); !ok {
		return
	}

	p := handler.BindPaginationWithDefaults(c, 1, 20)

	withdrawals, total, err := h.distributionService.GetPendingWithdrawals(c.Request.Context(), p.GetOffset(), p.GetLimit())
	handler.MustSucceedPage(c, err, withdrawals, total, p.Page, p.PageSize)
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
	operatorID, ok := handler.RequireAdminID(c)
	if !ok {
		return
	}

	id, ok := handler.ParseID(c, "提现")
	if !ok {
		return
	}

	var req WithdrawalApproveRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "参数错误")
		return
	}

	var err error
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

	handler.MustSucceed(c, err, nil)
}

// GetStats 获取分销统计
// @Summary 获取分销统计
// @Tags 管理-分销
// @Produce json
// @Security Bearer
// @Success 200 {object} response.Response{data=adminService.DistributionStats}
// @Router /api/v1/admin/distribution/stats [get]
func (h *DistributionHandler) GetStats(c *gin.Context) {
	if _, ok := handler.RequireAdminID(c); !ok {
		return
	}

	stats, err := h.distributionService.GetStats(c.Request.Context())
	handler.MustSucceed(c, err, stats)
}
