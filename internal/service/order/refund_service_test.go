// Package order 订单服务单元测试
package order

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"

	"github.com/dumeirei/smart-locker-backend/internal/models"
	"github.com/dumeirei/smart-locker-backend/internal/repository"
)

func setupTestDB(t *testing.T) *gorm.DB {
	t.Helper()

	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Silent),
	})
	require.NoError(t, err)

	require.NoError(t, db.AutoMigrate(
		&models.User{},
		&models.MemberLevel{},
		&models.Order{},
		&models.OrderItem{},
		&models.Payment{},
		&models.Refund{},
	))

	db.Create(&models.MemberLevel{ID: 1, Name: "普通会员", Level: 1, MinPoints: 0, Discount: 1.0})

	return db
}

func createTestUser(t *testing.T, db *gorm.DB, phone string) *models.User {
	t.Helper()

	u := &models.User{
		Phone:         &phone,
		Nickname:      "测试用户",
		MemberLevelID: 1,
		Status:        models.UserStatusActive,
	}
	require.NoError(t, db.Create(u).Error)
	return u
}

func createPaidOrder(t *testing.T, db *gorm.DB, userID int64, status string, actualAmount float64) *models.Order {
	t.Helper()

	order := &models.Order{
		OrderNo:        fmt.Sprintf("O%d", time.Now().UnixNano()),
		UserID:        userID,
		Type:          models.OrderTypeMall,
		OriginalAmount: actualAmount,
		ActualAmount:  actualAmount,
		Status:        status,
	}
	require.NoError(t, db.Create(order).Error)
	return order
}

func createPayment(t *testing.T, db *gorm.DB, userID, orderID int64, orderNo string, amount float64, status int8) *models.Payment {
	t.Helper()
	now := time.Now()
	p := &models.Payment{
		PaymentNo:      fmt.Sprintf("P%d", time.Now().UnixNano()),
		OrderID:        orderID,
		OrderNo:        orderNo,
		UserID:         userID,
		Amount:         amount,
		PaymentMethod:  models.PaymentMethodBalance,
		PaymentChannel: models.PaymentChannelMiniProgram,
		Status:         status,
		PaidAt:         &now,
	}
	require.NoError(t, db.Create(p).Error)
	return p
}

func setupRefundService(db *gorm.DB) *RefundService {
	refundRepo := repository.NewRefundRepository(db)
	orderRepo := repository.NewOrderRepository(db)
	paymentRepo := repository.NewPaymentRepository(db)
	return NewRefundService(db, refundRepo, orderRepo, paymentRepo)
}

