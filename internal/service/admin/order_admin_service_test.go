// Package admin 订单管理服务单元测试
package admin

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
)

// setupOrderAdminTestDB 创建订单管理测试数据库
func setupOrderAdminTestDB(t *testing.T) *gorm.DB {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Silent),
	})
	require.NoError(t, err)

	err = db.AutoMigrate(&models.User{}, &models.Order{}, &models.OrderItem{})
	require.NoError(t, err)

	return db
}

// setupOrderAdminService 创建测试用的 OrderAdminService
func setupOrderAdminService(t *testing.T) (*OrderAdminService, *gorm.DB) {
	db := setupOrderAdminTestDB(t)
	orderRepo := repository.NewOrderRepository(db)
	service := NewOrderAdminService(db, orderRepo)
	return service, db
}

// createTestUserForOrderAdmin 创建测试用户
func createTestUserForOrderAdmin(t *testing.T, db *gorm.DB) *models.User {
	phone := "13800138000"
	user := &models.User{
		Phone:    &phone,
		Nickname: "测试用户",
		Status:   models.UserStatusActive,
	}
	err := db.Create(user).Error
	require.NoError(t, err)
	return user
}

// createTestOrder 创建测试订单
func createTestOrder(t *testing.T, db *gorm.DB, userID int64, orderNo, orderType, status string) *models.Order {
	order := &models.Order{
		OrderNo:        orderNo,
		UserID:         userID,
		Type:           orderType,
		OriginalAmount: 100.00,
		DiscountAmount: 10.00,
		ActualAmount:   90.00,
		Status:         status,
	}
	err := db.Create(order).Error
	require.NoError(t, err)
	return order
}

func TestOrderAdminService_List(t *testing.T) {
	service, db := setupOrderAdminService(t)
	ctx := context.Background()

	user := createTestUserForOrderAdmin(t, db)

	// 创建多个订单
	createTestOrder(t, db, user.ID, "ORD001", models.OrderTypeRental, models.OrderStatusPending)
	createTestOrder(t, db, user.ID, "ORD002", models.OrderTypeMall, models.OrderStatusPaid)
	createTestOrder(t, db, user.ID, "ORD003", models.OrderTypeHotel, models.OrderStatusCompleted)
	createTestOrder(t, db, user.ID, "ORD004", models.OrderTypeMall, models.OrderStatusCancelled)

	t.Run("获取全部订单列表", func(t *testing.T) {
		results, total, err := service.List(ctx, 1, 10, nil)
		require.NoError(t, err)
		assert.Equal(t, int64(4), total)
		assert.Len(t, results, 4)
	})

	t.Run("按订单号筛选", func(t *testing.T) {
		filters := &OrderListFilters{OrderNo: "ORD00"}
		results, total, err := service.List(ctx, 1, 10, filters)
		require.NoError(t, err)
		assert.Equal(t, int64(4), total)
		assert.Len(t, results, 4)
	})

	t.Run("按类型筛选", func(t *testing.T) {
		filters := &OrderListFilters{Type: models.OrderTypeMall}
		results, total, err := service.List(ctx, 1, 10, filters)
		require.NoError(t, err)
		assert.Equal(t, int64(2), total)
		assert.Len(t, results, 2)
	})

	t.Run("按状态筛选", func(t *testing.T) {
		filters := &OrderListFilters{Status: models.OrderStatusCompleted}
		results, total, err := service.List(ctx, 1, 10, filters)
		require.NoError(t, err)
		assert.Equal(t, int64(1), total)
		assert.Len(t, results, 1)
	})

	t.Run("按用户ID筛选", func(t *testing.T) {
		filters := &OrderListFilters{UserID: user.ID}
		results, total, err := service.List(ctx, 1, 10, filters)
		require.NoError(t, err)
		assert.Equal(t, int64(4), total)
		assert.Len(t, results, 4)
	})

	t.Run("分页", func(t *testing.T) {
		results, total, err := service.List(ctx, 1, 2, nil)
		require.NoError(t, err)
		assert.Equal(t, int64(4), total)
		assert.Len(t, results, 2)
	})
}

func TestOrderAdminService_GetByID(t *testing.T) {
	service, db := setupOrderAdminService(t)
	ctx := context.Background()

	user := createTestUserForOrderAdmin(t, db)
	order := createTestOrder(t, db, user.ID, "ORD001", models.OrderTypeMall, models.OrderStatusPaid)

	t.Run("获取存在的订单", func(t *testing.T) {
		result, err := service.GetByID(ctx, order.ID)
		require.NoError(t, err)
		assert.Equal(t, order.ID, result.ID)
		assert.Equal(t, "ORD001", result.OrderNo)
	})

	t.Run("获取不存在的订单", func(t *testing.T) {
		_, err := service.GetByID(ctx, 99999)
		assert.Error(t, err)
	})
}

