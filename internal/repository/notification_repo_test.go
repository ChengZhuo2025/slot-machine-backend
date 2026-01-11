// Package repository 通知仓储单元测试
package repository

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"

	"github.com/dumeirei/smart-locker-backend/internal/models"
)

func setupNotificationTestDB(t *testing.T) *gorm.DB {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Silent),
	})
	require.NoError(t, err)

	err = db.AutoMigrate(&models.Notification{})
	require.NoError(t, err)

	return db
}

// int64Ptr 返回int64指针
func int64Ptr(i int64) *int64 {
	return &i
}

func TestNotificationRepository_Create(t *testing.T) {
	db := setupNotificationTestDB(t)
	repo := NewNotificationRepository(db)
	ctx := context.Background()

	userID := int64(1)
	notification := &models.Notification{
		UserID:  &userID,
		Type:    models.NotificationTypeSystem,
		Title:   "系统通知",
		Content: "这是一条系统通知",
		IsRead:  false,
	}

	err := repo.Create(ctx, notification)
	require.NoError(t, err)
	assert.NotZero(t, notification.ID)
}

func TestNotificationRepository_GetByID(t *testing.T) {
	db := setupNotificationTestDB(t)
	repo := NewNotificationRepository(db)
	ctx := context.Background()

	userID := int64(1)
	notification := &models.Notification{
		UserID:  &userID,
		Type:    models.NotificationTypeOrder,
		Title:   "订单通知",
		Content: "您的订单已发货",
		IsRead:  false,
	}
	db.Create(notification)

	found, err := repo.GetByID(ctx, notification.ID)
	require.NoError(t, err)
	assert.Equal(t, notification.ID, found.ID)
	assert.Equal(t, "订单通知", found.Title)
}

func TestNotificationRepository_GetByIDAndUserID(t *testing.T) {
	db := setupNotificationTestDB(t)
	repo := NewNotificationRepository(db)
	ctx := context.Background()

	userID := int64(100)
	notification := &models.Notification{
		UserID:  &userID,
		Type:    models.NotificationTypeSystem,
		Title:   "用户通知",
		Content: "内容",
		IsRead:  false,
	}
	db.Create(notification)

	// 正确的用户ID
	found, err := repo.GetByIDAndUserID(ctx, notification.ID, 100)
	require.NoError(t, err)
	assert.Equal(t, notification.ID, found.ID)

	// 错误的用户ID
	found, err = repo.GetByIDAndUserID(ctx, notification.ID, 999)
	assert.Error(t, err)
	assert.Nil(t, found)
}

func TestNotificationRepository_ListByUserID(t *testing.T) {
	db := setupNotificationTestDB(t)
	repo := NewNotificationRepository(db)
	ctx := context.Background()

	userID := int64(200)
	notifications := []*models.Notification{
		{UserID: &userID, Type: models.NotificationTypeSystem, Title: "通知1", Content: "内容1", IsRead: false},
		{UserID: &userID, Type: models.NotificationTypeOrder, Title: "通知2", Content: "内容2", IsRead: true},
		{UserID: &userID, Type: models.NotificationTypeMarketing, Title: "通知3", Content: "内容3", IsRead: false},
		{UserID: int64Ptr(999), Type: models.NotificationTypeSystem, Title: "其他用户", Content: "内容", IsRead: false},
	}
	for _, n := range notifications {
		db.Create(n)
	}

	list, total, err := repo.ListByUserID(ctx, userID, 0, 10, "", nil)
	require.NoError(t, err)
	assert.Equal(t, int64(3), total)
	assert.Equal(t, 3, len(list))

	// 按类型过滤
	list, total, err = repo.ListByUserID(ctx, userID, 0, 10, models.NotificationTypeSystem, nil)
	require.NoError(t, err)
	assert.Equal(t, int64(1), total)

	// 按已读状态过滤
	isRead := false
	list, total, err = repo.ListByUserID(ctx, userID, 0, 10, "", &isRead)
	require.NoError(t, err)
	assert.Equal(t, int64(2), total)
}

