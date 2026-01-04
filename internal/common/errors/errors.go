// Package errors 定义业务错误码和错误处理
package errors

import (
	"fmt"
)

// AppError 应用错误
type AppError struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
	Err     error  `json:"-"`
}

// Error 实现 error 接口
func (e *AppError) Error() string {
	if e.Err != nil {
		return fmt.Sprintf("[%d] %s: %v", e.Code, e.Message, e.Err)
	}
	return fmt.Sprintf("[%d] %s", e.Code, e.Message)
}

// Unwrap 实现 errors.Unwrap
func (e *AppError) Unwrap() error {
	return e.Err
}

// New 创建新的应用错误
func New(code int, message string) *AppError {
	return &AppError{
		Code:    code,
		Message: message,
	}
}

// Wrap 包装错误
func Wrap(code int, message string, err error) *AppError {
	return &AppError{
		Code:    code,
		Message: message,
		Err:     err,
	}
}

// WithMessage 修改错误消息
func (e *AppError) WithMessage(message string) *AppError {
	return &AppError{
		Code:    e.Code,
		Message: message,
		Err:     e.Err,
	}
}

// WithError 添加原始错误
func (e *AppError) WithError(err error) *AppError {
	return &AppError{
		Code:    e.Code,
		Message: e.Message,
		Err:     err,
	}
}

// 通用错误码 (1000-1999)
var (
	ErrUnknown          = New(1000, "未知错误")
	ErrInvalidParams    = New(1001, "参数错误")
	ErrNotFound         = New(1002, "资源不存在")
	ErrAlreadyExists    = New(1003, "资源已存在")
	ErrDatabaseError    = New(1004, "数据库错误")
	ErrCacheError       = New(1005, "缓存错误")
	ErrInternalError    = New(1006, "内部错误")
	ErrExternalService  = New(1007, "外部服务错误")
	ErrRateLimitExceed  = New(1008, "请求过于频繁")
	ErrOperationFailed  = New(1009, "操作失败")
	ErrResourceNotFound = New(1010, "资源不存在")
)

// 认证错误码 (2000-2999)
var (
	ErrUnauthorized     = New(2000, "未登录")
	ErrTokenExpired     = New(2001, "登录已过期")
	ErrTokenInvalid     = New(2002, "无效的令牌")
	ErrTokenRefreshFail = New(2003, "刷新令牌失败")
	ErrPermissionDenied = New(2004, "权限不足")
	ErrAccountDisabled  = New(2005, "账号已禁用")
	ErrAccountLocked    = New(2006, "账号已锁定")
	ErrPasswordError    = New(2007, "密码错误")
	ErrCaptchaError     = New(2008, "验证码错误")
	ErrSmsCodeError     = New(2009, "短信验证码错误")
	ErrSmsCodeExpired   = New(2010, "短信验证码已过期")
	ErrSmsSendFail      = New(2011, "短信发送失败")
	ErrSmsSendTooFast   = New(2012, "短信发送过于频繁")
)

// 用户错误码 (3000-3999)
var (
	ErrUserNotFound      = New(3000, "用户不存在")
	ErrUserExists        = New(3001, "用户已存在")
	ErrPhoneExists       = New(3002, "手机号已被注册")
	ErrPhoneInvalid      = New(3003, "无效的手机号")
	ErrRealNameVerified  = New(3004, "已完成实名认证")
	ErrRealNameFailed    = New(3005, "实名认证失败")
	ErrBalanceInsufficient = New(3006, "余额不足")
	ErrWithdrawFailed    = New(3007, "提现失败")
)

// 设备错误码 (4000-4999)
var (
	ErrDeviceNotFound    = New(4000, "设备不存在")
	ErrDeviceOffline     = New(4001, "设备离线")
	ErrDeviceBusy        = New(4002, "设备繁忙")
	ErrDeviceDisabled    = New(4003, "设备已禁用")
	ErrDeviceMaintenance = New(4004, "设备维护中")
	ErrDeviceFault       = New(4005, "设备故障")
	ErrSlotNotAvailable  = New(4006, "无可用格口")
	ErrUnlockFailed      = New(4007, "开锁失败")
	ErrLockFailed        = New(4008, "锁定失败")
	ErrDeviceNoSlot      = New(4009, "设备无可用槽位")
	ErrVenueNotFound     = New(4010, "场地不存在")
	ErrVenueDisabled     = New(4011, "场地已禁用")
	ErrPricingNotFound   = New(4012, "定价方案不存在")
)

// 订单错误码 (5000-5999)
var (
	ErrOrderNotFound     = New(5000, "订单不存在")
	ErrOrderStatusError  = New(5001, "订单状态异常")
	ErrOrderExpired      = New(5002, "订单已过期")
	ErrOrderCancelled    = New(5003, "订单已取消")
	ErrOrderPaid         = New(5004, "订单已支付")
	ErrOrderCannotCancel = New(5005, "订单无法取消")
	ErrCartEmpty         = New(5006, "购物车为空")
	ErrProductNotFound   = New(5007, "商品不存在")
	ErrProductOffShelf   = New(5008, "商品已下架")
	ErrStockInsufficient = New(5009, "库存不足")
)

// 支付错误码 (6000-6999)
var (
	ErrPaymentNotFound     = New(6000, "支付记录不存在")
	ErrPaymentFailed       = New(6001, "支付失败")
	ErrPaymentExpired      = New(6002, "支付已过期")
	ErrRefundNotFound      = New(6003, "退款记录不存在")
	ErrRefundFailed        = New(6004, "退款失败")
	ErrRefundAmountExceed  = New(6005, "退款金额超限")
	ErrPaymentMethodError  = New(6006, "支付方式错误")
	ErrPaymentCallbackError = New(6007, "支付回调错误")
)

// 租借错误码 (7000-7999)
var (
	ErrRentalNotFound    = New(7000, "租借订单不存在")
	ErrRentalStatusError = New(7001, "租借状态异常")
	ErrRentalExpired     = New(7002, "租借已过期")
	ErrRentalInProgress  = New(7003, "存在进行中的租借")
	ErrRentalReturned    = New(7004, "已归还")
	ErrRentalOverdue     = New(7005, "租借超时")
	ErrDepositNotPaid    = New(7006, "押金未支付")
)

// 预订错误码 (8000-8999)
var (
	ErrBookingNotFound    = New(8000, "预订不存在")
	ErrBookingStatusError = New(8001, "预订状态异常")
	ErrBookingConflict    = New(8002, "时段已被预订")
	ErrBookingExpired     = New(8003, "预订已过期")
	ErrRoomNotAvailable   = New(8004, "房间不可用")
	ErrTimeSlotInvalid    = New(8005, "无效的时段")
)

// 营销错误码 (9000-9999)
var (
	ErrCouponNotFound     = New(9000, "优惠券不存在")
	ErrCouponExpired      = New(9001, "优惠券已过期")
	ErrCouponUsed         = New(9002, "优惠券已使用")
	ErrCouponNotApplicable = New(9003, "优惠券不适用")
	ErrCouponLimitExceed  = New(9004, "优惠券领取已达上限")
	ErrCouponNotEnough    = New(9005, "优惠券已领完")
	ErrCampaignNotFound   = New(9006, "活动不存在")
	ErrCampaignExpired    = New(9007, "活动已结束")
)

// IsAppError 判断是否为应用错误
func IsAppError(err error) bool {
	_, ok := err.(*AppError)
	return ok
}

// GetAppError 获取应用错误
func GetAppError(err error) *AppError {
	if appErr, ok := err.(*AppError); ok {
		return appErr
	}
	return ErrUnknown.WithError(err)
}
