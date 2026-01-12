// Package user 提供用户相关的 HTTP Handler
package user

import (
	"github.com/gin-gonic/gin"

	"github.com/dumeirei/smart-locker-backend/internal/common/handler"
	"github.com/dumeirei/smart-locker-backend/internal/common/response"
	userService "github.com/dumeirei/smart-locker-backend/internal/service/user"
)

// MemberHandler 会员处理器
type MemberHandler struct {
	memberLevelService   *userService.MemberLevelService
	memberPackageService *userService.MemberPackageService
	pointsService        *userService.PointsService
}

// NewMemberHandler 创建会员处理器
func NewMemberHandler(
	memberLevelSvc *userService.MemberLevelService,
	memberPackageSvc *userService.MemberPackageService,
	pointsSvc *userService.PointsService,
) *MemberHandler {
	return &MemberHandler{
		memberLevelService:   memberLevelSvc,
		memberPackageService: memberPackageSvc,
		pointsService:        pointsSvc,
	}
}

// GetMemberInfo 获取会员信息
// @Summary 获取会员信息
// @Description 获取当前用户的会员等级、积分、权益等信息
// @Tags 会员
// @Produce json
// @Security Bearer
// @Success 200 {object} response.Response{data=userService.UserMemberInfo}
// @Router /api/v1/member/info [get]
func (h *MemberHandler) GetMemberInfo(c *gin.Context) {
	userID, ok := handler.RequireUserID(c)
	if !ok {
		return
	}

	info, err := h.memberLevelService.GetUserMemberInfo(c.Request.Context(), userID)
	handler.MustSucceed(c, err, info)
}

// GetMemberLevels 获取所有会员等级
// @Summary 获取会员等级列表
// @Description 获取系统中所有会员等级及其权益
// @Tags 会员
// @Produce json
// @Success 200 {object} response.Response{data=[]userService.MemberLevelInfo}
// @Router /api/v1/member/levels [get]
func (h *MemberHandler) GetMemberLevels(c *gin.Context) {
	levels, err := h.memberLevelService.GetAllLevels(c.Request.Context())
	handler.MustSucceed(c, err, levels)
}

// GetMemberPackages 获取会员套餐列表
// @Summary 获取会员套餐列表
// @Description 获取所有可购买的会员套餐
// @Tags 会员
// @Produce json
// @Success 200 {object} response.Response{data=[]userService.PackageInfo}
// @Router /api/v1/member/packages [get]
func (h *MemberHandler) GetMemberPackages(c *gin.Context) {
	packages, err := h.memberPackageService.GetActivePackages(c.Request.Context())
	handler.MustSucceed(c, err, packages)
}

// GetRecommendedPackages 获取推荐套餐
// @Summary 获取推荐套餐
// @Description 获取推荐的会员套餐
// @Tags 会员
// @Produce json
// @Success 200 {object} response.Response{data=[]userService.PackageInfo}
// @Router /api/v1/member/packages/recommended [get]
func (h *MemberHandler) GetRecommendedPackages(c *gin.Context) {
	packages, err := h.memberPackageService.GetRecommendedPackages(c.Request.Context())
	handler.MustSucceed(c, err, packages)
}

// GetPackageDetail 获取套餐详情
// @Summary 获取套餐详情
// @Description 获取指定会员套餐的详细信息
// @Tags 会员
// @Produce json
// @Param id path int true "套餐ID"
// @Success 200 {object} response.Response{data=userService.PackageInfo}
// @Router /api/v1/member/packages/{id} [get]
func (h *MemberHandler) GetPackageDetail(c *gin.Context) {
	id, ok := handler.ParseID(c, "套餐")
	if !ok {
		return
	}

	pkg, err := h.memberPackageService.GetPackageByID(c.Request.Context(), id)
	handler.MustSucceed(c, err, pkg)
}

// PurchasePackageRequest 购买套餐请求
type PurchasePackageRequest struct {
	PackageID int64 `json:"package_id" binding:"required"`
}

// PurchasePackage 购买会员套餐
// @Summary 购买会员套餐
// @Description 购买指定的会员套餐，成功后升级会员等级并赠送积分
// @Tags 会员
// @Accept json
// @Produce json
// @Security Bearer
// @Param request body PurchasePackageRequest true "请求参数"
// @Success 200 {object} response.Response{data=userService.PurchaseResult}
// @Router /api/v1/member/packages/purchase [post]
func (h *MemberHandler) PurchasePackage(c *gin.Context) {
	userID, ok := handler.RequireUserID(c)
	if !ok {
		return
	}

	var req PurchasePackageRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "参数错误")
		return
	}

	result, err := h.memberPackageService.PurchasePackage(c.Request.Context(), userID, req.PackageID)
	handler.MustSucceed(c, err, result)
}

