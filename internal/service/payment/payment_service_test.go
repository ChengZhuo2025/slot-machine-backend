// Package payment 支付服务单元测试
package payment

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"

	appErrors "github.com/dumeirei/smart-locker-backend/internal/common/errors"
	"github.com/dumeirei/smart-locker-backend/internal/models"
	"github.com/dumeirei/smart-locker-backend/internal/repository"
	"github.com/dumeirei/smart-locker-backend/pkg/wechatpay"
)

// setupTestDB 创建测试数据库
func setupTestDB(t *testing.T) *gorm.DB {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Silent),
	})
	require.NoError(t, err)

	err = db.AutoMigrate(
		&models.User{},
		&models.MemberLevel{},
		&models.Payment{},
		&models.Refund{},
		&models.Rental{},
	)
	require.NoError(t, err)

	// 创建默认会员等级
	level := &models.MemberLevel{
		ID:        1,
		Name:      "普通会员",
		Level:     1,
		MinPoints: 0,
		Discount:  1.0,
	}
	db.Create(level)

	return db
}

// testPaymentService 测试用支付服务
type testPaymentService struct {
	*PaymentService
	db *gorm.DB
}

// setupTestPaymentService 创建测试用的 PaymentService
func setupTestPaymentService(t *testing.T) *testPaymentService {
	db := setupTestDB(t)
	paymentRepo := repository.NewPaymentRepository(db)
	refundRepo := repository.NewRefundRepository(db)
	rentalRepo := repository.NewRentalRepository(db)

	// 不使用微信支付客户端，传入 nil
	service := NewPaymentService(db, paymentRepo, refundRepo, rentalRepo, nil)

	return &testPaymentService{
		PaymentService: service,
		db:             db,
	}
}

// createTestUser 创建测试用户
func createTestUser(t *testing.T, db *gorm.DB) *models.User {
	phone := "13800138000"
	user := &models.User{
		Phone:         &phone,
		Nickname:      "测试用户",
		MemberLevelID: 1,
		Status:        models.UserStatusActive,
	}
	err := db.Create(user).Error
	require.NoError(t, err)
	return user
}

func TestPaymentService_CreatePayment(t *testing.T) {
	svc := setupTestPaymentService(t)
	ctx := context.Background()
	user := createTestUser(t, svc.db)

	t.Run("创建余额支付成功", func(t *testing.T) {
		req := &CreatePaymentRequest{
			OrderID:        1,
			OrderNo:        "R20240101001",
			OrderType:      "rental",
			Amount:         60.0,
			PaymentMethod:  models.PaymentMethodBalance,
			PaymentChannel: models.PaymentChannelMiniProgram,
			Description:    "租借订单支付",
		}

		resp, err := svc.CreatePayment(ctx, user.ID, req)
		require.NoError(t, err)
		assert.NotNil(t, resp)
		assert.NotEmpty(t, resp.PaymentNo)
		assert.False(t, resp.ExpiredAt.IsZero())

		// 验证支付记录已创建
		var payment models.Payment
		svc.db.Where("payment_no = ?", resp.PaymentNo).First(&payment)
		assert.Equal(t, req.Amount, payment.Amount)
		assert.EqualValues(t, models.PaymentStatusPending, payment.Status)
	})

	t.Run("创建微信支付无客户端", func(t *testing.T) {
		req := &CreatePaymentRequest{
			OrderID:        2,
			OrderNo:        "R20240101002",
			OrderType:      "rental",
			Amount:         60.0,
			PaymentMethod:  models.PaymentMethodWechat,
			PaymentChannel: models.PaymentChannelMiniProgram,
			OpenID:         "oXXXX",
		}

		// 由于没有微信支付客户端，不会调用微信接口
		resp, err := svc.CreatePayment(ctx, user.ID, req)
		require.NoError(t, err)
		assert.NotNil(t, resp)
		assert.Nil(t, resp.PayParams) // 没有微信支付参数
	})
}

