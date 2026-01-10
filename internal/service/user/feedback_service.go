// Package user 用户服务
package user

import (
	"context"
	"time"

	"github.com/dumeirei/smart-locker-backend/internal/models"
	"github.com/dumeirei/smart-locker-backend/internal/repository"
)

// FeedbackService 用户反馈服务
type FeedbackService struct {
	feedbackRepo *repository.FeedbackRepository
}

// NewFeedbackService 创建用户反馈服务
func NewFeedbackService(feedbackRepo *repository.FeedbackRepository) *FeedbackService {
	return &FeedbackService{feedbackRepo: feedbackRepo}
}

// CreateFeedbackRequest 创建反馈请求
type CreateFeedbackRequest struct {
	Type    string   `json:"type" binding:"required,oneof=suggestion bug complaint other"`
	Content string   `json:"content" binding:"required,max=2000"`
	Images  []string `json:"images"`
	Contact string   `json:"contact"`
}

// Create 创建反馈
func (s *FeedbackService) Create(ctx context.Context, userID int64, req *CreateFeedbackRequest) (*models.UserFeedback, error) {
	feedback := &models.UserFeedback{
		UserID:  userID,
		Type:    req.Type,
		Content: req.Content,
		Status:  models.FeedbackStatusPending,
	}

	if len(req.Images) > 0 {
		feedback.Images = models.JSON{"images": req.Images}
	}
	if req.Contact != "" {
		feedback.Contact = &req.Contact
	}

	if err := s.feedbackRepo.Create(ctx, feedback); err != nil {
		return nil, err
	}

	return feedback, nil
}

// GetByID 根据 ID 获取反馈
func (s *FeedbackService) GetByID(ctx context.Context, id int64) (*models.UserFeedback, error) {
	return s.feedbackRepo.GetByID(ctx, id)
}

// ListByUser 获取用户的反馈列表
func (s *FeedbackService) ListByUser(ctx context.Context, userID int64, page, pageSize int) ([]*models.UserFeedback, int64, error) {
	offset := (page - 1) * pageSize
	return s.feedbackRepo.ListByUser(ctx, userID, offset, pageSize)
}

// Delete 删除反馈（用户只能删除自己的）
func (s *FeedbackService) Delete(ctx context.Context, id, userID int64) error {
	feedback, err := s.feedbackRepo.GetByID(ctx, id)
	if err != nil {
		return err
	}
	if feedback.UserID != userID {
		return ErrNotOwner
	}
	return s.feedbackRepo.Delete(ctx, id)
}

// 错误定义
var (
	ErrNotOwner = &FeedbackError{Code: "NOT_OWNER", Message: "无权操作此反馈"}
)

// FeedbackError 反馈错误
type FeedbackError struct {
	Code    string
	Message string
}

func (e *FeedbackError) Error() string {
	return e.Message
}

// FeedbackAdminService 反馈管理服务（管理端）
type FeedbackAdminService struct {
	feedbackRepo *repository.FeedbackRepository
}

// NewFeedbackAdminService 创建反馈管理服务
func NewFeedbackAdminService(feedbackRepo *repository.FeedbackRepository) *FeedbackAdminService {
	return &FeedbackAdminService{feedbackRepo: feedbackRepo}
}

// List 获取反馈列表
func (s *FeedbackAdminService) List(ctx context.Context, page, pageSize int, feedbackType string, status *int8, startDate, endDate *time.Time) ([]*models.UserFeedback, int64, error) {
	offset := (page - 1) * pageSize
	filters := &repository.FeedbackListFilters{
		Type:      feedbackType,
		Status:    status,
		StartDate: startDate,
		EndDate:   endDate,
	}
	return s.feedbackRepo.List(ctx, offset, pageSize, filters)
}

// GetByID 获取反馈详情
func (s *FeedbackAdminService) GetByID(ctx context.Context, id int64) (*models.UserFeedback, error) {
	return s.feedbackRepo.GetByID(ctx, id)
}

// UpdateStatus 更新反馈状态
func (s *FeedbackAdminService) UpdateStatus(ctx context.Context, id int64, status int8) error {
	return s.feedbackRepo.UpdateStatus(ctx, id, status)
}

// Reply 回复反馈
func (s *FeedbackAdminService) Reply(ctx context.Context, id int64, reply string, adminID int64) error {
	return s.feedbackRepo.Reply(ctx, id, reply, adminID)
}

// GetStatistics 获取反馈统计
func (s *FeedbackAdminService) GetStatistics(ctx context.Context) (*FeedbackStatistics, error) {
	stats := &FeedbackStatistics{}

	// 按状态统计
	statusCounts, err := s.feedbackRepo.CountByStatus(ctx)
	if err != nil {
		return nil, err
	}
	stats.StatusCounts = statusCounts

	// 按类型统计
	typeCounts, err := s.feedbackRepo.CountByType(ctx)
	if err != nil {
		return nil, err
	}
	stats.TypeCounts = typeCounts

	// 待处理数量
	stats.PendingCount, _ = s.feedbackRepo.GetPendingCount(ctx)

	return stats, nil
}

// FeedbackStatistics 反馈统计
type FeedbackStatistics struct {
	PendingCount int64            `json:"pending_count"`
	StatusCounts map[int8]int64   `json:"status_counts"`
	TypeCounts   map[string]int64 `json:"type_counts"`
}
