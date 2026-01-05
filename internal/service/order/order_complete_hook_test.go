package order

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/dumeirei/smart-locker-backend/internal/models"
	"github.com/dumeirei/smart-locker-backend/internal/service/distribution"
)

type stubCommissionService struct {
	calculateFn func(ctx context.Context, req *distribution.CalculateRequest) (*distribution.CalculateResponse, error)
	cancelFn    func(ctx context.Context, orderID int64) error
}

func (s *stubCommissionService) Calculate(ctx context.Context, req *distribution.CalculateRequest) (*distribution.CalculateResponse, error) {
	if s.calculateFn == nil {
		return &distribution.CalculateResponse{}, nil
	}
	return s.calculateFn(ctx, req)
}

func (s *stubCommissionService) CancelByOrderID(ctx context.Context, orderID int64) error {
	if s.cancelFn == nil {
		return nil
	}
	return s.cancelFn(ctx, orderID)
}

func TestOrderCompleteHook_OnOrderCompleted(t *testing.T) {
	ctx := context.Background()

	t.Run("非已完成订单不触发", func(t *testing.T) {
		h := NewOrderCompleteHook(&stubCommissionService{
			calculateFn: func(ctx context.Context, req *distribution.CalculateRequest) (*distribution.CalculateResponse, error) {
				t.Fatalf("should not be called")
				return nil, nil
			},
		})

		err := h.OnOrderCompleted(ctx, &models.Order{Status: models.OrderStatusPaid})
		assert.NoError(t, err)
	})

	t.Run("佣金服务为空直接返回", func(t *testing.T) {
		h := NewOrderCompleteHook(nil)
		err := h.OnOrderCompleted(ctx, &models.Order{Status: models.OrderStatusCompleted})
		assert.NoError(t, err)
	})

	t.Run("佣金计算失败不影响订单完成", func(t *testing.T) {
		h := NewOrderCompleteHook(&stubCommissionService{
			calculateFn: func(ctx context.Context, req *distribution.CalculateRequest) (*distribution.CalculateResponse, error) {
				return nil, errors.New("boom")
			},
		})
		err := h.OnOrderCompleted(ctx, &models.Order{ID: 1, OrderNo: "O1", UserID: 2, ActualAmount: 10, Status: models.OrderStatusCompleted})
		assert.NoError(t, err)
	})

	t.Run("佣金计算成功", func(t *testing.T) {
		h := NewOrderCompleteHook(&stubCommissionService{
			calculateFn: func(ctx context.Context, req *distribution.CalculateRequest) (*distribution.CalculateResponse, error) {
				return &distribution.CalculateResponse{
					TotalAmount:       1,
					DirectCommission:  &models.Commission{Amount: 0.5},
					IndirectCommission: &models.Commission{Amount: 0.5},
				}, nil
			},
		})
		err := h.OnOrderCompleted(ctx, &models.Order{ID: 1, OrderNo: "O1", UserID: 2, ActualAmount: 10, Status: models.OrderStatusCompleted})
		assert.NoError(t, err)
	})
}

func TestOrderCompleteHook_OnOrderRefunded(t *testing.T) {
	ctx := context.Background()

	t.Run("佣金服务为空直接返回", func(t *testing.T) {
		h := NewOrderCompleteHook(nil)
		err := h.OnOrderRefunded(ctx, &models.Order{ID: 1, OrderNo: "O1"})
		assert.NoError(t, err)
	})

	t.Run("取消佣金失败不影响退款流程", func(t *testing.T) {
		h := NewOrderCompleteHook(&stubCommissionService{
			cancelFn: func(ctx context.Context, orderID int64) error {
				return errors.New("boom")
			},
		})
		err := h.OnOrderRefunded(ctx, &models.Order{ID: 1, OrderNo: "O1"})
		assert.NoError(t, err)
	})

	t.Run("取消佣金成功", func(t *testing.T) {
		h := NewOrderCompleteHook(&stubCommissionService{})
		err := h.OnOrderRefunded(ctx, &models.Order{ID: 1, OrderNo: "O1"})
		assert.NoError(t, err)
	})
}

func TestGetCommissionAmount(t *testing.T) {
	assert.Equal(t, 0.0, getCommissionAmount(nil))
	assert.Equal(t, 12.5, getCommissionAmount(&models.Commission{Amount: 12.5}))
}
