// Package order 积分钩子单元测试
package order

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"

	"github.com/dumeirei/smart-locker-backend/internal/models"
)

type stubPointsAdder struct {
	addConsumeFn func(ctx context.Context, tx *gorm.DB, userID int64, amount float64, orderNo string) error
	refundFn     func(ctx context.Context, tx *gorm.DB, userID int64, amount float64, orderNo string) error
}

func (s *stubPointsAdder) AddConsumePointsTx(ctx context.Context, tx *gorm.DB, userID int64, amount float64, orderNo string) error {
	if s.addConsumeFn == nil {
		return nil
	}
	return s.addConsumeFn(ctx, tx, userID, amount, orderNo)
}

func (s *stubPointsAdder) RefundPointsTx(ctx context.Context, tx *gorm.DB, userID int64, amount float64, orderNo string) error {
	if s.refundFn == nil {
		return nil
	}
	return s.refundFn(ctx, tx, userID, amount, orderNo)
}

func setupPointsHookTestDB(t *testing.T) *gorm.DB {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Silent),
	})
	require.NoError(t, err)
	return db
}

func TestPointsHook_OnOrderCompleted(t *testing.T) {
	ctx := context.Background()
	db := setupPointsHookTestDB(t)

	t.Run("非已完成订单不触发", func(t *testing.T) {
		hook := NewPointsHook(db, &stubPointsAdder{
			addConsumeFn: func(ctx context.Context, tx *gorm.DB, userID int64, amount float64, orderNo string) error {
				t.Fatalf("should not be called")
				return nil
			},
		})

		require.NoError(t, hook.OnOrderCompleted(ctx, &models.Order{Status: models.OrderStatusPaid}))
	})

	t.Run("积分服务为空直接返回", func(t *testing.T) {
		hook := NewPointsHook(db, nil)
		require.NoError(t, hook.OnOrderCompleted(ctx, &models.Order{Status: models.OrderStatusCompleted}))
	})

	t.Run("会员套餐订单不触发积分", func(t *testing.T) {
		hook := NewPointsHook(db, &stubPointsAdder{
			addConsumeFn: func(ctx context.Context, tx *gorm.DB, userID int64, amount float64, orderNo string) error {
				t.Fatalf("should not be called")
				return nil
			},
		})
		require.NoError(t, hook.OnOrderCompleted(ctx, &models.Order{Status: models.OrderStatusCompleted, Type: "member_package"}))
	})

	t.Run("积分添加失败不影响订单完成", func(t *testing.T) {
		hook := NewPointsHook(db, &stubPointsAdder{
			addConsumeFn: func(ctx context.Context, tx *gorm.DB, userID int64, amount float64, orderNo string) error {
				return errors.New("boom")
			},
		})
		require.NoError(t, hook.OnOrderCompleted(ctx, &models.Order{ID: 1, OrderNo: "O1", UserID: 2, ActualAmount: 10, Status: models.OrderStatusCompleted}))
	})

	t.Run("正常添加积分", func(t *testing.T) {
		called := false
		hook := NewPointsHook(db, &stubPointsAdder{
			addConsumeFn: func(ctx context.Context, tx *gorm.DB, userID int64, amount float64, orderNo string) error {
				called = true
				assert.Equal(t, int64(2), userID)
				assert.Equal(t, 10.0, amount)
				assert.Equal(t, "O1", orderNo)
				return nil
			},
		})
		require.NoError(t, hook.OnOrderCompleted(ctx, &models.Order{ID: 1, OrderNo: "O1", UserID: 2, ActualAmount: 10, Status: models.OrderStatusCompleted}))
		assert.True(t, called)
	})
}

