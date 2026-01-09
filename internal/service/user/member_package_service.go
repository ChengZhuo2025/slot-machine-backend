// Package user 提供用户服务
package user

import (
	"context"
	"fmt"
	"time"

	"gorm.io/gorm"

	"github.com/dumeirei/smart-locker-backend/internal/common/errors"
	"github.com/dumeirei/smart-locker-backend/internal/common/utils"
	"github.com/dumeirei/smart-locker-backend/internal/models"
	"github.com/dumeirei/smart-locker-backend/internal/repository"
)

// MemberPackageService 会员套餐服务
type MemberPackageService struct {
	db                *gorm.DB
	userRepo          *repository.UserRepository
	memberPackageRepo *repository.MemberPackageRepository
	memberLevelRepo   *repository.MemberLevelRepository
	orderRepo         *repository.OrderRepository
	pointsSvc         *PointsService
}

// NewMemberPackageService 创建会员套餐服务
func NewMemberPackageService(
	db *gorm.DB,
	userRepo *repository.UserRepository,
	memberPackageRepo *repository.MemberPackageRepository,
	memberLevelRepo *repository.MemberLevelRepository,
	orderRepo *repository.OrderRepository,
	pointsSvc *PointsService,
) *MemberPackageService {
	return &MemberPackageService{
		db:                db,
		userRepo:          userRepo,
		memberPackageRepo: memberPackageRepo,
		memberLevelRepo:   memberLevelRepo,
		orderRepo:         orderRepo,
		pointsSvc:         pointsSvc,
	}
}

// PackageInfo 套餐信息
type PackageInfo struct {
	ID             int64                  `json:"id"`
	Name           string                 `json:"name"`
	MemberLevelID  int64                  `json:"member_level_id"`
	MemberLevel    *MemberLevelInfo       `json:"member_level,omitempty"`
	Duration       int                    `json:"duration"`
	DurationUnit   string                 `json:"duration_unit"`
	DurationText   string                 `json:"duration_text"`
	Price          float64                `json:"price"`
	OriginalPrice  *float64               `json:"original_price,omitempty"`
	GiftPoints     int                    `json:"gift_points"`
	Description    *string                `json:"description,omitempty"`
	Benefits       map[string]interface{} `json:"benefits,omitempty"`
	Sort           int                    `json:"sort"`
	IsRecommend    bool                   `json:"is_recommend"`
	Status         int8                   `json:"status"`
}

// PurchaseResult 购买结果
type PurchaseResult struct {
	OrderID       int64     `json:"order_id"`
	OrderNo       string    `json:"order_no"`
	Amount        float64   `json:"amount"`
	ExpireAt      time.Time `json:"expire_at"`
	GiftPoints    int       `json:"gift_points"`
	NewLevelID    int64     `json:"new_level_id"`
	NewLevelName  string    `json:"new_level_name"`
}

// GetActivePackages 获取所有启用的会员套餐
func (s *MemberPackageService) GetActivePackages(ctx context.Context) ([]*PackageInfo, error) {
	packages, err := s.memberPackageRepo.GetActive(ctx)
	if err != nil {
		return nil, errors.ErrDatabaseError.WithError(err)
	}

	result := make([]*PackageInfo, len(packages))
	for i, pkg := range packages {
		result[i] = s.toPackageInfo(pkg)
	}

	return result, nil
}

// GetRecommendedPackages 获取推荐的会员套餐
func (s *MemberPackageService) GetRecommendedPackages(ctx context.Context) ([]*PackageInfo, error) {
	packages, err := s.memberPackageRepo.GetRecommended(ctx)
	if err != nil {
		return nil, errors.ErrDatabaseError.WithError(err)
	}

	result := make([]*PackageInfo, len(packages))
	for i, pkg := range packages {
		result[i] = s.toPackageInfo(pkg)
	}

	return result, nil
}

// GetPackageByID 根据ID获取会员套餐
func (s *MemberPackageService) GetPackageByID(ctx context.Context, id int64) (*PackageInfo, error) {
	pkg, err := s.memberPackageRepo.GetByID(ctx, id)
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, errors.New(errors.ErrResourceNotFound.Code, "会员套餐不存在")
		}
		return nil, errors.ErrDatabaseError.WithError(err)
	}

	return s.toPackageInfo(pkg), nil
}

// GetPackagesByLevel 根据会员等级获取套餐
func (s *MemberPackageService) GetPackagesByLevel(ctx context.Context, levelID int64) ([]*PackageInfo, error) {
	packages, err := s.memberPackageRepo.GetByLevelID(ctx, levelID)
	if err != nil {
		return nil, errors.ErrDatabaseError.WithError(err)
	}

	result := make([]*PackageInfo, len(packages))
	for i, pkg := range packages {
		result[i] = s.toPackageInfo(pkg)
	}

	return result, nil
}