func TestRefundService_CreateRefund(t *testing.T) {
	db := setupTestDB(t)
	svc := setupRefundService(db)
	ctx := context.Background()

	user := createTestUser(t, db, "13800138000")

	t.Run("创建退款成功", func(t *testing.T) {
		order := createPaidOrder(t, db, user.ID, models.OrderStatusPaid, 100.0)
		createPayment(t, db, user.ID, order.ID, order.OrderNo, 100.0, models.PaymentStatusSuccess)

		refund, err := svc.CreateRefund(ctx, user.ID, &CreateRefundRequest{
			OrderID: order.ID,
			Amount:  80.0,
			Reason:  "不想要了",
		})
		require.NoError(t, err)
		require.NotNil(t, refund)
		assert.NotEmpty(t, refund.RefundNo)
		assert.Equal(t, order.ID, refund.OrderID)
		assert.Equal(t, 80.0, refund.Amount)
		assert.EqualValues(t, models.RefundStatusPending, refund.Status)

		var updated models.Order
		require.NoError(t, db.First(&updated, order.ID).Error)
		assert.Equal(t, models.OrderStatusRefunding, updated.Status)
	})

	t.Run("订单不存在", func(t *testing.T) {
		_, err := svc.CreateRefund(ctx, user.ID, &CreateRefundRequest{
			OrderID: 999,
			Amount:  10.0,
			Reason:  "测试",
		})
		assert.Error(t, err)
	})

	t.Run("订单不属于用户", func(t *testing.T) {
		another := createTestUser(t, db, "13800138001")
		order := createPaidOrder(t, db, another.ID, models.OrderStatusPaid, 100.0)
		createPayment(t, db, another.ID, order.ID, order.OrderNo, 100.0, models.PaymentStatusSuccess)

		_, err := svc.CreateRefund(ctx, user.ID, &CreateRefundRequest{
			OrderID: order.ID,
			Amount:  10.0,
			Reason:  "测试",
		})
		assert.Error(t, err)
	})

	t.Run("订单状态不允许", func(t *testing.T) {
		order := createPaidOrder(t, db, user.ID, models.OrderStatusPending, 100.0)
		createPayment(t, db, user.ID, order.ID, order.OrderNo, 100.0, models.PaymentStatusSuccess)

		_, err := svc.CreateRefund(ctx, user.ID, &CreateRefundRequest{
			OrderID: order.ID,
			Amount:  10.0,
			Reason:  "测试",
		})
		assert.Error(t, err)
	})

	t.Run("退款金额超过实付", func(t *testing.T) {
		order := createPaidOrder(t, db, user.ID, models.OrderStatusPaid, 50.0)
		createPayment(t, db, user.ID, order.ID, order.OrderNo, 50.0, models.PaymentStatusSuccess)

		_, err := svc.CreateRefund(ctx, user.ID, &CreateRefundRequest{
			OrderID: order.ID,
			Amount:  60.0,
			Reason:  "测试",
		})
		assert.Error(t, err)
	})

	t.Run("已存在待处理退款", func(t *testing.T) {
		order := createPaidOrder(t, db, user.ID, models.OrderStatusPaid, 100.0)
		payment := createPayment(t, db, user.ID, order.ID, order.OrderNo, 100.0, models.PaymentStatusSuccess)

		existing := &models.Refund{
			RefundNo:  fmt.Sprintf("R%d", time.Now().UnixNano()),
			OrderID:   order.ID,
			OrderNo:   order.OrderNo,
			PaymentID: payment.ID,
			PaymentNo: payment.PaymentNo,
			UserID:    user.ID,
			Amount:    10.0,
			Reason:    "测试",
			Status:    models.RefundStatusPending,
		}
		require.NoError(t, db.Create(existing).Error)

		_, err := svc.CreateRefund(ctx, user.ID, &CreateRefundRequest{
			OrderID: order.ID,
			Amount:  10.0,
			Reason:  "测试",
		})
		assert.Error(t, err)
	})

	t.Run("支付记录不存在", func(t *testing.T) {
		order := createPaidOrder(t, db, user.ID, models.OrderStatusPaid, 100.0)
		_, err := svc.CreateRefund(ctx, user.ID, &CreateRefundRequest{
			OrderID: order.ID,
			Amount:  10.0,
			Reason:  "测试",
		})
		assert.Error(t, err)
	})
}

func TestRefundService_CancelRefund(t *testing.T) {
	db := setupTestDB(t)
	svc := setupRefundService(db)
	ctx := context.Background()

	user := createTestUser(t, db, "13800138000")
	another := createTestUser(t, db, "13800138001")

	order := createPaidOrder(t, db, user.ID, models.OrderStatusRefunding, 100.0)
	payment := createPayment(t, db, user.ID, order.ID, order.OrderNo, 100.0, models.PaymentStatusSuccess)

	refund := &models.Refund{
		RefundNo:  fmt.Sprintf("R%d", time.Now().UnixNano()),
		OrderID:   order.ID,
		OrderNo:   order.OrderNo,
		PaymentID: payment.ID,
		PaymentNo: payment.PaymentNo,
		UserID:    user.ID,
		Amount:    10.0,
		Reason:    "测试",
		Status:    models.RefundStatusPending,
	}
	require.NoError(t, db.Create(refund).Error)

	t.Run("取消退款成功并恢复订单状态", func(t *testing.T) {
		require.NoError(t, svc.CancelRefund(ctx, user.ID, refund.ID))

		var updatedRefund models.Refund
		require.NoError(t, db.First(&updatedRefund, refund.ID).Error)
		assert.EqualValues(t, models.RefundStatusRejected, updatedRefund.Status)
		assert.NotNil(t, updatedRefund.RejectedAt)

		var updatedOrder models.Order
		require.NoError(t, db.First(&updatedOrder, order.ID).Error)
		assert.Equal(t, models.OrderStatusPaid, updatedOrder.Status)
	})

	t.Run("取消不存在的退款", func(t *testing.T) {
		err := svc.CancelRefund(ctx, user.ID, 99999)
		assert.Error(t, err)
	})

	t.Run("取消不属于用户的退款", func(t *testing.T) {
		err := svc.CancelRefund(ctx, another.ID, refund.ID)
		assert.Error(t, err)
	})

	t.Run("非待处理状态不能取消", func(t *testing.T) {
		require.NoError(t, db.Model(&models.Refund{}).Where("id = ?", refund.ID).
			UpdateColumn("status", models.RefundStatusApproved).Error)
		err := svc.CancelRefund(ctx, user.ID, refund.ID)
		assert.Error(t, err)
	})
}
