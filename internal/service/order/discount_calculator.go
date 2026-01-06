// Package order 提供订单相关服务
package order

import (
	"context"

	"github.com/dumeirei/smart-locker-backend/internal/models"
	marketingService "github.com/dumeirei/smart-locker-backend/internal/service/marketing"
)

// DiscountCalculator 订单优惠计算器
type DiscountCalculator struct {
	couponService   *marketingService.CouponService
	campaignService *marketingService.CampaignService
}

// NewDiscountCalculator 创建订单优惠计算器
func NewDiscountCalculator(couponSvc *marketingService.CouponService, campaignSvc *marketingService.CampaignService) *DiscountCalculator {
	return &DiscountCalculator{
		couponService:   couponSvc,
		campaignService: campaignSvc,
	}
}

// DiscountResult 优惠结果
type DiscountResult struct {
	OriginalAmount   float64                  `json:"original_amount"`    // 原始金额
	FinalAmount      float64                  `json:"final_amount"`       // 最终金额
	TotalDiscount    float64                  `json:"total_discount"`     // 总优惠金额
	CouponDiscount   float64                  `json:"coupon_discount"`    // 优惠券优惠金额
	CampaignDiscount float64                  `json:"campaign_discount"`  // 活动优惠金额
	UserCoupon       *models.UserCoupon       `json:"user_coupon,omitempty"`
	Campaign         *models.Campaign         `json:"campaign,omitempty"`
	DiscountDetails  []*DiscountDetail        `json:"discount_details"`   // 优惠明细
}

// DiscountDetail 优惠明细
type DiscountDetail struct {
	Type        string  `json:"type"`        // 优惠类型：coupon/campaign
	Name        string  `json:"name"`        // 优惠名称
	Amount      float64 `json:"amount"`      // 优惠金额
	Description string  `json:"description"` // 优惠描述
}

// CalculateOrderDiscount 计算订单优惠
// 参数:
//   - userID: 用户ID
//   - orderType: 订单类型 (rental/mall/hotel)
//   - orderAmount: 订单金额
//   - userCouponID: 用户选择的优惠券ID（可选）
func (c *DiscountCalculator) CalculateOrderDiscount(ctx context.Context, userID int64, orderType string, orderAmount float64, userCouponID *int64) (*DiscountResult, error) {
	result := &DiscountResult{
		OriginalAmount:  orderAmount,
		FinalAmount:     orderAmount,
		DiscountDetails: make([]*DiscountDetail, 0),
	}

	// 1. 计算活动优惠（满减等）
	campaignDiscount, campaign, err := c.campaignService.CalculateDiscountCampaign(ctx, orderAmount)
	if err != nil {
		return nil, err
	}
	if campaignDiscount > 0 && campaign != nil {
		result.CampaignDiscount = campaignDiscount
		result.Campaign = campaign
		result.DiscountDetails = append(result.DiscountDetails, &DiscountDetail{
			Type:        "campaign",
			Name:        campaign.Name,
			Amount:      campaignDiscount,
			Description: "满减活动优惠",
		})
	}

	// 2. 计算优惠券优惠
	// 优惠券在活动优惠后计算，基于活动优惠后的金额
	afterCampaignAmount := orderAmount - campaignDiscount

	if userCouponID != nil {
		// 用户指定了优惠券
		bestCoupon, couponDiscount, err := c.couponService.GetBestCouponForOrder(ctx, userID, orderType, afterCampaignAmount)
		if err != nil {
			return nil, err
		}

		// 验证用户指定的优惠券
		if bestCoupon != nil && bestCoupon.ID == *userCouponID {
			result.CouponDiscount = couponDiscount
			result.UserCoupon = bestCoupon
			if bestCoupon.Coupon != nil {
				result.DiscountDetails = append(result.DiscountDetails, &DiscountDetail{
					Type:        "coupon",
					Name:        bestCoupon.Coupon.Name,
					Amount:      couponDiscount,
					Description: c.getCouponDescription(bestCoupon.Coupon),
				})
			}
		}
	} else {
		// 自动选择最优优惠券
		bestCoupon, couponDiscount, err := c.couponService.GetBestCouponForOrder(ctx, userID, orderType, afterCampaignAmount)
		if err != nil {
			return nil, err
		}
		if bestCoupon != nil && couponDiscount > 0 {
			result.CouponDiscount = couponDiscount
			result.UserCoupon = bestCoupon
			if bestCoupon.Coupon != nil {
				result.DiscountDetails = append(result.DiscountDetails, &DiscountDetail{
					Type:        "coupon",
					Name:        bestCoupon.Coupon.Name,
					Amount:      couponDiscount,
					Description: c.getCouponDescription(bestCoupon.Coupon),
				})
			}
		}
	}

	// 3. 计算最终金额
	result.TotalDiscount = result.CampaignDiscount + result.CouponDiscount
	result.FinalAmount = orderAmount - result.TotalDiscount

	// 确保最终金额不为负
	if result.FinalAmount < 0 {
		result.FinalAmount = 0
	}

	return result, nil
}

