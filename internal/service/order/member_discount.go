// Package order 提供订单相关服务
package order

import (
	"context"
	"fmt"

	"gorm.io/gorm"

	"github.com/dumeirei/smart-locker-backend/internal/models"
	"github.com/dumeirei/smart-locker-backend/internal/repository"
)

// MemberDiscountService 会员折扣服务
type MemberDiscountService struct {
	db              *gorm.DB
	userRepo        *repository.UserRepository
	memberLevelRepo *repository.MemberLevelRepository
}

// NewMemberDiscountService 创建会员折扣服务
func NewMemberDiscountService(
	db *gorm.DB,
	userRepo *repository.UserRepository,
	memberLevelRepo *repository.MemberLevelRepository,
) *MemberDiscountService {
	return &MemberDiscountService{
		db:              db,
		userRepo:        userRepo,
		memberLevelRepo: memberLevelRepo,
	}
}

// MemberDiscountResult 会员折扣结果
type MemberDiscountResult struct {
	OriginalAmount   float64 `json:"original_amount"`   // 原始金额
	DiscountRate     float64 `json:"discount_rate"`     // 折扣率（如 0.95 表示95折）
	DiscountAmount   float64 `json:"discount_amount"`   // 折扣金额
	FinalAmount      float64 `json:"final_amount"`      // 折后金额
	MemberLevelID    int64   `json:"member_level_id"`   // 会员等级ID
	MemberLevelName  string  `json:"member_level_name"` // 会员等级名称
	HasMemberDiscount bool   `json:"has_member_discount"` // 是否有会员折扣
}

// CalculateMemberDiscount 计算会员折扣
func (s *MemberDiscountService) CalculateMemberDiscount(ctx context.Context, userID int64, amount float64) (*MemberDiscountResult, error) {
	result := &MemberDiscountResult{
		OriginalAmount:    amount,
		DiscountRate:      1.0,
		DiscountAmount:    0,
		FinalAmount:       amount,
		HasMemberDiscount: false,
	}

	if amount <= 0 {
		return result, nil
	}

	// 获取用户会员信息
	user, err := s.userRepo.GetByIDWithMemberLevel(ctx, userID)
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return result, nil
		}
		return nil, err
	}

	if user.MemberLevel == nil {
		return result, nil
	}

	result.MemberLevelID = user.MemberLevel.ID
	result.MemberLevelName = user.MemberLevel.Name
	result.DiscountRate = user.MemberLevel.Discount

	// 计算折扣（discount 小于 1 才有优惠）
	if user.MemberLevel.Discount < 1.0 {
		result.FinalAmount = amount * user.MemberLevel.Discount
		result.DiscountAmount = amount - result.FinalAmount
		result.HasMemberDiscount = true

		// 保留两位小数
		result.FinalAmount = float64(int(result.FinalAmount*100)) / 100
		result.DiscountAmount = float64(int(result.DiscountAmount*100)) / 100
	}

	return result, nil
}

// GetMemberDiscount 获取用户会员折扣率
func (s *MemberDiscountService) GetMemberDiscount(ctx context.Context, userID int64) (float64, error) {
	user, err := s.userRepo.GetByIDWithMemberLevel(ctx, userID)
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return 1.0, nil
		}
		return 1.0, err
	}

	if user.MemberLevel == nil {
		return 1.0, nil
	}

	return user.MemberLevel.Discount, nil
}

// GetMemberInfo 获取用户会员信息（用于订单展示）
func (s *MemberDiscountService) GetMemberInfo(ctx context.Context, userID int64) (*models.MemberLevel, error) {
	user, err := s.userRepo.GetByIDWithMemberLevel(ctx, userID)
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, nil
		}
		return nil, err
	}

	return user.MemberLevel, nil
}

// GetDiscountDescription 获取折扣描述文本
func (s *MemberDiscountService) GetDiscountDescription(discount float64, levelName string) string {
	if discount >= 1.0 {
		return ""
	}

	// 将折扣率转换为"几折"的描述
	// 例如 0.95 -> "95折"，0.9 -> "9折"
	discountPercent := int(discount * 100)
	if discountPercent%10 == 0 {
		return fmt.Sprintf("%s享%d折", levelName, discountPercent/10)
	}
	return fmt.Sprintf("%s享%.1f折", levelName, float64(discountPercent)/10)
}

