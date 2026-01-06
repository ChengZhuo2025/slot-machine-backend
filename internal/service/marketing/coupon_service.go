// Package marketing 提供营销相关服务
package marketing

import (
	"context"
	"time"

	"gorm.io/gorm"

	"github.com/dumeirei/smart-locker-backend/internal/models"
	"github.com/dumeirei/smart-locker-backend/internal/repository"
)

// CouponService 优惠券服务
type CouponService struct {
	db             *gorm.DB
	couponRepo     *repository.CouponRepository
	userCouponRepo *repository.UserCouponRepository
}

// NewCouponService 创建优惠券服务
func NewCouponService(db *gorm.DB, couponRepo *repository.CouponRepository, userCouponRepo *repository.UserCouponRepository) *CouponService {
	return &CouponService{
		db:             db,
		couponRepo:     couponRepo,
		userCouponRepo: userCouponRepo,
	}
}

// CouponListRequest 优惠券列表请求
type CouponListRequest struct {
	Page     int
	PageSize int
}

// CouponListResponse 优惠券列表响应
type CouponListResponse struct {
	List  []*CouponItem `json:"list"`
	Total int64         `json:"total"`
}

// CouponItem 优惠券项
type CouponItem struct {
	ID              int64      `json:"id"`
	Name            string     `json:"name"`
	Type            string     `json:"type"`
	Value           float64    `json:"value"`
	MinAmount       float64    `json:"min_amount"`
	MaxDiscount     *float64   `json:"max_discount,omitempty"`
	ApplicableScope string     `json:"applicable_scope"`
	StartTime       time.Time  `json:"start_time"`
	EndTime         time.Time  `json:"end_time"`
	TotalCount      int        `json:"total_count"`
	ReceivedCount   int        `json:"received_count"`
	RemainCount     int        `json:"remain_count"`
	PerUserLimit    int        `json:"per_user_limit"`
	Description     *string    `json:"description,omitempty"`
	Status          int8       `json:"status"`
	CanReceive      bool       `json:"can_receive"`
	ReceivedByUser  int64      `json:"received_by_user,omitempty"` // 当前用户已领取数量
}

// GetCouponList 获取可领取的优惠券列表（用户端）
func (s *CouponService) GetCouponList(ctx context.Context, req *CouponListRequest, userID int64) (*CouponListResponse, error) {
	offset := (req.Page - 1) * req.PageSize

	coupons, total, err := s.couponRepo.ListActive(ctx, offset, req.PageSize)
	if err != nil {
		return nil, err
	}

	list := make([]*CouponItem, 0, len(coupons))
	for _, c := range coupons {
		// 获取用户已领取数量
		receivedCount, _ := s.userCouponRepo.CountByUserIDAndCouponID(ctx, userID, c.ID)
		canReceive := receivedCount < int64(c.PerUserLimit) && c.ReceivedCount < c.TotalCount

		item := &CouponItem{
			ID:              c.ID,
			Name:            c.Name,
			Type:            c.Type,
			Value:           c.Value,
			MinAmount:       c.MinAmount,
			MaxDiscount:     c.MaxDiscount,
			ApplicableScope: c.ApplicableScope,
			StartTime:       c.StartTime,
			EndTime:         c.EndTime,
			TotalCount:      c.TotalCount,
			ReceivedCount:   c.ReceivedCount,
			RemainCount:     c.TotalCount - c.ReceivedCount,
			PerUserLimit:    c.PerUserLimit,
			Description:     c.Description,
			Status:          c.Status,
			CanReceive:      canReceive,
			ReceivedByUser:  receivedCount,
		}
		list = append(list, item)
	}

	return &CouponListResponse{
		List:  list,
		Total: total,
	}, nil
}

// GetCouponDetail 获取优惠券详情
func (s *CouponService) GetCouponDetail(ctx context.Context, couponID int64, userID int64) (*CouponItem, error) {
	coupon, err := s.couponRepo.GetByID(ctx, couponID)
	if err != nil {
		return nil, err
	}

	// 获取用户已领取数量
	receivedCount, _ := s.userCouponRepo.CountByUserIDAndCouponID(ctx, userID, couponID)
	canReceive := receivedCount < int64(coupon.PerUserLimit) && coupon.ReceivedCount < coupon.TotalCount

	now := time.Now()
	if coupon.Status != models.CouponStatusActive ||
		now.Before(coupon.StartTime) ||
		now.After(coupon.EndTime) {
		canReceive = false
	}

	return &CouponItem{
		ID:              coupon.ID,
		Name:            coupon.Name,
		Type:            coupon.Type,
		Value:           coupon.Value,
		MinAmount:       coupon.MinAmount,
		MaxDiscount:     coupon.MaxDiscount,
		ApplicableScope: coupon.ApplicableScope,
		StartTime:       coupon.StartTime,
		EndTime:         coupon.EndTime,
		TotalCount:      coupon.TotalCount,
		ReceivedCount:   coupon.ReceivedCount,
		RemainCount:     coupon.TotalCount - coupon.ReceivedCount,
		PerUserLimit:    coupon.PerUserLimit,
		Description:     coupon.Description,
		Status:          coupon.Status,
		CanReceive:      canReceive,
		ReceivedByUser:  receivedCount,
	}, nil
}

