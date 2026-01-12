// Package marketing 提供营销相关的 HTTP Handler
package marketing

import (
	"strconv"

	"github.com/gin-gonic/gin"

	"github.com/dumeirei/smart-locker-backend/internal/common/handler"
	"github.com/dumeirei/smart-locker-backend/internal/common/response"
	marketingService "github.com/dumeirei/smart-locker-backend/internal/service/marketing"
)

// CouponHandler 优惠券处理器
type CouponHandler struct {
	couponService     *marketingService.CouponService
	userCouponService *marketingService.UserCouponService
}

// NewCouponHandler 创建优惠券处理器
func NewCouponHandler(couponSvc *marketingService.CouponService, userCouponSvc *marketingService.UserCouponService) *CouponHandler {
	return &CouponHandler{
		couponService:     couponSvc,
		userCouponService: userCouponSvc,
	}
}

// GetCouponList 获取可领取的优惠券列表
// @Summary 获取可领取的优惠券列表
// @Tags 营销-优惠券
// @Produce json
// @Security Bearer
// @Param page query int false "页码" default(1)
// @Param page_size query int false "每页数量" default(10)
// @Success 200 {object} response.Response{data=marketing.CouponListResponse}
// @Router /api/v1/marketing/coupons [get]
func (h *CouponHandler) GetCouponList(c *gin.Context) {
	userID, ok := handler.RequireUserID(c)
	if !ok {
		return
	}

	p := handler.BindPagination(c)

	req := &marketingService.CouponListRequest{
		Page:     p.Page,
		PageSize: p.PageSize,
	}

	result, err := h.couponService.GetCouponList(c.Request.Context(), req, userID)
	if handler.HandleError(c, err) {
		return
	}

	response.SuccessPage(c, result.List, result.Total, p.Page, p.PageSize)
}

// GetCouponDetail 获取优惠券详情
// @Summary 获取优惠券详情
// @Tags 营销-优惠券
// @Produce json
// @Security Bearer
// @Param id path int true "优惠券ID"
// @Success 200 {object} response.Response{data=marketing.CouponItem}
// @Router /api/v1/marketing/coupons/{id} [get]
func (h *CouponHandler) GetCouponDetail(c *gin.Context) {
	userID, ok := handler.RequireUserID(c)
	if !ok {
		return
	}

	couponID, ok := handler.ParseID(c, "优惠券")
	if !ok {
		return
	}

	coupon, err := h.couponService.GetCouponDetail(c.Request.Context(), couponID, userID)
	handler.MustSucceed(c, err, coupon)
}

// ReceiveCoupon 领取优惠券
// @Summary 领取优惠券
// @Tags 营销-优惠券
// @Accept json
// @Produce json
// @Security Bearer
// @Param id path int true "优惠券ID"
// @Success 200 {object} response.Response
// @Router /api/v1/marketing/coupons/{id}/receive [post]
func (h *CouponHandler) ReceiveCoupon(c *gin.Context) {
	userID, ok := handler.RequireUserID(c)
	if !ok {
		return
	}

	couponID, ok := handler.ParseID(c, "优惠券")
	if !ok {
		return
	}

	userCoupon, err := h.couponService.ReceiveCoupon(c.Request.Context(), couponID, userID)
	if err != nil {
		response.Error(c, 400, err.Error())
		return
	}

	response.SuccessWithMessage(c, "领取成功", gin.H{
		"user_coupon_id": userCoupon.ID,
		"expired_at":     userCoupon.ExpiredAt,
	})
}

// GetUserCoupons 获取用户优惠券列表
// @Summary 获取用户优惠券列表
// @Tags 营销-用户优惠券
// @Produce json
// @Security Bearer
// @Param status query int false "状态：0-未使用 1-已使用 2-已过期"
// @Param page query int false "页码" default(1)
// @Param page_size query int false "每页数量" default(10)
// @Success 200 {object} response.Response{data=marketing.UserCouponListResponse}
// @Router /api/v1/marketing/user-coupons [get]
func (h *CouponHandler) GetUserCoupons(c *gin.Context) {
	userID, ok := handler.RequireUserID(c)
	if !ok {
		return
	}

	p := handler.BindPagination(c)

	req := &marketingService.UserCouponListRequest{
		Page:     p.Page,
		PageSize: p.PageSize,
	}

	// 处理状态筛选
	if statusStr := c.Query("status"); statusStr != "" {
		status, err := strconv.ParseInt(statusStr, 10, 8)
		if err == nil {
			s := int8(status)
			req.Status = &s
		}
	}

	result, err := h.userCouponService.GetUserCoupons(c.Request.Context(), userID, req)
	if handler.HandleError(c, err) {
		return
	}

	response.SuccessPage(c, result.List, result.Total, p.Page, p.PageSize)
}

