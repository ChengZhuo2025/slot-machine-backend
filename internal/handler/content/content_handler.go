// Package content 提供内容管理相关的 HTTP Handler
package content

import (
	"strconv"

	"github.com/gin-gonic/gin"

	"github.com/dumeirei/smart-locker-backend/internal/common/errors"
	"github.com/dumeirei/smart-locker-backend/internal/common/response"
	"github.com/dumeirei/smart-locker-backend/internal/middleware"
	contentService "github.com/dumeirei/smart-locker-backend/internal/service/content"
)

// Handler 内容管理处理器
type Handler struct {
	contentService      *contentService.ContentService
	notificationService *contentService.NotificationService
}

// NewHandler 创建内容管理处理器
func NewHandler(
	contentSvc *contentService.ContentService,
	notificationSvc *contentService.NotificationService,
) *Handler {
	return &Handler{
		contentService:      contentSvc,
		notificationService: notificationSvc,
	}
}

// ==================== 文章管理（管理端）====================

// CreateArticle 创建文章
// @Summary 创建文章
// @Tags 内容管理-管理端
// @Accept json
// @Produce json
// @Security Bearer
// @Param request body contentService.CreateArticleRequest true "创建文章请求"
// @Success 200 {object} response.Response{data=models.Article}
// @Router /api/v1/admin/articles [post]
func (h *Handler) CreateArticle(c *gin.Context) {
	var req contentService.CreateArticleRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "参数错误: "+err.Error())
		return
	}

	article, err := h.contentService.CreateArticle(c.Request.Context(), &req)
	if err != nil {
		if appErr, ok := err.(*errors.AppError); ok {
			response.Error(c, appErr.Code, appErr.Message)
			return
		}
		response.InternalError(c, err.Error())
		return
	}

	response.Success(c, article)
}

// UpdateArticle 更新文章
// @Summary 更新文章
// @Tags 内容管理-管理端
// @Accept json
// @Produce json
// @Security Bearer
// @Param id path int true "文章ID"
// @Param request body contentService.UpdateArticleRequest true "更新文章请求"
// @Success 200 {object} response.Response{data=models.Article}
// @Router /api/v1/admin/articles/{id} [put]
func (h *Handler) UpdateArticle(c *gin.Context) {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		response.BadRequest(c, "无效的文章ID")
		return
	}

	var req contentService.UpdateArticleRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "参数错误: "+err.Error())
		return
	}

	article, err := h.contentService.UpdateArticle(c.Request.Context(), id, &req)
	if err != nil {
		if appErr, ok := err.(*errors.AppError); ok {
			response.Error(c, appErr.Code, appErr.Message)
			return
		}
		response.InternalError(c, err.Error())
		return
	}

	response.Success(c, article)
}

// GetArticle 获取文章详情（管理端）
// @Summary 获取文章详情
// @Tags 内容管理-管理端
// @Produce json
// @Security Bearer
// @Param id path int true "文章ID"
// @Success 200 {object} response.Response{data=models.Article}
// @Router /api/v1/admin/articles/{id} [get]
func (h *Handler) GetArticle(c *gin.Context) {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		response.BadRequest(c, "无效的文章ID")
		return
	}

	article, err := h.contentService.GetArticle(c.Request.Context(), id)
	if err != nil {
		if appErr, ok := err.(*errors.AppError); ok {
			response.Error(c, appErr.Code, appErr.Message)
			return
		}
		response.InternalError(c, err.Error())
		return
	}

	response.Success(c, article)
}

// DeleteArticle 删除文章
// @Summary 删除文章
// @Tags 内容管理-管理端
// @Produce json
// @Security Bearer
// @Param id path int true "文章ID"
// @Success 200 {object} response.Response
// @Router /api/v1/admin/articles/{id} [delete]
func (h *Handler) DeleteArticle(c *gin.Context) {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		response.BadRequest(c, "无效的文章ID")
		return
	}

	if err := h.contentService.DeleteArticle(c.Request.Context(), id); err != nil {
		if appErr, ok := err.(*errors.AppError); ok {
			response.Error(c, appErr.Code, appErr.Message)
			return
		}
		response.InternalError(c, err.Error())
		return
	}

	response.Success(c, nil)
}