func TestPaymentService_QueryPayment(t *testing.T) {
	svc := setupTestPaymentService(t)
	ctx := context.Background()
	user := createTestUser(t, svc.db)

	// 创建支付记录
	payment := &models.Payment{
		PaymentNo:      "P20240101001",
		OrderID:        1,
		OrderNo:        "R20240101001",
		UserID:         user.ID,
		Amount:         60.0,
		PaymentMethod:  models.PaymentMethodBalance,
		PaymentChannel: models.PaymentChannelMiniProgram,
		Status:         models.PaymentStatusPending,
	}
	svc.db.Create(payment)

	t.Run("查询支付记录成功", func(t *testing.T) {
		info, err := svc.QueryPayment(ctx, payment.PaymentNo)
		require.NoError(t, err)
		assert.Equal(t, payment.PaymentNo, info.PaymentNo)
		assert.Equal(t, payment.Amount, info.Amount)
		assert.Equal(t, "待支付", info.StatusName)
	})

	t.Run("查询不存在的支付记录", func(t *testing.T) {
		_, err := svc.QueryPayment(ctx, "P99999999999")
		assert.Error(t, err)
	})
}

func TestPaymentService_CreateRefund(t *testing.T) {
	svc := setupTestPaymentService(t)
	ctx := context.Background()
	user := createTestUser(t, svc.db)

	// 创建成功支付的记录
	now := time.Now()
	payment := &models.Payment{
		PaymentNo:      "P20240101002",
		OrderID:        1,
		OrderNo:        "R20240101002",
		UserID:         user.ID,
		Amount:         60.0,
		PaymentMethod:  models.PaymentMethodBalance,
		PaymentChannel: models.PaymentChannelMiniProgram,
		Status:         models.PaymentStatusSuccess,
		PaidAt:         &now,
	}
	svc.db.Create(payment)

	t.Run("创建退款成功", func(t *testing.T) {
		req := &CreateRefundRequest{
			PaymentNo: payment.PaymentNo,
			Amount:    30.0,
			Reason:    "测试退款",
		}

		err := svc.CreateRefund(ctx, user.ID, req)
		require.NoError(t, err)

		// 验证退款记录已创建
		var refund models.Refund
		svc.db.Where("payment_no = ?", payment.PaymentNo).First(&refund)
		assert.Equal(t, req.Amount, refund.Amount)
		assert.EqualValues(t, models.RefundStatusPending, refund.Status)
	})

	t.Run("退款金额超过支付金额", func(t *testing.T) {
		req := &CreateRefundRequest{
			PaymentNo: payment.PaymentNo,
			Amount:    100.0, // 超过原支付金额
			Reason:    "测试退款",
		}

		err := svc.CreateRefund(ctx, user.ID, req)
		assert.Error(t, err)
	})

	t.Run("非支付用户退款失败", func(t *testing.T) {
		anotherPhone := "13800138001"
		anotherUser := &models.User{
			Phone:         &anotherPhone,
			Nickname:      "另一个用户",
			MemberLevelID: 1,
			Status:        models.UserStatusActive,
		}
		svc.db.Create(anotherUser)

		req := &CreateRefundRequest{
			PaymentNo: payment.PaymentNo,
			Amount:    10.0,
			Reason:    "测试退款",
		}

		err := svc.CreateRefund(ctx, anotherUser.ID, req)
		assert.Error(t, err)
	})

	t.Run("待支付状态不能退款", func(t *testing.T) {
		pendingPayment := &models.Payment{
			PaymentNo:      "P20240101003",
			OrderID:        2,
			OrderNo:        "R20240101003",
			UserID:         user.ID,
			Amount:         60.0,
			PaymentMethod:  models.PaymentMethodBalance,
			PaymentChannel: models.PaymentChannelMiniProgram,
			Status:         models.PaymentStatusPending,
		}
		svc.db.Create(pendingPayment)

		req := &CreateRefundRequest{
			PaymentNo: pendingPayment.PaymentNo,
			Amount:    30.0,
			Reason:    "测试退款",
		}

		err := svc.CreateRefund(ctx, user.ID, req)
		assert.Error(t, err)
	})
}

