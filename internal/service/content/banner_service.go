// Package content 内容服务
package content

import (
	"context"
	"time"

	"github.com/dumeirei/smart-locker-backend/internal/models"
	"github.com/dumeirei/smart-locker-backend/internal/repository"
)

// BannerService 轮播图服务（用户端）
type BannerService struct {
	bannerRepo *repository.BannerRepository
}

// NewBannerService 创建轮播图服务
func NewBannerService(bannerRepo *repository.BannerRepository) *BannerService {
	return &BannerService{bannerRepo: bannerRepo}
}

// BannerResponse 轮播图响应
type BannerResponse struct {
	ID        int64   `json:"id"`
	Title     string  `json:"title"`
	Image     string  `json:"image"`
	LinkType  string  `json:"link_type,omitempty"`
	LinkValue string  `json:"link_value,omitempty"`
	Position  string  `json:"position"`
}

// ListByPosition 获取指定位置的轮播图
func (s *BannerService) ListByPosition(ctx context.Context, position string, limit int) ([]*BannerResponse, error) {
	if limit <= 0 {
		limit = 10
	}

	banners, err := s.bannerRepo.ListByPosition(ctx, position, limit)
	if err != nil {
		return nil, err
	}

	results := make([]*BannerResponse, len(banners))
	for i, banner := range banners {
		results[i] = s.toBannerResponse(banner)
	}

	return results, nil
}

// RecordClick 记录点击
func (s *BannerService) RecordClick(ctx context.Context, id int64) error {
	return s.bannerRepo.IncrementClickCount(ctx, id)
}

// toBannerResponse 转换为响应
func (s *BannerService) toBannerResponse(banner *models.Banner) *BannerResponse {
	resp := &BannerResponse{
		ID:       banner.ID,
		Title:    banner.Title,
		Image:    banner.Image,
		Position: banner.Position,
	}
	if banner.LinkType != nil {
		resp.LinkType = *banner.LinkType
	}
	if banner.LinkValue != nil {
		resp.LinkValue = *banner.LinkValue
	}
	return resp
}

// BannerAdminService 轮播图管理服务（管理端）
type BannerAdminService struct {
	bannerRepo *repository.BannerRepository
}

// NewBannerAdminService 创建轮播图管理服务
func NewBannerAdminService(bannerRepo *repository.BannerRepository) *BannerAdminService {
	return &BannerAdminService{bannerRepo: bannerRepo}
}

// CreateBannerRequest 创建轮播图请求
type CreateBannerRequest struct {
	Title     string     `json:"title" binding:"required,max=100"`
	Image     string     `json:"image" binding:"required"`
	LinkType  string     `json:"link_type"`
	LinkValue string     `json:"link_value"`
	Position  string     `json:"position" binding:"required"`
	Sort      int        `json:"sort"`
	StartTime *time.Time `json:"start_time"`
	EndTime   *time.Time `json:"end_time"`
	IsActive  bool       `json:"is_active"`
}

// UpdateBannerRequest 更新轮播图请求
type UpdateBannerRequest struct {
	Title     *string    `json:"title"`
	Image     *string    `json:"image"`
	LinkType  *string    `json:"link_type"`
	LinkValue *string    `json:"link_value"`
	Position  *string    `json:"position"`
	Sort      *int       `json:"sort"`
	StartTime *time.Time `json:"start_time"`
	EndTime   *time.Time `json:"end_time"`
	IsActive  *bool      `json:"is_active"`
}

// Create 创建轮播图
func (s *BannerAdminService) Create(ctx context.Context, req *CreateBannerRequest) (*models.Banner, error) {
	banner := &models.Banner{
		Title:     req.Title,
		Image:     req.Image,
		Position:  req.Position,
		Sort:      req.Sort,
		StartTime: req.StartTime,
		EndTime:   req.EndTime,
		IsActive:  req.IsActive,
	}

	if req.LinkType != "" {
		banner.LinkType = &req.LinkType
	}
	if req.LinkValue != "" {
		banner.LinkValue = &req.LinkValue
	}

	if err := s.bannerRepo.Create(ctx, banner); err != nil {
		return nil, err
	}

	return banner, nil
}

// GetByID 根据 ID 获取轮播图
func (s *BannerAdminService) GetByID(ctx context.Context, id int64) (*models.Banner, error) {
	return s.bannerRepo.GetByID(ctx, id)
}

// Update 更新轮播图
func (s *BannerAdminService) Update(ctx context.Context, id int64, req *UpdateBannerRequest) (*models.Banner, error) {
	banner, err := s.bannerRepo.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}

	if req.Title != nil {
		banner.Title = *req.Title
	}
	if req.Image != nil {
		banner.Image = *req.Image
	}
	if req.LinkType != nil {
		banner.LinkType = req.LinkType
	}
	if req.LinkValue != nil {
		banner.LinkValue = req.LinkValue
	}
	if req.Position != nil {
		banner.Position = *req.Position
	}
	if req.Sort != nil {
		banner.Sort = *req.Sort
	}
	if req.StartTime != nil {
		banner.StartTime = req.StartTime
	}
	if req.EndTime != nil {
		banner.EndTime = req.EndTime
	}
	if req.IsActive != nil {
		banner.IsActive = *req.IsActive
	}

	if err := s.bannerRepo.Update(ctx, banner); err != nil {
		return nil, err
	}

	return banner, nil
}

// Delete 删除轮播图
func (s *BannerAdminService) Delete(ctx context.Context, id int64) error {
	return s.bannerRepo.Delete(ctx, id)
}

// List 获取轮播图列表
func (s *BannerAdminService) List(ctx context.Context, page, pageSize int, position string, isActive *bool, keyword string) ([]*models.Banner, int64, error) {
	offset := (page - 1) * pageSize
	filters := &repository.BannerListFilters{
		Position: position,
		IsActive: isActive,
		Keyword:  keyword,
	}
	return s.bannerRepo.List(ctx, offset, pageSize, filters)
}

// UpdateStatus 更新轮播图状态
func (s *BannerAdminService) UpdateStatus(ctx context.Context, id int64, isActive bool) error {
	return s.bannerRepo.UpdateStatus(ctx, id, isActive)
}

// UpdateSort 更新排序
func (s *BannerAdminService) UpdateSort(ctx context.Context, id int64, sort int) error {
	return s.bannerRepo.UpdateSort(ctx, id, sort)
}

// GetStatistics 获取轮播图统计
func (s *BannerAdminService) GetStatistics(ctx context.Context) (*BannerStatistics, error) {
	stats := &BannerStatistics{}

	// 按位置统计
	positionCounts, err := s.bannerRepo.CountByPosition(ctx)
	if err != nil {
		return nil, err
	}
	stats.PositionCounts = positionCounts

	// 活跃数量
	stats.ActiveCount, _ = s.bannerRepo.GetActiveCount(ctx, "")

	return stats, nil
}

// BannerStatistics 轮播图统计
type BannerStatistics struct {
	ActiveCount    int64            `json:"active_count"`
	PositionCounts map[string]int64 `json:"position_counts"`
}
