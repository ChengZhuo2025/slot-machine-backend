// Package marketing 提供营销相关服务
package marketing

import (
	"context"
	"time"

	"gorm.io/gorm"

	"github.com/dumeirei/smart-locker-backend/internal/models"
	"github.com/dumeirei/smart-locker-backend/internal/repository"
)

// UserCouponService 用户优惠券服务
type UserCouponService struct {
	db             *gorm.DB
	couponRepo     *repository.CouponRepository
	userCouponRepo *repository.UserCouponRepository
}

// NewUserCouponService 创建用户优惠券服务
func NewUserCouponService(db *gorm.DB, couponRepo *repository.CouponRepository, userCouponRepo *repository.UserCouponRepository) *UserCouponService {
	return &UserCouponService{
		db:             db,
		couponRepo:     couponRepo,
		userCouponRepo: userCouponRepo,
	}
}

// UserCouponListRequest 用户优惠券列表请求
type UserCouponListRequest struct {
	Page     int
	PageSize int
	Status   *int8 // nil: 全部, 0: 未使用, 1: 已使用, 2: 已过期
}

// UserCouponListResponse 用户优惠券列表响应
type UserCouponListResponse struct {
	List  []*UserCouponItem `json:"list"`
	Total int64             `json:"total"`
}

// UserCouponItem 用户优惠券项
type UserCouponItem struct {
	ID              int64      `json:"id"`
	CouponID        int64      `json:"coupon_id"`
	CouponName      string     `json:"coupon_name"`
	CouponType      string     `json:"coupon_type"`
	Value           float64    `json:"value"`
	MinAmount       float64    `json:"min_amount"`
	MaxDiscount     *float64   `json:"max_discount,omitempty"`
	ApplicableScope string     `json:"applicable_scope"`
	Description     *string    `json:"description,omitempty"`
	Status          int8       `json:"status"`
	StatusText      string     `json:"status_text"`
	ExpiredAt       time.Time  `json:"expired_at"`
	ReceivedAt      time.Time  `json:"received_at"`
	UsedAt          *time.Time `json:"used_at,omitempty"`
	OrderID         *int64     `json:"order_id,omitempty"`
	IsAvailable     bool       `json:"is_available"`
	DaysRemaining   int        `json:"days_remaining"` // 剩余天数
}

// GetUserCoupons 获取用户优惠券列表
func (s *UserCouponService) GetUserCoupons(ctx context.Context, userID int64, req *UserCouponListRequest) (*UserCouponListResponse, error) {
	offset := (req.Page - 1) * req.PageSize

	params := repository.UserCouponListParams{
		UserID: userID,
		Offset: offset,
		Limit:  req.PageSize,
		Status: req.Status,
	}

	userCoupons, total, err := s.userCouponRepo.List(ctx, params)
	if err != nil {
		return nil, err
	}

	now := time.Now()
	list := make([]*UserCouponItem, 0, len(userCoupons))
	for _, uc := range userCoupons {
		item := s.buildUserCouponItem(uc, now)
		list = append(list, item)
	}

	return &UserCouponListResponse{
		List:  list,
		Total: total,
	}, nil
}

// GetAvailableCoupons 获取用户可用优惠券列表
func (s *UserCouponService) GetAvailableCoupons(ctx context.Context, userID int64, page, pageSize int) (*UserCouponListResponse, error) {
	offset := (page - 1) * pageSize

	userCoupons, total, err := s.userCouponRepo.ListAvailableByUserID(ctx, userID, offset, pageSize)
	if err != nil {
		return nil, err
	}

	now := time.Now()
	list := make([]*UserCouponItem, 0, len(userCoupons))
	for _, uc := range userCoupons {
		item := s.buildUserCouponItem(uc, now)
		list = append(list, item)
	}

	return &UserCouponListResponse{
		List:  list,
		Total: total,
	}, nil
}