// ListArticles 获取文章列表（管理端）
// @Summary 获取文章列表
// @Tags 内容管理-管理端
// @Produce json
// @Security Bearer
// @Param category query string false "分类"
// @Param is_published query bool false "是否发布"
// @Param keyword query string false "关键词"
// @Param page query int false "页码" default(1)
// @Param page_size query int false "每页数量" default(20)
// @Success 200 {object} response.Response{data=response.ListData{list=[]models.Article}}
// @Router /api/v1/admin/articles [get]
func (h *Handler) ListArticles(c *gin.Context) {
	var req contentService.ArticleListRequest
	if err := c.ShouldBindQuery(&req); err != nil {
		response.BadRequest(c, "参数错误: "+err.Error())
		return
	}

	articles, total, err := h.contentService.ListArticles(c.Request.Context(), &req)
	if err != nil {
		if appErr, ok := err.(*errors.AppError); ok {
			response.Error(c, appErr.Code, appErr.Message)
			return
		}
		response.InternalError(c, err.Error())
		return
	}

	response.SuccessList(c, articles, total)
}

// PublishArticle 发布文章
// @Summary 发布文章
// @Tags 内容管理-管理端
// @Produce json
// @Security Bearer
// @Param id path int true "文章ID"
// @Success 200 {object} response.Response
// @Router /api/v1/admin/articles/{id}/publish [post]
func (h *Handler) PublishArticle(c *gin.Context) {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		response.BadRequest(c, "无效的文章ID")
		return
	}

	if err := h.contentService.PublishArticle(c.Request.Context(), id); err != nil {
		if appErr, ok := err.(*errors.AppError); ok {
			response.Error(c, appErr.Code, appErr.Message)
			return
		}
		response.InternalError(c, err.Error())
		return
	}

	response.Success(c, nil)
}

// UnpublishArticle 取消发布文章
// @Summary 取消发布文章
// @Tags 内容管理-管理端
// @Produce json
// @Security Bearer
// @Param id path int true "文章ID"
// @Success 200 {object} response.Response
// @Router /api/v1/admin/articles/{id}/unpublish [post]
func (h *Handler) UnpublishArticle(c *gin.Context) {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		response.BadRequest(c, "无效的文章ID")
		return
	}

	if err := h.contentService.UnpublishArticle(c.Request.Context(), id); err != nil {
		if appErr, ok := err.(*errors.AppError); ok {
			response.Error(c, appErr.Code, appErr.Message)
			return
		}
		response.InternalError(c, err.Error())
		return
	}

	response.Success(c, nil)
}

// GetCategoryCounts 获取分类统计
// @Summary 获取分类统计
// @Tags 内容管理-管理端
// @Produce json
// @Security Bearer
// @Success 200 {object} response.Response{data=map[string]int64}
// @Router /api/v1/admin/articles/category-counts [get]
func (h *Handler) GetCategoryCounts(c *gin.Context) {
	counts, err := h.contentService.GetCategoryCounts(c.Request.Context())
	if err != nil {
		if appErr, ok := err.(*errors.AppError); ok {
			response.Error(c, appErr.Code, appErr.Message)
			return
		}
		response.InternalError(c, err.Error())
		return
	}

	response.Success(c, counts)
}

// ==================== 文章查询（用户端）====================

// GetPublishedArticle 获取已发布文章详情
// @Summary 获取已发布文章详情
// @Tags 内容管理-用户端
// @Produce json
// @Param id path int true "文章ID"
// @Success 200 {object} response.Response{data=models.Article}
// @Router /api/v1/articles/{id} [get]
func (h *Handler) GetPublishedArticle(c *gin.Context) {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		response.BadRequest(c, "无效的文章ID")
		return
	}

	article, err := h.contentService.GetArticleWithViewCount(c.Request.Context(), id)
	if err != nil {
		if appErr, ok := err.(*errors.AppError); ok {
			response.Error(c, appErr.Code, appErr.Message)
			return
		}
		response.InternalError(c, err.Error())
		return
	}

	response.Success(c, article)
}