func TestNotificationRepository_MarkAsRead(t *testing.T) {
	db := setupNotificationTestDB(t)
	repo := NewNotificationRepository(db)
	ctx := context.Background()

	userID := int64(1)
	notification := &models.Notification{
		UserID:  &userID,
		Type:    models.NotificationTypeSystem,
		Title:   "待读通知",
		Content: "内容",
		IsRead:  false,
	}
	db.Create(notification)

	err := repo.MarkAsRead(ctx, notification.ID, userID)
	require.NoError(t, err)

	var found models.Notification
	db.First(&found, notification.ID)
	assert.True(t, found.IsRead)
	assert.NotNil(t, found.ReadAt)
}

func TestNotificationRepository_MarkAllAsRead(t *testing.T) {
	db := setupNotificationTestDB(t)
	repo := NewNotificationRepository(db)
	ctx := context.Background()

	userID := int64(400)
	notifications := []*models.Notification{
		{UserID: &userID, Type: models.NotificationTypeSystem, Title: "通知1", Content: "内容1", IsRead: false},
		{UserID: &userID, Type: models.NotificationTypeOrder, Title: "通知2", Content: "内容2", IsRead: false},
		{UserID: &userID, Type: models.NotificationTypeMarketing, Title: "通知3", Content: "内容3", IsRead: true}, // 已读
		{UserID: int64Ptr(999), Type: models.NotificationTypeSystem, Title: "其他用户", Content: "内容", IsRead: false},
	}
	for _, n := range notifications {
		db.Create(n)
	}

	err := repo.MarkAllAsRead(ctx, userID)
	require.NoError(t, err)

	// 验证该用户的所有通知都已读
	var unreadCount int64
	db.Model(&models.Notification{}).Where("user_id = ? AND is_read = ?", userID, false).Count(&unreadCount)
	assert.Equal(t, int64(0), unreadCount)

	// 验证其他用户的通知不受影响
	var otherUserNotif models.Notification
	db.Where("user_id = ?", 999).First(&otherUserNotif)
	assert.False(t, otherUserNotif.IsRead)
}

func TestNotificationRepository_Delete(t *testing.T) {
	db := setupNotificationTestDB(t)
	repo := NewNotificationRepository(db)
	ctx := context.Background()

	userID := int64(1)
	notification := &models.Notification{
		UserID:  &userID,
		Type:    models.NotificationTypeSystem,
		Title:   "待删除通知",
		Content: "内容",
		IsRead:  false,
	}
	db.Create(notification)

	err := repo.Delete(ctx, notification.ID)
	require.NoError(t, err)

	var count int64
	db.Model(&models.Notification{}).Where("id = ?", notification.ID).Count(&count)
	assert.Equal(t, int64(0), count)
}

func TestNotificationRepository_DeleteByUserID(t *testing.T) {
	db := setupNotificationTestDB(t)
	repo := NewNotificationRepository(db)
	ctx := context.Background()

	userID := int64(500)
	notifications := []*models.Notification{
		{UserID: &userID, Type: models.NotificationTypeSystem, Title: "通知1", Content: "内容1", IsRead: false},
		{UserID: &userID, Type: models.NotificationTypeOrder, Title: "通知2", Content: "内容2", IsRead: true},
		{UserID: int64Ptr(999), Type: models.NotificationTypeSystem, Title: "其他用户", Content: "内容", IsRead: false},
	}
	for _, n := range notifications {
		db.Create(n)
	}

	err := repo.DeleteByUserID(ctx, userID)
	require.NoError(t, err)

	var count int64
	db.Model(&models.Notification{}).Where("user_id = ?", userID).Count(&count)
	assert.Equal(t, int64(0), count)

	// 验证其他用户的通知不受影响
	db.Model(&models.Notification{}).Where("user_id = ?", 999).Count(&count)
	assert.Equal(t, int64(1), count)
}

