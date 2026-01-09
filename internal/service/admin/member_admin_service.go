// Package admin 提供管理端服务
package admin

import (
	"context"
	"fmt"
	"time"

	"gorm.io/gorm"

	"github.com/dumeirei/smart-locker-backend/internal/common/errors"
	"github.com/dumeirei/smart-locker-backend/internal/models"
	"github.com/dumeirei/smart-locker-backend/internal/repository"
)

// MemberAdminService 会员管理服务
type MemberAdminService struct {
	db                *gorm.DB
	memberLevelRepo   *repository.MemberLevelRepository
	memberPackageRepo *repository.MemberPackageRepository
	userRepo          *repository.UserRepository
}

// NewMemberAdminService 创建会员管理服务
func NewMemberAdminService(
	db *gorm.DB,
	memberLevelRepo *repository.MemberLevelRepository,
	memberPackageRepo *repository.MemberPackageRepository,
	userRepo *repository.UserRepository,
) *MemberAdminService {
	return &MemberAdminService{
		db:                db,
		memberLevelRepo:   memberLevelRepo,
		memberPackageRepo: memberPackageRepo,
		userRepo:          userRepo,
	}
}

// ===================== 会员等级管理 =====================

// AdminMemberLevelItem 管理端会员等级项
type AdminMemberLevelItem struct {
	ID          int64                  `json:"id"`
	Name        string                 `json:"name"`
	Level       int                    `json:"level"`
	MinPoints   int                    `json:"min_points"`
	Discount    float64                `json:"discount"`
	Benefits    map[string]interface{} `json:"benefits,omitempty"`
	Icon        *string                `json:"icon,omitempty"`
	UserCount   int64                  `json:"user_count"`
	CreatedAt   time.Time              `json:"created_at"`
}

// GetMemberLevelList 获取会员等级列表
func (s *MemberAdminService) GetMemberLevelList(ctx context.Context) ([]*AdminMemberLevelItem, error) {
	levels, err := s.memberLevelRepo.GetAll(ctx)
	if err != nil {
		return nil, errors.ErrDatabaseError.WithError(err)
	}

	result := make([]*AdminMemberLevelItem, len(levels))
	for i, level := range levels {
		userCount, _ := s.countUsersByLevel(ctx, level.ID)
		result[i] = &AdminMemberLevelItem{
			ID:        level.ID,
			Name:      level.Name,
			Level:     level.Level,
			MinPoints: level.MinPoints,
			Discount:  level.Discount,
			Icon:      level.Icon,
			UserCount: userCount,
			CreatedAt: level.CreatedAt,
		}
		if level.Benefits != nil {
			result[i].Benefits = level.Benefits
		}
	}

	return result, nil
}

// GetMemberLevelDetail 获取会员等级详情
func (s *MemberAdminService) GetMemberLevelDetail(ctx context.Context, id int64) (*AdminMemberLevelItem, error) {
	level, err := s.memberLevelRepo.GetByID(ctx, id)
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, errors.New(errors.ErrResourceNotFound.Code, "会员等级不存在")
		}
		return nil, errors.ErrDatabaseError.WithError(err)
	}

	userCount, _ := s.countUsersByLevel(ctx, level.ID)
	item := &AdminMemberLevelItem{
		ID:        level.ID,
		Name:      level.Name,
		Level:     level.Level,
		MinPoints: level.MinPoints,
		Discount:  level.Discount,
		Icon:      level.Icon,
		UserCount: userCount,
		CreatedAt: level.CreatedAt,
	}
	if level.Benefits != nil {
		item.Benefits = level.Benefits
	}

	return item, nil
}

// CreateMemberLevelRequest 创建会员等级请求
type CreateMemberLevelRequest struct {
	Name      string                 `json:"name" binding:"required"`
	Level     int                    `json:"level" binding:"required"`
	MinPoints int                    `json:"min_points"`
	Discount  float64                `json:"discount" binding:"required"`
	Benefits  map[string]interface{} `json:"benefits,omitempty"`
	Icon      *string                `json:"icon"`
}

