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

func TestDashboardService_GetDeviceStatusSummary(t *testing.T) {
	db := setupDashboardServiceTestDB(t)
	svc := NewDashboardService(db)
	ctx := context.Background()

	// 创建不同状态的设备
	devices := []models.Device{
		{DeviceNo: "D1", Name: "设备1", Type: models.DeviceTypeStandard, VenueID: 1, QRCode: "qr1", ProductName: "商品", OnlineStatus: models.DeviceOnline, Status: models.DeviceStatusActive, RentalStatus: models.DeviceRentalFree},
		{DeviceNo: "D2", Name: "设备2", Type: models.DeviceTypeStandard, VenueID: 1, QRCode: "qr2", ProductName: "商品", OnlineStatus: models.DeviceOnline, Status: models.DeviceStatusActive, RentalStatus: models.DeviceRentalInUse},
		{DeviceNo: "D3", Name: "设备3", Type: models.DeviceTypeStandard, VenueID: 1, QRCode: "qr3", ProductName: "商品", OnlineStatus: models.DeviceOffline, Status: models.DeviceStatusMaintenance, RentalStatus: models.DeviceRentalFree},
		{DeviceNo: "D4", Name: "设备4", Type: models.DeviceTypeStandard, VenueID: 1, QRCode: "qr4", ProductName: "商品", OnlineStatus: models.DeviceOffline, Status: models.DeviceStatusFault, RentalStatus: models.DeviceRentalFree},
	}
	for i := range devices {
		require.NoError(t, db.Create(&devices[i]).Error)
	}

	summary, err := svc.GetDeviceStatusSummary(ctx)
	require.NoError(t, err)
	require.NotNil(t, summary)
	assert.GreaterOrEqual(t, len(summary), 6) // online, offline, normal, maintenance, faulty, idle, in_use
}

func TestDashboardService_GetOrderTypeSummary(t *testing.T) {
	db := setupDashboardServiceTestDB(t)
	svc := NewDashboardService(db)
	ctx := context.Background()

	user := &models.User{Nickname: "U1", MemberLevelID: 1, Status: models.UserStatusActive}
	require.NoError(t, db.Create(user).Error)

	// 创建不同类型的已支付订单
	orders := []models.Order{
		{OrderNo: "O1", UserID: user.ID, Type: models.OrderTypeRental, OriginalAmount: 10, ActualAmount: 10, Status: models.OrderStatusCompleted},
		{OrderNo: "O2", UserID: user.ID, Type: models.OrderTypeRental, OriginalAmount: 20, ActualAmount: 20, Status: models.OrderStatusCompleted},
		{OrderNo: "O3", UserID: user.ID, Type: models.OrderTypeHotel, OriginalAmount: 100, ActualAmount: 100, Status: models.OrderStatusCompleted},
		{OrderNo: "O4", UserID: user.ID, Type: models.OrderTypeMall, OriginalAmount: 50, ActualAmount: 50, Status: models.OrderStatusCompleted},
		{OrderNo: "O5", UserID: user.ID, Type: models.OrderTypeRental, OriginalAmount: 30, ActualAmount: 30, Status: models.OrderStatusPending}, // 不计入
	}
	for i := range orders {
		require.NoError(t, db.Create(&orders[i]).Error)
	}

	summary, err := svc.GetOrderTypeSummary(ctx, nil, nil)
	require.NoError(t, err)
	require.NotNil(t, summary)
	assert.GreaterOrEqual(t, len(summary), 1)
}

func TestDashboardService_GetOrderTypeSummary_WithDateFilter(t *testing.T) {
	db := setupDashboardServiceTestDB(t)
	svc := NewDashboardService(db)
	ctx := context.Background()

	startDate := time.Now().AddDate(0, 0, -7)
	endDate := time.Now()
	summary, err := svc.GetOrderTypeSummary(ctx, &startDate, &endDate)
	require.NoError(t, err)
	require.NotNil(t, summary)
}

