// Package repository 提供数据访问层
package repository

import (
	"context"
	"time"

	"gorm.io/gorm"

	"github.com/dumeirei/smart-locker-backend/internal/models"
)

// NotificationRepository 通知仓储
type NotificationRepository struct {
	db *gorm.DB
}

// NewNotificationRepository 创建通知仓储
func NewNotificationRepository(db *gorm.DB) *NotificationRepository {
	return &NotificationRepository{db: db}
}

// Create 创建通知
func (r *NotificationRepository) Create(ctx context.Context, notification *models.Notification) error {
	return r.db.WithContext(ctx).Create(notification).Error
}

// CreateBatch 批量创建通知
func (r *NotificationRepository) CreateBatch(ctx context.Context, notifications []*models.Notification) error {
	return r.db.WithContext(ctx).Create(&notifications).Error
}

// GetByID 根据 ID 获取通知
func (r *NotificationRepository) GetByID(ctx context.Context, id int64) (*models.Notification, error) {
	var notification models.Notification
	err := r.db.WithContext(ctx).First(&notification, id).Error
	if err != nil {
		return nil, err
	}
	return &notification, nil
}

// GetByIDAndUserID 根据 ID 和用户 ID 获取通知
func (r *NotificationRepository) GetByIDAndUserID(ctx context.Context, id, userID int64) (*models.Notification, error) {
	var notification models.Notification
	err := r.db.WithContext(ctx).
		Where("id = ? AND (user_id = ? OR user_id IS NULL)", id, userID).
		First(&notification).Error
	if err != nil {
		return nil, err
	}
	return &notification, nil
}

// Delete 删除通知
func (r *NotificationRepository) Delete(ctx context.Context, id int64) error {
	return r.db.WithContext(ctx).Delete(&models.Notification{}, id).Error
}

// DeleteByUserID 删除用户的所有通知
func (r *NotificationRepository) DeleteByUserID(ctx context.Context, userID int64) error {
	return r.db.WithContext(ctx).Where("user_id = ?", userID).Delete(&models.Notification{}).Error
}

// NotificationListFilters 通知列表筛选条件
type NotificationListFilters struct {
	UserID *int64
	Type   string
	IsRead *bool
}

// List 获取通知列表
func (r *NotificationRepository) List(ctx context.Context, offset, limit int, filters *NotificationListFilters) ([]*models.Notification, int64, error) {
	var notifications []*models.Notification
	var total int64

	query := r.db.WithContext(ctx).Model(&models.Notification{})

	if filters != nil {
		if filters.UserID != nil {
			query = query.Where("user_id = ? OR user_id IS NULL", *filters.UserID)
		}
		if filters.Type != "" {
			query = query.Where("type = ?", filters.Type)
		}
		if filters.IsRead != nil {
			query = query.Where("is_read = ?", *filters.IsRead)
		}
	}

	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	if err := query.Order("id DESC").Offset(offset).Limit(limit).Find(&notifications).Error; err != nil {
		return nil, 0, err
	}

	return notifications, total, nil
}

// ListByUserID 获取用户的通知列表
func (r *NotificationRepository) ListByUserID(ctx context.Context, userID int64, offset, limit int, notificationType string, isRead *bool) ([]*models.Notification, int64, error) {
	filters := &NotificationListFilters{
		UserID: &userID,
		Type:   notificationType,
		IsRead: isRead,
	}
	return r.List(ctx, offset, limit, filters)
}

// MarkAsRead 标记为已读
func (r *NotificationRepository) MarkAsRead(ctx context.Context, id, userID int64) error {
	now := time.Now()
	return r.db.WithContext(ctx).Model(&models.Notification{}).
		Where("id = ? AND (user_id = ? OR user_id IS NULL)", id, userID).
		Updates(map[string]interface{}{
			"is_read": true,
			"read_at": now,
		}).Error
}

// MarkAllAsRead 标记用户的所有通知为已读
func (r *NotificationRepository) MarkAllAsRead(ctx context.Context, userID int64) error {
	now := time.Now()
	return r.db.WithContext(ctx).Model(&models.Notification{}).
		Where("(user_id = ? OR user_id IS NULL) AND is_read = ?", userID, false).
		Updates(map[string]interface{}{
			"is_read": true,
			"read_at": now,
		}).Error
}

// CountUnread 统计未读通知数量
func (r *NotificationRepository) CountUnread(ctx context.Context, userID int64) (int64, error) {
	var count int64
	err := r.db.WithContext(ctx).Model(&models.Notification{}).
		Where("(user_id = ? OR user_id IS NULL) AND is_read = ?", userID, false).
		Count(&count).Error
	return count, err
}

// CountUnreadByType 按类型统计未读通知数量
func (r *NotificationRepository) CountUnreadByType(ctx context.Context, userID int64) (map[string]int64, error) {
	type Result struct {
		Type  string
		Count int64
	}

	var results []Result
	err := r.db.WithContext(ctx).Model(&models.Notification{}).
		Select("type, COUNT(*) as count").
		Where("(user_id = ? OR user_id IS NULL) AND is_read = ?", userID, false).
		Group("type").
		Find(&results).Error

	if err != nil {
		return nil, err
	}

	counts := make(map[string]int64)
	for _, r := range results {
		counts[r.Type] = r.Count
	}
	return counts, nil
}

// DeleteRead 删除用户的已读通知
func (r *NotificationRepository) DeleteRead(ctx context.Context, userID int64) error {
	return r.db.WithContext(ctx).
		Where("user_id = ? AND is_read = ?", userID, true).
		Delete(&models.Notification{}).Error
}

// DeleteOlderThan 删除指定时间之前的通知
func (r *NotificationRepository) DeleteOlderThan(ctx context.Context, before time.Time) error {
	return r.db.WithContext(ctx).
		Where("created_at < ?", before).
		Delete(&models.Notification{}).Error
}

// CreateSystemNotification 创建系统通知（发送给所有用户）
func (r *NotificationRepository) CreateSystemNotification(ctx context.Context, title, content string) error {
	notification := &models.Notification{
		UserID:  nil, // NULL 表示发送给所有用户
		Type:    models.NotificationTypeSystem,
		Title:   title,
		Content: content,
	}
	return r.Create(ctx, notification)
}

// CreateUserNotification 创建用户通知
func (r *NotificationRepository) CreateUserNotification(ctx context.Context, userID int64, notificationType, title, content string, link *string) error {
	notification := &models.Notification{
		UserID:  &userID,
		Type:    notificationType,
		Title:   title,
		Content: content,
		Link:    link,
	}
	return r.Create(ctx, notification)
}