func TestNotificationRepository_CountUnread(t *testing.T) {
	db := setupNotificationTestDB(t)
	repo := NewNotificationRepository(db)
	ctx := context.Background()

	userID := int64(600)
	notifications := []*models.Notification{
		{UserID: &userID, Type: models.NotificationTypeSystem, Title: "未读1", Content: "内容1", IsRead: false},
		{UserID: &userID, Type: models.NotificationTypeOrder, Title: "未读2", Content: "内容2", IsRead: false},
		{UserID: &userID, Type: models.NotificationTypeMarketing, Title: "已读1", Content: "内容3", IsRead: true},
		{UserID: int64Ptr(999), Type: models.NotificationTypeSystem, Title: "其他用户未读", Content: "内容", IsRead: false},
	}
	for _, n := range notifications {
		db.Create(n)
	}

	count, err := repo.CountUnread(ctx, userID)
	require.NoError(t, err)
	assert.Equal(t, int64(2), count)

	// 测试不存在的用户
	count, err = repo.CountUnread(ctx, 9999)
	require.NoError(t, err)
	assert.Equal(t, int64(0), count)
}

func TestNotificationRepository_DeleteRead(t *testing.T) {
	db := setupNotificationTestDB(t)
	repo := NewNotificationRepository(db)
	ctx := context.Background()

	userID := int64(700)
	notifications := []*models.Notification{
		{UserID: &userID, Type: models.NotificationTypeSystem, Title: "未读", Content: "内容1", IsRead: false},
		{UserID: &userID, Type: models.NotificationTypeOrder, Title: "已读1", Content: "内容2", IsRead: true},
		{UserID: &userID, Type: models.NotificationTypeMarketing, Title: "已读2", Content: "内容3", IsRead: true},
		{UserID: int64Ptr(999), Type: models.NotificationTypeSystem, Title: "其他用户已读", Content: "内容", IsRead: true},
	}
	for _, n := range notifications {
		db.Create(n)
	}

	err := repo.DeleteRead(ctx, userID)
	require.NoError(t, err)

	// 验证该用户的已读通知被删除
	var count int64
	db.Model(&models.Notification{}).Where("user_id = ? AND is_read = ?", userID, true).Count(&count)
	assert.Equal(t, int64(0), count)

	// 验证该用户的未读通知仍然存在
	db.Model(&models.Notification{}).Where("user_id = ? AND is_read = ?", userID, false).Count(&count)
	assert.Equal(t, int64(1), count)

	// 验证其他用户的通知不受影响
	db.Model(&models.Notification{}).Where("user_id = ?", 999).Count(&count)
	assert.Equal(t, int64(1), count)
}

func TestNotificationRepository_CreateSystemNotification(t *testing.T) {
	db := setupNotificationTestDB(t)
	repo := NewNotificationRepository(db)
	ctx := context.Background()

	err := repo.CreateSystemNotification(ctx, "系统公告", "重要系统更新通知")
	require.NoError(t, err)

	var notification models.Notification
	err = db.Where("title = ?", "系统公告").First(&notification).Error
	require.NoError(t, err)
	assert.Nil(t, notification.UserID) // 系统通知的 UserID 为 NULL
	assert.Equal(t, models.NotificationTypeSystem, notification.Type)
}

func TestNotificationRepository_CreateUserNotification(t *testing.T) {
	db := setupNotificationTestDB(t)
	repo := NewNotificationRepository(db)
	ctx := context.Background()

	userID := int64(800)
	link := "/orders/123"
	err := repo.CreateUserNotification(ctx, userID, models.NotificationTypeOrder, "订单通知", "您的订单已完成", &link)
	require.NoError(t, err)

	var notification models.Notification
	err = db.Where("title = ?", "订单通知").First(&notification).Error
	require.NoError(t, err)
	assert.NotNil(t, notification.UserID)
	assert.Equal(t, userID, *notification.UserID)
	assert.Equal(t, models.NotificationTypeOrder, notification.Type)
	assert.NotNil(t, notification.Link)
	assert.Equal(t, link, *notification.Link)
}
