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

	appErrors "github.com/dumeirei/smart-locker-backend/internal/common/errors"
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

	t.Run("待发货订单可以退款", func(t *testing.T) {
		order := createPaidOrder(t, db, user.ID, models.OrderStatusPendingShip, 100.0)
		createPayment(t, db, user.ID, order.ID, order.OrderNo, 100.0, models.PaymentStatusSuccess)

		refund, err := svc.CreateRefund(ctx, user.ID, &CreateRefundRequest{
			OrderID: order.ID,
			Amount:  80.0,
			Reason:  "不想要了",
		})
		require.NoError(t, err)
		require.NotNil(t, refund)
		assert.Equal(t, order.ID, refund.OrderID)
	})

	t.Run("已发货订单可以退款", func(t *testing.T) {
		order := createPaidOrder(t, db, user.ID, models.OrderStatusShipped, 100.0)
		createPayment(t, db, user.ID, order.ID, order.OrderNo, 100.0, models.PaymentStatusSuccess)

		refund, err := svc.CreateRefund(ctx, user.ID, &CreateRefundRequest{
			OrderID: order.ID,
			Amount:  80.0,
			Reason:  "不想要了",
		})
		require.NoError(t, err)
		require.NotNil(t, refund)
		assert.Equal(t, order.ID, refund.OrderID)
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

func TestRefundService_GetRefundDetail(t *testing.T) {
	db := setupTestDB(t)
	svc := setupRefundService(db)
	ctx := context.Background()

	user := createTestUser(t, db, "13800138000")
	another := createTestUser(t, db, "13800138001")

	order := createPaidOrder(t, db, user.ID, models.OrderStatusRefunding, 100.0)
	payment := createPayment(t, db, user.ID, order.ID, order.OrderNo, 100.0, models.PaymentStatusSuccess)
	txID := "T123"
	now := time.Now()
	refund := &models.Refund{
		RefundNo:      fmt.Sprintf("R%d", time.Now().UnixNano()),
		OrderID:       order.ID,
		OrderNo:       order.OrderNo,
		PaymentID:     payment.ID,
		PaymentNo:     payment.PaymentNo,
		UserID:        user.ID,
		Amount:        10.0,
		Reason:        "测试",
		Status:        models.RefundStatusSuccess,
		TransactionID: &txID,
		RefundedAt:    &now,
	}
	require.NoError(t, db.Create(refund).Error)

	t.Run("获取成功", func(t *testing.T) {
		got, err := svc.GetRefundDetail(ctx, user.ID, refund.ID)
		require.NoError(t, err)
		require.NotNil(t, got)
		assert.Equal(t, refund.ID, got.ID)
		assert.Equal(t, txID, got.TransactionID)
		assert.NotEmpty(t, got.RefundedAt)
		assert.Equal(t, "退款成功", got.StatusName)
	})

	t.Run("退款不存在", func(t *testing.T) {
		_, err := svc.GetRefundDetail(ctx, user.ID, 99999)
		require.Error(t, err)
		appErr, ok := err.(*appErrors.AppError)
		require.True(t, ok)
		assert.Equal(t, appErrors.ErrRefundNotFound.Code, appErr.Code)
	})

	t.Run("不属于用户返回不存在", func(t *testing.T) {
		_, err := svc.GetRefundDetail(ctx, another.ID, refund.ID)
		require.Error(t, err)
		appErr, ok := err.(*appErrors.AppError)
		require.True(t, ok)
		assert.Equal(t, appErrors.ErrRefundNotFound.Code, appErr.Code)
	})
}

func TestRefundService_GetUserRefunds(t *testing.T) {
	db := setupTestDB(t)
	svc := setupRefundService(db)
	ctx := context.Background()

	user := createTestUser(t, db, "13800138000")
	another := createTestUser(t, db, "13800138001")

	// user: 2 条退款
	for i := 0; i < 2; i++ {
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
	}

	// another: 1 条退款（不应出现在 user 列表）
	{
		order := createPaidOrder(t, db, another.ID, models.OrderStatusRefunding, 50.0)
		payment := createPayment(t, db, another.ID, order.ID, order.OrderNo, 50.0, models.PaymentStatusSuccess)
		refund := &models.Refund{
			RefundNo:  fmt.Sprintf("R%d", time.Now().UnixNano()),
			OrderID:   order.ID,
			OrderNo:   order.OrderNo,
			PaymentID: payment.ID,
			PaymentNo: payment.PaymentNo,
			UserID:    another.ID,
			Amount:    5.0,
			Reason:    "测试",
			Status:    models.RefundStatusPending,
		}
		require.NoError(t, db.Create(refund).Error)
	}

	t.Run("默认分页参数", func(t *testing.T) {
		resp, err := svc.GetUserRefunds(ctx, user.ID, 0, 0)
		require.NoError(t, err)
		require.NotNil(t, resp)
		assert.Equal(t, int64(2), resp.Total)
		assert.Equal(t, 1, resp.Page)
		assert.Equal(t, 10, resp.PageSize)
		assert.Equal(t, 1, resp.TotalPages)
		require.Len(t, resp.List, 2)
	})

	t.Run("分页计算 totalPages", func(t *testing.T) {
		resp, err := svc.GetUserRefunds(ctx, user.ID, 1, 1)
		require.NoError(t, err)
		assert.Equal(t, int64(2), resp.Total)
		assert.Equal(t, 2, resp.TotalPages)
		require.Len(t, resp.List, 1)
	})
}

func TestRefundService_ApproveRefund(t *testing.T) {
	db := setupTestDB(t)
	svc := setupRefundService(db)
	ctx := context.Background()

	user := createTestUser(t, db, "13800138000")
	adminID := int64(99)

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

	t.Run("审批通过更新退款状态与操作人", func(t *testing.T) {
		require.NoError(t, svc.ApproveRefund(ctx, adminID, refund.ID))

		var updated models.Refund
		require.NoError(t, db.First(&updated, refund.ID).Error)
		assert.EqualValues(t, models.RefundStatusApproved, updated.Status)
		require.NotNil(t, updated.OperatorID)
		assert.Equal(t, adminID, *updated.OperatorID)
		require.NotNil(t, updated.OperatorType)
		assert.Equal(t, models.RefundOperatorAdmin, *updated.OperatorType)
	})

	t.Run("退款不存在", func(t *testing.T) {
		err := svc.ApproveRefund(ctx, adminID, 99999)
		require.Error(t, err)
		appErr, ok := err.(*appErrors.AppError)
		require.True(t, ok)
		assert.Equal(t, appErrors.ErrRefundNotFound.Code, appErr.Code)
	})

	t.Run("非待处理状态不允许审批", func(t *testing.T) {
		require.NoError(t, db.Model(&models.Refund{}).Where("id = ?", refund.ID).
			UpdateColumn("status", models.RefundStatusRejected).Error)
		err := svc.ApproveRefund(ctx, adminID, refund.ID)
		require.Error(t, err)
		appErr, ok := err.(*appErrors.AppError)
		require.True(t, ok)
		assert.Equal(t, appErrors.ErrOperationFailed.Code, appErr.Code)
		assert.Contains(t, appErr.Message, "不允许审批")
	})
}

func TestRefundService_RejectRefund(t *testing.T) {
	db := setupTestDB(t)
	svc := setupRefundService(db)
	ctx := context.Background()

	user := createTestUser(t, db, "13800138000")
	adminID := int64(99)

	t.Run("拒绝退款更新退款信息并恢复订单状态(已发货->已发货)", func(t *testing.T) {
		order := createPaidOrder(t, db, user.ID, models.OrderStatusRefunding, 100.0)
		shippedAt := time.Now().Add(-time.Hour)
		require.NoError(t, db.Model(&models.Order{}).Where("id = ?", order.ID).UpdateColumn("shipped_at", shippedAt).Error)

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

		require.NoError(t, svc.RejectRefund(ctx, adminID, refund.ID, "不同意"))

		var updatedRefund models.Refund
		require.NoError(t, db.First(&updatedRefund, refund.ID).Error)
		assert.EqualValues(t, models.RefundStatusRejected, updatedRefund.Status)
		require.NotNil(t, updatedRefund.RejectedAt)
		require.NotNil(t, updatedRefund.RejectReason)
		assert.Equal(t, "不同意", *updatedRefund.RejectReason)
		require.NotNil(t, updatedRefund.OperatorType)
		assert.Equal(t, models.RefundOperatorAdmin, *updatedRefund.OperatorType)

		var updatedOrder models.Order
		require.NoError(t, db.First(&updatedOrder, order.ID).Error)
		assert.Equal(t, models.OrderStatusShipped, updatedOrder.Status)
	})

	t.Run("拒绝退款恢复订单状态(未发货->已支付)", func(t *testing.T) {
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

		require.NoError(t, svc.RejectRefund(ctx, adminID, refund.ID, "不同意"))

		var updatedOrder models.Order
		require.NoError(t, db.First(&updatedOrder, order.ID).Error)
		assert.Equal(t, models.OrderStatusPaid, updatedOrder.Status)
	})

	t.Run("非待处理状态不允许拒绝", func(t *testing.T) {
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
			Status:    models.RefundStatusApproved,
		}
		require.NoError(t, db.Create(refund).Error)

		err := svc.RejectRefund(ctx, adminID, refund.ID, "不同意")
		require.Error(t, err)
		appErr, ok := err.(*appErrors.AppError)
		require.True(t, ok)
		assert.Equal(t, appErrors.ErrOperationFailed.Code, appErr.Code)
		assert.Contains(t, appErr.Message, "不允许拒绝")
	})
}

func TestRefundService_getStatusName(t *testing.T) {
	db := setupTestDB(t)
	svc := setupRefundService(db)

	tests := []struct {
		name     string
		status   int8
		expected string
	}{
		{"待处理", models.RefundStatusPending, "待处理"},
		{"已批准", models.RefundStatusApproved, "已批准"},
		{"已拒绝", models.RefundStatusRejected, "已拒绝"},
		{"处理中", models.RefundStatusProcessing, "处理中"},
		{"退款成功", models.RefundStatusSuccess, "退款成功"},
		{"退款失败", models.RefundStatusFailed, "退款失败"},
		{"未知状态", 99, "未知状态(99)"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := svc.getStatusName(tt.status)
			assert.Equal(t, tt.expected, result)
		})
	}
}

// TestRefundService_RejectRefund_NotFound 测试退款不存在
func TestRefundService_RejectRefund_NotFound(t *testing.T) {
	db := setupTestDB(t)
	svc := setupRefundService(db)
	ctx := context.Background()

	adminID := int64(99)
	err := svc.RejectRefund(ctx, adminID, 99999, "不同意")
	require.Error(t, err)
	appErr, ok := err.(*appErrors.AppError)
	require.True(t, ok)
	assert.Equal(t, appErrors.ErrRefundNotFound.Code, appErr.Code)
}

// TestRefundService_RejectRefund_DBError 测试数据库错误
func TestRefundService_RejectRefund_DBError(t *testing.T) {
	db := setupTestDB(t)
	svc := setupRefundService(db)
	ctx := context.Background()

	user := createTestUser(t, db, "13800138111")
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

	// 关闭数据库连接模拟错误
	sqlDB, _ := db.DB()
	sqlDB.Close()

	adminID := int64(99)
	err := svc.RejectRefund(ctx, adminID, refund.ID, "不同意")
	require.Error(t, err)
}
