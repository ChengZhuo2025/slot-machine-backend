// Package repository 用户反馈仓储单元测试
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

func setupFeedbackTestDB(t *testing.T) *gorm.DB {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Silent),
	})
	require.NoError(t, err)

	err = db.AutoMigrate(&models.UserFeedback{}, &models.User{})
	require.NoError(t, err)

	return db
}

func TestFeedbackRepository_Create(t *testing.T) {
	db := setupFeedbackTestDB(t)
	repo := NewFeedbackRepository(db)
	ctx := context.Background()

	feedback := &models.UserFeedback{
		UserID:  1,
		Type:    models.FeedbackTypeSuggestion,
		Content: "建议改进",
		Images:  models.JSON{"images": []string{"img1.jpg"}},
	}

	err := repo.Create(ctx, feedback)
	require.NoError(t, err)
	assert.NotZero(t, feedback.ID)
}

func TestFeedbackRepository_GetByID(t *testing.T) {
	db := setupFeedbackTestDB(t)
	repo := NewFeedbackRepository(db)
	ctx := context.Background()

	phone := "13800138000"
	user := &models.User{
		Phone: &phone,
	}
	db.Create(user)

	feedback := &models.UserFeedback{
		UserID:  user.ID,
		Type:    models.FeedbackTypeSuggestion,
		Content: "测试反馈",
		Images:  models.JSON{"images": []string{}},
	}
	db.Create(feedback)

	found, err := repo.GetByID(ctx, feedback.ID)
	require.NoError(t, err)
	assert.Equal(t, feedback.ID, found.ID)
	assert.NotNil(t, found.User)
	assert.Equal(t, user.ID, found.User.ID)
}

func TestFeedbackRepository_Update(t *testing.T) {
	db := setupFeedbackTestDB(t)
	repo := NewFeedbackRepository(db)
	ctx := context.Background()

	feedback := &models.UserFeedback{
		UserID:  1,
		Type:    models.FeedbackTypeSuggestion,
		Content: "原内容",
		Images:  models.JSON{"images": []string{}},
	}
	db.Create(feedback)

	feedback.Content = "新内容"
	err := repo.Update(ctx, feedback)
	require.NoError(t, err)

	var found models.UserFeedback
	db.First(&found, feedback.ID)
	assert.Equal(t, "新内容", found.Content)
}

func TestFeedbackRepository_Delete(t *testing.T) {
	db := setupFeedbackTestDB(t)
	repo := NewFeedbackRepository(db)
	ctx := context.Background()

	feedback := &models.UserFeedback{
		UserID:  1,
		Type:    models.FeedbackTypeBug,
		Content: "待删除",
		Images:  models.JSON{"images": []string{}},
	}
	db.Create(feedback)

	err := repo.Delete(ctx, feedback.ID)
	require.NoError(t, err)

	var count int64
	db.Model(&models.UserFeedback{}).Where("id = ?", feedback.ID).Count(&count)
	assert.Equal(t, int64(0), count)
}

func TestFeedbackRepository_List(t *testing.T) {
	db := setupFeedbackTestDB(t)
	repo := NewFeedbackRepository(db)
	ctx := context.Background()

	db.Create(&models.UserFeedback{
		UserID: 1, Type: models.FeedbackTypeSuggestion, Content: "建议1",
		Images: models.JSON{"images": []string{}}, Status: models.FeedbackStatusPending,
	})

	db.Create(&models.UserFeedback{
		UserID: 1, Type: models.FeedbackTypeBug, Content: "问题1",
		Images: models.JSON{"images": []string{}}, Status: models.FeedbackStatusPending,
	})

	db.Model(&models.UserFeedback{}).Create(map[string]interface{}{
		"user_id": 2, "type": models.FeedbackTypeSuggestion, "content": "建议2",
		"images": models.JSON{"images": []string{}}, "status": models.FeedbackStatusProcessed,
	})

	// 获取所有反馈
	list, total, err := repo.List(ctx, 0, 10, nil)
	require.NoError(t, err)
	assert.Equal(t, int64(3), total)
	assert.Equal(t, 3, len(list))

	// 按用户过滤
	filters := &FeedbackListFilters{UserID: 1}
	list, total, err = repo.List(ctx, 0, 10, filters)
	require.NoError(t, err)
	assert.Equal(t, int64(2), total)

	// 按类型过滤
	filters = &FeedbackListFilters{Type: models.FeedbackTypeBug}
	list, total, err = repo.List(ctx, 0, 10, filters)
	require.NoError(t, err)
	assert.Equal(t, int64(1), total)

	// 按状态过滤
	status := int8(models.FeedbackStatusPending)
	filters = &FeedbackListFilters{Status: &status}
	list, total, err = repo.List(ctx, 0, 10, filters)
	require.NoError(t, err)
	assert.Equal(t, int64(2), total)
}

func TestFeedbackRepository_ListByUser(t *testing.T) {
	db := setupFeedbackTestDB(t)
	repo := NewFeedbackRepository(db)
	ctx := context.Background()

	db.Create(&models.UserFeedback{
		UserID: 1, Type: models.FeedbackTypeSuggestion, Content: "用户1反馈1",
		Images: models.JSON{"images": []string{}},
	})

	db.Create(&models.UserFeedback{
		UserID: 1, Type: models.FeedbackTypeBug, Content: "用户1反馈2",
		Images: models.JSON{"images": []string{}},
	})

	db.Create(&models.UserFeedback{
		UserID: 2, Type: models.FeedbackTypeSuggestion, Content: "用户2反馈",
		Images: models.JSON{"images": []string{}},
	})

	list, total, err := repo.ListByUser(ctx, 1, 0, 10)
	require.NoError(t, err)
	assert.Equal(t, int64(2), total)
	assert.Equal(t, 2, len(list))
}