// CalculateWithSpecificCoupon 使用指定优惠券计算优惠
func (c *DiscountCalculator) CalculateWithSpecificCoupon(ctx context.Context, userID int64, orderType string, orderAmount float64, userCouponID int64) (*DiscountResult, error) {
	return c.CalculateOrderDiscount(ctx, userID, orderType, orderAmount, &userCouponID)
}

// GetBestCoupon 获取最优优惠券
func (c *DiscountCalculator) GetBestCoupon(ctx context.Context, userID int64, orderType string, orderAmount float64) (*models.UserCoupon, float64, error) {
	return c.couponService.GetBestCouponForOrder(ctx, userID, orderType, orderAmount)
}

// PreviewDiscount 预览订单优惠（不使用优惠券）
func (c *DiscountCalculator) PreviewDiscount(ctx context.Context, orderAmount float64) (*DiscountResult, error) {
	result := &DiscountResult{
		OriginalAmount:  orderAmount,
		FinalAmount:     orderAmount,
		DiscountDetails: make([]*DiscountDetail, 0),
	}

	// 只计算活动优惠
	campaignDiscount, campaign, err := c.campaignService.CalculateDiscountCampaign(ctx, orderAmount)
	if err != nil {
		return nil, err
	}
	if campaignDiscount > 0 && campaign != nil {
		result.CampaignDiscount = campaignDiscount
		result.Campaign = campaign
		result.TotalDiscount = campaignDiscount
		result.FinalAmount = orderAmount - campaignDiscount
		result.DiscountDetails = append(result.DiscountDetails, &DiscountDetail{
			Type:        "campaign",
			Name:        campaign.Name,
			Amount:      campaignDiscount,
			Description: "满减活动优惠",
		})
	}

	if result.FinalAmount < 0 {
		result.FinalAmount = 0
	}

	return result, nil
}

// getCouponDescription 获取优惠券描述
func (c *DiscountCalculator) getCouponDescription(coupon *models.Coupon) string {
	if coupon == nil {
		return ""
	}

	switch coupon.Type {
	case models.CouponTypeFixed:
		if coupon.MinAmount > 0 {
			return "满" + formatAmount(coupon.MinAmount) + "减" + formatAmount(coupon.Value)
		}
		return "立减" + formatAmount(coupon.Value) + "元"
	case models.CouponTypePercent:
		discount := int(coupon.Value * 100)
		if coupon.MinAmount > 0 {
			return "满" + formatAmount(coupon.MinAmount) + "享" + formatPercent(discount) + "折"
		}
		return formatPercent(discount) + "折优惠"
	default:
		return "优惠券"
	}
}

// formatAmount 格式化金额
func formatAmount(amount float64) string {
	if amount == float64(int(amount)) {
		return string(rune(int(amount) + '0'))
	}
	return string(rune(int(amount)))
}

// formatPercent 格式化折扣百分比
func formatPercent(percent int) string {
	// 将百分比转换为折扣描述（如 10% 优惠 = 9折）
	discount := 100 - percent
	if discount%10 == 0 {
		return string(rune(discount/10 + '0'))
	}
	return string(rune(discount/10+'0')) + "." + string(rune(discount%10+'0'))
}