// ListPublishedArticles 获取已发布文章列表
// @Summary 获取已发布文章列表
// @Tags 内容管理-用户端
// @Produce json
// @Param category query string false "分类"
// @Param page query int false "页码" default(1)
// @Param page_size query int false "每页数量" default(20)
// @Success 200 {object} response.Response{data=response.ListData{list=[]models.Article}}
// @Router /api/v1/articles [get]
func (h *Handler) ListPublishedArticles(c *gin.Context) {
	category := c.Query("category")
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "20"))

	articles, total, err := h.contentService.ListPublishedArticles(c.Request.Context(), category, page, pageSize)
	if err != nil {
		if appErr, ok := err.(*errors.AppError); ok {
			response.Error(c, appErr.Code, appErr.Message)
			return
		}
		response.InternalError(c, err.Error())
		return
	}

	response.SuccessList(c, articles, total)
}

// GetArticlesByCategory 按分类获取文章
// @Summary 按分类获取文章
// @Tags 内容管理-用户端
// @Produce json
// @Param category path string true "分类"
// @Param limit query int false "数量限制" default(10)
// @Success 200 {object} response.Response{data=[]models.Article}
// @Router /api/v1/articles/category/{category} [get]
func (h *Handler) GetArticlesByCategory(c *gin.Context) {
	category := c.Param("category")
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "10"))

	articles, err := h.contentService.GetArticlesByCategory(c.Request.Context(), category, limit)
	if err != nil {
		if appErr, ok := err.(*errors.AppError); ok {
			response.Error(c, appErr.Code, appErr.Message)
			return
		}
		response.InternalError(c, err.Error())
		return
	}

	response.Success(c, articles)
}

// ==================== 通知管理（管理端）====================

// CreateNotification 创建通知
// @Summary 创建通知
// @Tags 内容管理-管理端
// @Accept json
// @Produce json
// @Security Bearer
// @Param request body contentService.CreateNotificationRequest true "创建通知请求"
// @Success 200 {object} response.Response{data=models.Notification}
// @Router /api/v1/admin/notifications [post]
func (h *Handler) CreateNotification(c *gin.Context) {
	var req contentService.CreateNotificationRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "参数错误: "+err.Error())
		return
	}

	notification, err := h.notificationService.CreateNotification(c.Request.Context(), &req)
	if err != nil {
		if appErr, ok := err.(*errors.AppError); ok {
			response.Error(c, appErr.Code, appErr.Message)
			return
		}
		response.InternalError(c, err.Error())
		return
	}

	response.Success(c, notification)
}

// CreateSystemNotification 创建系统通知
// @Summary 创建系统通知（发送给所有用户）
// @Tags 内容管理-管理端
// @Accept json
// @Produce json
// @Security Bearer
// @Param request body SystemNotificationRequest true "系统通知请求"
// @Success 200 {object} response.Response
// @Router /api/v1/admin/notifications/system [post]
func (h *Handler) CreateSystemNotification(c *gin.Context) {
	var req SystemNotificationRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "参数错误: "+err.Error())
		return
	}

	if err := h.notificationService.CreateSystemNotification(c.Request.Context(), req.Title, req.Content); err != nil {
		if appErr, ok := err.(*errors.AppError); ok {
			response.Error(c, appErr.Code, appErr.Message)
			return
		}
		response.InternalError(c, err.Error())
		return
	}

	response.Success(c, nil)
}

// SystemNotificationRequest 系统通知请求
type SystemNotificationRequest struct {
	Title   string `json:"title" binding:"required"`
	Content string `json:"content" binding:"required"`
}