func TestPointsHook_OnOrderRefunded(t *testing.T) {
	ctx := context.Background()
	db := setupPointsHookTestDB(t)

	t.Run("积分服务为空直接返回", func(t *testing.T) {
		hook := NewPointsHook(db, nil)
		require.NoError(t, hook.OnOrderRefunded(ctx, &models.Order{ID: 1, OrderNo: "O1"}))
	})

	t.Run("会员套餐订单不触发积分", func(t *testing.T) {
		hook := NewPointsHook(db, &stubPointsAdder{
			refundFn: func(ctx context.Context, tx *gorm.DB, userID int64, amount float64, orderNo string) error {
				t.Fatalf("should not be called")
				return nil
			},
		})
		require.NoError(t, hook.OnOrderRefunded(ctx, &models.Order{ID: 1, OrderNo: "O1", Type: "member_package"}))
	})

	t.Run("积分扣减失败不影响退款流程", func(t *testing.T) {
		hook := NewPointsHook(db, &stubPointsAdder{
			refundFn: func(ctx context.Context, tx *gorm.DB, userID int64, amount float64, orderNo string) error {
				return errors.New("boom")
			},
		})
		require.NoError(t, hook.OnOrderRefunded(ctx, &models.Order{ID: 1, OrderNo: "O1", UserID: 2, ActualAmount: 10}))
	})

	t.Run("正常扣减积分", func(t *testing.T) {
		called := false
		hook := NewPointsHook(db, &stubPointsAdder{
			refundFn: func(ctx context.Context, tx *gorm.DB, userID int64, amount float64, orderNo string) error {
				called = true
				assert.Equal(t, int64(2), userID)
				assert.Equal(t, 10.0, amount)
				assert.Equal(t, "O1", orderNo)
				return nil
			},
		})
		require.NoError(t, hook.OnOrderRefunded(ctx, &models.Order{ID: 1, OrderNo: "O1", UserID: 2, ActualAmount: 10}))
		assert.True(t, called)
	})
}

type stubOrderEventHandler struct {
	onCompleted func(ctx context.Context, order *models.Order) error
	onRefunded  func(ctx context.Context, order *models.Order) error
}

func (s *stubOrderEventHandler) OnOrderCompleted(ctx context.Context, order *models.Order) error {
	if s.onCompleted == nil {
		return nil
	}
	return s.onCompleted(ctx, order)
}

func (s *stubOrderEventHandler) OnOrderRefunded(ctx context.Context, order *models.Order) error {
	if s.onRefunded == nil {
		return nil
	}
	return s.onRefunded(ctx, order)
}

func TestCompositeOrderEventHandler(t *testing.T) {
	ctx := context.Background()
	order := &models.Order{ID: 1, OrderNo: "O1"}

	t.Run("跳过 nil handler，且遇到错误继续执行", func(t *testing.T) {
		var completedCalls []string
		var refundedCalls []string

		h := NewCompositeOrderEventHandler(
			nil,
			&stubOrderEventHandler{
				onCompleted: func(ctx context.Context, order *models.Order) error {
					completedCalls = append(completedCalls, "h1")
					return errors.New("boom")
				},
				onRefunded: func(ctx context.Context, order *models.Order) error {
					refundedCalls = append(refundedCalls, "h1")
					return errors.New("boom")
				},
			},
			&stubOrderEventHandler{
				onCompleted: func(ctx context.Context, order *models.Order) error {
					completedCalls = append(completedCalls, "h2")
					return nil
				},
				onRefunded: func(ctx context.Context, order *models.Order) error {
					refundedCalls = append(refundedCalls, "h2")
					return nil
				},
			},
		)

		require.NoError(t, h.OnOrderCompleted(ctx, order))
		require.NoError(t, h.OnOrderRefunded(ctx, order))

		assert.Equal(t, []string{"h1", "h2"}, completedCalls)
		assert.Equal(t, []string{"h1", "h2"}, refundedCalls)
	})

	t.Run("AddHandler 忽略 nil", func(t *testing.T) {
		h := NewCompositeOrderEventHandler()
		h.AddHandler(nil)
		require.NoError(t, h.OnOrderCompleted(ctx, order))
	})

	t.Run("AddHandler 添加真实 handler", func(t *testing.T) {
		h := NewCompositeOrderEventHandler()
		called := false
		h.AddHandler(&stubOrderEventHandler{
			onCompleted: func(ctx context.Context, order *models.Order) error {
				called = true
				return nil
			},
		})
		require.NoError(t, h.OnOrderCompleted(ctx, order))
		assert.True(t, called)
	})
}
