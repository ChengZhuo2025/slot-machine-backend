// Package errors 错误码和错误处理单元测试
package errors

import (
	stderrors "errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// ==================== AppError 基础测试 ====================

func TestNew(t *testing.T) {
	err := New(1001, "参数错误")
	require.NotNil(t, err)
	assert.Equal(t, 1001, err.Code)
	assert.Equal(t, "参数错误", err.Message)
	assert.Nil(t, err.Err)
}

func TestWrap(t *testing.T) {
	originalErr := stderrors.New("database connection failed")
	err := Wrap(1004, "数据库错误", originalErr)

	require.NotNil(t, err)
	assert.Equal(t, 1004, err.Code)
	assert.Equal(t, "数据库错误", err.Message)
	assert.Equal(t, originalErr, err.Err)
}

// ==================== AppError 方法测试 ====================

func TestAppError_Error(t *testing.T) {
	tests := []struct {
		name     string
		appError *AppError
		want     string
	}{
		{
			name:     "Error without underlying error",
			appError: New(1001, "参数错误"),
			want:     "[1001] 参数错误",
		},
		{
			name:     "Error with underlying error",
			appError: Wrap(1004, "数据库错误", stderrors.New("connection timeout")),
			want:     "[1004] 数据库错误: connection timeout",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.appError.Error()
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestAppError_Unwrap(t *testing.T) {
	originalErr := stderrors.New("original error")
	err := Wrap(1000, "wrapped error", originalErr)

	unwrapped := err.Unwrap()
	assert.Equal(t, originalErr, unwrapped)
}

func TestAppError_WithMessage(t *testing.T) {
	original := New(1001, "原始消息")
	modified := original.WithMessage("修改后的消息")

	assert.Equal(t, 1001, modified.Code)
	assert.Equal(t, "修改后的消息", modified.Message)
	assert.Nil(t, modified.Err)

	// 验证原始错误未被修改
	assert.Equal(t, "原始消息", original.Message)
}

func TestAppError_WithError(t *testing.T) {
	original := New(1001, "参数错误")
	underlyingErr := stderrors.New("validation failed")
	modified := original.WithError(underlyingErr)

	assert.Equal(t, 1001, modified.Code)
	assert.Equal(t, "参数错误", modified.Message)
	assert.Equal(t, underlyingErr, modified.Err)

	// 验证原始错误未被修改
	assert.Nil(t, original.Err)
}

// ==================== 错误码常量测试 ====================

func TestCommonErrors(t *testing.T) {
	tests := []struct {
		name string
		err  *AppError
		code int
	}{
		{"ErrUnknown", ErrUnknown, 1000},
		{"ErrInvalidParams", ErrInvalidParams, 1001},
		{"ErrNotFound", ErrNotFound, 1002},
		{"ErrAlreadyExists", ErrAlreadyExists, 1003},
		{"ErrDatabaseError", ErrDatabaseError, 1004},
		{"ErrCacheError", ErrCacheError, 1005},
		{"ErrInternalError", ErrInternalError, 1006},
		{"ErrExternalService", ErrExternalService, 1007},
		{"ErrRateLimitExceed", ErrRateLimitExceed, 1008},
		{"ErrOperationFailed", ErrOperationFailed, 1009},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.code, tt.err.Code)
			assert.NotEmpty(t, tt.err.Message)
		})
	}
}

func TestAuthErrors(t *testing.T) {
	tests := []struct {
		name string
		err  *AppError
		code int
	}{
		{"ErrUnauthorized", ErrUnauthorized, 2000},
		{"ErrTokenExpired", ErrTokenExpired, 2001},
		{"ErrTokenInvalid", ErrTokenInvalid, 2002},
		{"ErrPermissionDenied", ErrPermissionDenied, 2004},
		{"ErrAccountDisabled", ErrAccountDisabled, 2005},
		{"ErrPasswordError", ErrPasswordError, 2007},
		{"ErrCaptchaError", ErrCaptchaError, 2008},
		{"ErrSmsCodeError", ErrSmsCodeError, 2009},
		{"ErrSmsCodeExpired", ErrSmsCodeExpired, 2010},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.code, tt.err.Code)
			assert.NotEmpty(t, tt.err.Message)
		})
	}
}

