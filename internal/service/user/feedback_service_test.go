// Package user 用户反馈服务单元测试
package user

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"

	"github.com/dumeirei/smart-locker-backend/internal/models"
	"github.com/dumeirei/smart-locker-backend/internal/repository"
)

// setupFeedbackTestDB 创建反馈测试数据库
func setupFeedbackTestDB(t *testing.T) *gorm.DB {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Silent),
	})
	require.NoError(t, err)

	// 创建必要的表
	err = db.AutoMigrate(&models.User{}, &models.UserFeedback{})
	require.NoError(t, err)

	return db
}

// setupFeedbackService 创建测试用的 FeedbackService
func setupFeedbackService(t *testing.T) (*FeedbackService, *gorm.DB) {
	db := setupFeedbackTestDB(t)
	feedbackRepo := repository.NewFeedbackRepository(db)
	service := NewFeedbackService(feedbackRepo)
	return service, db
}

// setupFeedbackAdminService 创建测试用的 FeedbackAdminService
func setupFeedbackAdminService(t *testing.T) (*FeedbackAdminService, *gorm.DB) {
	db := setupFeedbackTestDB(t)
	feedbackRepo := repository.NewFeedbackRepository(db)
	service := NewFeedbackAdminService(feedbackRepo)
	return service, db
}

// createTestUser 创建测试用户
func createTestUserForFeedback(t *testing.T, db *gorm.DB) *models.User {
	phone := "13800138000"
	user := &models.User{
		Phone:    &phone,
		Nickname: "测试用户",
		Status:   1,
	}
	err := db.Create(user).Error
	require.NoError(t, err)
	return user
}

func TestFeedbackService_Create(t *testing.T) {
	service, db := setupFeedbackService(t)
	ctx := context.Background()

	user := createTestUserForFeedback(t, db)

	t.Run("创建建议反馈", func(t *testing.T) {
		req := &CreateFeedbackRequest{
			Type:    "suggestion",
			Content: "建议增加深色模式功能",
			Contact: "13800138000",
		}

		feedback, err := service.Create(ctx, user.ID, req)
		require.NoError(t, err)
		assert.NotZero(t, feedback.ID)
		assert.Equal(t, user.ID, feedback.UserID)
		assert.Equal(t, "suggestion", feedback.Type)
		assert.Equal(t, "建议增加深色模式功能", feedback.Content)
		assert.Equal(t, int8(models.FeedbackStatusPending), feedback.Status)
		assert.NotNil(t, feedback.Contact)
		assert.Equal(t, "13800138000", *feedback.Contact)
	})

	t.Run("创建Bug反馈带图片", func(t *testing.T) {
		req := &CreateFeedbackRequest{
			Type:    "bug",
			Content: "支付页面无法加载",
			Images:  []string{"https://example.com/img1.png", "https://example.com/img2.png"},
		}

		feedback, err := service.Create(ctx, user.ID, req)
		require.NoError(t, err)
		assert.Equal(t, "bug", feedback.Type)
		assert.NotNil(t, feedback.Images)
	})

	t.Run("创建投诉反馈", func(t *testing.T) {
		req := &CreateFeedbackRequest{
			Type:    "complaint",
			Content: "服务态度差",
		}

		feedback, err := service.Create(ctx, user.ID, req)
		require.NoError(t, err)
		assert.Equal(t, "complaint", feedback.Type)
		assert.Nil(t, feedback.Contact)
	})
}

func TestFeedbackService_GetByID(t *testing.T) {
	service, db := setupFeedbackService(t)
	ctx := context.Background()

	user := createTestUserForFeedback(t, db)

	// 创建测试反馈
	feedback := &models.UserFeedback{
		UserID:  user.ID,
		Type:    "suggestion",
		Content: "测试反馈内容",
		Status:  models.FeedbackStatusPending,
	}
	db.Create(feedback)

	t.Run("获取存在的反馈", func(t *testing.T) {
		result, err := service.GetByID(ctx, feedback.ID)
		require.NoError(t, err)
		assert.Equal(t, feedback.ID, result.ID)
		assert.Equal(t, "测试反馈内容", result.Content)
	})

	t.Run("获取不存在的反馈", func(t *testing.T) {
		_, err := service.GetByID(ctx, 99999)
		assert.Error(t, err)
	})
}

