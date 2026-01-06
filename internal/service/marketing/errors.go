// Package marketing 提供营销相关服务
package marketing

import "errors"

// 营销模块错误定义
var (
	// 优惠券相关错误
	ErrCouponNotFound      = errors.New("优惠券不存在")
	ErrCouponNotActive     = errors.New("优惠券未启用")
	ErrCouponNotStarted    = errors.New("优惠券活动未开始")
	ErrCouponExpired       = errors.New("优惠券已过期")
	ErrCouponSoldOut       = errors.New("优惠券已领完")
	ErrCouponLimitExceeded = errors.New("已达到领取上限")
	ErrCouponNotAvailable  = errors.New("优惠券不可用")
	ErrCouponAlreadyUsed   = errors.New("优惠券已使用")
	ErrCouponAmountNotMet  = errors.New("未达到使用门槛")

	// 用户优惠券相关错误
	ErrUserCouponNotFound = errors.New("用户优惠券不存在")
	ErrUserCouponExpired  = errors.New("用户优惠券已过期")
	ErrUserCouponUsed     = errors.New("用户优惠券已使用")

	// 活动相关错误
	ErrCampaignNotFound    = errors.New("活动不存在")
	ErrCampaignNotActive   = errors.New("活动未启用")
	ErrCampaignNotStarted  = errors.New("活动未开始")
	ErrCampaignExpired     = errors.New("活动已结束")
	ErrCampaignRuleInvalid = errors.New("活动规则无效")
)