// CreateMemberLevel 创建会员等级
func (s *MemberAdminService) CreateMemberLevel(ctx context.Context, req *CreateMemberLevelRequest) (*models.MemberLevel, error) {
	// 检查等级序号是否已存在
	existing, err := s.memberLevelRepo.GetByLevel(ctx, req.Level)
	if err == nil && existing != nil {
		return nil, errors.New(errors.ErrOperationFailed.Code, "该等级序号已存在")
	}

	level := &models.MemberLevel{
		Name:      req.Name,
		Level:     req.Level,
		MinPoints: req.MinPoints,
		Discount:  req.Discount,
		Icon:      req.Icon,
	}

	if req.Benefits != nil {
		level.Benefits = req.Benefits
	}

	if err := s.memberLevelRepo.Create(ctx, level); err != nil {
		return nil, errors.ErrDatabaseError.WithError(err)
	}

	return level, nil
}

// UpdateMemberLevelRequest 更新会员等级请求
type UpdateMemberLevelRequest struct {
	Name      *string                `json:"name"`
	MinPoints *int                   `json:"min_points"`
	Discount  *float64               `json:"discount"`
	Benefits  map[string]interface{} `json:"benefits"`
	Icon      *string                `json:"icon"`
}

// UpdateMemberLevel 更新会员等级
func (s *MemberAdminService) UpdateMemberLevel(ctx context.Context, id int64, req *UpdateMemberLevelRequest) error {
	level, err := s.memberLevelRepo.GetByID(ctx, id)
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return errors.New(errors.ErrResourceNotFound.Code, "会员等级不存在")
		}
		return errors.ErrDatabaseError.WithError(err)
	}

	if req.Name != nil {
		level.Name = *req.Name
	}
	if req.MinPoints != nil {
		level.MinPoints = *req.MinPoints
	}
	if req.Discount != nil {
		level.Discount = *req.Discount
	}
	if req.Benefits != nil {
		level.Benefits = req.Benefits
	}
	if req.Icon != nil {
		level.Icon = req.Icon
	}

	if err := s.memberLevelRepo.Update(ctx, level); err != nil {
		return errors.ErrDatabaseError.WithError(err)
	}

	return nil
}

// DeleteMemberLevel 删除会员等级
func (s *MemberAdminService) DeleteMemberLevel(ctx context.Context, id int64) error {
	// 检查是否有用户在使用该等级
	userCount, err := s.countUsersByLevel(ctx, id)
	if err != nil {
		return errors.ErrDatabaseError.WithError(err)
	}
	if userCount > 0 {
		return errors.New(errors.ErrOperationFailed.Code, "该等级下有用户，无法删除")
	}

	if err := s.memberLevelRepo.Delete(ctx, id); err != nil {
		return errors.ErrDatabaseError.WithError(err)
	}

	return nil
}

// ===================== 会员套餐管理 =====================

// AdminMemberPackageItem 管理端会员套餐项
type AdminMemberPackageItem struct {
	ID             int64                  `json:"id"`
	Name           string                 `json:"name"`
	MemberLevelID  int64                  `json:"member_level_id"`
	MemberLevel    *AdminMemberLevelItem  `json:"member_level,omitempty"`
	Duration       int                    `json:"duration"`
	DurationUnit   string                 `json:"duration_unit"`
	DurationText   string                 `json:"duration_text"`
	Price          float64                `json:"price"`
	OriginalPrice  *float64               `json:"original_price,omitempty"`
	GiftPoints     int                    `json:"gift_points"`
	GiftCouponIDs  []int64                `json:"gift_coupon_ids,omitempty"`
	Description    *string                `json:"description,omitempty"`
	Benefits       map[string]interface{} `json:"benefits,omitempty"`
	Sort           int                    `json:"sort"`
	IsRecommend    bool                   `json:"is_recommend"`
	Status         int8                   `json:"status"`
	StatusText     string                 `json:"status_text"`
	PurchaseCount  int64                  `json:"purchase_count"`
	CreatedAt      time.Time              `json:"created_at"`
}

// AdminPackageListRequest 管理端套餐列表请求
type AdminPackageListRequest struct {
	Page          int
	PageSize      int
	Status        *int8
	MemberLevelID *int64
	IsRecommend   *bool
}

// AdminPackageListResponse 管理端套餐列表响应
type AdminPackageListResponse struct {
	List  []*AdminMemberPackageItem `json:"list"`
	Total int64                     `json:"total"`
}