// EnhancedDiscountResult 增强的优惠结果（包含会员折扣）
type EnhancedDiscountResult struct {
	OriginalAmount   float64                  `json:"original_amount"`    // 原始金额
	FinalAmount      float64                  `json:"final_amount"`       // 最终金额
	TotalDiscount    float64                  `json:"total_discount"`     // 总优惠金额
	MemberDiscount   float64                  `json:"member_discount"`    // 会员折扣金额
	CouponDiscount   float64                  `json:"coupon_discount"`    // 优惠券优惠金额
	CampaignDiscount float64                  `json:"campaign_discount"`  // 活动优惠金额
	MemberLevel      *MemberLevelInfo         `json:"member_level,omitempty"`
	UserCoupon       *models.UserCoupon       `json:"user_coupon,omitempty"`
	Campaign         *models.Campaign         `json:"campaign,omitempty"`
	DiscountDetails  []*EnhancedDiscountDetail `json:"discount_details"`
}

// EnhancedDiscountDetail 增强的优惠明细
type EnhancedDiscountDetail struct {
	Type        string  `json:"type"`        // 优惠类型：member/coupon/campaign
	Name        string  `json:"name"`        // 优惠名称
	Amount      float64 `json:"amount"`      // 优惠金额
	Description string  `json:"description"` // 优惠描述
}

// MemberLevelInfo 会员等级信息
type MemberLevelInfo struct {
	ID       int64   `json:"id"`
	Name     string  `json:"name"`
	Discount float64 `json:"discount"`
}

// CalculateWithMemberDiscount 计算包含会员折扣的订单优惠
// 优惠计算顺序：
// 1. 先计算会员折扣（基于原始金额）
// 2. 再计算活动优惠（基于会员折后金额）
// 3. 最后计算优惠券（基于活动优惠后金额）
func (s *MemberDiscountService) CalculateWithMemberDiscount(
	ctx context.Context,
	userID int64,
	amount float64,
	discountCalc *DiscountCalculator,
	orderType string,
	userCouponID *int64,
) (*EnhancedDiscountResult, error) {
	result := &EnhancedDiscountResult{
		OriginalAmount:  amount,
		FinalAmount:     amount,
		DiscountDetails: make([]*EnhancedDiscountDetail, 0),
	}

	// 1. 计算会员折扣
	memberResult, err := s.CalculateMemberDiscount(ctx, userID, amount)
	if err != nil {
		return nil, err
	}

	afterMemberAmount := amount
	if memberResult.HasMemberDiscount {
		result.MemberDiscount = memberResult.DiscountAmount
		afterMemberAmount = memberResult.FinalAmount
		result.MemberLevel = &MemberLevelInfo{
			ID:       memberResult.MemberLevelID,
			Name:     memberResult.MemberLevelName,
			Discount: memberResult.DiscountRate,
		}
		result.DiscountDetails = append(result.DiscountDetails, &EnhancedDiscountDetail{
			Type:        "member",
			Name:        memberResult.MemberLevelName,
			Amount:      memberResult.DiscountAmount,
			Description: s.GetDiscountDescription(memberResult.DiscountRate, memberResult.MemberLevelName),
		})
	}

	// 2. 计算活动和优惠券优惠（如果有 DiscountCalculator）
	if discountCalc != nil {
		discountResult, err := discountCalc.CalculateOrderDiscount(ctx, userID, orderType, afterMemberAmount, userCouponID)
		if err != nil {
			return nil, err
		}

		result.CampaignDiscount = discountResult.CampaignDiscount
		result.CouponDiscount = discountResult.CouponDiscount
		result.Campaign = discountResult.Campaign
		result.UserCoupon = discountResult.UserCoupon

		// 添加活动优惠明细
		for _, detail := range discountResult.DiscountDetails {
			result.DiscountDetails = append(result.DiscountDetails, &EnhancedDiscountDetail{
				Type:        detail.Type,
				Name:        detail.Name,
				Amount:      detail.Amount,
				Description: detail.Description,
			})
		}

		result.FinalAmount = discountResult.FinalAmount
	} else {
		result.FinalAmount = afterMemberAmount
	}

	// 3. 计算总优惠
	result.TotalDiscount = result.MemberDiscount + result.CampaignDiscount + result.CouponDiscount

	// 确保最终金额不为负
	if result.FinalAmount < 0 {
		result.FinalAmount = 0
	}

	return result, nil
}
