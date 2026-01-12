// Package distribution 提供分销相关的 HTTP Handler
package distribution

import (
	"strconv"

	"github.com/gin-gonic/gin"

	"github.com/dumeirei/smart-locker-backend/internal/common/handler"
	"github.com/dumeirei/smart-locker-backend/internal/common/response"
	"github.com/dumeirei/smart-locker-backend/internal/service/distribution"
)

// Handler 分销处理器
type Handler struct {
	distributorService *distribution.DistributorService
	commissionService  *distribution.CommissionService
	inviteService      *distribution.InviteService
	withdrawService    *distribution.WithdrawService
}

// NewHandler 创建分销处理器
func NewHandler(
	distributorSvc *distribution.DistributorService,
	commissionSvc *distribution.CommissionService,
	inviteSvc *distribution.InviteService,
	withdrawSvc *distribution.WithdrawService,
) *Handler {
	return &Handler{
		distributorService: distributorSvc,
		commissionService:  commissionSvc,
		inviteService:      inviteSvc,
		withdrawService:    withdrawSvc,
	}
}

// ApplyRequest 申请成为分销商请求
type ApplyRequest struct {
	InviteCode string `json:"invite_code"` // 上级邀请码（可选）
}

// Apply 申请成为分销商
// @Summary 申请成为分销商
// @Tags 分销
// @Accept json
// @Produce json
// @Security Bearer
// @Param request body ApplyRequest true "请求参数"
// @Success 200 {object} response.Response{data=distribution.ApplyResponse}
// @Router /api/v1/distribution/apply [post]
func (h *Handler) Apply(c *gin.Context) {
	userID, ok := handler.RequireUserID(c)
	if !ok {
		return
	}

	var req ApplyRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "参数错误")
		return
	}

	applyReq := &distribution.ApplyRequest{
		UserID: userID,
	}
	if req.InviteCode != "" {
		applyReq.InviteCode = &req.InviteCode
	}

	result, err := h.distributorService.Apply(c.Request.Context(), applyReq)
	handler.MustSucceed(c, err, result)
}

// GetInfo 获取分销商信息
// @Summary 获取分销商信息
// @Tags 分销
// @Produce json
// @Security Bearer
// @Success 200 {object} response.Response{data=models.Distributor}
// @Router /api/v1/distribution/info [get]
func (h *Handler) GetInfo(c *gin.Context) {
	userID, ok := handler.RequireUserID(c)
	if !ok {
		return
	}

	distributor, err := h.distributorService.GetByUserID(c.Request.Context(), userID)
	handler.MustSucceed(c, err, distributor)
}

// GetDashboard 获取分销商仪表盘数据
// @Summary 获取分销商仪表盘数据
// @Tags 分销
// @Produce json
// @Security Bearer
// @Success 200 {object} response.Response{data=distribution.DashboardData}
// @Router /api/v1/distribution/dashboard [get]
func (h *Handler) GetDashboard(c *gin.Context) {
	userID, ok := handler.RequireUserID(c)
	if !ok {
		return
	}

	// 获取分销商信息
	distributor, err := h.distributorService.GetByUserID(c.Request.Context(), userID)
	if handler.HandleError(c, err) {
		return
	}

	dashboard, err := h.distributorService.GetDashboard(c.Request.Context(), distributor.ID)
	handler.MustSucceed(c, err, dashboard)
}

// GetTeamStats 获取团队统计
// @Summary 获取团队统计
// @Tags 分销
// @Produce json
// @Security Bearer
// @Success 200 {object} response.Response{data=distribution.TeamStats}
// @Router /api/v1/distribution/team/stats [get]
func (h *Handler) GetTeamStats(c *gin.Context) {
	userID, ok := handler.RequireUserID(c)
	if !ok {
		return
	}

	distributor, err := h.distributorService.GetByUserID(c.Request.Context(), userID)
	if handler.HandleError(c, err) {
		return
	}

	stats, err := h.distributorService.GetTeamStats(c.Request.Context(), distributor.ID)
	handler.MustSucceed(c, err, stats)
}