// GetMemberPackageList 获取会员套餐列表
func (s *MemberAdminService) GetMemberPackageList(ctx context.Context, req *AdminPackageListRequest) (*AdminPackageListResponse, error) {
	offset := (req.Page - 1) * req.PageSize

	filters := make(map[string]interface{})
	if req.Status != nil {
		filters["status"] = *req.Status
	}
	if req.MemberLevelID != nil {
		filters["member_level_id"] = *req.MemberLevelID
	}
	if req.IsRecommend != nil {
		filters["is_recommend"] = *req.IsRecommend
	}

	packages, total, err := s.memberPackageRepo.List(ctx, offset, req.PageSize, filters)
	if err != nil {
		return nil, errors.ErrDatabaseError.WithError(err)
	}

	list := make([]*AdminMemberPackageItem, len(packages))
	for i, pkg := range packages {
		list[i] = s.buildAdminPackageItem(pkg)
	}

	return &AdminPackageListResponse{
		List:  list,
		Total: total,
	}, nil
}

// GetMemberPackageDetail 获取会员套餐详情
func (s *MemberAdminService) GetMemberPackageDetail(ctx context.Context, id int64) (*AdminMemberPackageItem, error) {
	pkg, err := s.memberPackageRepo.GetByID(ctx, id)
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, errors.New(errors.ErrResourceNotFound.Code, "会员套餐不存在")
		}
		return nil, errors.ErrDatabaseError.WithError(err)
	}

	return s.buildAdminPackageItem(pkg), nil
}

// CreateMemberPackageRequest 创建会员套餐请求
type CreateMemberPackageRequest struct {
	Name          string                 `json:"name" binding:"required"`
	MemberLevelID int64                  `json:"member_level_id" binding:"required"`
	Duration      int                    `json:"duration" binding:"required"`
	DurationUnit  string                 `json:"duration_unit" binding:"required,oneof=day month year"`
	Price         float64                `json:"price" binding:"required"`
	OriginalPrice *float64               `json:"original_price"`
	GiftPoints    int                    `json:"gift_points"`
	GiftCouponIDs []int64                `json:"gift_coupon_ids"`
	Description   *string                `json:"description"`
	Benefits      map[string]interface{} `json:"benefits"`
	Sort          int                    `json:"sort"`
	IsRecommend   bool                   `json:"is_recommend"`
}

// CreateMemberPackage 创建会员套餐
func (s *MemberAdminService) CreateMemberPackage(ctx context.Context, req *CreateMemberPackageRequest) (*models.MemberPackage, error) {
	// 验证会员等级是否存在
	_, err := s.memberLevelRepo.GetByID(ctx, req.MemberLevelID)
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, errors.New(errors.ErrInvalidParams.Code, "会员等级不存在")
		}
		return nil, errors.ErrDatabaseError.WithError(err)
	}

	pkg := &models.MemberPackage{
		Name:          req.Name,
		MemberLevelID: req.MemberLevelID,
		Duration:      req.Duration,
		DurationUnit:  req.DurationUnit,
		Price:         req.Price,
		OriginalPrice: req.OriginalPrice,
		GiftPoints:    req.GiftPoints,
		Description:   req.Description,
		Sort:          req.Sort,
		IsRecommend:   req.IsRecommend,
		Status:        models.MemberPackageStatusActive,
	}

	if req.Benefits != nil {
		pkg.Benefits = req.Benefits
	}

	if len(req.GiftCouponIDs) > 0 {
		giftCouponIDsMap := make(models.JSON)
		giftCouponIDsMap["ids"] = req.GiftCouponIDs
		pkg.GiftCouponIDs = giftCouponIDsMap
	}

	if err := s.memberPackageRepo.Create(ctx, pkg); err != nil {
		return nil, errors.ErrDatabaseError.WithError(err)
	}

	return pkg, nil
}

// UpdateMemberPackageRequest 更新会员套餐请求
type UpdateMemberPackageRequest struct {
	Name          *string                `json:"name"`
	MemberLevelID *int64                 `json:"member_level_id"`
	Duration      *int                   `json:"duration"`
	DurationUnit  *string                `json:"duration_unit"`
	Price         *float64               `json:"price"`
	OriginalPrice *float64               `json:"original_price"`
	GiftPoints    *int                   `json:"gift_points"`
	GiftCouponIDs []int64                `json:"gift_coupon_ids"`
	Description   *string                `json:"description"`
	Benefits      map[string]interface{} `json:"benefits"`
	Sort          *int                   `json:"sort"`
	IsRecommend   *bool                  `json:"is_recommend"`
}