func TestUserErrors(t *testing.T) {
	tests := []struct {
		name string
		err  *AppError
		code int
	}{
		{"ErrUserNotFound", ErrUserNotFound, 3000},
		{"ErrUserExists", ErrUserExists, 3001},
		{"ErrPhoneExists", ErrPhoneExists, 3002},
		{"ErrPhoneInvalid", ErrPhoneInvalid, 3003},
		{"ErrBalanceInsufficient", ErrBalanceInsufficient, 3006},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.code, tt.err.Code)
			assert.NotEmpty(t, tt.err.Message)
		})
	}
}

func TestDeviceErrors(t *testing.T) {
	tests := []struct {
		name string
		err  *AppError
		code int
	}{
		{"ErrDeviceNotFound", ErrDeviceNotFound, 4000},
		{"ErrDeviceOffline", ErrDeviceOffline, 4001},
		{"ErrDeviceBusy", ErrDeviceBusy, 4002},
		{"ErrDeviceDisabled", ErrDeviceDisabled, 4003},
		{"ErrSlotNotAvailable", ErrSlotNotAvailable, 4006},
		{"ErrUnlockFailed", ErrUnlockFailed, 4007},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.code, tt.err.Code)
			assert.NotEmpty(t, tt.err.Message)
		})
	}
}

func TestOrderErrors(t *testing.T) {
	tests := []struct {
		name string
		err  *AppError
		code int
	}{
		{"ErrOrderNotFound", ErrOrderNotFound, 5000},
		{"ErrOrderStatusError", ErrOrderStatusError, 5001},
		{"ErrOrderExpired", ErrOrderExpired, 5002},
		{"ErrOrderCancelled", ErrOrderCancelled, 5003},
		{"ErrProductNotFound", ErrProductNotFound, 5007},
		{"ErrStockInsufficient", ErrStockInsufficient, 5009},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.code, tt.err.Code)
			assert.NotEmpty(t, tt.err.Message)
		})
	}
}

func TestPaymentErrors(t *testing.T) {
	tests := []struct {
		name string
		err  *AppError
		code int
	}{
		{"ErrPaymentNotFound", ErrPaymentNotFound, 6000},
		{"ErrPaymentFailed", ErrPaymentFailed, 6001},
		{"ErrPaymentExpired", ErrPaymentExpired, 6002},
		{"ErrRefundNotFound", ErrRefundNotFound, 6003},
		{"ErrRefundFailed", ErrRefundFailed, 6004},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.code, tt.err.Code)
			assert.NotEmpty(t, tt.err.Message)
		})
	}
}

func TestRentalErrors(t *testing.T) {
	tests := []struct {
		name string
		err  *AppError
		code int
	}{
		{"ErrRentalNotFound", ErrRentalNotFound, 7000},
		{"ErrRentalStatusError", ErrRentalStatusError, 7001},
		{"ErrRentalExpired", ErrRentalExpired, 7002},
		{"ErrRentalInProgress", ErrRentalInProgress, 7003},
		{"ErrDepositNotPaid", ErrDepositNotPaid, 7006},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.code, tt.err.Code)
			assert.NotEmpty(t, tt.err.Message)
		})
	}
}

func TestHotelAndBookingErrors(t *testing.T) {
	tests := []struct {
		name string
		err  *AppError
		code int
	}{
		{"ErrHotelNotFound", ErrHotelNotFound, 8000},
		{"ErrRoomNotFound", ErrRoomNotFound, 8010},
		{"ErrRoomNotAvailable", ErrRoomNotAvailable, 8011},
		{"ErrBookingNotFound", ErrBookingNotFound, 8500},
		{"ErrBookingConflict", ErrBookingConflict, 8502},
		{"ErrVerificationCodeInvalid", ErrVerificationCodeInvalid, 8510},
		{"ErrUnlockCodeInvalid", ErrUnlockCodeInvalid, 8511},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.code, tt.err.Code)
			assert.NotEmpty(t, tt.err.Message)
		})
	}
}

