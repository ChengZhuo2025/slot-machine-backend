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
	userID := middleware.GetUserID(c)
	if userID == 0 {
		response.Unauthorized(c, "请先登录")
		return
	}

	var req mallService.CreateReviewRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "参数错误")
		return
	}

	review, err := h.reviewService.CreateReview(c.Request.Context(), userID, &req)
	if err != nil {
		if appErr, ok := err.(*errors.AppError); ok {
			response.Error(c, appErr.Code, appErr.Message)
			return
		}
		response.InternalError(c, err.Error())
		return
	}

	response.Success(c, review)
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
	productID, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		response.BadRequest(c, "无效的商品ID")
		return
	}

	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "10"))

	result, err := h.reviewService.GetProductReviews(c.Request.Context(), productID, page, pageSize)
	if err != nil {
		if appErr, ok := err.(*errors.AppError); ok {
			response.Error(c, appErr.Code, appErr.Message)
			return
		}
		response.InternalError(c, err.Error())
		return
	}

	response.Success(c, result)
}

// GetProductReviewStats 获取商品评价统计
// @Summary 获取商品评价统计
// @Tags 评价
// @Produce json
// @Param id path int true "商品ID"
// @Success 200 {object} response.Response{data=mall.ReviewStats}
// @Router /api/v1/products/{id}/review-stats [get]
func (h *ReviewHandler) GetProductReviewStats(c *gin.Context) {
	productID, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		response.BadRequest(c, "无效的商品ID")
		return
	}

	stats, err := h.reviewService.GetProductReviewStats(c.Request.Context(), productID)
	if err != nil {
		if appErr, ok := err.(*errors.AppError); ok {
			response.Error(c, appErr.Code, appErr.Message)
			return
		}
		response.InternalError(c, err.Error())
		return
	}

	response.Success(c, stats)
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
	userID := middleware.GetUserID(c)
	if userID == 0 {
		response.Unauthorized(c, "请先登录")
		return
	}

	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "10"))

	result, err := h.reviewService.GetUserReviews(c.Request.Context(), userID, page, pageSize)
	if err != nil {
		if appErr, ok := err.(*errors.AppError); ok {
			response.Error(c, appErr.Code, appErr.Message)
			return
		}
		response.InternalError(c, err.Error())
		return
	}

	response.Success(c, result)
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
	userID := middleware.GetUserID(c)
	if userID == 0 {
		response.Unauthorized(c, "请先登录")
		return
	}

	reviewID, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		response.BadRequest(c, "无效的评价ID")
		return
	}

	if err := h.reviewService.DeleteReview(c.Request.Context(), userID, reviewID); err != nil {
		if appErr, ok := err.(*errors.AppError); ok {
			response.Error(c, appErr.Code, appErr.Message)
			return
		}
		response.InternalError(c, err.Error())
		return
	}

	response.Success(c, nil)
}