// UpdateMemberPackage 更新会员套餐
func (s *MemberAdminService) UpdateMemberPackage(ctx context.Context, id int64, req *UpdateMemberPackageRequest) error {
	pkg, err := s.memberPackageRepo.GetByID(ctx, id)
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return errors.New(errors.ErrResourceNotFound.Code, "会员套餐不存在")
		}
		return errors.ErrDatabaseError.WithError(err)
	}

	if req.Name != nil {
		pkg.Name = *req.Name
	}
	if req.MemberLevelID != nil {
		// 验证会员等级是否存在
		_, err := s.memberLevelRepo.GetByID(ctx, *req.MemberLevelID)
		if err != nil {
			if err == gorm.ErrRecordNotFound {
				return errors.New(errors.ErrInvalidParams.Code, "会员等级不存在")
			}
			return errors.ErrDatabaseError.WithError(err)
		}
		pkg.MemberLevelID = *req.MemberLevelID
	}
	if req.Duration != nil {
		pkg.Duration = *req.Duration
	}
	if req.DurationUnit != nil {
		pkg.DurationUnit = *req.DurationUnit
	}
	if req.Price != nil {
		pkg.Price = *req.Price
	}
	if req.OriginalPrice != nil {
		pkg.OriginalPrice = req.OriginalPrice
	}
	if req.GiftPoints != nil {
		pkg.GiftPoints = *req.GiftPoints
	}
	if req.GiftCouponIDs != nil {
		giftCouponIDsMap := make(models.JSON)
		giftCouponIDsMap["ids"] = req.GiftCouponIDs
		pkg.GiftCouponIDs = giftCouponIDsMap
	}
	if req.Description != nil {
		pkg.Description = req.Description
	}
	if req.Benefits != nil {
		pkg.Benefits = req.Benefits
	}
	if req.Sort != nil {
		pkg.Sort = *req.Sort
	}
	if req.IsRecommend != nil {
		pkg.IsRecommend = *req.IsRecommend
	}

	if err := s.memberPackageRepo.Update(ctx, pkg); err != nil {
		return errors.ErrDatabaseError.WithError(err)
	}

	return nil
}

// UpdateMemberPackageStatus 更新套餐状态
func (s *MemberAdminService) UpdateMemberPackageStatus(ctx context.Context, id int64, status int8) error {
	if err := s.memberPackageRepo.UpdateStatus(ctx, id, status); err != nil {
		return errors.ErrDatabaseError.WithError(err)
	}
	return nil
}

// DeleteMemberPackage 删除会员套餐
func (s *MemberAdminService) DeleteMemberPackage(ctx context.Context, id int64) error {
	if err := s.memberPackageRepo.Delete(ctx, id); err != nil {
		return errors.ErrDatabaseError.WithError(err)
	}
	return nil
}

// ===================== 会员统计 =====================

// MemberStats 会员统计信息
type MemberStats struct {
	TotalUsers          int64            `json:"total_users"`
	LevelDistribution   []*LevelStatItem `json:"level_distribution"`
	TotalPackageSales   int64            `json:"total_package_sales"`
	TotalPackageRevenue float64          `json:"total_package_revenue"`
}

// LevelStatItem 等级统计项
type LevelStatItem struct {
	LevelID   int64   `json:"level_id"`
	LevelName string  `json:"level_name"`
	UserCount int64   `json:"user_count"`
	Percent   float64 `json:"percent"`
}