func TestFeedbackRepository_UpdateStatus(t *testing.T) {
	db := setupFeedbackTestDB(t)
	repo := NewFeedbackRepository(db)
	ctx := context.Background()

	feedback := &models.UserFeedback{
		UserID:  1,
		Type:    models.FeedbackTypeSuggestion,
		Content: "测试",
		Images:  models.JSON{"images": []string{}},
		Status:  models.FeedbackStatusPending,
	}
	db.Create(feedback)

	err := repo.UpdateStatus(ctx, feedback.ID, models.FeedbackStatusProcessing)
	require.NoError(t, err)

	var found models.UserFeedback
	db.First(&found, feedback.ID)
	assert.Equal(t, int8(models.FeedbackStatusProcessing), found.Status)
}

func TestFeedbackRepository_Reply(t *testing.T) {
	db := setupFeedbackTestDB(t)
	repo := NewFeedbackRepository(db)
	ctx := context.Background()

	feedback := &models.UserFeedback{
		UserID:  1,
		Type:    models.FeedbackTypeSuggestion,
		Content: "测试反馈",
		Images:  models.JSON{"images": []string{}},
		Status:  models.FeedbackStatusPending,
	}
	db.Create(feedback)

	err := repo.Reply(ctx, feedback.ID, "感谢您的反馈", 100)
	require.NoError(t, err)

	var found models.UserFeedback
	db.First(&found, feedback.ID)
	assert.NotNil(t, found.Reply)
	assert.Equal(t, "感谢您的反馈", *found.Reply)
	assert.NotNil(t, found.RepliedBy)
	assert.Equal(t, int64(100), *found.RepliedBy)
	assert.NotNil(t, found.RepliedAt)
	assert.Equal(t, int8(models.FeedbackStatusProcessed), found.Status)
}

func TestFeedbackRepository_CountByStatus(t *testing.T) {
	db := setupFeedbackTestDB(t)
	repo := NewFeedbackRepository(db)
	ctx := context.Background()

	db.Create(&models.UserFeedback{
		UserID: 1, Type: models.FeedbackTypeSuggestion, Content: "待处理1",
		Images: models.JSON{"images": []string{}}, Status: models.FeedbackStatusPending,
	})

	db.Create(&models.UserFeedback{
		UserID: 1, Type: models.FeedbackTypeBug, Content: "待处理2",
		Images: models.JSON{"images": []string{}}, Status: models.FeedbackStatusPending,
	})

	db.Model(&models.UserFeedback{}).Create(map[string]interface{}{
		"user_id": 1, "type": models.FeedbackTypeSuggestion, "content": "已处理",
		"images": models.JSON{"images": []string{}}, "status": models.FeedbackStatusProcessed,
	})

	counts, err := repo.CountByStatus(ctx)
	require.NoError(t, err)
	assert.Equal(t, int64(2), counts[models.FeedbackStatusPending])
	assert.Equal(t, int64(1), counts[models.FeedbackStatusProcessed])
}

func TestFeedbackRepository_CountByType(t *testing.T) {
	db := setupFeedbackTestDB(t)
	repo := NewFeedbackRepository(db)
	ctx := context.Background()

	db.Create(&models.UserFeedback{
		UserID: 1, Type: models.FeedbackTypeSuggestion, Content: "建议1",
		Images: models.JSON{"images": []string{}},
	})

	db.Create(&models.UserFeedback{
		UserID: 1, Type: models.FeedbackTypeSuggestion, Content: "建议2",
		Images: models.JSON{"images": []string{}},
	})

	db.Create(&models.UserFeedback{
		UserID: 1, Type: models.FeedbackTypeBug, Content: "问题1",
		Images: models.JSON{"images": []string{}},
	})

	counts, err := repo.CountByType(ctx)
	require.NoError(t, err)
	assert.Equal(t, int64(2), counts[models.FeedbackTypeSuggestion])
	assert.Equal(t, int64(1), counts[models.FeedbackTypeBug])
}

func TestFeedbackRepository_GetPendingCount(t *testing.T) {
	db := setupFeedbackTestDB(t)
	repo := NewFeedbackRepository(db)
	ctx := context.Background()

	db.Create(&models.UserFeedback{
		UserID: 1, Type: models.FeedbackTypeSuggestion, Content: "待处理1",
		Images: models.JSON{"images": []string{}}, Status: models.FeedbackStatusPending,
	})

	db.Create(&models.UserFeedback{
		UserID: 1, Type: models.FeedbackTypeBug, Content: "待处理2",
		Images: models.JSON{"images": []string{}}, Status: models.FeedbackStatusPending,
	})

	db.Model(&models.UserFeedback{}).Create(map[string]interface{}{
		"user_id": 1, "type": models.FeedbackTypeSuggestion, "content": "已处理",
		"images": models.JSON{"images": []string{}}, "status": models.FeedbackStatusProcessed,
	})

	count, err := repo.GetPendingCount(ctx)
	require.NoError(t, err)
	assert.Equal(t, int64(2), count)
}