func TestPaymentService_CloseExpiredPayments(t *testing.T) {
	svc := setupTestPaymentService(t)
	ctx := context.Background()
	user := createTestUser(t, svc.db)

	// 创建过期的待支付记录
	expiredTime := time.Now().Add(-1 * time.Hour)
	expiredPayment := &models.Payment{
		PaymentNo:      "P20240101004",
		OrderID:        3,
		OrderNo:        "R20240101004",
		UserID:         user.ID,
		Amount:         60.0,
		PaymentMethod:  models.PaymentMethodBalance,
		PaymentChannel: models.PaymentChannelMiniProgram,
		Status:         models.PaymentStatusPending,
		ExpiredAt:      &expiredTime,
	}
	svc.db.Create(expiredPayment)

	t.Run("关闭过期支付成功", func(t *testing.T) {
		err := svc.CloseExpiredPayments(ctx)
		require.NoError(t, err)

		// 验证支付状态已关闭
		var payment models.Payment
		svc.db.First(&payment, expiredPayment.ID)
		assert.EqualValues(t, models.PaymentStatusClosed, payment.Status)
	})
}

func TestPaymentService_getStatusName(t *testing.T) {
	svc := setupTestPaymentService(t)

	tests := []struct {
		status   int8
		expected string
	}{
		{models.PaymentStatusPending, "待支付"},
		{models.PaymentStatusSuccess, "支付成功"},
		{models.PaymentStatusFailed, "支付失败"},
		{models.PaymentStatusClosed, "已关闭"},
		{models.PaymentStatusRefunded, "已退款"},
		{99, "未知"},
	}

	for _, tt := range tests {
		name := svc.getStatusName(tt.status)
		assert.Equal(t, tt.expected, name)
	}
}

func TestPaymentService_toPaymentInfo(t *testing.T) {
	svc := setupTestPaymentService(t)

	now := time.Now()
	transactionID := "wx123456789"
	payment := &models.Payment{
		ID:             1,
		PaymentNo:      "P20240101001",
		OrderNo:        "R20240101001",
		Amount:         60.0,
		PaymentMethod:  models.PaymentMethodWechat,
		PaymentChannel: models.PaymentChannelMiniProgram,
		Status:         models.PaymentStatusSuccess,
		TransactionID:  &transactionID,
		PaidAt:         &now,
		ExpiredAt:      &now,
		CreatedAt:      now,
	}

	info := svc.toPaymentInfo(payment)

	assert.Equal(t, payment.ID, info.ID)
	assert.Equal(t, payment.PaymentNo, info.PaymentNo)
	assert.Equal(t, payment.OrderNo, info.OrderNo)
	assert.Equal(t, payment.Amount, info.Amount)
	assert.Equal(t, payment.PaymentMethod, info.PaymentMethod)
	assert.Equal(t, payment.PaymentChannel, info.PaymentChannel)
	assert.Equal(t, payment.Status, info.Status)
	assert.Equal(t, "支付成功", info.StatusName)
	assert.Equal(t, &transactionID, info.TransactionID)
	assert.NotNil(t, info.PaidAt)
}

