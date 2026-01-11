package admin

import (
	"context"
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"

	"github.com/dumeirei/smart-locker-backend/internal/models"
)

func setupDashboardServiceTestDB(t *testing.T) *gorm.DB {
	t.Helper()

	dsn := fmt.Sprintf("file:%s?mode=memory&cache=shared", strings.ReplaceAll(t.Name(), "/", "_"))
	db, err := gorm.Open(sqlite.Open(dsn), &gorm.Config{Logger: logger.Default.LogMode(logger.Silent)})
	require.NoError(t, err)

	sqlDB, err := db.DB()
	require.NoError(t, err)
	sqlDB.SetMaxOpenConns(1)
	sqlDB.SetMaxIdleConns(1)

	require.NoError(t, db.AutoMigrate(
		&models.User{},
		&models.Order{},
		&models.Payment{},
		&models.Device{},
		&models.Merchant{},
		&models.Rental{},
		&models.Booking{},
	))
	return db
}

func TestDashboardService_GetPlatformOverview(t *testing.T) {
	db := setupDashboardServiceTestDB(t)
	svc := NewDashboardService(db)
	ctx := context.Background()

	user := &models.User{Nickname: "U1", MemberLevelID: 1, Status: models.UserStatusActive}
	require.NoError(t, db.Create(user).Error)

	merchant := &models.Merchant{Name: "M1", ContactName: "C", ContactPhone: "138", CommissionRate: 0.2, SettlementType: models.SettlementTypeMonthly, Status: models.MerchantStatusActive}
	require.NoError(t, db.Create(merchant).Error)

	deviceOnline := &models.Device{
		DeviceNo:     "D1",
		Name:         "设备1",
		Type:         models.DeviceTypeStandard,
		VenueID:      1,
		QRCode:       "qr",
		ProductName:  "商品",
		OnlineStatus: models.DeviceOnline,
		Status:       models.DeviceStatusActive,
	}
	deviceFault := &models.Device{
		DeviceNo:     "D2",
		Name:         "设备2",
		Type:         models.DeviceTypeStandard,
		VenueID:      1,
		QRCode:       "qr",
		ProductName:  "商品",
		OnlineStatus: models.DeviceOffline,
		Status:       models.DeviceStatusFault,
	}
	require.NoError(t, db.Create(deviceOnline).Error)
	require.NoError(t, db.Create(deviceFault).Error)

	order := &models.Order{
		OrderNo:         "O1",
		UserID:          user.ID,
		Type:            models.OrderTypeRental,
		OriginalAmount:  10,
		DiscountAmount:  0,
		ActualAmount:    10,
		DepositAmount:   0,
		Status:          models.OrderStatusPending,
	}
	require.NoError(t, db.Create(order).Error)

	now := time.Now()
	payment := &models.Payment{
		PaymentNo:      "P1",
		OrderID:        order.ID,
		OrderNo:        order.OrderNo,
		UserID:         user.ID,
		Amount:         10,
		PaymentMethod:  models.PaymentMethodBalance,
		PaymentChannel: models.PaymentChannelMiniProgram,
		Status:         models.PaymentStatusSuccess,
		PaidAt:         &now,
	}
	require.NoError(t, db.Create(payment).Error)

	rental := &models.Rental{
		OrderID:       order.ID,
		UserID:        user.ID,
		DeviceID:      1,
		DurationHours: 1,
		RentalFee:     10,
		Deposit:       0,
		OvertimeRate:  0,
		Status:        models.RentalStatusInUse,
	}
	require.NoError(t, db.Create(rental).Error)

	booking := &models.Booking{
		BookingNo:        "B1",
		OrderID:          order.ID,
		UserID:           user.ID,
		HotelID:          1,
		RoomID:           1,
		CheckInTime:      now.Add(-time.Hour),
		CheckOutTime:     now.Add(time.Hour),
		DurationHours:    2,
		Amount:           10,
		VerificationCode: "V1234567890123456789",
		UnlockCode:       "123456",
		QRCode:           "/qr",
		Status:           models.BookingStatusVerified,
	}
	require.NoError(t, db.Create(booking).Error)

	overview, err := svc.GetPlatformOverview(ctx)
	require.NoError(t, err)
	require.NotNil(t, overview)

	assert.Equal(t, int64(1), overview.TotalUsers)
	assert.Equal(t, int64(1), overview.TotalOrders)
	assert.Equal(t, int64(1), overview.PendingOrders)
	assert.Equal(t, float64(10), overview.TotalRevenue)
	assert.Equal(t, int64(2), overview.TotalDevices)
	assert.Equal(t, int64(1), overview.OnlineDevices)
	assert.Equal(t, int64(1), overview.OfflineDevices)
	assert.Equal(t, int64(1), overview.FaultyDevices)
	assert.Equal(t, int64(1), overview.TotalMerchants)
	assert.Equal(t, int64(1), overview.ActiveMerchants)
	assert.Equal(t, int64(1), overview.ActiveRentals)
	assert.Equal(t, int64(1), overview.ActiveBookings)
}

func TestDashboardService_GetOrderTrend_Bounds(t *testing.T) {
	db := setupDashboardServiceTestDB(t)
	svc := NewDashboardService(db)
	ctx := context.Background()

	trends, err := svc.GetOrderTrend(ctx, 0)
	require.NoError(t, err)
	require.Len(t, trends, 7)

	trends, err = svc.GetOrderTrend(ctx, 999)
	require.NoError(t, err)
	require.Len(t, trends, 30)
}

