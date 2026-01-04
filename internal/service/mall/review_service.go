// Package mall 提供商城服务
package mall

import (
	"context"
	"encoding/json"
	"time"

	"gorm.io/gorm"

	"github.com/dumeirei/smart-locker-backend/internal/common/errors"
	"github.com/dumeirei/smart-locker-backend/internal/models"
	"github.com/dumeirei/smart-locker-backend/internal/repository"
)

// ReviewService 评价服务
type ReviewService struct {
	db         *gorm.DB
	reviewRepo *repository.ReviewRepository
	orderRepo  *repository.OrderRepository
}

// NewReviewService 创建评价服务
func NewReviewService(
	db *gorm.DB,
	reviewRepo *repository.ReviewRepository,
	orderRepo *repository.OrderRepository,
) *ReviewService {
	return &ReviewService{
		db:         db,
		reviewRepo: reviewRepo,
		orderRepo:  orderRepo,
	}
}

// ReviewInfo 评价信息
type ReviewInfo struct {
	ID          int64    `json:"id"`
	OrderID     int64    `json:"order_id"`
	ProductID   int64    `json:"product_id"`
	UserID      int64    `json:"user_id"`
	UserName    string   `json:"user_name"`
	UserAvatar  string   `json:"user_avatar,omitempty"`
	Rating      int      `json:"rating"`
	Content     string   `json:"content"`
	Images      []string `json:"images,omitempty"`
	IsAnonymous bool     `json:"is_anonymous"`
	Reply       string   `json:"reply,omitempty"`
	RepliedAt   string   `json:"replied_at,omitempty"`
	CreatedAt   string   `json:"created_at"`
}

// ReviewListResponse 评价列表响应
type ReviewListResponse struct {
	List       []*ReviewInfo `json:"list"`
	Total      int64         `json:"total"`
	Page       int           `json:"page"`
	PageSize   int           `json:"page_size"`
	TotalPages int           `json:"total_pages"`
}

// ReviewStats 评价统计
type ReviewStats struct {
	TotalCount   int64          `json:"total_count"`
	AverageRating float64        `json:"average_rating"`
	Distribution map[int16]int64 `json:"distribution"`
}

// CreateReviewRequest 创建评价请求
type CreateReviewRequest struct {
	OrderID     int64    `json:"order_id" binding:"required"`
	ProductID   int64    `json:"product_id" binding:"required"`
	Rating      int16    `json:"rating" binding:"required,min=1,max=5"`
	Content     string   `json:"content"`
	Images      []string `json:"images"`
	IsAnonymous bool     `json:"is_anonymous"`
}

// CreateReview 创建评价
func (s *ReviewService) CreateReview(ctx context.Context, userID int64, req *CreateReviewRequest) (*ReviewInfo, error) {
	// 检查订单是否存在且属于该用户
	order, err := s.orderRepo.GetByID(ctx, req.OrderID)
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, errors.ErrResourceNotFound
		}
		return nil, errors.ErrDatabaseError.WithError(err)
	}

	if order.UserID != userID {
		return nil, errors.ErrResourceNotFound
	}

	// 检查订单状态是否已完成
	if order.Status != models.OrderStatusCompleted {
		return nil, errors.ErrOrderStatusError.WithMessage("订单未完成，无法评价")
	}

	// 检查是否已评价
	exists, err := s.reviewRepo.ExistsByOrderAndProduct(ctx, req.OrderID, req.ProductID)
	if err != nil {
		return nil, errors.ErrDatabaseError.WithError(err)
	}
	if exists {
		return nil, errors.ErrAlreadyExists.WithMessage("该商品已评价")
	}

	// 创建评价
	var imagesJSON json.RawMessage
	if len(req.Images) > 0 {
		imagesJSON, _ = json.Marshal(req.Images)
	}

	var content *string
	if req.Content != "" {
		content = &req.Content
	}

	review := &models.Review{
		OrderID:     req.OrderID,
		ProductID:   req.ProductID,
		UserID:      userID,
		Rating:      req.Rating,
		Content:     content,
		Images:      imagesJSON,
		IsAnonymous: req.IsAnonymous,
		Status:      int16(models.ReviewStatusVisible),
	}

	if err := s.reviewRepo.Create(ctx, review); err != nil {
		return nil, errors.ErrDatabaseError.WithError(err)
	}

	return s.toReviewInfo(review), nil
}