// GetPointsInfo 获取积分信息
// @Summary 获取积分信息
// @Description 获取用户积分余额及会员等级升级进度
// @Tags 会员-积分
// @Produce json
// @Security Bearer
// @Success 200 {object} response.Response{data=userService.PointsInfo}
// @Router /api/v1/member/points [get]
func (h *MemberHandler) GetPointsInfo(c *gin.Context) {
	userID, ok := handler.RequireUserID(c)
	if !ok {
		return
	}

	info, err := h.pointsService.GetPointsInfo(c.Request.Context(), userID)
	handler.MustSucceed(c, err, info)
}

// GetPointsHistory 获取积分历史
// @Summary 获取积分历史
// @Description 获取用户积分变动历史记录
// @Tags 会员-积分
// @Produce json
// @Security Bearer
// @Param page query int false "页码" default(1)
// @Param page_size query int false "每页数量" default(10)
// @Param type query string false "积分类型"
// @Success 200 {object} response.Response{data=response.PageData}
// @Router /api/v1/member/points/history [get]
func (h *MemberHandler) GetPointsHistory(c *gin.Context) {
	userID, ok := handler.RequireUserID(c)
	if !ok {
		return
	}

	p := handler.BindPagination(c)

	pointsType := c.Query("type")

	records, total, err := h.pointsService.GetPointsHistory(
		c.Request.Context(),
		userID,
		p.GetOffset(),
		p.GetLimit(),
		pointsType,
	)
	handler.MustSucceedPage(c, err, records, total, p.Page, p.PageSize)
}

// GetMemberBenefits 获取会员权益
// @Summary 获取会员权益
// @Description 获取当前会员等级的所有权益
// @Tags 会员
// @Produce json
// @Security Bearer
// @Success 200 {object} response.Response
// @Router /api/v1/member/benefits [get]
func (h *MemberHandler) GetMemberBenefits(c *gin.Context) {
	userID, ok := handler.RequireUserID(c)
	if !ok {
		return
	}

	info, err := h.memberLevelService.GetUserMemberInfo(c.Request.Context(), userID)
	if handler.HandleError(c, err) {
		return
	}

	// 返回当前等级的权益
	benefits := gin.H{
		"level_name": "",
		"discount":   1.0,
		"benefits":   map[string]interface{}{},
	}

	if info.CurrentLevel != nil {
		benefits["level_name"] = info.CurrentLevel.Name
		benefits["discount"] = info.CurrentLevel.Discount
		if info.CurrentLevel.Benefits != nil {
			benefits["benefits"] = info.CurrentLevel.Benefits
		}
	}

	response.Success(c, benefits)
}

// GetDiscount 获取会员折扣
// @Summary 获取会员折扣
// @Description 获取当前用户的会员折扣率
// @Tags 会员
// @Produce json
// @Security Bearer
// @Success 200 {object} response.Response
// @Router /api/v1/member/discount [get]
func (h *MemberHandler) GetDiscount(c *gin.Context) {
	userID, ok := handler.RequireUserID(c)
	if !ok {
		return
	}

	discount, err := h.memberLevelService.GetDiscount(c.Request.Context(), userID)
	handler.MustSucceed(c, err, gin.H{"discount": discount})
}

// RegisterRoutes 注册会员相关路由
func (h *MemberHandler) RegisterRoutes(r *gin.RouterGroup) {
	member := r.Group("/member")
	{
		// 会员信息
		member.GET("/info", h.GetMemberInfo)
		member.GET("/levels", h.GetMemberLevels)
		member.GET("/benefits", h.GetMemberBenefits)
		member.GET("/discount", h.GetDiscount)

		// 会员套餐
		member.GET("/packages", h.GetMemberPackages)
		member.GET("/packages/recommended", h.GetRecommendedPackages)
		member.GET("/packages/:id", h.GetPackageDetail)
		member.POST("/packages/purchase", h.PurchasePackage)

		// 积分
		member.GET("/points", h.GetPointsInfo)
		member.GET("/points/history", h.GetPointsHistory)
	}
}
