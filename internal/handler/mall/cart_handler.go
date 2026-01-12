// Package mall 提供商城相关的 HTTP Handler
package mall

import (
	"github.com/gin-gonic/gin"

	"github.com/dumeirei/smart-locker-backend/internal/common/handler"
	"github.com/dumeirei/smart-locker-backend/internal/common/response"
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
	userID, ok := handler.RequireUserID(c)
	if !ok {
		return
	}

	cart, err := h.cartService.GetCart(c.Request.Context(), userID)
	handler.MustSucceed(c, err, cart)
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
	userID, ok := handler.RequireUserID(c)
	if !ok {
		return
	}

	var req mallService.AddCartItemRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "参数错误")
		return
	}

	item, err := h.cartService.AddItem(c.Request.Context(), userID, &req)
	handler.MustSucceed(c, err, item)
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
	userID, itemID, ok := handler.RequireUserAndParseID(c, "购物车项")
	if !ok {
		return
	}

	var req mallService.UpdateCartItemRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "参数错误")
		return
	}

	item, err := h.cartService.UpdateItem(c.Request.Context(), userID, itemID, &req)
	handler.MustSucceed(c, err, item)
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
	userID, itemID, ok := handler.RequireUserAndParseID(c, "购物车项")
	if !ok {
		return
	}

	handler.MustSucceed(c, h.cartService.RemoveItem(c.Request.Context(), userID, itemID), nil)
}

// ClearCart 清空购物车
// @Summary 清空购物车
// @Tags 购物车
// @Produce json
// @Security Bearer
// @Success 200 {object} response.Response
// @Router /api/v1/cart [delete]
func (h *CartHandler) ClearCart(c *gin.Context) {
	userID, ok := handler.RequireUserID(c)
	if !ok {
		return
	}

	handler.MustSucceed(c, h.cartService.ClearCart(c.Request.Context(), userID), nil)
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
	userID, ok := handler.RequireUserID(c)
	if !ok {
		return
	}

	selected := c.Query("selected") == "true"

	handler.MustSucceed(c, h.cartService.SelectAll(c.Request.Context(), userID, selected), nil)
}

// GetCartCount 获取购物车商品数量
// @Summary 获取购物车商品数量
// @Tags 购物车
// @Produce json
// @Security Bearer
// @Success 200 {object} response.Response{data=int}
// @Router /api/v1/cart/count [get]
func (h *CartHandler) GetCartCount(c *gin.Context) {
	userID, ok := handler.RequireUserID(c)
	if !ok {
		return
	}

	count, err := h.cartService.GetCartCount(c.Request.Context(), userID)
	handler.MustSucceed(c, err, gin.H{"count": count})
}