// GetProductReviews 获取商品评价列表
func (s *ReviewService) GetProductReviews(ctx context.Context, productID int64, page, pageSize int) (*ReviewListResponse, error) {
	if page == 0 {
		page = 1
	}
	if pageSize == 0 {
		pageSize = 10
	}

	offset := (page - 1) * pageSize

	reviews, total, err := s.reviewRepo.ListByProductID(ctx, productID, offset, pageSize)
	if err != nil {
		return nil, errors.ErrDatabaseError.WithError(err)
	}

	list := make([]*ReviewInfo, len(reviews))
	for i, r := range reviews {
		list[i] = s.toReviewInfo(r)
	}

	totalPages := int(total) / pageSize
	if int(total)%pageSize > 0 {
		totalPages++
	}

	return &ReviewListResponse{
		List:       list,
		Total:      total,
		Page:       page,
		PageSize:   pageSize,
		TotalPages: totalPages,
	}, nil
}

// GetUserReviews 获取用户评价列表
func (s *ReviewService) GetUserReviews(ctx context.Context, userID int64, page, pageSize int) (*ReviewListResponse, error) {
	if page == 0 {
		page = 1
	}
	if pageSize == 0 {
		pageSize = 10
	}

	offset := (page - 1) * pageSize

	reviews, total, err := s.reviewRepo.ListByUserID(ctx, userID, offset, pageSize)
	if err != nil {
		return nil, errors.ErrDatabaseError.WithError(err)
	}

	list := make([]*ReviewInfo, len(reviews))
	for i, r := range reviews {
		list[i] = s.toReviewInfo(r)
	}

	totalPages := int(total) / pageSize
	if int(total)%pageSize > 0 {
		totalPages++
	}

	return &ReviewListResponse{
		List:       list,
		Total:      total,
		Page:       page,
		PageSize:   pageSize,
		TotalPages: totalPages,
	}, nil
}

// GetProductReviewStats 获取商品评价统计
func (s *ReviewService) GetProductReviewStats(ctx context.Context, productID int64) (*ReviewStats, error) {
	totalCount, err := s.reviewRepo.CountByProductID(ctx, productID)
	if err != nil {
		return nil, errors.ErrDatabaseError.WithError(err)
	}

	avgRating, err := s.reviewRepo.GetAverageRatingByProductID(ctx, productID)
	if err != nil {
		return nil, errors.ErrDatabaseError.WithError(err)
	}

	distribution, err := s.reviewRepo.GetRatingDistribution(ctx, productID)
	if err != nil {
		return nil, errors.ErrDatabaseError.WithError(err)
	}

	return &ReviewStats{
		TotalCount:    totalCount,
		AverageRating: avgRating,
		Distribution:  distribution,
	}, nil
}

// GetReviewByID 根据ID获取评价
func (s *ReviewService) GetReviewByID(ctx context.Context, id int64) (*ReviewInfo, error) {
	review, err := s.reviewRepo.GetByIDWithUser(ctx, id)
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, errors.ErrResourceNotFound
		}
		return nil, errors.ErrDatabaseError.WithError(err)
	}

	return s.toReviewInfo(review), nil
}

// DeleteReview 删除评价（用户自己的）
func (s *ReviewService) DeleteReview(ctx context.Context, userID, reviewID int64) error {
	review, err := s.reviewRepo.GetByID(ctx, reviewID)
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return errors.ErrResourceNotFound
		}
		return errors.ErrDatabaseError.WithError(err)
	}

	if review.UserID != userID {
		return errors.ErrResourceNotFound
	}

	return s.reviewRepo.Delete(ctx, reviewID)
}

// ReplyReview 商家回复评价
func (s *ReviewService) ReplyReview(ctx context.Context, reviewID int64, reply string) error {
	now := time.Now()
	return s.reviewRepo.UpdateFields(ctx, reviewID, map[string]interface{}{
		"reply":      reply,
		"replied_at": now,
	})
}

// toReviewInfo 转换为评价信息
func (s *ReviewService) toReviewInfo(r *models.Review) *ReviewInfo {
	info := &ReviewInfo{
		ID:          r.ID,
		OrderID:     r.OrderID,
		ProductID:   r.ProductID,
		UserID:      r.UserID,
		Rating:      int(r.Rating),
		IsAnonymous: r.IsAnonymous,
		CreatedAt:   r.CreatedAt.Format("2006-01-02 15:04:05"),
	}

	if r.Content != nil {
		info.Content = *r.Content
	}
	if r.Reply != nil {
		info.Reply = *r.Reply
	}
	if r.RepliedAt != nil {
		info.RepliedAt = r.RepliedAt.Format("2006-01-02 15:04:05")
	}

	// 解析图片 JSON
	if r.Images != nil {
		_ = json.Unmarshal(r.Images, &info.Images)
	}

	// 用户信息
	if r.User != nil {
		if r.IsAnonymous {
			// 匿名显示
			info.UserName = "匿名用户"
		} else {
			info.UserName = r.User.Nickname
			if r.User.Avatar != nil {
				info.UserAvatar = *r.User.Avatar
			}
		}
	}

	return info
}
