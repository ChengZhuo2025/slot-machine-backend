// Package user 用户端 HTTP Handler
package user

import (
	"github.com/gin-gonic/gin"

	"github.com/dumeirei/smart-locker-backend/internal/common/handler"
	"github.com/dumeirei/smart-locker-backend/internal/common/response"
	userService "github.com/dumeirei/smart-locker-backend/internal/service/user"
)

// AddressHandler 地址处理器
type AddressHandler struct {
	addressService *userService.AddressService
}

// NewAddressHandler 创建地址处理器
func NewAddressHandler(addressService *userService.AddressService) *AddressHandler {
	return &AddressHandler{addressService: addressService}
}

// Create 创建地址
// @Summary 添加收货地址
// @Tags 用户-地址
// @Accept json
// @Produce json
// @Security Bearer
// @Param request body userService.CreateAddressRequest true "地址信息"
// @Success 200 {object} response.Response{data=models.Address}
// @Router /api/v1/user/addresses [post]
func (h *AddressHandler) Create(c *gin.Context) {
	userID, ok := handler.RequireUserID(c)
	if !ok {
		return
	}

	var req userService.CreateAddressRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, err.Error())
		return
	}

	address, err := h.addressService.Create(c.Request.Context(), userID, &req)
	handler.MustSucceed(c, err, address)
}

// List 获取地址列表
// @Summary 获取收货地址列表
// @Tags 用户-地址
// @Produce json
// @Security Bearer
// @Success 200 {object} response.Response{data=[]models.Address}
// @Router /api/v1/user/addresses [get]
func (h *AddressHandler) List(c *gin.Context) {
	userID, ok := handler.RequireUserID(c)
	if !ok {
		return
	}

	addresses, err := h.addressService.List(c.Request.Context(), userID)
	handler.MustSucceed(c, err, addresses)
}

// GetByID 获取地址详情
// @Summary 获取地址详情
// @Tags 用户-地址
// @Produce json
// @Security Bearer
// @Param id path int true "地址ID"
// @Success 200 {object} response.Response{data=models.Address}
// @Router /api/v1/user/addresses/{id} [get]
func (h *AddressHandler) GetByID(c *gin.Context) {
	userID, id, ok := handler.RequireUserAndParseID(c, "地址")
	if !ok {
		return
	}

	address, err := h.addressService.GetByID(c.Request.Context(), id, userID)
	handler.MustSucceed(c, err, address)
}

// Update 更新地址
// @Summary 更新收货地址
// @Tags 用户-地址
// @Accept json
// @Produce json
// @Security Bearer
// @Param id path int true "地址ID"
// @Param request body userService.UpdateAddressRequest true "地址信息"
// @Success 200 {object} response.Response{data=models.Address}
// @Router /api/v1/user/addresses/{id} [put]
func (h *AddressHandler) Update(c *gin.Context) {
	userID, id, ok := handler.RequireUserAndParseID(c, "地址")
	if !ok {
		return
	}

	var req userService.UpdateAddressRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, err.Error())
		return
	}

	address, err := h.addressService.Update(c.Request.Context(), id, userID, &req)
	handler.MustSucceed(c, err, address)
}

// Delete 删除地址
// @Summary 删除收货地址
// @Tags 用户-地址
// @Produce json
// @Security Bearer
// @Param id path int true "地址ID"
// @Success 200 {object} response.Response
// @Router /api/v1/user/addresses/{id} [delete]
func (h *AddressHandler) Delete(c *gin.Context) {
	userID, id, ok := handler.RequireUserAndParseID(c, "地址")
	if !ok {
		return
	}

	err := h.addressService.Delete(c.Request.Context(), id, userID)
	handler.MustSucceed(c, err, nil)
}

// GetDefault 获取默认地址
// @Summary 获取默认收货地址
// @Tags 用户-地址
// @Produce json
// @Security Bearer
// @Success 200 {object} response.Response{data=models.Address}
// @Router /api/v1/user/addresses/default [get]
func (h *AddressHandler) GetDefault(c *gin.Context) {
	userID, ok := handler.RequireUserID(c)
	if !ok {
		return
	}

	address, err := h.addressService.GetDefault(c.Request.Context(), userID)
	handler.MustSucceed(c, err, address)
}

// SetDefault 设置默认地址
// @Summary 设置默认收货地址
// @Tags 用户-地址
// @Produce json
// @Security Bearer
// @Param id path int true "地址ID"
// @Success 200 {object} response.Response
// @Router /api/v1/user/addresses/{id}/default [put]
func (h *AddressHandler) SetDefault(c *gin.Context) {
	userID, id, ok := handler.RequireUserAndParseID(c, "地址")
	if !ok {
		return
	}

	err := h.addressService.SetDefault(c.Request.Context(), id, userID)
	handler.MustSucceed(c, err, nil)
}