// ReceiveCoupon 领取优惠券
func (s *CouponService) ReceiveCoupon(ctx context.Context, couponID, userID int64) (*models.UserCoupon, error) {
	var userCoupon *models.UserCoupon

	err := s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		// 获取优惠券
		var coupon models.Coupon
		if err := tx.First(&coupon, couponID).Error; err != nil {
			return err
		}

		// 检查优惠券状态
		now := time.Now()
		if coupon.Status != models.CouponStatusActive {
			return ErrCouponNotActive
		}
		if now.Before(coupon.StartTime) {
			return ErrCouponNotStarted
		}
		if now.After(coupon.EndTime) {
			return ErrCouponExpired
		}
		if coupon.ReceivedCount >= coupon.TotalCount {
			return ErrCouponSoldOut
		}

		// 检查用户领取数量
		var receivedCount int64
		if err := tx.Model(&models.UserCoupon{}).
			Where("user_id = ? AND coupon_id = ?", userID, couponID).
			Count(&receivedCount).Error; err != nil {
			return err
		}
		if receivedCount >= int64(coupon.PerUserLimit) {
			return ErrCouponLimitExceeded
		}

		// 计算过期时间
		var expireAt time.Time
		if coupon.ValidDays != nil && *coupon.ValidDays > 0 {
			expireAt = now.AddDate(0, 0, *coupon.ValidDays)
			// 过期时间不能超过优惠券本身的结束时间
			if expireAt.After(coupon.EndTime) {
				expireAt = coupon.EndTime
			}
		} else {
			expireAt = coupon.EndTime
		}

		// 创建用户优惠券
		userCoupon = &models.UserCoupon{
			UserID:     userID,
			CouponID:   couponID,
			Status:     models.UserCouponStatusUnused,
			ExpiredAt:  expireAt,
			ReceivedAt: now,
		}
		if err := tx.Create(userCoupon).Error; err != nil {
			return err
		}

		// 增加已发放数量
		result := tx.Model(&models.Coupon{}).
			Where("id = ? AND total_count > received_count", couponID).
			UpdateColumn("received_count", gorm.Expr("received_count + 1"))
		if result.Error != nil {
			return result.Error
		}
		if result.RowsAffected == 0 {
			return ErrCouponSoldOut
		}

		return nil
	})

	if err != nil {
		return nil, err
	}

	return userCoupon, nil
}

// CalculateDiscount 计算优惠金额
func (s *CouponService) CalculateDiscount(coupon *models.Coupon, orderAmount float64) float64 {
	if orderAmount < coupon.MinAmount {
		return 0
	}

	var discount float64
	switch coupon.Type {
	case models.CouponTypeFixed:
		// 固定金额优惠
		discount = coupon.Value
	case models.CouponTypePercent:
		// 百分比折扣，value 是折扣比例（如 0.1 表示打9折，优惠10%）
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

// GetBestCouponForOrder 获取订单最优优惠券
func (s *CouponService) GetBestCouponForOrder(ctx context.Context, userID int64, orderType string, orderAmount float64) (*models.UserCoupon, float64, error) {
	userCoupons, err := s.userCouponRepo.ListAvailableForOrder(ctx, userID, orderType, orderAmount)
	if err != nil {
		return nil, 0, err
	}

	if len(userCoupons) == 0 {
		return nil, 0, nil
	}

	var bestCoupon *models.UserCoupon
	var maxDiscount float64

	for _, uc := range userCoupons {
		if uc.Coupon == nil {
			continue
		}
		discount := s.CalculateDiscount(uc.Coupon, orderAmount)
		if discount > maxDiscount {
			maxDiscount = discount
			bestCoupon = uc
		}
	}

	return bestCoupon, maxDiscount, nil
}