// GetAvailableCouponsForOrder 获取用户可用于订单的优惠券列表
func (s *UserCouponService) GetAvailableCouponsForOrder(ctx context.Context, userID int64, orderType string, orderAmount float64) ([]*UserCouponItem, error) {
	userCoupons, err := s.userCouponRepo.ListAvailableForOrder(ctx, userID, orderType, orderAmount)
	if err != nil {
		return nil, err
	}

	now := time.Now()
	list := make([]*UserCouponItem, 0, len(userCoupons))
	for _, uc := range userCoupons {
		item := s.buildUserCouponItem(uc, now)
		if uc.Coupon != nil {
			// 计算可优惠金额
			item.Value = s.calculateDiscount(uc.Coupon, orderAmount)
		}
		list = append(list, item)
	}

	return list, nil
}

// GetUserCouponDetail 获取用户优惠券详情
func (s *UserCouponService) GetUserCouponDetail(ctx context.Context, userID, userCouponID int64) (*UserCouponItem, error) {
	userCoupon, err := s.userCouponRepo.GetByIDWithCoupon(ctx, userCouponID)
	if err != nil {
		return nil, err
	}

	if userCoupon.UserID != userID {
		return nil, ErrUserCouponNotFound
	}

	now := time.Now()
	return s.buildUserCouponItem(userCoupon, now), nil
}

// UseCoupon 使用优惠券
func (s *UserCouponService) UseCoupon(ctx context.Context, userCouponID, orderID int64, orderAmount float64) (*models.UserCoupon, float64, error) {
	var userCoupon *models.UserCoupon
	var discount float64

	err := s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		// 获取用户优惠券
		var uc models.UserCoupon
		if err := tx.Preload("Coupon").First(&uc, userCouponID).Error; err != nil {
			return err
		}
		userCoupon = &uc

		// 检查状态
		if uc.Status != models.UserCouponStatusUnused {
			return ErrUserCouponUsed
		}
		now := time.Now()
		if now.After(uc.ExpiredAt) {
			return ErrUserCouponExpired
		}

		// 检查优惠券
		if uc.Coupon == nil {
			return ErrCouponNotFound
		}
		if orderAmount < uc.Coupon.MinAmount {
			return ErrCouponAmountNotMet
		}

		// 计算优惠金额
		discount = s.calculateDiscount(uc.Coupon, orderAmount)

		// 标记为已使用
		if err := tx.Model(&models.UserCoupon{}).
			Where("id = ? AND status = ?", userCouponID, models.UserCouponStatusUnused).
			Updates(map[string]interface{}{
				"status":   models.UserCouponStatusUsed,
				"order_id": orderID,
				"used_at":  now,
			}).Error; err != nil {
			return err
		}

		// 增加优惠券使用数量
		if err := tx.Model(&models.Coupon{}).
			Where("id = ?", uc.CouponID).
			UpdateColumn("used_count", gorm.Expr("used_count + 1")).Error; err != nil {
			return err
		}

		return nil
	})

	if err != nil {
		return nil, 0, err
	}

	return userCoupon, discount, nil
}

// UnuseCoupon 取消使用优惠券（退款时）
func (s *UserCouponService) UnuseCoupon(ctx context.Context, userCouponID int64) error {
	return s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		// 获取用户优惠券
		var uc models.UserCoupon
		if err := tx.First(&uc, userCouponID).Error; err != nil {
			return err
		}

		if uc.Status != models.UserCouponStatusUsed {
			return nil // 不是已使用状态，无需恢复
		}

		// 恢复为未使用
		if err := tx.Model(&models.UserCoupon{}).
			Where("id = ?", userCouponID).
			Updates(map[string]interface{}{
				"status":   models.UserCouponStatusUnused,
				"order_id": nil,
				"used_at":  nil,
			}).Error; err != nil {
			return err
		}

		// 减少优惠券使用数量
		if err := tx.Model(&models.Coupon{}).
			Where("id = ? AND used_count > 0", uc.CouponID).
			UpdateColumn("used_count", gorm.Expr("used_count - 1")).Error; err != nil {
			return err
		}

		return nil
	})
}