// GetTeamMembers 获取团队成员列表
// @Summary 获取团队成员列表
// @Tags 分销
// @Produce json
// @Security Bearer
// @Param page query int false "页码" default(1)
// @Param page_size query int false "每页数量" default(10)
// @Param type query string false "成员类型: direct/all" default(direct)
// @Success 200 {object} response.Response{data=response.PageData}
// @Router /api/v1/distribution/team/members [get]
func (h *Handler) GetTeamMembers(c *gin.Context) {
	userID, ok := handler.RequireUserID(c)
	if !ok {
		return
	}

	p := handler.BindPagination(c)
	memberType := c.DefaultQuery("type", "direct")

	distributor, err := h.distributorService.GetByUserID(c.Request.Context(), userID)
	if handler.HandleError(c, err) {
		return
	}

	members, total, err := h.distributorService.GetTeamMembers(c.Request.Context(), distributor.ID, p.GetOffset(), p.GetLimit(), memberType)
	handler.MustSucceedPage(c, err, members, total, p.Page, p.PageSize)
}

// GetInviteInfo 获取邀请信息
// @Summary 获取邀请信息
// @Tags 分销
// @Produce json
// @Security Bearer
// @Success 200 {object} response.Response{data=distribution.InviteInfo}
// @Router /api/v1/distribution/invite [get]
func (h *Handler) GetInviteInfo(c *gin.Context) {
	userID, ok := handler.RequireUserID(c)
	if !ok {
		return
	}

	distributor, err := h.distributorService.GetByUserID(c.Request.Context(), userID)
	if handler.HandleError(c, err) {
		return
	}

	inviteInfo, err := h.inviteService.GenerateInviteInfo(c.Request.Context(), distributor.ID)
	handler.MustSucceed(c, err, inviteInfo)
}

// GetShareContent 获取分享内容
// @Summary 获取分享内容
// @Tags 分销
// @Produce json
// @Security Bearer
// @Success 200 {object} response.Response{data=distribution.ShareContent}
// @Router /api/v1/distribution/share [get]
func (h *Handler) GetShareContent(c *gin.Context) {
	userID, ok := handler.RequireUserID(c)
	if !ok {
		return
	}

	distributor, err := h.distributorService.GetByUserID(c.Request.Context(), userID)
	if handler.HandleError(c, err) {
		return
	}

	shareContent, err := h.inviteService.GenerateShareContent(c.Request.Context(), distributor.ID)
	handler.MustSucceed(c, err, shareContent)
}

// ValidateInviteCode 验证邀请码
// @Summary 验证邀请码
// @Tags 分销
// @Produce json
// @Param code query string true "邀请码"
// @Success 200 {object} response.Response{data=models.Distributor}
// @Router /api/v1/distribution/invite/validate [get]
func (h *Handler) ValidateInviteCode(c *gin.Context) {
	code := c.Query("code")
	if code == "" {
		response.BadRequest(c, "邀请码不能为空")
		return
	}

	distributor, err := h.inviteService.ValidateInviteCode(c.Request.Context(), code)
	if err != nil {
		response.BadRequest(c, err.Error())
		return
	}

	response.Success(c, distributor)
}

// GetCommissions 获取佣金记录
// @Summary 获取佣金记录
// @Tags 分销
// @Produce json
// @Security Bearer
// @Param page query int false "页码" default(1)
// @Param page_size query int false "每页数量" default(10)
// @Success 200 {object} response.Response{data=response.PageData}
// @Router /api/v1/distribution/commissions [get]
func (h *Handler) GetCommissions(c *gin.Context) {
	userID, ok := handler.RequireUserID(c)
	if !ok {
		return
	}

	p := handler.BindPagination(c)

	distributor, err := h.distributorService.GetByUserID(c.Request.Context(), userID)
	if handler.HandleError(c, err) {
		return
	}

	commissions, total, err := h.commissionService.GetByDistributorID(c.Request.Context(), distributor.ID, p.GetOffset(), p.GetLimit())
	handler.MustSucceedPage(c, err, commissions, total, p.Page, p.PageSize)
}

// GetCommissionStats 获取佣金统计
// @Summary 获取佣金统计
// @Tags 分销
// @Produce json
// @Security Bearer
// @Success 200 {object} response.Response
// @Router /api/v1/distribution/commissions/stats [get]
func (h *Handler) GetCommissionStats(c *gin.Context) {
	userID, ok := handler.RequireUserID(c)
	if !ok {
		return
	}

	distributor, err := h.distributorService.GetByUserID(c.Request.Context(), userID)
	if handler.HandleError(c, err) {
		return
	}

	stats, err := h.commissionService.GetStats(c.Request.Context(), distributor.ID)
	handler.MustSucceed(c, err, stats)
}

