// Package mall 提供商城相关的 HTTP Handler
package mall

import (
	"strconv"

	"github.com/gin-gonic/gin"

	"github.com/dumeirei/smart-locker-backend/internal/common/errors"
	"github.com/dumeirei/smart-locker-backend/internal/common/response"
	"github.com/dumeirei/smart-locker-backend/internal/middleware"
	mallService "github.com/dumeirei/smart-locker-backend/internal/service/mall"
)

// CartHandler 购物车处理器
type CartHandler struct {
	cartService *mallService.CartService
}

// NewCartHandler 创建购物车处理器
func NewCartHandler(cartSvc *mallService.CartService) *CartHandler {
	return &CartHandler{
		cartService: cartSvc,
	}
}

// GetCart 获取购物车
// @Summary 获取购物车
// @Tags 购物车
// @Produce json
// @Security Bearer
// @Success 200 {object} response.Response{data=mall.CartInfo}
// @Router /api/v1/cart [get]
func (h *CartHandler) GetCart(c *gin.Context) {
	userID := middleware.GetUserID(c)
	if userID == 0 {
		response.Unauthorized(c, "请先登录")
		return
	}

	cart, err := h.cartService.GetCart(c.Request.Context(), userID)
	if err != nil {
		if appErr, ok := err.(*errors.AppError); ok {
			response.Error(c, appErr.Code, appErr.Message)
			return
		}
		response.InternalError(c, err.Error())
		return
	}

	response.Success(c, cart)
}

// AddItem 添加商品到购物车
// @Summary 添加商品到购物车
// @Tags 购物车
// @Accept json
// @Produce json
// @Security Bearer
// @Param request body mall.AddCartItemRequest true "请求参数"
// @Success 200 {object} response.Response{data=mall.CartItemInfo}
// @Router /api/v1/cart [post]
func (h *CartHandler) AddItem(c *gin.Context) {
	userID := middleware.GetUserID(c)
	if userID == 0 {
		response.Unauthorized(c, "请先登录")
		return
	}

	var req mallService.AddCartItemRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "参数错误")
		return
	}

	item, err := h.cartService.AddItem(c.Request.Context(), userID, &req)
	if err != nil {
		if appErr, ok := err.(*errors.AppError); ok {
			response.Error(c, appErr.Code, appErr.Message)
			return
		}
		response.InternalError(c, err.Error())
		return
	}

	response.Success(c, item)
}

// UpdateItem 更新购物车项
// @Summary 更新购物车项
// @Tags 购物车
// @Accept json
// @Produce json
// @Security Bearer
// @Param id path int true "购物车项ID"
// @Param request body mall.UpdateCartItemRequest true "请求参数"
// @Success 200 {object} response.Response{data=mall.CartItemInfo}
// @Router /api/v1/cart/{id} [put]
func (h *CartHandler) UpdateItem(c *gin.Context) {
	userID := middleware.GetUserID(c)
	if userID == 0 {
		response.Unauthorized(c, "请先登录")
		return
	}

	itemID, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		response.BadRequest(c, "无效的购物车项ID")
		return
	}

	var req mallService.UpdateCartItemRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "参数错误")
		return
	}

	item, err := h.cartService.UpdateItem(c.Request.Context(), userID, itemID, &req)
	if err != nil {
		if appErr, ok := err.(*errors.AppError); ok {
			response.Error(c, appErr.Code, appErr.Message)
			return
		}
		response.InternalError(c, err.Error())
		return
	}

	response.Success(c, item)
}

// RemoveItem 移除购物车项
// @Summary 移除购物车项
// @Tags 购物车
// @Produce json
// @Security Bearer
// @Param id path int true "购物车项ID"
// @Success 200 {object} response.Response
// @Router /api/v1/cart/{id} [delete]
func (h *CartHandler) RemoveItem(c *gin.Context) {
	userID := middleware.GetUserID(c)
	if userID == 0 {
		response.Unauthorized(c, "请先登录")
		return
	}

	itemID, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		response.BadRequest(c, "无效的购物车项ID")
		return
	}

	if err := h.cartService.RemoveItem(c.Request.Context(), userID, itemID); err != nil {
		if appErr, ok := err.(*errors.AppError); ok {
			response.Error(c, appErr.Code, appErr.Message)
			return
		}
		response.InternalError(c, err.Error())
		return
	}

	response.Success(c, nil)
}

// ClearCart 清空购物车
// @Summary 清空购物车
// @Tags 购物车
// @Produce json
// @Security Bearer
// @Success 200 {object} response.Response
// @Router /api/v1/cart [delete]
func (h *CartHandler) ClearCart(c *gin.Context) {
	userID := middleware.GetUserID(c)
	if userID == 0 {
		response.Unauthorized(c, "请先登录")
		return
	}

	if err := h.cartService.ClearCart(c.Request.Context(), userID); err != nil {
		if appErr, ok := err.(*errors.AppError); ok {
			response.Error(c, appErr.Code, appErr.Message)
			return
		}
		response.InternalError(c, err.Error())
		return
	}

	response.Success(c, nil)
}

// SelectAll 全选/取消全选
// @Summary 全选/取消全选
// @Tags 购物车
// @Accept json
// @Produce json
// @Security Bearer
// @Param selected query bool true "是否选中"
// @Success 200 {object} response.Response
// @Router /api/v1/cart/select-all [put]
func (h *CartHandler) SelectAll(c *gin.Context) {
	userID := middleware.GetUserID(c)
	if userID == 0 {
		response.Unauthorized(c, "请先登录")
		return
	}

	selected := c.Query("selected") == "true"

	if err := h.cartService.SelectAll(c.Request.Context(), userID, selected); err != nil {
		if appErr, ok := err.(*errors.AppError); ok {
			response.Error(c, appErr.Code, appErr.Message)
			return
		}
		response.InternalError(c, err.Error())
		return
	}

	response.Success(c, nil)
}

// GetCartCount 获取购物车商品数量
// @Summary 获取购物车商品数量
// @Tags 购物车
// @Produce json
// @Security Bearer
// @Success 200 {object} response.Response{data=int}
// @Router /api/v1/cart/count [get]
func (h *CartHandler) GetCartCount(c *gin.Context) {
	userID := middleware.GetUserID(c)
	if userID == 0 {
		response.Unauthorized(c, "请先登录")
		return
	}

	count, err := h.cartService.GetCartCount(c.Request.Context(), userID)
	if err != nil {
		if appErr, ok := err.(*errors.AppError); ok {
			response.Error(c, appErr.Code, appErr.Message)
			return
		}
		response.InternalError(c, err.Error())
		return
	}

	response.Success(c, gin.H{"count": count})
}