// ExpireUserCoupons 过期处理用户优惠券
func (s *UserCouponService) ExpireUserCoupons(ctx context.Context) (int64, error) {
	return s.userCouponRepo.BatchMarkAsExpired(ctx)
}

// GetCouponCountByStatus 获取各状态优惠券数量
func (s *UserCouponService) GetCouponCountByStatus(ctx context.Context, userID int64) (map[string]int64, error) {
	result := make(map[string]int64)

	// 未使用
	unusedStatus := int8(models.UserCouponStatusUnused)
	_, unused, err := s.userCouponRepo.List(ctx, repository.UserCouponListParams{
		UserID: userID,
		Status: &unusedStatus,
		Limit:  1,
	})
	if err != nil {
		return nil, err
	}
	result["unused"] = unused

	// 已使用
	usedStatus := int8(models.UserCouponStatusUsed)
	_, used, err := s.userCouponRepo.List(ctx, repository.UserCouponListParams{
		UserID: userID,
		Status: &usedStatus,
		Limit:  1,
	})
	if err != nil {
		return nil, err
	}
	result["used"] = used

	// 已过期
	expiredStatus := int8(models.UserCouponStatusExpired)
	_, expired, err := s.userCouponRepo.List(ctx, repository.UserCouponListParams{
		UserID: userID,
		Status: &expiredStatus,
		Limit:  1,
	})
	if err != nil {
		return nil, err
	}
	result["expired"] = expired

	return result, nil
}

// buildUserCouponItem 构建用户优惠券项
func (s *UserCouponService) buildUserCouponItem(uc *models.UserCoupon, now time.Time) *UserCouponItem {
	item := &UserCouponItem{
		ID:         uc.ID,
		CouponID:   uc.CouponID,
		Status:     uc.Status,
		ExpiredAt:  uc.ExpiredAt,
		ReceivedAt: uc.ReceivedAt,
		UsedAt:     uc.UsedAt,
		OrderID:    uc.OrderID,
	}

	// 设置状态文本
	switch uc.Status {
	case models.UserCouponStatusUnused:
		if now.After(uc.ExpiredAt) {
			item.StatusText = "已过期"
			item.IsAvailable = false
		} else {
			item.StatusText = "未使用"
			item.IsAvailable = true
		}
	case models.UserCouponStatusUsed:
		item.StatusText = "已使用"
		item.IsAvailable = false
	case models.UserCouponStatusExpired:
		item.StatusText = "已过期"
		item.IsAvailable = false
	}

	// 计算剩余天数
	if item.IsAvailable {
		item.DaysRemaining = int(uc.ExpiredAt.Sub(now).Hours() / 24)
		if item.DaysRemaining < 0 {
			item.DaysRemaining = 0
		}
	}

	// 填充优惠券信息
	if uc.Coupon != nil {
		item.CouponName = uc.Coupon.Name
		item.CouponType = uc.Coupon.Type
		item.Value = uc.Coupon.Value
		item.MinAmount = uc.Coupon.MinAmount
		item.MaxDiscount = uc.Coupon.MaxDiscount
		item.ApplicableScope = uc.Coupon.ApplicableScope
		item.Description = uc.Coupon.Description
	}

	return item
}

// calculateDiscount 计算优惠金额
func (s *UserCouponService) calculateDiscount(coupon *models.Coupon, orderAmount float64) float64 {
	if orderAmount < coupon.MinAmount {
		return 0
	}

	var discount float64
	switch coupon.Type {
	case models.CouponTypeFixed:
		discount = coupon.Value
	case models.CouponTypePercent:
		discount = orderAmount * coupon.Value
	default:
		return 0
	}

	// 限制最大优惠金额
	if coupon.MaxDiscount != nil && discount > *coupon.MaxDiscount {
		discount = *coupon.MaxDiscount
	}

	// 优惠金额不能超过订单金额
	if discount > orderAmount {
		discount = orderAmount
	}

	return discount
}