func TestFeedbackService_ListByUser(t *testing.T) {
	service, db := setupFeedbackService(t)
	ctx := context.Background()

	user1 := createTestUserForFeedback(t, db)
	phone2 := "13900139000"
	user2 := &models.User{Phone: &phone2, Nickname: "用户2", Status: 1}
	db.Create(user2)

	// 创建多个反馈
	feedbacks := []*models.UserFeedback{
		{UserID: user1.ID, Type: "suggestion", Content: "反馈1", Status: 0},
		{UserID: user1.ID, Type: "bug", Content: "反馈2", Status: 0},
		{UserID: user1.ID, Type: "complaint", Content: "反馈3", Status: 0},
		{UserID: user2.ID, Type: "other", Content: "用户2的反馈", Status: 0},
	}
	for _, f := range feedbacks {
		db.Create(f)
	}

	t.Run("获取用户反馈列表", func(t *testing.T) {
		results, total, err := service.ListByUser(ctx, user1.ID, 1, 10)
		require.NoError(t, err)
		assert.Equal(t, int64(3), total)
		assert.Len(t, results, 3)
	})

	t.Run("分页获取", func(t *testing.T) {
		results, total, err := service.ListByUser(ctx, user1.ID, 1, 2)
		require.NoError(t, err)
		assert.Equal(t, int64(3), total)
		assert.Len(t, results, 2)

		// 第二页
		results2, _, err := service.ListByUser(ctx, user1.ID, 2, 2)
		require.NoError(t, err)
		assert.Len(t, results2, 1)
	})

	t.Run("获取其他用户的反馈", func(t *testing.T) {
		results, total, err := service.ListByUser(ctx, user2.ID, 1, 10)
		require.NoError(t, err)
		assert.Equal(t, int64(1), total)
		assert.Len(t, results, 1)
		assert.Equal(t, "用户2的反馈", results[0].Content)
	})
}

func TestFeedbackService_Delete(t *testing.T) {
	service, db := setupFeedbackService(t)
	ctx := context.Background()

	user1 := createTestUserForFeedback(t, db)
	phone2 := "13900139000"
	user2 := &models.User{Phone: &phone2, Nickname: "用户2", Status: 1}
	db.Create(user2)

	// 创建测试反馈
	feedback := &models.UserFeedback{
		UserID:  user1.ID,
		Type:    "suggestion",
		Content: "待删除反馈",
		Status:  0,
	}
	db.Create(feedback)

	t.Run("用户删除自己的反馈", func(t *testing.T) {
		err := service.Delete(ctx, feedback.ID, user1.ID)
		require.NoError(t, err)

		// 验证已删除
		_, err = service.GetByID(ctx, feedback.ID)
		assert.Error(t, err)
	})

	t.Run("用户无法删除他人反馈", func(t *testing.T) {
		// 创建新反馈
		anotherFeedback := &models.UserFeedback{
			UserID:  user1.ID,
			Type:    "bug",
			Content: "用户1的反馈",
			Status:  0,
		}
		db.Create(anotherFeedback)

		err := service.Delete(ctx, anotherFeedback.ID, user2.ID)
		assert.Error(t, err)
		assert.Equal(t, ErrNotOwner, err)

		// 验证未删除
		result, err := service.GetByID(ctx, anotherFeedback.ID)
		require.NoError(t, err)
		assert.NotNil(t, result)
	})

	t.Run("删除不存在的反馈", func(t *testing.T) {
		err := service.Delete(ctx, 99999, user1.ID)
		assert.Error(t, err)
	})
}

func TestFeedbackAdminService_List(t *testing.T) {
	service, db := setupFeedbackAdminService(t)
	ctx := context.Background()

	user := createTestUserForFeedback(t, db)

	// 创建多个反馈
	now := time.Now()
	yesterday := now.AddDate(0, 0, -1)
	feedbacks := []*models.UserFeedback{
		{UserID: user.ID, Type: "suggestion", Content: "建议1", Status: 0, CreatedAt: now},
		{UserID: user.ID, Type: "suggestion", Content: "建议2", Status: 1, CreatedAt: now},
		{UserID: user.ID, Type: "bug", Content: "Bug1", Status: 0, CreatedAt: yesterday},
		{UserID: user.ID, Type: "complaint", Content: "投诉1", Status: 2, CreatedAt: yesterday},
	}
	for _, f := range feedbacks {
		db.Create(f)
	}

	t.Run("获取全部反馈列表", func(t *testing.T) {
		results, total, err := service.List(ctx, 1, 10, "", nil, nil, nil)
		require.NoError(t, err)
		assert.Equal(t, int64(4), total)
		assert.Len(t, results, 4)
	})

	t.Run("按类型筛选", func(t *testing.T) {
		results, total, err := service.List(ctx, 1, 10, "suggestion", nil, nil, nil)
		require.NoError(t, err)
		assert.Equal(t, int64(2), total)
		for _, r := range results {
			assert.Equal(t, "suggestion", r.Type)
		}
	})

	t.Run("按状态筛选", func(t *testing.T) {
		status := int8(0)
		results, total, err := service.List(ctx, 1, 10, "", &status, nil, nil)
		require.NoError(t, err)
		assert.Equal(t, int64(2), total)
		for _, r := range results {
			assert.Equal(t, int8(0), r.Status)
		}
	})

	t.Run("组合筛选", func(t *testing.T) {
		status := int8(0)
		results, total, err := service.List(ctx, 1, 10, "bug", &status, nil, nil)
		require.NoError(t, err)
		assert.Equal(t, int64(1), total)
		assert.Equal(t, "Bug1", results[0].Content)
	})

	t.Run("分页", func(t *testing.T) {
		results, total, err := service.List(ctx, 1, 2, "", nil, nil, nil)
		require.NoError(t, err)
		assert.Equal(t, int64(4), total)
		assert.Len(t, results, 2)
	})
}