func TestPaymentService_HandlePaymentCallback(t *testing.T) {
	ctx := context.Background()

	t.Run("wechat client not initialized", func(t *testing.T) {
		svc := setupTestPaymentService(t)
		err := svc.HandlePaymentCallback(ctx, []byte(`{}`))
		require.Error(t, err)
		var appErr *appErrors.AppError
		require.ErrorAs(t, err, &appErr)
		assert.Equal(t, appErrors.ErrPaymentCallbackError.Code, appErr.Code)
	})

	t.Run("invalid payload", func(t *testing.T) {
		svc := setupTestPaymentService(t)
		wp, err := wechatpay.NewClient(&wechatpay.Config{})
		require.NoError(t, err)
		svc.wechatPay = wp

		err = svc.HandlePaymentCallback(ctx, []byte(`not-json`))
		require.Error(t, err)
		var appErr *appErrors.AppError
		require.ErrorAs(t, err, &appErr)
		assert.Equal(t, appErrors.ErrPaymentCallbackError.Code, appErr.Code)
	})

	t.Run("payment not found", func(t *testing.T) {
		svc := setupTestPaymentService(t)
		wp, err := wechatpay.NewClient(&wechatpay.Config{})
		require.NoError(t, err)
		svc.wechatPay = wp

		resource := map[string]any{
			"out_trade_no":   "P_NOT_EXISTS",
			"transaction_id": "wx_txn",
			"trade_type":     "JSAPI",
			"trade_state":    wechatpay.TradeStateSuccess,
			"success_time":   time.Now().Format(time.RFC3339),
			"payer":          map[string]any{"openid": "o_x"},
			"amount":         map[string]any{"total": 6000, "payer_total": 6000, "currency": "CNY"},
		}
		payload, _ := json.Marshal(map[string]any{
			"id":            "1",
			"create_time":   time.Now().Format(time.RFC3339),
			"resource_type": "encrypt-resource",
			"event_type":    "TRANSACTION.SUCCESS",
			"summary":       "ok",
			"resource":      resource,
		})

		err = svc.HandlePaymentCallback(ctx, payload)
		require.Error(t, err)
		var appErr *appErrors.AppError
		require.ErrorAs(t, err, &appErr)
		assert.Equal(t, appErrors.ErrPaymentNotFound.Code, appErr.Code)
	})

	t.Run("success callback updates payment and rental, duplicate is idempotent", func(t *testing.T) {
		svc := setupTestPaymentService(t)
		user := createTestUser(t, svc.db)
		wp, err := wechatpay.NewClient(&wechatpay.Config{})
		require.NoError(t, err)
		svc.wechatPay = wp

		payment := &models.Payment{
			PaymentNo:      "P_CB_SUCCESS",
			OrderID:        1001,
			OrderNo:        "O1001",
			UserID:         user.ID,
			Amount:         60.0,
			PaymentMethod:  models.PaymentMethodWechat,
			PaymentChannel: models.PaymentChannelMiniProgram,
			Status:         models.PaymentStatusPending,
		}
		require.NoError(t, svc.db.Create(payment).Error)
		require.NoError(t, svc.db.Create(&models.Rental{
			OrderID:       payment.OrderID,
			UserID:        user.ID,
			DeviceID:      1,
			DurationHours: 1,
			RentalFee:     10,
			Deposit:       50,
			OvertimeRate:  1.5,
			OvertimeFee:   0,
			Status:        models.RentalStatusPending,
		}).Error)

		resource := map[string]any{
			"out_trade_no":   payment.PaymentNo,
			"transaction_id": "wx_txn_1",
			"trade_type":     "JSAPI",
			"trade_state":    wechatpay.TradeStateSuccess,
			"success_time":   time.Now().Format(time.RFC3339),
			"payer":          map[string]any{"openid": "o_x"},
			"amount":         map[string]any{"total": 6000, "payer_total": 6000, "currency": "CNY"},
		}
		payload, _ := json.Marshal(map[string]any{
			"id":            "1",
			"create_time":   time.Now().Format(time.RFC3339),
			"resource_type": "encrypt-resource",
			"event_type":    "TRANSACTION.SUCCESS",
			"summary":       "ok",
			"resource":      resource,
		})

		require.NoError(t, svc.HandlePaymentCallback(ctx, payload))

		var updated models.Payment
		require.NoError(t, svc.db.Where("payment_no = ?", payment.PaymentNo).First(&updated).Error)
		assert.EqualValues(t, models.PaymentStatusSuccess, updated.Status)
		require.NotNil(t, updated.TransactionID)
		assert.Equal(t, "wx_txn_1", *updated.TransactionID)
		require.NotNil(t, updated.PaidAt)

		var rental models.Rental
		require.NoError(t, svc.db.Where("order_id = ?", payment.OrderID).First(&rental).Error)
		assert.Equal(t, models.RentalStatusPaid, rental.Status)

		// duplicate callback is idempotent
		require.NoError(t, svc.HandlePaymentCallback(ctx, payload))
	})

	t.Run("amount mismatch returns callback error and does not update payment", func(t *testing.T) {
		svc := setupTestPaymentService(t)
		user := createTestUser(t, svc.db)
		wp, err := wechatpay.NewClient(&wechatpay.Config{})
		require.NoError(t, err)
		svc.wechatPay = wp

		payment := &models.Payment{
			PaymentNo:      "P_CB_MISMATCH",
			OrderID:        1002,
			OrderNo:        "O1002",
			UserID:         user.ID,
			Amount:         60.0,
			PaymentMethod:  models.PaymentMethodWechat,
			PaymentChannel: models.PaymentChannelMiniProgram,
			Status:         models.PaymentStatusPending,
		}
		require.NoError(t, svc.db.Create(payment).Error)

		resource := map[string]any{
			"out_trade_no":   payment.PaymentNo,
			"transaction_id": "wx_txn_2",
			"trade_type":     "JSAPI",
			"trade_state":    wechatpay.TradeStateSuccess,
			"success_time":   time.Now().Format(time.RFC3339),
			"payer":          map[string]any{"openid": "o_x"},
			"amount":         map[string]any{"total": 6100, "payer_total": 6100, "currency": "CNY"},
		}
		payload, _ := json.Marshal(map[string]any{
			"id":            "1",
			"create_time":   time.Now().Format(time.RFC3339),
			"resource_type": "encrypt-resource",
			"event_type":    "TRANSACTION.SUCCESS",
			"summary":       "ok",
			"resource":      resource,
		})

		err = svc.HandlePaymentCallback(ctx, payload)
		require.Error(t, err)
		var appErr *appErrors.AppError
		require.ErrorAs(t, err, &appErr)
		assert.Equal(t, appErrors.ErrPaymentCallbackError.Code, appErr.Code)

		var updated models.Payment
		require.NoError(t, svc.db.Where("payment_no = ?", payment.PaymentNo).First(&updated).Error)
		assert.EqualValues(t, models.PaymentStatusPending, updated.Status)
	})

	t.Run("failed trade state sets payment failed and does not update rental", func(t *testing.T) {
		svc := setupTestPaymentService(t)
		user := createTestUser(t, svc.db)
		wp, err := wechatpay.NewClient(&wechatpay.Config{})
		require.NoError(t, err)
		svc.wechatPay = wp

		payment := &models.Payment{
			PaymentNo:      "P_CB_FAIL",
			OrderID:        1003,
			OrderNo:        "O1003",
			UserID:         user.ID,
			Amount:         60.0,
			PaymentMethod:  models.PaymentMethodWechat,
			PaymentChannel: models.PaymentChannelMiniProgram,
			Status:         models.PaymentStatusPending,
		}
		require.NoError(t, svc.db.Create(payment).Error)
		require.NoError(t, svc.db.Create(&models.Rental{
			OrderID:       payment.OrderID,
			UserID:        user.ID,
			DeviceID:      1,
			DurationHours: 1,
			RentalFee:     10,
			Deposit:       50,
			OvertimeRate:  1.5,
			OvertimeFee:   0,
			Status:        models.RentalStatusPending,
		}).Error)

		resource := map[string]any{
			"out_trade_no":   payment.PaymentNo,
			"transaction_id": "wx_txn_3",
			"trade_type":     "JSAPI",
			"trade_state":    wechatpay.TradeStateClosed,
			"success_time":   time.Now().Format(time.RFC3339),
			"payer":          map[string]any{"openid": "o_x"},
			"amount":         map[string]any{"total": 6000, "payer_total": 6000, "currency": "CNY"},
		}
		payload, _ := json.Marshal(map[string]any{
			"id":            "1",
			"create_time":   time.Now().Format(time.RFC3339),
			"resource_type": "encrypt-resource",
			"event_type":    "TRANSACTION.SUCCESS",
			"summary":       "ok",
			"resource":      resource,
		})

		require.NoError(t, svc.HandlePaymentCallback(ctx, payload))

		var updated models.Payment
		require.NoError(t, svc.db.Where("payment_no = ?", payment.PaymentNo).First(&updated).Error)
		assert.EqualValues(t, models.PaymentStatusFailed, updated.Status)
		require.NotNil(t, updated.ErrorMessage)
		assert.Equal(t, wechatpay.TradeStateClosed, *updated.ErrorMessage)

		var rental models.Rental
		require.NoError(t, svc.db.Where("order_id = ?", payment.OrderID).First(&rental).Error)
		assert.Equal(t, models.RentalStatusPending, rental.Status)
	})
}