func TestOrderAdminService_GetByOrderNo(t *testing.T) {
	service, db := setupOrderAdminService(t)
	ctx := context.Background()

	user := createTestUserForOrderAdmin(t, db)
	createTestOrder(t, db, user.ID, "ORD_UNIQUE_001", models.OrderTypeMall, models.OrderStatusPaid)

	t.Run("获取存在的订单", func(t *testing.T) {
		result, err := service.GetByOrderNo(ctx, "ORD_UNIQUE_001")
		require.NoError(t, err)
		assert.Equal(t, "ORD_UNIQUE_001", result.OrderNo)
	})

	t.Run("获取不存在的订单", func(t *testing.T) {
		_, err := service.GetByOrderNo(ctx, "NONEXISTENT")
		assert.Error(t, err)
	})
}

func TestOrderAdminService_CancelOrder(t *testing.T) {
	service, db := setupOrderAdminService(t)
	ctx := context.Background()

	user := createTestUserForOrderAdmin(t, db)

	t.Run("取消待支付订单", func(t *testing.T) {
		order := createTestOrder(t, db, user.ID, "ORD_CANCEL_001", models.OrderTypeMall, models.OrderStatusPending)

		err := service.CancelOrder(ctx, order.ID, "用户申请取消")
		require.NoError(t, err)

		// 验证状态已更新
		var updated models.Order
		db.First(&updated, order.ID)
		assert.Equal(t, models.OrderStatusCancelled, updated.Status)
		assert.NotNil(t, updated.CancelledAt)
	})

	t.Run("无法取消已支付订单", func(t *testing.T) {
		order := createTestOrder(t, db, user.ID, "ORD_CANCEL_002", models.OrderTypeMall, models.OrderStatusPaid)

		err := service.CancelOrder(ctx, order.ID, "尝试取消")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "只能取消待支付的订单")
	})

	t.Run("无法取消已完成订单", func(t *testing.T) {
		order := createTestOrder(t, db, user.ID, "ORD_CANCEL_003", models.OrderTypeMall, models.OrderStatusCompleted)

		err := service.CancelOrder(ctx, order.ID, "尝试取消")
		assert.Error(t, err)
	})
}

func TestOrderAdminService_ShipOrder(t *testing.T) {
	service, db := setupOrderAdminService(t)
	ctx := context.Background()

	user := createTestUserForOrderAdmin(t, db)

	t.Run("商城订单发货", func(t *testing.T) {
		order := createTestOrder(t, db, user.ID, "ORD_SHIP_001", models.OrderTypeMall, models.OrderStatusPendingShip)

		err := service.ShipOrder(ctx, order.ID, "顺丰快递", "SF123456789")
		require.NoError(t, err)

		// 验证状态已更新
		var updated models.Order
		db.First(&updated, order.ID)
		assert.Equal(t, models.OrderStatusShipped, updated.Status)
		assert.NotNil(t, updated.ShippedAt)
	})

	t.Run("非商城订单不能发货", func(t *testing.T) {
		order := createTestOrder(t, db, user.ID, "ORD_SHIP_002", models.OrderTypeRental, models.OrderStatusPendingShip)

		err := service.ShipOrder(ctx, order.ID, "顺丰快递", "SF123456789")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "只能对商城订单执行发货操作")
	})

	t.Run("非待发货订单不能发货", func(t *testing.T) {
		order := createTestOrder(t, db, user.ID, "ORD_SHIP_003", models.OrderTypeMall, models.OrderStatusPaid)

		err := service.ShipOrder(ctx, order.ID, "顺丰快递", "SF123456789")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "只能对待发货订单执行发货操作")
	})
}

func TestOrderAdminService_ConfirmReceipt(t *testing.T) {
	service, db := setupOrderAdminService(t)
	ctx := context.Background()

	user := createTestUserForOrderAdmin(t, db)

	t.Run("确认收货", func(t *testing.T) {
		order := createTestOrder(t, db, user.ID, "ORD_RECV_001", models.OrderTypeMall, models.OrderStatusShipped)

		err := service.ConfirmReceipt(ctx, order.ID)
		require.NoError(t, err)

		// 验证状态已更新
		var updated models.Order
		db.First(&updated, order.ID)
		assert.Equal(t, models.OrderStatusCompleted, updated.Status)
		assert.NotNil(t, updated.ReceivedAt)
		assert.NotNil(t, updated.CompletedAt)
	})

	t.Run("非已发货订单不能确认收货", func(t *testing.T) {
		order := createTestOrder(t, db, user.ID, "ORD_RECV_002", models.OrderTypeMall, models.OrderStatusPaid)

		err := service.ConfirmReceipt(ctx, order.ID)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "只能对已发货订单确认收货")
	})
}