func TestDashboardService_GetTopVenues(t *testing.T) {
	db := setupDashboardServiceTestDB(t)
	// 需要加入 Venue 模型
	require.NoError(t, db.AutoMigrate(&models.Venue{}))
	svc := NewDashboardService(db)
	ctx := context.Background()

	// 创建场地和设备
	venue := &models.Venue{
		MerchantID: 1,
		Name:       "测试场地",
		Type:       models.VenueTypeMall,
		Province:   "广东省",
		City:       "深圳市",
		District:   "南山区",
		Address:    "测试地址",
		Status:     models.VenueStatusActive,
	}
	require.NoError(t, db.Create(venue).Error)

	device := &models.Device{
		DeviceNo:    "D1",
		Name:        "设备1",
		Type:        models.DeviceTypeStandard,
		VenueID:     venue.ID,
		QRCode:      "qr1",
		ProductName: "商品",
		Status:      models.DeviceStatusActive,
	}
	require.NoError(t, db.Create(device).Error)

	// 创建租借记录
	user := &models.User{Nickname: "U1", MemberLevelID: 1, Status: models.UserStatusActive}
	require.NoError(t, db.Create(user).Error)

	rental := &models.Rental{
		OrderID:       1,
		UserID:        user.ID,
		DeviceID:      device.ID,
		DurationHours: 1,
		RentalFee:     10,
		Status:        models.RentalStatusInUse,
	}
	require.NoError(t, db.Create(rental).Error)

	topVenues, err := svc.GetTopVenues(ctx, 10, nil, nil)
	require.NoError(t, err)
	require.NotNil(t, topVenues)
}

func TestDashboardService_GetTopVenues_DefaultLimit(t *testing.T) {
	db := setupDashboardServiceTestDB(t)
	svc := NewDashboardService(db)
	ctx := context.Background()

	topVenues, err := svc.GetTopVenues(ctx, 0, nil, nil)
	require.NoError(t, err)
	require.NotNil(t, topVenues)
}

func TestDashboardService_GetRecentOrders(t *testing.T) {
	db := setupDashboardServiceTestDB(t)
	svc := NewDashboardService(db)
	ctx := context.Background()

	phone := "13800138000"
	user := &models.User{Phone: &phone, Nickname: "U1", MemberLevelID: 1, Status: models.UserStatusActive}
	require.NoError(t, db.Create(user).Error)

	// 创建订单
	for i := 0; i < 5; i++ {
		order := &models.Order{
			OrderNo:        fmt.Sprintf("O%d", i),
			UserID:         user.ID,
			Type:           models.OrderTypeRental,
			OriginalAmount: float64(10 * (i + 1)),
			ActualAmount:   float64(10 * (i + 1)),
			Status:         models.OrderStatusCompleted,
		}
		require.NoError(t, db.Create(order).Error)
	}

	orders, err := svc.GetRecentOrders(ctx, 10)
	require.NoError(t, err)
	assert.Len(t, orders, 5)
}

func TestDashboardService_GetRecentOrders_DefaultLimit(t *testing.T) {
	db := setupDashboardServiceTestDB(t)
	svc := NewDashboardService(db)
	ctx := context.Background()

	orders, err := svc.GetRecentOrders(ctx, 0)
	require.NoError(t, err)
	require.NotNil(t, orders)
}

func TestDashboardService_GetAlerts(t *testing.T) {
	db := setupDashboardServiceTestDB(t)
	svc := NewDashboardService(db)
	ctx := context.Background()

	// 创建一些告警条件
	// 故障设备
	faultyDevice := &models.Device{
		DeviceNo:     "D1",
		Name:         "故障设备",
		Type:         models.DeviceTypeStandard,
		VenueID:      1,
		QRCode:       "qr1",
		ProductName:  "商品",
		OnlineStatus: models.DeviceOffline,
		Status:       models.DeviceStatusFault,
	}
	require.NoError(t, db.Create(faultyDevice).Error)

	alerts, err := svc.GetAlerts(ctx, 10)
	require.NoError(t, err)
	require.NotNil(t, alerts)
}