func TestFeedbackAdminService_UpdateStatus(t *testing.T) {
	service, db := setupFeedbackAdminService(t)
	ctx := context.Background()

	user := createTestUserForFeedback(t, db)

	feedback := &models.UserFeedback{
		UserID:  user.ID,
		Type:    "suggestion",
		Content: "测试反馈",
		Status:  models.FeedbackStatusPending,
	}
	db.Create(feedback)

	t.Run("更新状态为处理中", func(t *testing.T) {
		err := service.UpdateStatus(ctx, feedback.ID, models.FeedbackStatusProcessing)
		require.NoError(t, err)

		// 验证状态已更新
		result, _ := service.GetByID(ctx, feedback.ID)
		assert.Equal(t, int8(models.FeedbackStatusProcessing), result.Status)
	})

	t.Run("更新状态为已处理", func(t *testing.T) {
		err := service.UpdateStatus(ctx, feedback.ID, models.FeedbackStatusProcessed)
		require.NoError(t, err)

		result, _ := service.GetByID(ctx, feedback.ID)
		assert.Equal(t, int8(models.FeedbackStatusProcessed), result.Status)
	})
}

func TestFeedbackAdminService_Reply(t *testing.T) {
	service, db := setupFeedbackAdminService(t)
	ctx := context.Background()

	user := createTestUserForFeedback(t, db)
	adminID := int64(1)

	feedback := &models.UserFeedback{
		UserID:  user.ID,
		Type:    "suggestion",
		Content: "建议增加某功能",
		Status:  models.FeedbackStatusPending,
	}
	db.Create(feedback)

	t.Run("回复反馈", func(t *testing.T) {
		err := service.Reply(ctx, feedback.ID, "感谢您的建议，我们会认真考虑", adminID)
		require.NoError(t, err)

		// 验证回复已保存
		result, _ := service.GetByID(ctx, feedback.ID)
		assert.NotNil(t, result.Reply)
		assert.Equal(t, "感谢您的建议，我们会认真考虑", *result.Reply)
		assert.NotNil(t, result.RepliedBy)
		assert.Equal(t, adminID, *result.RepliedBy)
		assert.NotNil(t, result.RepliedAt)
		// 回复后状态应变为已处理
		assert.Equal(t, int8(models.FeedbackStatusProcessed), result.Status)
	})
}

func TestFeedbackAdminService_GetStatistics(t *testing.T) {
	service, db := setupFeedbackAdminService(t)
	ctx := context.Background()

	user := createTestUserForFeedback(t, db)

	// 创建不同状态和类型的反馈
	feedbacks := []*models.UserFeedback{
		{UserID: user.ID, Type: "suggestion", Content: "建议1", Status: 0},
		{UserID: user.ID, Type: "suggestion", Content: "建议2", Status: 0},
		{UserID: user.ID, Type: "bug", Content: "Bug1", Status: 1},
		{UserID: user.ID, Type: "complaint", Content: "投诉1", Status: 2},
		{UserID: user.ID, Type: "other", Content: "其他1", Status: 2},
	}
	for _, f := range feedbacks {
		db.Create(f)
	}

	stats, err := service.GetStatistics(ctx)
	require.NoError(t, err)

	t.Run("待处理数量", func(t *testing.T) {
		assert.Equal(t, int64(2), stats.PendingCount)
	})

	t.Run("状态统计", func(t *testing.T) {
		assert.Equal(t, int64(2), stats.StatusCounts[0]) // 待处理
		assert.Equal(t, int64(1), stats.StatusCounts[1]) // 处理中
		assert.Equal(t, int64(2), stats.StatusCounts[2]) // 已处理
	})

	t.Run("类型统计", func(t *testing.T) {
		assert.Equal(t, int64(2), stats.TypeCounts["suggestion"])
		assert.Equal(t, int64(1), stats.TypeCounts["bug"])
		assert.Equal(t, int64(1), stats.TypeCounts["complaint"])
		assert.Equal(t, int64(1), stats.TypeCounts["other"])
	})
}
