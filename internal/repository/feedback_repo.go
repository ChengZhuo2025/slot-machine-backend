// Package repository 提供数据访问层
package repository

import (
	"context"
	"time"

	"gorm.io/gorm"

	"github.com/dumeirei/smart-locker-backend/internal/models"
)

// FeedbackRepository 用户反馈仓储
type FeedbackRepository struct {
	db *gorm.DB
}

// NewFeedbackRepository 创建用户反馈仓储
func NewFeedbackRepository(db *gorm.DB) *FeedbackRepository {
	return &FeedbackRepository{db: db}
}

// Create 创建反馈
func (r *FeedbackRepository) Create(ctx context.Context, feedback *models.UserFeedback) error {
	return r.db.WithContext(ctx).Create(feedback).Error
}

// GetByID 根据 ID 获取反馈
func (r *FeedbackRepository) GetByID(ctx context.Context, id int64) (*models.UserFeedback, error) {
	var feedback models.UserFeedback
	err := r.db.WithContext(ctx).Preload("User").First(&feedback, id).Error
	if err != nil {
		return nil, err
	}
	return &feedback, nil
}

// Update 更新反馈
func (r *FeedbackRepository) Update(ctx context.Context, feedback *models.UserFeedback) error {
	return r.db.WithContext(ctx).Save(feedback).Error
}

// Delete 删除反馈
func (r *FeedbackRepository) Delete(ctx context.Context, id int64) error {
	return r.db.WithContext(ctx).Delete(&models.UserFeedback{}, id).Error
}

// FeedbackListFilters 反馈列表筛选条件
type FeedbackListFilters struct {
	UserID    int64
	Type      string
	Status    *int8
	StartDate *time.Time
	EndDate   *time.Time
}

// List 获取反馈列表
func (r *FeedbackRepository) List(ctx context.Context, offset, limit int, filters *FeedbackListFilters) ([]*models.UserFeedback, int64, error) {
	var feedbacks []*models.UserFeedback
	var total int64

	query := r.db.WithContext(ctx).Model(&models.UserFeedback{})

	if filters != nil {
		if filters.UserID > 0 {
			query = query.Where("user_id = ?", filters.UserID)
		}
		if filters.Type != "" {
			query = query.Where("type = ?", filters.Type)
		}
		if filters.Status != nil {
			query = query.Where("status = ?", *filters.Status)
		}
		if filters.StartDate != nil {
			query = query.Where("created_at >= ?", *filters.StartDate)
		}
		if filters.EndDate != nil {
			query = query.Where("created_at <= ?", *filters.EndDate)
		}
	}

	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	if err := query.Preload("User").Order("id DESC").Offset(offset).Limit(limit).Find(&feedbacks).Error; err != nil {
		return nil, 0, err
	}

	return feedbacks, total, nil
}

// ListByUser 获取用户的反馈列表
func (r *FeedbackRepository) ListByUser(ctx context.Context, userID int64, offset, limit int) ([]*models.UserFeedback, int64, error) {
	filters := &FeedbackListFilters{UserID: userID}
	return r.List(ctx, offset, limit, filters)
}

// UpdateStatus 更新反馈状态
func (r *FeedbackRepository) UpdateStatus(ctx context.Context, id int64, status int8) error {
	return r.db.WithContext(ctx).Model(&models.UserFeedback{}).
		Where("id = ?", id).
		Update("status", status).Error
}

// Reply 回复反馈
func (r *FeedbackRepository) Reply(ctx context.Context, id int64, reply string, repliedBy int64) error {
	now := time.Now()
	return r.db.WithContext(ctx).Model(&models.UserFeedback{}).
		Where("id = ?", id).
		Updates(map[string]interface{}{
			"reply":      reply,
			"replied_by": repliedBy,
			"replied_at": now,
			"status":     models.FeedbackStatusProcessed,
		}).Error
}

// CountByStatus 按状态统计反馈数量
func (r *FeedbackRepository) CountByStatus(ctx context.Context) (map[int8]int64, error) {
	type Result struct {
		Status int8
		Count  int64
	}

	var results []Result
	err := r.db.WithContext(ctx).Model(&models.UserFeedback{}).
		Select("status, COUNT(*) as count").
		Group("status").
		Find(&results).Error

	if err != nil {
		return nil, err
	}

	counts := make(map[int8]int64)
	for _, r := range results {
		counts[r.Status] = r.Count
	}
	return counts, nil
}

// CountByType 按类型统计反馈数量
func (r *FeedbackRepository) CountByType(ctx context.Context) (map[string]int64, error) {
	type Result struct {
		Type  string
		Count int64
	}

	var results []Result
	err := r.db.WithContext(ctx).Model(&models.UserFeedback{}).
		Select("type, COUNT(*) as count").
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

// GetPendingCount 获取待处理数量
func (r *FeedbackRepository) GetPendingCount(ctx context.Context) (int64, error) {
	var count int64
	err := r.db.WithContext(ctx).Model(&models.UserFeedback{}).
		Where("status = ?", models.FeedbackStatusPending).
		Count(&count).Error
	return count, err
}
