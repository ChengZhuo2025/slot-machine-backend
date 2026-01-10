// Package user 用户端 HTTP Handler
package user

import (
	"strconv"

	"github.com/gin-gonic/gin"

	"github.com/dumeirei/smart-locker-backend/internal/common/response"
	userService "github.com/dumeirei/smart-locker-backend/internal/service/user"
)

// FeedbackHandler 用户反馈处理器
type FeedbackHandler struct {
	feedbackService *userService.FeedbackService
}

// NewFeedbackHandler 创建用户反馈处理器
func NewFeedbackHandler(feedbackService *userService.FeedbackService) *FeedbackHandler {
	return &FeedbackHandler{feedbackService: feedbackService}
}

// Create 创建反馈
// @Summary 提交用户反馈
// @Tags 用户-反馈
// @Accept json
// @Produce json
// @Security Bearer
// @Param request body userService.CreateFeedbackRequest true "反馈内容"
// @Success 200 {object} response.Response{data=models.UserFeedback}
// @Router /api/v1/user/feedbacks [post]
func (h *FeedbackHandler) Create(c *gin.Context) {
	userID := c.GetInt64("user_id")
	if userID == 0 {
		response.Unauthorized(c, "请先登录")
		return
	}

	var req userService.CreateFeedbackRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, err.Error())
		return
	}

	feedback, err := h.feedbackService.Create(c.Request.Context(), userID, &req)
	if err != nil {
		response.InternalError(c, err.Error())
		return
	}

	response.Success(c, feedback)
}

// List 获取我的反馈列表
// @Summary 获取我的反馈列表
// @Tags 用户-反馈
// @Produce json
// @Security Bearer
// @Param page query int false "页码" default(1)
// @Param page_size query int false "每页数量" default(20)
// @Success 200 {object} response.Response{data=response.ListData}
// @Router /api/v1/user/feedbacks [get]
func (h *FeedbackHandler) List(c *gin.Context) {
	userID := c.GetInt64("user_id")
	if userID == 0 {
		response.Unauthorized(c, "请先登录")
		return
	}

	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "20"))

	feedbacks, total, err := h.feedbackService.ListByUser(c.Request.Context(), userID, page, pageSize)
	if err != nil {
		response.InternalError(c, err.Error())
		return
	}

	response.SuccessList(c, feedbacks, total, page, pageSize)
}

// GetByID 获取反馈详情
// @Summary 获取反馈详情
// @Tags 用户-反馈
// @Produce json
// @Security Bearer
// @Param id path int true "反馈ID"
// @Success 200 {object} response.Response{data=models.UserFeedback}
// @Router /api/v1/user/feedbacks/{id} [get]
func (h *FeedbackHandler) GetByID(c *gin.Context) {
	userID := c.GetInt64("user_id")
	if userID == 0 {
		response.Unauthorized(c, "请先登录")
		return
	}

	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		response.BadRequest(c, "无效的反馈ID")
		return
	}

	feedback, err := h.feedbackService.GetByID(c.Request.Context(), id)
	if err != nil {
		response.NotFound(c, "反馈不存在")
		return
	}

	// 检查是否是自己的反馈
	if feedback.UserID != userID {
		response.Forbidden(c, "无权查看此反馈")
		return
	}

	response.Success(c, feedback)
}

// Delete 删除反馈
// @Summary 删除反馈
// @Tags 用户-反馈
// @Produce json
// @Security Bearer
// @Param id path int true "反馈ID"
// @Success 200 {object} response.Response
// @Router /api/v1/user/feedbacks/{id} [delete]
func (h *FeedbackHandler) Delete(c *gin.Context) {
	userID := c.GetInt64("user_id")
	if userID == 0 {
		response.Unauthorized(c, "请先登录")
		return
	}

	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		response.BadRequest(c, "无效的反馈ID")
		return
	}

	if err := h.feedbackService.Delete(c.Request.Context(), id, userID); err != nil {
		if err == userService.ErrNotOwner {
			response.Forbidden(c, err.Error())
			return
		}
		response.InternalError(c, err.Error())
		return
	}

	response.Success(c, nil)
}