// GetUserCouponDetail 获取用户优惠券详情
// @Summary 获取用户优惠券详情
// @Tags 营销-用户优惠券
// @Produce json
// @Security Bearer
// @Param id path int true "用户优惠券ID"
// @Success 200 {object} response.Response{data=marketing.UserCouponItem}
// @Router /api/v1/marketing/user-coupons/{id} [get]
func (h *CouponHandler) GetUserCouponDetail(c *gin.Context) {
	userID, ok := handler.RequireUserID(c)
	if !ok {
		return
	}

	userCouponID, ok := handler.ParseID(c, "用户优惠券")
	if !ok {
		return
	}

	coupon, err := h.userCouponService.GetUserCouponDetail(c.Request.Context(), userID, userCouponID)
	if err != nil {
		if err == marketingService.ErrUserCouponNotFound {
			response.NotFound(c, "用户优惠券不存在")
			return
		}
		response.InternalError(c, err.Error())
		return
	}

	response.Success(c, coupon)
}

// GetAvailableCoupons 获取可用优惠券列表
// @Summary 获取可用优惠券列表
// @Tags 营销-用户优惠券
// @Produce json
// @Security Bearer
// @Param page query int false "页码" default(1)
// @Param page_size query int false "每页数量" default(10)
// @Success 200 {object} response.Response{data=marketing.UserCouponListResponse}
// @Router /api/v1/marketing/user-coupons/available [get]
func (h *CouponHandler) GetAvailableCoupons(c *gin.Context) {
	userID, ok := handler.RequireUserID(c)
	if !ok {
		return
	}

	p := handler.BindPagination(c)

	result, err := h.userCouponService.GetAvailableCoupons(c.Request.Context(), userID, p.Page, p.PageSize)
	if handler.HandleError(c, err) {
		return
	}

	response.SuccessPage(c, result.List, result.Total, p.Page, p.PageSize)
}

// GetAvailableCouponsForOrderRequest 订单可用优惠券请求
type GetAvailableCouponsForOrderRequest struct {
	OrderType   string  `form:"order_type" binding:"required"` // 订单类型：rental/mall/hotel
	OrderAmount float64 `form:"order_amount" binding:"required,gt=0"`
}

// GetAvailableCouponsForOrder 获取订单可用优惠券
// @Summary 获取订单可用优惠券
// @Tags 营销-用户优惠券
// @Produce json
// @Security Bearer
// @Param order_type query string true "订单类型：rental/mall/hotel"
// @Param order_amount query number true "订单金额"
// @Success 200 {object} response.Response{data=[]marketing.UserCouponItem}
// @Router /api/v1/marketing/user-coupons/for-order [get]
func (h *CouponHandler) GetAvailableCouponsForOrder(c *gin.Context) {
	userID, ok := handler.RequireUserID(c)
	if !ok {
		return
	}

	var req GetAvailableCouponsForOrderRequest
	if err := c.ShouldBindQuery(&req); err != nil {
		response.BadRequest(c, "参数错误：需要订单类型和订单金额")
		return
	}

	coupons, err := h.userCouponService.GetAvailableCouponsForOrder(c.Request.Context(), userID, req.OrderType, req.OrderAmount)
	handler.MustSucceed(c, err, coupons)
}

// GetCouponCountByStatus 获取各状态优惠券数量
// @Summary 获取各状态优惠券数量
// @Tags 营销-用户优惠券
// @Produce json
// @Security Bearer
// @Success 200 {object} response.Response{data=map[string]int64}
// @Router /api/v1/marketing/user-coupons/count [get]
func (h *CouponHandler) GetCouponCountByStatus(c *gin.Context) {
	userID, ok := handler.RequireUserID(c)
	if !ok {
		return
	}

	counts, err := h.userCouponService.GetCouponCountByStatus(c.Request.Context(), userID)
	handler.MustSucceed(c, err, counts)
}
