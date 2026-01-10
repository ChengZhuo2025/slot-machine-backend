// Package content 提供内容管理服务
package content

import (
	"context"

	"github.com/dumeirei/smart-locker-backend/internal/models"
	"github.com/dumeirei/smart-locker-backend/internal/repository"
)

// NotificationService 通知服务
type NotificationService struct {
	notificationRepo *repository.NotificationRepository
}

// NewNotificationService 创建通知服务
func NewNotificationService(notificationRepo *repository.NotificationRepository) *NotificationService {
	return &NotificationService{
		notificationRepo: notificationRepo,
	}
}

// CreateNotificationRequest 创建通知请求
type CreateNotificationRequest struct {
	UserID  *int64 `json:"user_id"` // nil 表示发送给所有用户
	Type    string `json:"type" binding:"required"`
	Title   string `json:"title" binding:"required"`
	Content string `json:"content" binding:"required"`
	Link    string `json:"link"`
}

// CreateNotification 创建通知
func (s *NotificationService) CreateNotification(ctx context.Context, req *CreateNotificationRequest) (*models.Notification, error) {
	var link *string
	if req.Link != "" {
		link = &req.Link
	}

	notification := &models.Notification{
		UserID:  req.UserID,
		Type:    req.Type,
		Title:   req.Title,
		Content: req.Content,
		Link:    link,
	}

	if err := s.notificationRepo.Create(ctx, notification); err != nil {
		return nil, err
	}

	return notification, nil
}

// CreateSystemNotification 创建系统通知（发送给所有用户）
func (s *NotificationService) CreateSystemNotification(ctx context.Context, title, content string) error {
	return s.notificationRepo.CreateSystemNotification(ctx, title, content)
}

// CreateUserNotification 创建用户通知
func (s *NotificationService) CreateUserNotification(ctx context.Context, userID int64, notificationType, title, content string, link *string) error {
	return s.notificationRepo.CreateUserNotification(ctx, userID, notificationType, title, content, link)
}

// SendOrderNotification 发送订单通知
func (s *NotificationService) SendOrderNotification(ctx context.Context, userID int64, title, content string, orderID int64) error {
	link := "/orders/" + string(rune(orderID))
	return s.notificationRepo.CreateUserNotification(ctx, userID, models.NotificationTypeOrder, title, content, &link)
}

// GetNotification 获取通知详情
func (s *NotificationService) GetNotification(ctx context.Context, id, userID int64) (*models.Notification, error) {
	return s.notificationRepo.GetByIDAndUserID(ctx, id, userID)
}

// NotificationListRequest 通知列表请求
type NotificationListRequest struct {
	Type     string `form:"type"`
	IsRead   *bool  `form:"is_read"`
	Page     int    `form:"page,default=1"`
	PageSize int    `form:"page_size,default=20"`
}

// ListNotifications 获取通知列表
func (s *NotificationService) ListNotifications(ctx context.Context, userID int64, req *NotificationListRequest) ([]*models.Notification, int64, error) {
	if req.Page < 1 {
		req.Page = 1
	}
	if req.PageSize < 1 || req.PageSize > 100 {
		req.PageSize = 20
	}

	offset := (req.Page - 1) * req.PageSize
	return s.notificationRepo.ListByUserID(ctx, userID, offset, req.PageSize, req.Type, req.IsRead)
}

// MarkAsRead 标记通知为已读
func (s *NotificationService) MarkAsRead(ctx context.Context, id, userID int64) error {
	return s.notificationRepo.MarkAsRead(ctx, id, userID)
}

// MarkAllAsRead 标记所有通知为已读
func (s *NotificationService) MarkAllAsRead(ctx context.Context, userID int64) error {
	return s.notificationRepo.MarkAllAsRead(ctx, userID)
}

// GetUnreadCount 获取未读通知数量
func (s *NotificationService) GetUnreadCount(ctx context.Context, userID int64) (int64, error) {
	return s.notificationRepo.CountUnread(ctx, userID)
}

// GetUnreadCountByType 按类型获取未读通知数量
func (s *NotificationService) GetUnreadCountByType(ctx context.Context, userID int64) (map[string]int64, error) {
	return s.notificationRepo.CountUnreadByType(ctx, userID)
}

// DeleteNotification 删除通知
func (s *NotificationService) DeleteNotification(ctx context.Context, id int64) error {
	return s.notificationRepo.Delete(ctx, id)
}

// DeleteReadNotifications 删除已读通知
func (s *NotificationService) DeleteReadNotifications(ctx context.Context, userID int64) error {
	return s.notificationRepo.DeleteRead(ctx, userID)
}

// BatchCreateNotificationsRequest 批量创建通知请求
type BatchCreateNotificationsRequest struct {
	UserIDs []int64 `json:"user_ids" binding:"required"`
	Type    string  `json:"type" binding:"required"`
	Title   string  `json:"title" binding:"required"`
	Content string  `json:"content" binding:"required"`
	Link    string  `json:"link"`
}

// BatchCreateNotifications 批量创建通知
func (s *NotificationService) BatchCreateNotifications(ctx context.Context, req *BatchCreateNotificationsRequest) error {
	var link *string
	if req.Link != "" {
		link = &req.Link
	}

	notifications := make([]*models.Notification, len(req.UserIDs))
	for i, userID := range req.UserIDs {
		uid := userID // 创建局部变量避免闭包问题
		notifications[i] = &models.Notification{
			UserID:  &uid,
			Type:    req.Type,
			Title:   req.Title,
			Content: req.Content,
			Link:    link,
		}
	}

	return s.notificationRepo.CreateBatch(ctx, notifications)
}

// NotificationSummary 通知摘要
type NotificationSummary struct {
	TotalUnread  int64            `json:"total_unread"`
	UnreadByType map[string]int64 `json:"unread_by_type"`
}

// GetNotificationSummary 获取通知摘要
func (s *NotificationService) GetNotificationSummary(ctx context.Context, userID int64) (*NotificationSummary, error) {
	totalUnread, err := s.notificationRepo.CountUnread(ctx, userID)
	if err != nil {
		return nil, err
	}

	unreadByType, err := s.notificationRepo.CountUnreadByType(ctx, userID)
	if err != nil {
		return nil, err
	}

	return &NotificationSummary{
		TotalUnread:  totalUnread,
		UnreadByType: unreadByType,
	}, nil
}