// GetMemberStats 获取会员统计信息
func (s *MemberAdminService) GetMemberStats(ctx context.Context) (*MemberStats, error) {
	var totalUsers int64
	if err := s.db.WithContext(ctx).Model(&models.User{}).Count(&totalUsers).Error; err != nil {
		return nil, errors.ErrDatabaseError.WithError(err)
	}

	levels, err := s.memberLevelRepo.GetAll(ctx)
	if err != nil {
		return nil, errors.ErrDatabaseError.WithError(err)
	}

	levelDistribution := make([]*LevelStatItem, len(levels))
	for i, level := range levels {
		userCount, _ := s.countUsersByLevel(ctx, level.ID)
		percent := 0.0
		if totalUsers > 0 {
			percent = float64(userCount) / float64(totalUsers) * 100
		}
		levelDistribution[i] = &LevelStatItem{
			LevelID:   level.ID,
			LevelName: level.Name,
			UserCount: userCount,
			Percent:   percent,
		}
	}

	// 统计套餐销售（简化，实际应从订单表统计）
	var packageSales int64
	var packageRevenue float64
	s.db.WithContext(ctx).Model(&models.Order{}).
		Where("type = ?", "member_package").
		Where("status = ?", models.OrderStatusCompleted).
		Count(&packageSales)
	s.db.WithContext(ctx).Model(&models.Order{}).
		Where("type = ?", "member_package").
		Where("status = ?", models.OrderStatusCompleted).
		Select("COALESCE(SUM(actual_amount), 0)").
		Scan(&packageRevenue)

	return &MemberStats{
		TotalUsers:          totalUsers,
		LevelDistribution:   levelDistribution,
		TotalPackageSales:   packageSales,
		TotalPackageRevenue: packageRevenue,
	}, nil
}

// ===================== 辅助方法 =====================

// countUsersByLevel 统计某等级的用户数量
func (s *MemberAdminService) countUsersByLevel(ctx context.Context, levelID int64) (int64, error) {
	var count int64
	err := s.db.WithContext(ctx).Model(&models.User{}).Where("member_level_id = ?", levelID).Count(&count).Error
	return count, err
}

// buildAdminPackageItem 构建管理端套餐项
func (s *MemberAdminService) buildAdminPackageItem(pkg *models.MemberPackage) *AdminMemberPackageItem {
	item := &AdminMemberPackageItem{
		ID:            pkg.ID,
		Name:          pkg.Name,
		MemberLevelID: pkg.MemberLevelID,
		Duration:      pkg.Duration,
		DurationUnit:  pkg.DurationUnit,
		DurationText:  s.getDurationText(pkg.Duration, pkg.DurationUnit),
		Price:         pkg.Price,
		OriginalPrice: pkg.OriginalPrice,
		GiftPoints:    pkg.GiftPoints,
		Description:   pkg.Description,
		Sort:          pkg.Sort,
		IsRecommend:   pkg.IsRecommend,
		Status:        pkg.Status,
		CreatedAt:     pkg.CreatedAt,
	}

	// 设置状态文本
	if pkg.Status == models.MemberPackageStatusActive {
		item.StatusText = "启用"
	} else {
		item.StatusText = "禁用"
	}

	// 处理会员等级
	if pkg.MemberLevel != nil {
		item.MemberLevel = &AdminMemberLevelItem{
			ID:        pkg.MemberLevel.ID,
			Name:      pkg.MemberLevel.Name,
			Level:     pkg.MemberLevel.Level,
			MinPoints: pkg.MemberLevel.MinPoints,
			Discount:  pkg.MemberLevel.Discount,
		}
	}

	// 处理权益
	if pkg.Benefits != nil {
		item.Benefits = pkg.Benefits
	}

	// 处理赠送优惠券ID
	if pkg.GiftCouponIDs != nil {
		if ids, ok := pkg.GiftCouponIDs["ids"]; ok {
			if idsSlice, ok := ids.([]interface{}); ok {
				for _, id := range idsSlice {
					if idFloat, ok := id.(float64); ok {
						item.GiftCouponIDs = append(item.GiftCouponIDs, int64(idFloat))
					}
				}
			}
		}
	}

	return item
}

// getDurationText 获取时长文本
func (s *MemberAdminService) getDurationText(duration int, unit string) string {
	switch unit {
	case models.PackageDurationUnitDay:
		return formatDuration(duration, "天")
	case models.PackageDurationUnitMonth:
		return formatDuration(duration, "个月")
	case models.PackageDurationUnitYear:
		return formatDuration(duration, "年")
	default:
		return formatDuration(duration, "个月")
	}
}

// formatDuration 格式化时长
func formatDuration(value int, unit string) string {
	return fmt.Sprintf("%d%s", value, unit)
}
