// Package repository 支付仓储单元测试
package repository

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
)

// setupPaymentTestDB 创建支付测试数据库
func setupPaymentTestDB(t *testing.T) *gorm.DB {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Silent),
	})
	require.NoError(t, err)

	err = db.AutoMigrate(
		&models.Payment{},
		&models.Refund{},
		&models.Order{},
		&models.User{},
		&models.MemberLevel{},
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

func createPaymentTestUser(t *testing.T, db *gorm.DB, phone string) *models.User {
	t.Helper()

	user := &models.User{
		Phone:         &phone,
		Nickname:      "测试用户",
		MemberLevelID: 1,
		Status:        models.UserStatusActive,
	}
	require.NoError(t, db.Create(user).Error)
	return user
}

func createPaymentTestOrder(t *testing.T, db *gorm.DB, userID int64, orderNo string) *models.Order {
	t.Helper()

	now := time.Now()
	order := &models.Order{
		OrderNo:        orderNo,
		UserID:         userID,
		Type:           models.OrderTypeMall,
		OriginalAmount: 100.0,
		ActualAmount:   100.0,
		Status:         models.OrderStatusPaid,
		PaidAt:         &now,
	}
	require.NoError(t, db.Create(order).Error)
	return order
}

func createTestPaymentForRepo(t *testing.T, db *gorm.DB, userID, orderID int64, paymentNo string, status int8) *models.Payment {
	t.Helper()

	now := time.Now()
	expiredAt := now.Add(30 * time.Minute)
	payment := &models.Payment{
		PaymentNo:     paymentNo,
		UserID:        userID,
		OrderID:       orderID,
		Amount:        100.0,
		PaymentMethod: "wechat",
		Status:        status,
		ExpiredAt:     &expiredAt,
		PaidAt:        &now,
	}
	require.NoError(t, db.Create(payment).Error)
	return payment
}

func TestPaymentRepository_Create(t *testing.T) {
	db := setupPaymentTestDB(t)
	repo := NewPaymentRepository(db)
	ctx := context.Background()

	user := createPaymentTestUser(t, db, "13800138000")
	order := createPaymentTestOrder(t, db, user.ID, "ORD001")

	payment := &models.Payment{
		PaymentNo:     "PAY001",
		UserID:        user.ID,
		OrderID:       order.ID,
		Amount:        100.0,
		PaymentMethod: "wechat",
		Status:        models.PaymentStatusPending,
	}

	err := repo.Create(ctx, payment)
	require.NoError(t, err)
	assert.NotZero(t, payment.ID)

	// 验证支付记录已创建
	var found models.Payment
	db.First(&found, payment.ID)
	assert.Equal(t, "PAY001", found.PaymentNo)
}

func TestPaymentRepository_GetByID(t *testing.T) {
	db := setupPaymentTestDB(t)
	repo := NewPaymentRepository(db)
	ctx := context.Background()

	user := createPaymentTestUser(t, db, "13800138001")
	order := createPaymentTestOrder(t, db, user.ID, "ORD002")
	payment := createTestPaymentForRepo(t, db, user.ID, order.ID, "PAY002", models.PaymentStatusSuccess)

	t.Run("获取存在的支付记录", func(t *testing.T) {
		found, err := repo.GetByID(ctx, payment.ID)
		require.NoError(t, err)
		assert.Equal(t, payment.ID, found.ID)
		assert.Equal(t, payment.PaymentNo, found.PaymentNo)
	})

	t.Run("获取不存在的支付记录", func(t *testing.T) {
		_, err := repo.GetByID(ctx, 99999)
		assert.Error(t, err)
	})
}

func TestPaymentRepository_GetByPaymentNo(t *testing.T) {
	db := setupPaymentTestDB(t)
	repo := NewPaymentRepository(db)
	ctx := context.Background()

	user := createPaymentTestUser(t, db, "13800138002")
	order := createPaymentTestOrder(t, db, user.ID, "ORD003")
	payment := createTestPaymentForRepo(t, db, user.ID, order.ID, "PAY003", models.PaymentStatusSuccess)

	t.Run("根据支付单号获取", func(t *testing.T) {
		found, err := repo.GetByPaymentNo(ctx, payment.PaymentNo)
		require.NoError(t, err)
		assert.Equal(t, payment.ID, found.ID)
	})

	t.Run("获取不存在的支付单号", func(t *testing.T) {
		_, err := repo.GetByPaymentNo(ctx, "INVALID_NO")
		assert.Error(t, err)
	})
}

func TestPaymentRepository_GetByTransactionID(t *testing.T) {
	db := setupPaymentTestDB(t)
	repo := NewPaymentRepository(db)
	ctx := context.Background()

	user := createPaymentTestUser(t, db, "13800138003")
	order := createPaymentTestOrder(t, db, user.ID, "ORD004")
	payment := createTestPaymentForRepo(t, db, user.ID, order.ID, "PAY004", models.PaymentStatusSuccess)

	// 设置第三方交易号
	transactionID := "TX12345678"
	db.Model(&payment).Update("transaction_id", transactionID)

	t.Run("根据第三方交易号获取", func(t *testing.T) {
		found, err := repo.GetByTransactionID(ctx, transactionID)
		require.NoError(t, err)
		assert.Equal(t, payment.ID, found.ID)
	})

	t.Run("获取不存在的交易号", func(t *testing.T) {
		_, err := repo.GetByTransactionID(ctx, "INVALID_TX")
		assert.Error(t, err)
	})
}

func TestPaymentRepository_GetByOrder(t *testing.T) {
	db := setupPaymentTestDB(t)
	repo := NewPaymentRepository(db)
	ctx := context.Background()

	user := createPaymentTestUser(t, db, "13800138004")
	order := createPaymentTestOrder(t, db, user.ID, "ORD005")
	payment := createTestPaymentForRepo(t, db, user.ID, order.ID, "PAY005", models.PaymentStatusSuccess)

	t.Run("获取订单的成功支付记录", func(t *testing.T) {
		found, err := repo.GetByOrder(ctx, order.ID)
		require.NoError(t, err)
		assert.Equal(t, payment.ID, found.ID)
	})

	t.Run("获取无成功支付的订单", func(t *testing.T) {
		newOrder := createPaymentTestOrder(t, db, user.ID, "ORD005_NEW")
		// 创建待支付记录
		createTestPaymentForRepo(t, db, user.ID, newOrder.ID, "PAY005_PENDING", models.PaymentStatusPending)

		_, err := repo.GetByOrder(ctx, newOrder.ID)
		assert.Error(t, err)
	})
}

func TestPaymentRepository_Update(t *testing.T) {
	db := setupPaymentTestDB(t)
	repo := NewPaymentRepository(db)
	ctx := context.Background()

	user := createPaymentTestUser(t, db, "13800138005")
	order := createPaymentTestOrder(t, db, user.ID, "ORD006")
	payment := createTestPaymentForRepo(t, db, user.ID, order.ID, "PAY006", models.PaymentStatusPending)

	payment.Status = models.PaymentStatusSuccess
	err := repo.Update(ctx, payment)
	require.NoError(t, err)

	var found models.Payment
	db.First(&found, payment.ID)
	assert.Equal(t, int8(models.PaymentStatusSuccess), found.Status)
}

func TestPaymentRepository_UpdateFields(t *testing.T) {
	db := setupPaymentTestDB(t)
	repo := NewPaymentRepository(db)
	ctx := context.Background()

	user := createPaymentTestUser(t, db, "13800138006")
	order := createPaymentTestOrder(t, db, user.ID, "ORD007")
	payment := createTestPaymentForRepo(t, db, user.ID, order.ID, "PAY007", models.PaymentStatusPending)

	txID := "TX_NEW_123"
	err := repo.UpdateFields(ctx, payment.ID, map[string]interface{}{
		"status":         models.PaymentStatusSuccess,
		"transaction_id": &txID,
	})
	require.NoError(t, err)

	var found models.Payment
	db.First(&found, payment.ID)
	assert.Equal(t, int8(models.PaymentStatusSuccess), found.Status)
	assert.NotNil(t, found.TransactionID)
	assert.Equal(t, txID, *found.TransactionID)
}

func TestPaymentRepository_List(t *testing.T) {
	db := setupPaymentTestDB(t)
	repo := NewPaymentRepository(db)
	ctx := context.Background()

	user := createPaymentTestUser(t, db, "13800138007")

	// 创建多个支付记录
	for i := 0; i < 5; i++ {
		order := createPaymentTestOrder(t, db, user.ID, fmt.Sprintf("ORD_LIST_%d", i))
		createTestPaymentForRepo(t, db, user.ID, order.ID, fmt.Sprintf("PAY_LIST_%d", i), models.PaymentStatusSuccess)
	}

	t.Run("获取支付记录列表", func(t *testing.T) {
		payments, total, err := repo.List(ctx, 0, 10, nil)
		require.NoError(t, err)
		assert.True(t, total >= 5)
		assert.True(t, len(payments) >= 5)
	})

	t.Run("分页获取", func(t *testing.T) {
		payments, total, err := repo.List(ctx, 0, 2, nil)
		require.NoError(t, err)
		assert.True(t, total >= 5)
		assert.Len(t, payments, 2)
	})

	t.Run("按用户筛选", func(t *testing.T) {
		payments, _, err := repo.List(ctx, 0, 10, map[string]interface{}{
			"user_id": user.ID,
		})
		require.NoError(t, err)
		for _, p := range payments {
			assert.Equal(t, user.ID, p.UserID)
		}
	})

	t.Run("按状态筛选", func(t *testing.T) {
		payments, _, err := repo.List(ctx, 0, 10, map[string]interface{}{
			"status": int8(models.PaymentStatusSuccess),
		})
		require.NoError(t, err)
		for _, p := range payments {
			assert.Equal(t, int8(models.PaymentStatusSuccess), p.Status)
		}
	})

	t.Run("按支付方式筛选", func(t *testing.T) {
		payments, _, err := repo.List(ctx, 0, 10, map[string]interface{}{
			"payment_method": "wechat",
		})
		require.NoError(t, err)
		for _, p := range payments {
			assert.Equal(t, "wechat", p.PaymentMethod)
		}
	})
}

func TestPaymentRepository_GetPendingExpired(t *testing.T) {
	db := setupPaymentTestDB(t)
	repo := NewPaymentRepository(db)
	ctx := context.Background()

	user := createPaymentTestUser(t, db, "13800138008")

	// 创建已过期的待支付记录
	order1 := createPaymentTestOrder(t, db, user.ID, "ORD_EXPIRED")
	expiredAt := time.Now().Add(-1 * time.Hour)
	expiredPayment := &models.Payment{
		PaymentNo:     "PAY_EXPIRED",
		UserID:        user.ID,
		OrderID:       order1.ID,
		Amount:        100.0,
		PaymentMethod: "wechat",
		Status:        models.PaymentStatusPending,
		ExpiredAt:     &expiredAt,
	}
	db.Create(expiredPayment)

	// 创建未过期的待支付记录
	order2 := createPaymentTestOrder(t, db, user.ID, "ORD_VALID")
	futureExpiredAt := time.Now().Add(1 * time.Hour)
	validPayment := &models.Payment{
		PaymentNo:     "PAY_VALID",
		UserID:        user.ID,
		OrderID:       order2.ID,
		Amount:        100.0,
		PaymentMethod: "wechat",
		Status:        models.PaymentStatusPending,
		ExpiredAt:     &futureExpiredAt,
	}
	db.Create(validPayment)

	payments, err := repo.GetPendingExpired(ctx, time.Now(), 10)
	require.NoError(t, err)

	// 应只包含过期记录
	foundExpired := false
	foundValid := false
	for _, p := range payments {
		if p.ID == expiredPayment.ID {
			foundExpired = true
		}
		if p.ID == validPayment.ID {
			foundValid = true
		}
	}
	assert.True(t, foundExpired)
	assert.False(t, foundValid)
}

// ================== RefundRepository Tests ==================

func createTestRefundForRepo(t *testing.T, db *gorm.DB, userID, orderID, paymentID int64, refundNo string, status int8) *models.Refund {
	t.Helper()

	refund := &models.Refund{
		RefundNo:  refundNo,
		OrderID:   orderID,
		PaymentID: paymentID,
		UserID:    userID,
		Amount:    50.0,
		Reason:    "测试退款",
		Status:    status,
	}
	require.NoError(t, db.Create(refund).Error)
	return refund
}

func TestRefundRepository_Create(t *testing.T) {
	db := setupPaymentTestDB(t)
	repo := NewRefundRepository(db)
	ctx := context.Background()

	user := createPaymentTestUser(t, db, "13800138010")
	order := createPaymentTestOrder(t, db, user.ID, "ORD_REF_1")
	payment := createTestPaymentForRepo(t, db, user.ID, order.ID, "PAY_REF_1", models.PaymentStatusSuccess)

	refund := &models.Refund{
		RefundNo:  "REF001",
		OrderID:   order.ID,
		PaymentID: payment.ID,
		UserID:    user.ID,
		Amount:    50.0,
		Reason:    "测试退款",
		Status:    models.RefundStatusPending,
	}

	err := repo.Create(ctx, refund)
	require.NoError(t, err)
	assert.NotZero(t, refund.ID)
}

func TestRefundRepository_GetByID(t *testing.T) {
	db := setupPaymentTestDB(t)
	repo := NewRefundRepository(db)
	ctx := context.Background()

	user := createPaymentTestUser(t, db, "13800138011")
	order := createPaymentTestOrder(t, db, user.ID, "ORD_REF_2")
	payment := createTestPaymentForRepo(t, db, user.ID, order.ID, "PAY_REF_2", models.PaymentStatusSuccess)
	refund := createTestRefundForRepo(t, db, user.ID, order.ID, payment.ID, "REF002", models.RefundStatusPending)

	t.Run("获取存在的退款记录", func(t *testing.T) {
		found, err := repo.GetByID(ctx, refund.ID)
		require.NoError(t, err)
		assert.Equal(t, refund.ID, found.ID)
		assert.Equal(t, refund.RefundNo, found.RefundNo)
	})

	t.Run("获取不存在的退款记录", func(t *testing.T) {
		_, err := repo.GetByID(ctx, 99999)
		assert.Error(t, err)
	})
}

func TestRefundRepository_GetByRefundNo(t *testing.T) {
	db := setupPaymentTestDB(t)
	repo := NewRefundRepository(db)
	ctx := context.Background()

	user := createPaymentTestUser(t, db, "13800138012")
	order := createPaymentTestOrder(t, db, user.ID, "ORD_REF_3")
	payment := createTestPaymentForRepo(t, db, user.ID, order.ID, "PAY_REF_3", models.PaymentStatusSuccess)
	refund := createTestRefundForRepo(t, db, user.ID, order.ID, payment.ID, "REF003", models.RefundStatusPending)

	t.Run("根据退款单号获取", func(t *testing.T) {
		found, err := repo.GetByRefundNo(ctx, refund.RefundNo)
		require.NoError(t, err)
		assert.Equal(t, refund.ID, found.ID)
	})

	t.Run("获取不存在的退款单号", func(t *testing.T) {
		_, err := repo.GetByRefundNo(ctx, "INVALID_REF")
		assert.Error(t, err)
	})
}

func TestRefundRepository_Update(t *testing.T) {
	db := setupPaymentTestDB(t)
	repo := NewRefundRepository(db)
	ctx := context.Background()

	user := createPaymentTestUser(t, db, "13800138013")
	order := createPaymentTestOrder(t, db, user.ID, "ORD_REF_4")
	payment := createTestPaymentForRepo(t, db, user.ID, order.ID, "PAY_REF_4", models.PaymentStatusSuccess)
	refund := createTestRefundForRepo(t, db, user.ID, order.ID, payment.ID, "REF004", models.RefundStatusPending)

	refund.Status = models.RefundStatusSuccess
	err := repo.Update(ctx, refund)
	require.NoError(t, err)

	var found models.Refund
	db.First(&found, refund.ID)
	assert.Equal(t, int8(models.RefundStatusSuccess), found.Status)
}

func TestRefundRepository_ListByPayment(t *testing.T) {
	db := setupPaymentTestDB(t)
	repo := NewRefundRepository(db)
	ctx := context.Background()

	user := createPaymentTestUser(t, db, "13800138014")
	order := createPaymentTestOrder(t, db, user.ID, "ORD_REF_5")
	payment := createTestPaymentForRepo(t, db, user.ID, order.ID, "PAY_REF_5", models.PaymentStatusSuccess)

	// 创建多个退款记录
	for i := 0; i < 3; i++ {
		createTestRefundForRepo(t, db, user.ID, order.ID, payment.ID, fmt.Sprintf("REF_PAY_%d", i), models.RefundStatusPending)
	}

	refunds, err := repo.ListByPayment(ctx, payment.ID)
	require.NoError(t, err)
	assert.Len(t, refunds, 3)
}

func TestRefundRepository_GetTotalRefunded(t *testing.T) {
	db := setupPaymentTestDB(t)
	repo := NewRefundRepository(db)
	ctx := context.Background()

	user := createPaymentTestUser(t, db, "13800138015")
	order := createPaymentTestOrder(t, db, user.ID, "ORD_REF_6")
	payment := createTestPaymentForRepo(t, db, user.ID, order.ID, "PAY_REF_6", models.PaymentStatusSuccess)

	// 创建不同状态的退款记录
	createTestRefundForRepo(t, db, user.ID, order.ID, payment.ID, "REF_TOTAL_1", models.RefundStatusSuccess)  // 50.0
	createTestRefundForRepo(t, db, user.ID, order.ID, payment.ID, "REF_TOTAL_2", models.RefundStatusPending)  // 50.0
	createTestRefundForRepo(t, db, user.ID, order.ID, payment.ID, "REF_TOTAL_3", models.RefundStatusRejected) // 不计入

	total, err := repo.GetTotalRefunded(ctx, payment.ID)
	require.NoError(t, err)
	// 成功和待处理的退款都计入
	assert.Equal(t, 100.0, total)
}

func TestRefundRepository_GetByOrderID(t *testing.T) {
	db := setupPaymentTestDB(t)
	repo := NewRefundRepository(db)
	ctx := context.Background()

	user := createPaymentTestUser(t, db, "13800138016")
	order := createPaymentTestOrder(t, db, user.ID, "ORD_REF_7")
	payment := createTestPaymentForRepo(t, db, user.ID, order.ID, "PAY_REF_7", models.PaymentStatusSuccess)
	refund := createTestRefundForRepo(t, db, user.ID, order.ID, payment.ID, "REF_ORDER", models.RefundStatusPending)

	t.Run("根据订单ID获取退款记录", func(t *testing.T) {
		found, err := repo.GetByOrderID(ctx, order.ID)
		require.NoError(t, err)
		assert.Equal(t, refund.ID, found.ID)
	})

	t.Run("获取无退款的订单", func(t *testing.T) {
		newOrder := createPaymentTestOrder(t, db, user.ID, "ORD_NO_REF")
		_, err := repo.GetByOrderID(ctx, newOrder.ID)
		assert.Error(t, err)
	})
}

func TestRefundRepository_List(t *testing.T) {
	db := setupPaymentTestDB(t)
	repo := NewRefundRepository(db)
	ctx := context.Background()

	user := createPaymentTestUser(t, db, "13800138017")

	// 创建多个退款记录
	for i := 0; i < 5; i++ {
		order := createPaymentTestOrder(t, db, user.ID, fmt.Sprintf("ORD_REF_LIST_%d", i))
		payment := createTestPaymentForRepo(t, db, user.ID, order.ID, fmt.Sprintf("PAY_REF_LIST_%d", i), models.PaymentStatusSuccess)
		createTestRefundForRepo(t, db, user.ID, order.ID, payment.ID, fmt.Sprintf("REF_LIST_%d", i), models.RefundStatusPending)
	}

	t.Run("获取退款记录列表", func(t *testing.T) {
		refunds, total, err := repo.List(ctx, RefundListParams{
			Offset: 0,
			Limit:  10,
		})
		require.NoError(t, err)
		assert.True(t, total >= 5)
		assert.True(t, len(refunds) >= 5)
	})

	t.Run("按用户筛选", func(t *testing.T) {
		refunds, _, err := repo.List(ctx, RefundListParams{
			Offset: 0,
			Limit:  10,
			UserID: user.ID,
		})
		require.NoError(t, err)
		for _, r := range refunds {
			assert.Equal(t, user.ID, r.UserID)
		}
	})

	t.Run("按状态筛选", func(t *testing.T) {
		status := int8(models.RefundStatusPending)
		refunds, _, err := repo.List(ctx, RefundListParams{
			Offset: 0,
			Limit:  10,
			Status: &status,
		})
		require.NoError(t, err)
		for _, r := range refunds {
			assert.Equal(t, int8(models.RefundStatusPending), r.Status)
		}
	})
}

func TestRefundRepository_ExistsPendingByOrderID(t *testing.T) {
	db := setupPaymentTestDB(t)
	repo := NewRefundRepository(db)
	ctx := context.Background()

	user := createPaymentTestUser(t, db, "13800138018")

	t.Run("有待处理退款", func(t *testing.T) {
		order := createPaymentTestOrder(t, db, user.ID, "ORD_PENDING_REF")
		payment := createTestPaymentForRepo(t, db, user.ID, order.ID, "PAY_PENDING_REF", models.PaymentStatusSuccess)
		createTestRefundForRepo(t, db, user.ID, order.ID, payment.ID, "REF_PENDING", models.RefundStatusPending)

		exists, err := repo.ExistsPendingByOrderID(ctx, order.ID)
		require.NoError(t, err)
		assert.True(t, exists)
	})

	t.Run("无待处理退款", func(t *testing.T) {
		order := createPaymentTestOrder(t, db, user.ID, "ORD_NO_PENDING_REF")
		payment := createTestPaymentForRepo(t, db, user.ID, order.ID, "PAY_NO_PENDING_REF", models.PaymentStatusSuccess)
		createTestRefundForRepo(t, db, user.ID, order.ID, payment.ID, "REF_SUCCESS", models.RefundStatusSuccess)

		exists, err := repo.ExistsPendingByOrderID(ctx, order.ID)
		require.NoError(t, err)
		assert.False(t, exists)
	})
}

func TestPaymentRepository_EdgeCases(t *testing.T) {
	db := setupPaymentTestDB(t)
	paymentRepo := NewPaymentRepository(db)
	refundRepo := NewRefundRepository(db)
	ctx := context.Background()

	t.Run("空支付记录列表", func(t *testing.T) {
		payments, total, err := paymentRepo.List(ctx, 0, 10, map[string]interface{}{
			"user_id": int64(99999),
		})
		require.NoError(t, err)
		assert.Equal(t, int64(0), total)
		assert.Empty(t, payments)
	})

	t.Run("空退款列表", func(t *testing.T) {
		refunds, total, err := refundRepo.List(ctx, RefundListParams{
			Offset: 0,
			Limit:  10,
			UserID: 99999,
		})
		require.NoError(t, err)
		assert.Equal(t, int64(0), total)
		assert.Empty(t, refunds)
	})
}