// WithdrawRequest 提现请求
type WithdrawRequest struct {
	Amount      float64 `json:"amount" binding:"required,gt=0"`  // 提现金额
	WithdrawTo  string  `json:"withdraw_to" binding:"required"`  // 提现方式: wechat/alipay/bank
	AccountInfo string  `json:"account_info" binding:"required"` // 账户信息（JSON格式）
}

// ApplyWithdraw 申请提现
// @Summary 申请提现
// @Tags 分销-提现
// @Accept json
// @Produce json
// @Security Bearer
// @Param request body WithdrawRequest true "请求参数"
// @Success 200 {object} response.Response{data=distribution.WithdrawResponse}
// @Router /api/v1/distribution/withdraw [post]
func (h *Handler) ApplyWithdraw(c *gin.Context) {
	userID, ok := handler.RequireUserID(c)
	if !ok {
		return
	}

	var req WithdrawRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "参数错误: "+err.Error())
		return
	}

	withdrawReq := &distribution.WithdrawRequest{
		UserID:      userID,
		Type:        "commission", // 分销商只能提佣金
		Amount:      req.Amount,
		WithdrawTo:  req.WithdrawTo,
		AccountInfo: req.AccountInfo,
	}

	result, err := h.withdrawService.Apply(c.Request.Context(), withdrawReq)
	handler.MustSucceed(c, err, result)
}

// GetWithdrawals 获取提现记录
// @Summary 获取提现记录
// @Tags 分销-提现
// @Produce json
// @Security Bearer
// @Param page query int false "页码" default(1)
// @Param page_size query int false "每页数量" default(10)
// @Success 200 {object} response.Response{data=response.PageData}
// @Router /api/v1/distribution/withdrawals [get]
func (h *Handler) GetWithdrawals(c *gin.Context) {
	userID, ok := handler.RequireUserID(c)
	if !ok {
		return
	}

	p := handler.BindPagination(c)

	withdrawals, total, err := h.withdrawService.GetByUserID(c.Request.Context(), userID, p.GetOffset(), p.GetLimit())
	handler.MustSucceedPage(c, err, withdrawals, total, p.Page, p.PageSize)
}

// GetWithdrawConfig 获取提现配置
// @Summary 获取提现配置
// @Tags 分销-提现
// @Produce json
// @Security Bearer
// @Success 200 {object} response.Response
// @Router /api/v1/distribution/withdraw/config [get]
func (h *Handler) GetWithdrawConfig(c *gin.Context) {
	config := h.withdrawService.GetConfig()
	response.Success(c, config)
}

// GetRanking 获取分销排行榜
// @Summary 获取分销排行榜
// @Tags 分销
// @Produce json
// @Param limit query int false "数量限制" default(10)
// @Success 200 {object} response.Response{data=[]models.Distributor}
// @Router /api/v1/distribution/ranking [get]
func (h *Handler) GetRanking(c *gin.Context) {
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "10"))
	if limit < 1 || limit > 100 {
		limit = 10
	}

	distributors, err := h.distributorService.GetTopDistributors(c.Request.Context(), limit)
	handler.MustSucceed(c, err, distributors)
}

// CheckStatus 检查是否是分销商
// @Summary 检查是否是分销商
// @Tags 分销
// @Produce json
// @Security Bearer
// @Success 200 {object} response.Response
// @Router /api/v1/distribution/check [get]
func (h *Handler) CheckStatus(c *gin.Context) {
	userID, ok := handler.RequireUserID(c)
	if !ok {
		return
	}

	isDistributor, err := h.distributorService.CheckIsDistributor(c.Request.Context(), userID)
	if handler.HandleError(c, err) {
		return
	}

	var status string
	var statusCode int
	if isDistributor {
		distributor, _ := h.distributorService.GetByUserID(c.Request.Context(), userID)
		if distributor != nil {
			statusCode = distributor.Status
			switch distributor.Status {
			case 0:
				status = "pending"
			case 1:
				status = "approved"
			case 2:
				status = "rejected"
			}
		}
	} else {
		status = "none"
		statusCode = -1
	}

	response.Success(c, gin.H{
		"is_distributor": isDistributor,
		"status":         status,
		"status_code":    statusCode,
	})
}