// BatchCreateNotifications 批量创建通知
// @Summary 批量创建通知
// @Tags 内容管理-管理端
// @Accept json
// @Produce json
// @Security Bearer
// @Param request body contentService.BatchCreateNotificationsRequest true "批量创建通知请求"
// @Success 200 {object} response.Response
// @Router /api/v1/admin/notifications/batch [post]
func (h *Handler) BatchCreateNotifications(c *gin.Context) {
	var req contentService.BatchCreateNotificationsRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "参数错误: "+err.Error())
		return
	}

	if err := h.notificationService.BatchCreateNotifications(c.Request.Context(), &req); err != nil {
		if appErr, ok := err.(*errors.AppError); ok {
			response.Error(c, appErr.Code, appErr.Message)
			return
		}
		response.InternalError(c, err.Error())
		return
	}

	response.Success(c, nil)
}

// DeleteNotification 删除通知（管理端）
// @Summary 删除通知
// @Tags 内容管理-管理端
// @Produce json
// @Security Bearer
// @Param id path int true "通知ID"
// @Success 200 {object} response.Response
// @Router /api/v1/admin/notifications/{id} [delete]
func (h *Handler) DeleteNotification(c *gin.Context) {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		response.BadRequest(c, "无效的通知ID")
		return
	}

	if err := h.notificationService.DeleteNotification(c.Request.Context(), id); err != nil {
		if appErr, ok := err.(*errors.AppError); ok {
			response.Error(c, appErr.Code, appErr.Message)
			return
		}
		response.InternalError(c, err.Error())
		return
	}

	response.Success(c, nil)
}

// ==================== 通知管理（用户端）====================

// GetUserNotifications 获取用户通知列表
// @Summary 获取用户通知列表
// @Tags 内容管理-用户端
// @Produce json
// @Security Bearer
// @Param type query string false "通知类型"
// @Param is_read query bool false "是否已读"
// @Param page query int false "页码" default(1)
// @Param page_size query int false "每页数量" default(20)
// @Success 200 {object} response.Response{data=response.ListData{list=[]models.Notification}}
// @Router /api/v1/notifications [get]
func (h *Handler) GetUserNotifications(c *gin.Context) {
	userID := middleware.GetUserID(c)
	if userID == 0 {
		response.Unauthorized(c, "请先登录")
		return
	}

	var req contentService.NotificationListRequest
	if err := c.ShouldBindQuery(&req); err != nil {
		response.BadRequest(c, "参数错误: "+err.Error())
		return
	}

	notifications, total, err := h.notificationService.ListNotifications(c.Request.Context(), userID, &req)
	if err != nil {
		if appErr, ok := err.(*errors.AppError); ok {
			response.Error(c, appErr.Code, appErr.Message)
			return
		}
		response.InternalError(c, err.Error())
		return
	}

	response.SuccessList(c, notifications, total)
}

// GetNotification 获取通知详情
// @Summary 获取通知详情
// @Tags 内容管理-用户端
// @Produce json
// @Security Bearer
// @Param id path int true "通知ID"
// @Success 200 {object} response.Response{data=models.Notification}
// @Router /api/v1/notifications/{id} [get]
func (h *Handler) GetNotification(c *gin.Context) {
	userID := middleware.GetUserID(c)
	if userID == 0 {
		response.Unauthorized(c, "请先登录")
		return
	}

	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		response.BadRequest(c, "无效的通知ID")
		return
	}

	notification, err := h.notificationService.GetNotification(c.Request.Context(), id, userID)
	if err != nil {
		if appErr, ok := err.(*errors.AppError); ok {
			response.Error(c, appErr.Code, appErr.Message)
			return
		}
		response.InternalError(c, err.Error())
		return
	}

	response.Success(c, notification)
}