// PurchasePackage 购买会员套餐
func (s *MemberPackageService) PurchasePackage(ctx context.Context, userID int64, packageID int64) (*PurchaseResult, error) {
	// 获取套餐信息
	pkg, err := s.memberPackageRepo.GetByID(ctx, packageID)
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, errors.New(errors.ErrResourceNotFound.Code, "会员套餐不存在")
		}
		return nil, errors.ErrDatabaseError.WithError(err)
	}

	// 检查套餐状态
	if pkg.Status != models.MemberPackageStatusActive {
		return nil, errors.New(errors.ErrOperationFailed.Code, "该套餐已下架")
	}

	// 获取目标会员等级
	level, err := s.memberLevelRepo.GetByID(ctx, pkg.MemberLevelID)
	if err != nil {
		return nil, errors.ErrDatabaseError.WithError(err)
	}

	var result *PurchaseResult

	err = s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		// 创建订单
		orderNo := utils.GenerateOrderNo("MP")
		now := time.Now()

		// 计算有效期
		var expireAt time.Time
		switch pkg.DurationUnit {
		case models.PackageDurationUnitDay:
			expireAt = now.AddDate(0, 0, pkg.Duration)
		case models.PackageDurationUnitMonth:
			expireAt = now.AddDate(0, pkg.Duration, 0)
		case models.PackageDurationUnitYear:
			expireAt = now.AddDate(pkg.Duration, 0, 0)
		default:
			expireAt = now.AddDate(0, pkg.Duration, 0)
		}

		order := &models.Order{
			OrderNo:        orderNo,
			UserID:         userID,
			Type:           "member_package",
			OriginalAmount: pkg.Price,
			DiscountAmount: 0,
			ActualAmount:   pkg.Price,
			DepositAmount:  0,
			Status:         models.OrderStatusPaid, // 假设直接支付成功
			Remark:         utils.StringPtr(fmt.Sprintf("购买会员套餐: %s", pkg.Name)),
			PaidAt:         &now,
			CompletedAt:    &now,
		}

		if err := tx.Create(order).Error; err != nil {
			return errors.ErrDatabaseError.WithError(err)
		}

		// 创建订单项
		orderItem := &models.OrderItem{
			OrderID:      order.ID,
			ProductName:  pkg.Name,
			Price:        pkg.Price,
			Quantity:     1,
			Subtotal:     pkg.Price,
		}
		if err := tx.Create(orderItem).Error; err != nil {
			return errors.ErrDatabaseError.WithError(err)
		}

		// 更新用户会员等级
		if err := tx.Model(&models.User{}).Where("id = ?", userID).
			Update("member_level_id", pkg.MemberLevelID).Error; err != nil {
			return errors.ErrDatabaseError.WithError(err)
		}

		// 赠送积分
		if pkg.GiftPoints > 0 && s.pointsSvc != nil {
			description := fmt.Sprintf("购买%s赠送积分", pkg.Name)
			if err := s.pointsSvc.AddPointsTx(ctx, tx, userID, pkg.GiftPoints, PointsTypePackagePurchase, description, &orderNo); err != nil {
				return err
			}
		}

		result = &PurchaseResult{
			OrderID:      order.ID,
			OrderNo:      orderNo,
			Amount:       pkg.Price,
			ExpireAt:     expireAt,
			GiftPoints:   pkg.GiftPoints,
			NewLevelID:   level.ID,
			NewLevelName: level.Name,
		}

		return nil
	})

	if err != nil {
		return nil, err
	}

	return result, nil
}

// toPackageInfo 转换为套餐信息
func (s *MemberPackageService) toPackageInfo(pkg *models.MemberPackage) *PackageInfo {
	if pkg == nil {
		return nil
	}

	info := &PackageInfo{
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
	}

	if pkg.Benefits != nil {
		info.Benefits = pkg.Benefits
	}

	if pkg.MemberLevel != nil {
		info.MemberLevel = &MemberLevelInfo{
			ID:       pkg.MemberLevel.ID,
			Name:     pkg.MemberLevel.Name,
			Level:    pkg.MemberLevel.Level,
			Discount: pkg.MemberLevel.Discount,
			Icon:     pkg.MemberLevel.Icon,
		}
	}

	return info
}

// getDurationText 获取时长文本
func (s *MemberPackageService) getDurationText(duration int, unit string) string {
	switch unit {
	case models.PackageDurationUnitDay:
		return fmt.Sprintf("%d天", duration)
	case models.PackageDurationUnitMonth:
		return fmt.Sprintf("%d个月", duration)
	case models.PackageDurationUnitYear:
		return fmt.Sprintf("%d年", duration)
	default:
		return fmt.Sprintf("%d个月", duration)
	}
}
