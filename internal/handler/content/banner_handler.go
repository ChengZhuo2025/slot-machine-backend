// Package content 内容 HTTP Handler
package content

import (
	"strconv"

	"github.com/gin-gonic/gin"

	"github.com/dumeirei/smart-locker-backend/internal/common/handler"
	"github.com/dumeirei/smart-locker-backend/internal/common/logger"
	"github.com/dumeirei/smart-locker-backend/internal/common/response"
	contentService "github.com/dumeirei/smart-locker-backend/internal/service/content"
)

// BannerHandler 轮播图处理器（用户端）
type BannerHandler struct {
	bannerService *contentService.BannerService
}

// NewBannerHandler 创建轮播图处理器
func NewBannerHandler(bannerService *contentService.BannerService) *BannerHandler {
	return &BannerHandler{bannerService: bannerService}
}

// ListByPosition 获取轮播图列表
// @Summary 获取轮播图列表
// @Tags 内容-轮播图
// @Produce json
// @Param position query string true "位置: home/mall/hotel"
// @Param limit query int false "数量" default(10)
// @Success 200 {object} response.Response{data=[]contentService.BannerResponse}
// @Router /api/v1/banners [get]
func (h *BannerHandler) ListByPosition(c *gin.Context) {
	position := c.Query("position")
	if position == "" {
		response.BadRequest(c, "位置参数不能为空")
		return
	}

	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "10"))

	banners, err := h.bannerService.ListByPosition(c.Request.Context(), position, limit)
	handler.MustSucceed(c, err, banners)
}

// RecordClick 记录点击
// @Summary 记录轮播图点击
// @Tags 内容-轮播图
// @Produce json
// @Param id path int true "轮播图ID"
// @Success 200 {object} response.Response
// @Router /api/v1/banners/{id}/click [post]
func (h *BannerHandler) RecordClick(c *gin.Context) {
	id, ok := handler.ParseID(c, "轮播图")
	if !ok {
		return
	}

	if err := h.bannerService.RecordClick(c.Request.Context(), id); err != nil {
		logger.Warn("记录轮播图点击失败",
			logger.Int64("banner_id", id),
			logger.Err(err),
		)
	}

	response.Success(c, nil)
}
