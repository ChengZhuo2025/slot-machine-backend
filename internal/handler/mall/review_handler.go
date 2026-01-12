// Package mall 提供商城相关的 HTTP Handler
package mall

import (
	"github.com/gin-gonic/gin"

	"github.com/dumeirei/smart-locker-backend/internal/common/handler"
	"github.com/dumeirei/smart-locker-backend/internal/common/response"
	mallService "github.com/dumeirei/smart-locker-backend/internal/service/mall"
)

// ReviewHandler 评价处理器
type ReviewHandler struct {
	reviewService *mallService.ReviewService
}

// NewReviewHandler 创建评价处理器
func NewReviewHandler(reviewSvc *mallService.ReviewService) *ReviewHandler {
	return &ReviewHandler{
		reviewService: reviewSvc,
	}
}

// CreateReview 创建评价
// @Summary 创建评价
// @Tags 评价
// @Accept json
// @Produce json
// @Security Bearer
// @Param request body mall.CreateReviewRequest true "请求参数"
// @Success 200 {object} response.Response{data=mall.ReviewInfo}
// @Router /api/v1/reviews [post]
func (h *ReviewHandler) CreateReview(c *gin.Context) {
	userID, ok := handler.RequireUserID(c)
	if !ok {
		return
	}

	var req mallService.CreateReviewRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "参数错误")
		return
	}

	review, err := h.reviewService.CreateReview(c.Request.Context(), userID, &req)
	handler.MustSucceed(c, err, review)
}

// GetProductReviews 获取商品评价列表
// @Summary 获取商品评价列表
// @Tags 评价
// @Produce json
// @Param id path int true "商品ID"
// @Param page query int false "页码"
// @Param page_size query int false "每页数量"
// @Success 200 {object} response.Response{data=mall.ReviewListResponse}
// @Router /api/v1/products/{id}/reviews [get]
func (h *ReviewHandler) GetProductReviews(c *gin.Context) {
	productID, ok := handler.ParseID(c, "商品")
	if !ok {
		return
	}

	p := handler.BindPagination(c)

	result, err := h.reviewService.GetProductReviews(c.Request.Context(), productID, p.Page, p.PageSize)
	handler.MustSucceed(c, err, result)
}

// GetProductReviewStats 获取商品评价统计
// @Summary 获取商品评价统计
// @Tags 评价
// @Produce json
// @Param id path int true "商品ID"
// @Success 200 {object} response.Response{data=mall.ReviewStats}
// @Router /api/v1/products/{id}/review-stats [get]
func (h *ReviewHandler) GetProductReviewStats(c *gin.Context) {
	productID, ok := handler.ParseID(c, "商品")
	if !ok {
		return
	}

	stats, err := h.reviewService.GetProductReviewStats(c.Request.Context(), productID)
	handler.MustSucceed(c, err, stats)
}

// GetUserReviews 获取用户评价列表
// @Summary 获取用户评价列表
// @Tags 评价
// @Produce json
// @Security Bearer
// @Param page query int false "页码"
// @Param page_size query int false "每页数量"
// @Success 200 {object} response.Response{data=mall.ReviewListResponse}
// @Router /api/v1/user/reviews [get]
func (h *ReviewHandler) GetUserReviews(c *gin.Context) {
	userID, ok := handler.RequireUserID(c)
	if !ok {
		return
	}

	p := handler.BindPagination(c)

	result, err := h.reviewService.GetUserReviews(c.Request.Context(), userID, p.Page, p.PageSize)
	handler.MustSucceed(c, err, result)
}

// DeleteReview 删除评价
// @Summary 删除评价
// @Tags 评价
// @Produce json
// @Security Bearer
// @Param id path int true "评价ID"
// @Success 200 {object} response.Response
// @Router /api/v1/reviews/{id} [delete]
func (h *ReviewHandler) DeleteReview(c *gin.Context) {
	userID, reviewID, ok := handler.RequireUserAndParseID(c, "评价")
	if !ok {
		return
	}

	handler.MustSucceed(c, h.reviewService.DeleteReview(c.Request.Context(), userID, reviewID), nil)
}
