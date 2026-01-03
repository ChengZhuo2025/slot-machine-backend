// Package integration 支付流程集成测试
package integration

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"

	"github.com/dumeirei/smart-locker-backend/internal/models"
	"github.com/dumeirei/smart-locker-backend/internal/repository"
	paymentService "github.com/dumeirei/smart-locker-backend/internal/service/payment"
)

// setupPaymentIntegrationDB 创建支付集成测试数据库
func setupPaymentIntegrationDB(t *testing.T) *gorm.DB {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Silent),
	})
	require.NoError(t, err)

	err = db.AutoMigrate(
		&models.User{},
		&models.UserWallet{},
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

// setupPaymentTestEnvironment 设置支付测试环境
func setupPaymentTestEnvironment(t *testing.T, db *gorm.DB) (*paymentService.PaymentService, *models.User) {
	// 创建用户
	phone := "13800138000"
	user := &models.User{
		Phone:         &phone,
		Nickname:      "测试用户",
		MemberLevelID: 1,
		Status:        models.UserStatusActive,
	}
	db.Create(user)

	// 创建钱包
	wallet := &models.UserWallet{
		UserID:  user.ID,
		Balance: 500.0,
	}
	db.Create(wallet)

	// 创建服务
	paymentRepo := repository.NewPaymentRepository(db)
	refundRepo := repository.NewRefundRepository(db)
	rentalRepo := repository.NewRentalRepository(db)

	svc := paymentService.NewPaymentService(db, paymentRepo, refundRepo, rentalRepo, nil)

	return svc, user
}

func TestPaymentFlow_BalancePayment(t *testing.T) {
	db := setupPaymentIntegrationDB(t)
	svc, user := setupPaymentTestEnvironment(t, db)
	ctx := context.Background()

	// 创建租借订单
	slotNo := 1
	rental := &models.Rental{
		RentalNo:      "R20240101001",
		UserID:        user.ID,
		DeviceID:      1,
		PricingID:     1,
		SlotNo:        &slotNo,
		Status:        models.RentalStatusPending,
		UnitPrice:     10.0,
		DepositAmount: 50.0,
		RentalAmount:  10.0,
		ActualAmount:  60.0,
	}
	db.Create(rental)

	// 1. 创建支付订单
	t.Run("步骤1: 创建支付订单", func(t *testing.T) {
		req := &paymentService.CreatePaymentRequest{
			OrderID:        rental.ID,
			OrderNo:        rental.RentalNo,
			OrderType:      "rental",
			Amount:         rental.ActualAmount,
			PaymentMethod:  models.PaymentMethodBalance,
			PaymentChannel: models.PaymentChannelMiniProgram,
			Description:    "租借订单支付",
		}

		resp, err := svc.CreatePayment(ctx, user.ID, req)
		require.NoError(t, err)
		assert.NotEmpty(t, resp.PaymentNo)
		assert.False(t, resp.ExpiredAt.IsZero())

		// 验证支付记录
		var payment models.Payment
		db.Where("payment_no = ?", resp.PaymentNo).First(&payment)
		assert.Equal(t, int8(models.PaymentStatusPending), payment.Status)
		assert.Equal(t, rental.ActualAmount, payment.Amount)
	})

	// 获取创建的支付记录
	var payment models.Payment
	db.Where("order_no = ?", rental.RentalNo).First(&payment)

	// 2. 查询支付状态
	t.Run("步骤2: 查询支付状态", func(t *testing.T) {
		info, err := svc.QueryPayment(ctx, payment.PaymentNo)
		require.NoError(t, err)
		assert.Equal(t, payment.PaymentNo, info.PaymentNo)
		assert.Equal(t, "待支付", info.StatusName)
	})
}

func TestPaymentFlow_RefundProcess(t *testing.T) {
	db := setupPaymentIntegrationDB(t)
	svc, user := setupPaymentTestEnvironment(t, db)
	ctx := context.Background()

	// 创建已支付的支付记录
	now := time.Now()
	payment := &models.Payment{
		PaymentNo:      "P20240101001",
		OrderID:        1,
		OrderNo:        "R20240101001",
		UserID:         user.ID,
		Amount:         60.0,
		PaymentMethod:  models.PaymentMethodBalance,
		PaymentChannel: models.PaymentChannelMiniProgram,
		Status:         models.PaymentStatusSuccess,
		PaidAt:         &now,
	}
	db.Create(payment)

	// 1. 创建退款申请
	t.Run("步骤1: 创建退款申请", func(t *testing.T) {
		req := &paymentService.CreateRefundRequest{
			PaymentNo: payment.PaymentNo,
			Amount:    30.0,
			Reason:    "测试退款",
		}

		err := svc.CreateRefund(ctx, user.ID, req)
		require.NoError(t, err)

		// 验证退款记录
		var refund models.Refund
		db.Where("payment_no = ?", payment.PaymentNo).First(&refund)
		assert.Equal(t, float64(30), refund.Amount)
		assert.Equal(t, int8(models.RefundStatusPending), refund.Status)
	})

	// 2. 再次退款（部分退款后继续退款）
	t.Run("步骤2: 再次部分退款", func(t *testing.T) {
		req := &paymentService.CreateRefundRequest{
			PaymentNo: payment.PaymentNo,
			Amount:    20.0,
			Reason:    "测试二次退款",
		}

		err := svc.CreateRefund(ctx, user.ID, req)
		require.NoError(t, err)

		// 验证退款记录数量
		var count int64
		db.Model(&models.Refund{}).Where("payment_no = ?", payment.PaymentNo).Count(&count)
		assert.Equal(t, int64(2), count)
	})

	// 3. 超额退款应失败
	t.Run("步骤3: 超额退款失败", func(t *testing.T) {
		req := &paymentService.CreateRefundRequest{
			PaymentNo: payment.PaymentNo,
			Amount:    50.0, // 超过剩余可退金额
			Reason:    "超额退款",
		}

		err := svc.CreateRefund(ctx, user.ID, req)
		assert.Error(t, err)
	})
}

func TestPaymentFlow_ExpiredPayment(t *testing.T) {
	db := setupPaymentIntegrationDB(t)
	svc, user := setupPaymentTestEnvironment(t, db)
	ctx := context.Background()

	// 创建已过期的待支付记录
	expiredTime := time.Now().Add(-1 * time.Hour)
	payment := &models.Payment{
		PaymentNo:      "P20240101002",
		OrderID:        2,
		OrderNo:        "R20240101002",
		UserID:         user.ID,
		Amount:         60.0,
		PaymentMethod:  models.PaymentMethodBalance,
		PaymentChannel: models.PaymentChannelMiniProgram,
		Status:         models.PaymentStatusPending,
		ExpiredAt:      &expiredTime,
	}
	db.Create(payment)

	// 关闭过期支付
	err := svc.CloseExpiredPayments(ctx)
	require.NoError(t, err)

	// 验证支付状态
	var updated models.Payment
	db.First(&updated, payment.ID)
	assert.Equal(t, int8(models.PaymentStatusClosed), updated.Status)
}

func TestPaymentFlow_MultiplePayments(t *testing.T) {
	db := setupPaymentIntegrationDB(t)
	svc, user := setupPaymentTestEnvironment(t, db)
	ctx := context.Background()

	// 创建多个支付订单
	for i := 0; i < 5; i++ {
		rental := &models.Rental{
			RentalNo:      "R2024010100" + string(rune('3'+i)),
			UserID:        user.ID,
			DeviceID:      1,
			PricingID:     1,
			Status:        models.RentalStatusPending,
			UnitPrice:     10.0,
			DepositAmount: 50.0,
			ActualAmount:  60.0,
		}
		db.Create(rental)

		req := &paymentService.CreatePaymentRequest{
			OrderID:        rental.ID,
			OrderNo:        rental.RentalNo,
			OrderType:      "rental",
			Amount:         rental.ActualAmount,
			PaymentMethod:  models.PaymentMethodBalance,
			PaymentChannel: models.PaymentChannelMiniProgram,
		}

		resp, err := svc.CreatePayment(ctx, user.ID, req)
		require.NoError(t, err)
		assert.NotEmpty(t, resp.PaymentNo)
	}

	// 验证创建了5个支付记录
	var count int64
	db.Model(&models.Payment{}).Where("user_id = ?", user.ID).Count(&count)
	assert.Equal(t, int64(5), count)
}

func TestPaymentFlow_UnauthorizedRefund(t *testing.T) {
	db := setupPaymentIntegrationDB(t)
	svc, user := setupPaymentTestEnvironment(t, db)
	ctx := context.Background()

	// 创建另一个用户
	anotherPhone := "13800138001"
	anotherUser := &models.User{
		Phone:         &anotherPhone,
		Nickname:      "另一个用户",
		MemberLevelID: 1,
		Status:        models.UserStatusActive,
	}
	db.Create(anotherUser)

	// 创建user的支付记录
	now := time.Now()
	payment := &models.Payment{
		PaymentNo:      "P20240101003",
		OrderID:        3,
		OrderNo:        "R20240101003",
		UserID:         user.ID,
		Amount:         60.0,
		PaymentMethod:  models.PaymentMethodBalance,
		PaymentChannel: models.PaymentChannelMiniProgram,
		Status:         models.PaymentStatusSuccess,
		PaidAt:         &now,
	}
	db.Create(payment)

	// 另一个用户尝试退款
	req := &paymentService.CreateRefundRequest{
		PaymentNo: payment.PaymentNo,
		Amount:    30.0,
		Reason:    "未授权退款",
	}

	err := svc.CreateRefund(ctx, anotherUser.ID, req)
	assert.Error(t, err) // 应该失败
}

func TestPaymentFlow_RefundPendingPayment(t *testing.T) {
	db := setupPaymentIntegrationDB(t)
	svc, user := setupPaymentTestEnvironment(t, db)
	ctx := context.Background()

	// 创建待支付的支付记录
	payment := &models.Payment{
		PaymentNo:      "P20240101004",
		OrderID:        4,
		OrderNo:        "R20240101004",
		UserID:         user.ID,
		Amount:         60.0,
		PaymentMethod:  models.PaymentMethodBalance,
		PaymentChannel: models.PaymentChannelMiniProgram,
		Status:         models.PaymentStatusPending, // 待支付状态
	}
	db.Create(payment)

	// 尝试对待支付订单退款
	req := &paymentService.CreateRefundRequest{
		PaymentNo: payment.PaymentNo,
		Amount:    30.0,
		Reason:    "待支付订单退款",
	}

	err := svc.CreateRefund(ctx, user.ID, req)
	assert.Error(t, err) // 应该失败
}

func TestPaymentFlow_PaymentStatusTransitions(t *testing.T) {
	db := setupPaymentIntegrationDB(t)
	_, user := setupPaymentTestEnvironment(t, db)

	// 测试不同状态的支付记录
	statuses := []int8{
		models.PaymentStatusPending,
		models.PaymentStatusSuccess,
		models.PaymentStatusFailed,
		models.PaymentStatusClosed,
		models.PaymentStatusRefunded,
	}

	for i, status := range statuses {
		payment := &models.Payment{
			PaymentNo:      "P2024010100" + string(rune('5'+i)),
			OrderID:        int64(5 + i),
			OrderNo:        "R2024010100" + string(rune('5'+i)),
			UserID:         user.ID,
			Amount:         60.0,
			PaymentMethod:  models.PaymentMethodBalance,
			PaymentChannel: models.PaymentChannelMiniProgram,
			Status:         status,
		}
		db.Create(payment)
	}

	// 验证创建了所有状态的支付记录
	var payments []models.Payment
	db.Where("user_id = ?", user.ID).Find(&payments)

	statusCount := make(map[int8]int)
	for _, p := range payments {
		statusCount[p.Status]++
	}

	for _, status := range statuses {
		assert.Equal(t, 1, statusCount[status])
	}
}