func TestMarketingErrors(t *testing.T) {
	tests := []struct {
		name string
		err  *AppError
		code int
	}{
		{"ErrCouponNotFound", ErrCouponNotFound, 9000},
		{"ErrCouponExpired", ErrCouponExpired, 9001},
		{"ErrCouponUsed", ErrCouponUsed, 9002},
		{"ErrCouponNotApplicable", ErrCouponNotApplicable, 9003},
		{"ErrCampaignNotFound", ErrCampaignNotFound, 9006},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.code, tt.err.Code)
			assert.NotEmpty(t, tt.err.Message)
		})
	}
}

func TestFinanceErrors(t *testing.T) {
	tests := []struct {
		name string
		err  *AppError
		code int
	}{
		{"ErrSettlementNotFound", ErrSettlementNotFound, 10000},
		{"ErrDuplicateRecord", ErrDuplicateRecord, 10001},
		{"ErrMerchantNotFound", ErrMerchantNotFound, 10002},
		{"ErrWithdrawalNotFound", ErrWithdrawalNotFound, 10004},
		{"ErrInsufficientBalance", ErrInsufficientBalance, 10006},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.code, tt.err.Code)
			assert.NotEmpty(t, tt.err.Message)
		})
	}
}

// ==================== 辅助函数测试 ====================

func TestIsAppError(t *testing.T) {
	tests := []struct {
		name string
		err  error
		want bool
	}{
		{"AppError", ErrUnknown, true},
		{"AppError created by New", New(1001, "test"), true},
		{"Standard error", stderrors.New("standard error"), false},
		{"Nil error", nil, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := IsAppError(tt.err)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestGetAppError(t *testing.T) {
	t.Run("From AppError", func(t *testing.T) {
		original := ErrInvalidParams
		got := GetAppError(original)
		assert.Equal(t, original, got)
	})

	t.Run("From standard error", func(t *testing.T) {
		standardErr := stderrors.New("standard error")
		got := GetAppError(standardErr)

		assert.Equal(t, ErrUnknown.Code, got.Code)
		assert.Equal(t, standardErr, got.Err)
	})

	t.Run("Preserves underlying error", func(t *testing.T) {
		underlyingErr := stderrors.New("database failed")
		appErr := Wrap(1004, "数据库错误", underlyingErr)

		got := GetAppError(appErr)
		assert.Equal(t, appErr, got)
	})
}

// ==================== 错误链测试 ====================

func TestErrorChaining(t *testing.T) {
	// 创建错误链
	originalErr := stderrors.New("connection timeout")
	wrappedErr := Wrap(1004, "数据库错误", originalErr)

	// 验证可以使用 errors.Is 和 errors.As
	unwrapped := wrappedErr.Unwrap()
	assert.Equal(t, originalErr, unwrapped)

	// 验证错误消息包含原始错误
	assert.Contains(t, wrappedErr.Error(), "connection timeout")
	assert.Contains(t, wrappedErr.Error(), "数据库错误")
	assert.Contains(t, wrappedErr.Error(), "1004")
}

// ==================== 边界条件测试 ====================

func TestAppError_EmptyMessage(t *testing.T) {
	err := New(9999, "")
	assert.Equal(t, 9999, err.Code)
	assert.Equal(t, "", err.Message)
	assert.Equal(t, "[9999] ", err.Error())
}

func TestAppError_ZeroCode(t *testing.T) {
	err := New(0, "零代码错误")
	assert.Equal(t, 0, err.Code)
	assert.Equal(t, "零代码错误", err.Message)
}

func TestAppError_NegativeCode(t *testing.T) {
	err := New(-1, "负数代码")
	assert.Equal(t, -1, err.Code)
	assert.Equal(t, "负数代码", err.Message)
}

// ==================== 修改链测试 ====================

func TestAppError_ChainedModifications(t *testing.T) {
	original := New(1001, "原始错误")

	// 链式修改
	modified := original.
		WithMessage("修改后的消息").
		WithError(stderrors.New("底层错误"))

	assert.Equal(t, 1001, modified.Code)
	assert.Equal(t, "修改后的消息", modified.Message)
	assert.NotNil(t, modified.Err)

	// 验证原始错误未被修改
	assert.Equal(t, "原始错误", original.Message)
	assert.Nil(t, original.Err)
}