// MarkNotificationAsRead 标记通知为已读
// @Summary 标记通知为已读
// @Tags 内容管理-用户端
// @Produce json
// @Security Bearer
// @Param id path int true "通知ID"
// @Success 200 {object} response.Response
// @Router /api/v1/notifications/{id}/read [post]
func (h *Handler) MarkNotificationAsRead(c *gin.Context) {
	userID := middleware.GetUserID(c)
	if userID == 0 {
		response.Unauthorized(c, "请先登录")
		return
	}

	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		response.BadRequest(c, "无效的通知ID")
		return
	}

	if err := h.notificationService.MarkAsRead(c.Request.Context(), id, userID); err != nil {
		if appErr, ok := err.(*errors.AppError); ok {
			response.Error(c, appErr.Code, appErr.Message)
			return
		}
		response.InternalError(c, err.Error())
		return
	}

	response.Success(c, nil)
}

// MarkAllNotificationsAsRead 标记所有通知为已读
// @Summary 标记所有通知为已读
// @Tags 内容管理-用户端
// @Produce json
// @Security Bearer
// @Success 200 {object} response.Response
// @Router /api/v1/notifications/read-all [post]
func (h *Handler) MarkAllNotificationsAsRead(c *gin.Context) {
	userID := middleware.GetUserID(c)
	if userID == 0 {
		response.Unauthorized(c, "请先登录")
		return
	}

	if err := h.notificationService.MarkAllAsRead(c.Request.Context(), userID); err != nil {
		if appErr, ok := err.(*errors.AppError); ok {
			response.Error(c, appErr.Code, appErr.Message)
			return
		}
		response.InternalError(c, err.Error())
		return
	}

	response.Success(c, nil)
}

// GetUnreadCount 获取未读通知数量
// @Summary 获取未读通知数量
// @Tags 内容管理-用户端
// @Produce json
// @Security Bearer
// @Success 200 {object} response.Response{data=UnreadCountResponse}
// @Router /api/v1/notifications/unread-count [get]
func (h *Handler) GetUnreadCount(c *gin.Context) {
	userID := middleware.GetUserID(c)
	if userID == 0 {
		response.Unauthorized(c, "请先登录")
		return
	}

	count, err := h.notificationService.GetUnreadCount(c.Request.Context(), userID)
	if err != nil {
		if appErr, ok := err.(*errors.AppError); ok {
			response.Error(c, appErr.Code, appErr.Message)
			return
		}
		response.InternalError(c, err.Error())
		return
	}

	response.Success(c, UnreadCountResponse{Count: count})
}

// UnreadCountResponse 未读数量响应
type UnreadCountResponse struct {
	Count int64 `json:"count"`
}

// GetNotificationSummary 获取通知摘要
// @Summary 获取通知摘要
// @Tags 内容管理-用户端
// @Produce json
// @Security Bearer
// @Success 200 {object} response.Response{data=contentService.NotificationSummary}
// @Router /api/v1/notifications/summary [get]
func (h *Handler) GetNotificationSummary(c *gin.Context) {
	userID := middleware.GetUserID(c)
	if userID == 0 {
		response.Unauthorized(c, "请先登录")
		return
	}

	summary, err := h.notificationService.GetNotificationSummary(c.Request.Context(), userID)
	if err != nil {
		if appErr, ok := err.(*errors.AppError); ok {
			response.Error(c, appErr.Code, appErr.Message)
			return
		}
		response.InternalError(c, err.Error())
		return
	}

	response.Success(c, summary)
}

// DeleteReadNotifications 删除已读通知
// @Summary 删除已读通知
// @Tags 内容管理-用户端
// @Produce json
// @Security Bearer
// @Success 200 {object} response.Response
// @Router /api/v1/notifications/read [delete]
func (h *Handler) DeleteReadNotifications(c *gin.Context) {
	userID := middleware.GetUserID(c)
	if userID == 0 {
		response.Unauthorized(c, "请先登录")
		return
	}

	if err := h.notificationService.DeleteReadNotifications(c.Request.Context(), userID); err != nil {
		if appErr, ok := err.(*errors.AppError); ok {
			response.Error(c, appErr.Code, appErr.Message)
			return
		}
		response.InternalError(c, err.Error())
		return
	}

	response.Success(c, nil)
}