func TestOrderAdminService_AddRemark(t *testing.T) {
	service, db := setupOrderAdminService(t)
	ctx := context.Background()

	user := createTestUserForOrderAdmin(t, db)
	order := createTestOrder(t, db, user.ID, "ORD_REMARK_001", models.OrderTypeMall, models.OrderStatusPaid)

	t.Run("添加备注", func(t *testing.T) {
		err := service.AddRemark(ctx, order.ID, "客户要求加急处理")
		require.NoError(t, err)

		// 验证备注已添加
		var updated models.Order
		db.First(&updated, order.ID)
		assert.NotNil(t, updated.Remark)
		assert.Equal(t, "客户要求加急处理", *updated.Remark)
	})
}

func TestOrderAdminService_GetStatistics(t *testing.T) {
	service, db := setupOrderAdminService(t)
	ctx := context.Background()

	user := createTestUserForOrderAdmin(t, db)

	// 创建多种状态和类型的订单
	createTestOrder(t, db, user.ID, "ORD_STAT_001", models.OrderTypeRental, models.OrderStatusPending)
	createTestOrder(t, db, user.ID, "ORD_STAT_002", models.OrderTypeMall, models.OrderStatusPaid)

	// 创建已完成订单
	completedOrder := createTestOrder(t, db, user.ID, "ORD_STAT_003", models.OrderTypeHotel, models.OrderStatusCompleted)
	completedOrder.CompletedAt = func() *time.Time { t := time.Now(); return &t }()
	db.Save(completedOrder)

	createTestOrder(t, db, user.ID, "ORD_STAT_004", models.OrderTypeMall, models.OrderStatusCompleted)

	stats, err := service.GetStatistics(ctx)
	require.NoError(t, err)

	t.Run("总订单数", func(t *testing.T) {
		assert.Equal(t, int64(4), stats.TotalOrders)
	})

	t.Run("状态统计", func(t *testing.T) {
		assert.Equal(t, int64(1), stats.StatusCounts[models.OrderStatusPending])
		assert.Equal(t, int64(1), stats.StatusCounts[models.OrderStatusPaid])
		assert.Equal(t, int64(2), stats.StatusCounts[models.OrderStatusCompleted])
	})

	t.Run("类型统计", func(t *testing.T) {
		assert.Equal(t, int64(1), stats.TypeCounts[models.OrderTypeRental])
		assert.Equal(t, int64(2), stats.TypeCounts[models.OrderTypeMall])
		assert.Equal(t, int64(1), stats.TypeCounts[models.OrderTypeHotel])
	})
}

func TestOrderAdminService_ExportOrders(t *testing.T) {
	service, db := setupOrderAdminService(t)
	ctx := context.Background()

	user := createTestUserForOrderAdmin(t, db)

	// 创建测试订单
	createTestOrder(t, db, user.ID, "EXP_001", models.OrderTypeMall, models.OrderStatusPaid)
	createTestOrder(t, db, user.ID, "EXP_002", models.OrderTypeRental, models.OrderStatusCompleted)

	t.Run("导出全部订单", func(t *testing.T) {
		orders, err := service.ExportOrders(ctx, nil)
		require.NoError(t, err)
		assert.Len(t, orders, 2)
	})

	t.Run("按类型导出", func(t *testing.T) {
		filters := &OrderListFilters{Type: models.OrderTypeMall}
		orders, err := service.ExportOrders(ctx, filters)
		require.NoError(t, err)
		assert.Len(t, orders, 1)
		assert.Equal(t, models.OrderTypeMall, orders[0].Type)
	})
}

func TestToOrderListResponse(t *testing.T) {
	service, _ := setupOrderAdminService(t)

	phone := "13800138000"
	now := time.Now()
	order := &models.Order{
		ID:             1,
		OrderNo:        "ORD123456",
		UserID:         100,
		User:           &models.User{Phone: &phone},
		Type:           models.OrderTypeMall,
		OriginalAmount: 100.00,
		DiscountAmount: 10.00,
		ActualAmount:   90.00,
		Status:         models.OrderStatusPaid,
		PaidAt:         &now,
		CreatedAt:      now,
	}

	resp := service.toOrderListResponse(order)

	assert.Equal(t, int64(1), resp.ID)
	assert.Equal(t, "ORD123456", resp.OrderNo)
	assert.Equal(t, int64(100), resp.UserID)
	assert.Equal(t, "13800138000", resp.UserPhone)
	assert.Equal(t, models.OrderTypeMall, resp.Type)
	assert.Equal(t, 100.00, resp.OriginalAmount)
	assert.Equal(t, 10.00, resp.DiscountAmount)
	assert.Equal(t, 90.00, resp.ActualAmount)
	assert.Equal(t, models.OrderStatusPaid, resp.Status)
	assert.NotNil(t, resp.PaidAt)
}
