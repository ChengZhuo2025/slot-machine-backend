// Package admin 管理端服务
package admin

import (
	"context"
	"time"

	"gorm.io/gorm"

	"github.com/dumeirei/smart-locker-backend/internal/models"
)

// DashboardService 平台管理员仪表盘服务
type DashboardService struct {
	db *gorm.DB
}

// NewDashboardService 创建仪表盘服务
func NewDashboardService(db *gorm.DB) *DashboardService {
	return &DashboardService{db: db}
}

// PlatformOverview 平台概览数据
type PlatformOverview struct {
	// 用户统计
	TotalUsers      int64 `json:"total_users"`
	TodayNewUsers   int64 `json:"today_new_users"`
	MonthNewUsers   int64 `json:"month_new_users"`
	ActiveUsers     int64 `json:"active_users"` // 最近7天活跃用户

	// 订单统计
	TotalOrders     int64   `json:"total_orders"`
	TodayOrders     int64   `json:"today_orders"`
	PendingOrders   int64   `json:"pending_orders"`
	TotalRevenue    float64 `json:"total_revenue"`
	TodayRevenue    float64 `json:"today_revenue"`
	MonthRevenue    float64 `json:"month_revenue"`

	// 设备统计
	TotalDevices    int64 `json:"total_devices"`
	OnlineDevices   int64 `json:"online_devices"`
	OfflineDevices  int64 `json:"offline_devices"`
	FaultyDevices   int64 `json:"faulty_devices"`

	// 商户统计
	TotalMerchants  int64 `json:"total_merchants"`
	ActiveMerchants int64 `json:"active_merchants"`

	// 租借统计
	TodayRentals    int64 `json:"today_rentals"`
	ActiveRentals   int64 `json:"active_rentals"`

	// 酒店统计
	TodayBookings   int64 `json:"today_bookings"`
	ActiveBookings  int64 `json:"active_bookings"`
}

// GetPlatformOverview 获取平台概览数据
func (s *DashboardService) GetPlatformOverview(ctx context.Context) (*PlatformOverview, error) {
	overview := &PlatformOverview{}

	now := time.Now()
	today := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())
	tomorrow := today.Add(24 * time.Hour)
	monthStart := time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, now.Location())
	weekAgo := now.Add(-7 * 24 * time.Hour)

	// 用户统计
	s.db.WithContext(ctx).Model(&models.User{}).Count(&overview.TotalUsers)
	s.db.WithContext(ctx).Model(&models.User{}).
		Where("created_at >= ? AND created_at < ?", today, tomorrow).
		Count(&overview.TodayNewUsers)
	s.db.WithContext(ctx).Model(&models.User{}).
		Where("created_at >= ?", monthStart).
		Count(&overview.MonthNewUsers)
	s.db.WithContext(ctx).Model(&models.User{}).
		Where("updated_at >= ?", weekAgo).
		Count(&overview.ActiveUsers)

	// 订单统计
	s.db.WithContext(ctx).Model(&models.Order{}).Count(&overview.TotalOrders)
	s.db.WithContext(ctx).Model(&models.Order{}).
		Where("created_at >= ? AND created_at < ?", today, tomorrow).
		Count(&overview.TodayOrders)
	s.db.WithContext(ctx).Model(&models.Order{}).
		Where("status = ?", models.OrderStatusPending).
		Count(&overview.PendingOrders)

	// 总收入
	s.db.WithContext(ctx).Model(&models.Payment{}).
		Where("status = ?", models.PaymentStatusSuccess).
		Select("COALESCE(SUM(amount), 0)").
		Row().Scan(&overview.TotalRevenue)

	// 今日收入
	s.db.WithContext(ctx).Model(&models.Payment{}).
		Where("status = ? AND pay_time >= ? AND pay_time < ?",
			models.PaymentStatusSuccess, today, tomorrow).
		Select("COALESCE(SUM(amount), 0)").
		Row().Scan(&overview.TodayRevenue)

	// 本月收入
	s.db.WithContext(ctx).Model(&models.Payment{}).
		Where("status = ? AND pay_time >= ?", models.PaymentStatusSuccess, monthStart).
		Select("COALESCE(SUM(amount), 0)").
		Row().Scan(&overview.MonthRevenue)

	// 设备统计
	s.db.WithContext(ctx).Model(&models.Device{}).Count(&overview.TotalDevices)
	s.db.WithContext(ctx).Model(&models.Device{}).
		Where("online_status = ?", models.DeviceOnline).
		Count(&overview.OnlineDevices)
	s.db.WithContext(ctx).Model(&models.Device{}).
		Where("online_status = ?", models.DeviceOffline).
		Count(&overview.OfflineDevices)
	s.db.WithContext(ctx).Model(&models.Device{}).
		Where("status = ?", models.DeviceStatusFault).
		Count(&overview.FaultyDevices)

	// 商户统计
	s.db.WithContext(ctx).Model(&models.Merchant{}).Count(&overview.TotalMerchants)
	s.db.WithContext(ctx).Model(&models.Merchant{}).
		Where("status = ?", 1).
		Count(&overview.ActiveMerchants)

	// 今日租借
	s.db.WithContext(ctx).Model(&models.Rental{}).
		Where("created_at >= ? AND created_at < ?", today, tomorrow).
		Count(&overview.TodayRentals)

	// 活跃租借（使用中）
	s.db.WithContext(ctx).Model(&models.Rental{}).
		Where("status = ?", models.RentalStatusInUse).
		Count(&overview.ActiveRentals)

	// 今日预订
	s.db.WithContext(ctx).Model(&models.Booking{}).
		Where("created_at >= ? AND created_at < ?", today, tomorrow).
		Count(&overview.TodayBookings)

	// 活跃预订（已核销待使用或使用中）
	s.db.WithContext(ctx).Model(&models.Booking{}).
		Where("status IN ?", []string{models.BookingStatusVerified, models.BookingStatusInUse}).
		Count(&overview.ActiveBookings)

	return overview, nil
}

// OrderTrend 订单趋势数据
type OrderTrend struct {
	Date     string  `json:"date"`
	Orders   int64   `json:"orders"`
	Revenue  float64 `json:"revenue"`
	Rentals  int64   `json:"rentals"`
	Bookings int64   `json:"bookings"`
	MallOrders int64 `json:"mall_orders"`
}

// GetOrderTrend 获取订单趋势（最近N天）
func (s *DashboardService) GetOrderTrend(ctx context.Context, days int) ([]OrderTrend, error) {
	if days <= 0 {
		days = 7
	}
	if days > 30 {
		days = 30
	}

	trends := make([]OrderTrend, days)
	now := time.Now()

	for i := days - 1; i >= 0; i-- {
		date := now.Add(-time.Duration(i) * 24 * time.Hour)
		startOfDay := time.Date(date.Year(), date.Month(), date.Day(), 0, 0, 0, 0, date.Location())
		endOfDay := startOfDay.Add(24 * time.Hour)

		trend := OrderTrend{
			Date: startOfDay.Format("2006-01-02"),
		}

		// 订单数
		s.db.WithContext(ctx).Model(&models.Order{}).
			Where("created_at >= ? AND created_at < ?", startOfDay, endOfDay).
			Count(&trend.Orders)

		// 收入
		s.db.WithContext(ctx).Model(&models.Payment{}).
			Where("status = ? AND pay_time >= ? AND pay_time < ?",
				models.PaymentStatusSuccess, startOfDay, endOfDay).
			Select("COALESCE(SUM(amount), 0)").
			Row().Scan(&trend.Revenue)

		// 租借订单
		s.db.WithContext(ctx).Model(&models.Order{}).
			Where("type = ? AND created_at >= ? AND created_at < ?",
				models.OrderTypeRental, startOfDay, endOfDay).
			Count(&trend.Rentals)

		// 酒店预订
		s.db.WithContext(ctx).Model(&models.Order{}).
			Where("type = ? AND created_at >= ? AND created_at < ?",
				models.OrderTypeHotel, startOfDay, endOfDay).
			Count(&trend.Bookings)

		// 商城订单
		s.db.WithContext(ctx).Model(&models.Order{}).
			Where("type = ? AND created_at >= ? AND created_at < ?",
				models.OrderTypeMall, startOfDay, endOfDay).
			Count(&trend.MallOrders)

		trends[days-1-i] = trend
	}

	return trends, nil
}

// DeviceStatusSummary 设备状态汇总
type DeviceStatusSummary struct {
	Status string `json:"status"`
	Count  int64  `json:"count"`
}

// GetDeviceStatusSummary 获取设备状态汇总
func (s *DashboardService) GetDeviceStatusSummary(ctx context.Context) ([]DeviceStatusSummary, error) {
	var results []DeviceStatusSummary

	// 在线状态
	var onlineCount, offlineCount int64
	s.db.WithContext(ctx).Model(&models.Device{}).
		Where("online_status = ?", models.DeviceOnline).
		Count(&onlineCount)
	s.db.WithContext(ctx).Model(&models.Device{}).
		Where("online_status = ?", models.DeviceOffline).
		Count(&offlineCount)

	results = append(results,
		DeviceStatusSummary{Status: "online", Count: onlineCount},
		DeviceStatusSummary{Status: "offline", Count: offlineCount},
	)

	// 设备状态
	var normalCount, maintainCount, faultyCount int64
	s.db.WithContext(ctx).Model(&models.Device{}).
		Where("status = ?", models.DeviceStatusActive).
		Count(&normalCount)
	s.db.WithContext(ctx).Model(&models.Device{}).
		Where("status = ?", models.DeviceStatusMaintenance).
		Count(&maintainCount)
	s.db.WithContext(ctx).Model(&models.Device{}).
		Where("status = ?", models.DeviceStatusFault).
		Count(&faultyCount)

	results = append(results,
		DeviceStatusSummary{Status: "normal", Count: normalCount},
		DeviceStatusSummary{Status: "maintenance", Count: maintainCount},
		DeviceStatusSummary{Status: "faulty", Count: faultyCount},
	)

	// 租借状态
	var idleCount, inUseCount int64
	s.db.WithContext(ctx).Model(&models.Device{}).
		Where("rental_status = ?", models.DeviceRentalFree).
		Count(&idleCount)
	s.db.WithContext(ctx).Model(&models.Device{}).
		Where("rental_status = ?", models.DeviceRentalInUse).
		Count(&inUseCount)

	results = append(results,
		DeviceStatusSummary{Status: "idle", Count: idleCount},
		DeviceStatusSummary{Status: "in_use", Count: inUseCount},
	)

	return results, nil
}

// OrderTypeSummary 订单类型汇总
type OrderTypeSummary struct {
	Type    string  `json:"type"`
	Count   int64   `json:"count"`
	Revenue float64 `json:"revenue"`
}

// GetOrderTypeSummary 获取订单类型汇总
func (s *DashboardService) GetOrderTypeSummary(ctx context.Context, startDate, endDate *time.Time) ([]OrderTypeSummary, error) {
	var results []OrderTypeSummary

	query := s.db.WithContext(ctx).Model(&models.Order{}).
		Select("type, COUNT(*) as count, COALESCE(SUM(actual_amount), 0) as revenue").
		Where("status NOT IN ?", []string{models.OrderStatusPending, models.OrderStatusCancelled}).
		Group("type")

	if startDate != nil {
		query = query.Where("created_at >= ?", *startDate)
	}
	if endDate != nil {
		query = query.Where("created_at <= ?", *endDate)
	}

	err := query.Find(&results).Error
	return results, err
}

// TopVenue 热门场地
type TopVenue struct {
	VenueID     int64   `json:"venue_id"`
	VenueName   string  `json:"venue_name"`
	MerchantID  int64   `json:"merchant_id"`
	OrderCount  int64   `json:"order_count"`
	Revenue     float64 `json:"revenue"`
}

// GetTopVenues 获取热门场地
func (s *DashboardService) GetTopVenues(ctx context.Context, limit int, startDate, endDate *time.Time) ([]TopVenue, error) {
	if limit <= 0 {
		limit = 10
	}

	var results []TopVenue

	// 通过租借记录统计场地订单量和收入
	query := s.db.WithContext(ctx).Table("rentals r").
		Select(`
			d.venue_id,
			v.name as venue_name,
			v.merchant_id,
			COUNT(*) as order_count,
			COALESCE(SUM(r.rental_fee + r.overtime_fee), 0) as revenue
		`).
		Joins("JOIN devices d ON r.device_id = d.id").
		Joins("JOIN venues v ON d.venue_id = v.id").
		Where("r.status != ?", models.RentalStatusPaid).
		Group("d.venue_id, v.name, v.merchant_id").
		Order("revenue DESC").
		Limit(limit)

	if startDate != nil {
		query = query.Where("r.created_at >= ?", *startDate)
	}
	if endDate != nil {
		query = query.Where("r.created_at <= ?", *endDate)
	}

	err := query.Find(&results).Error
	return results, err
}

// RecentOrder 最近订单
type RecentOrder struct {
	ID          int64     `json:"id"`
	OrderNo     string    `json:"order_no"`
	Type        string    `json:"type"`
	UserID      int64     `json:"user_id"`
	UserPhone   string    `json:"user_phone"`
	Amount      float64   `json:"amount"`
	Status      string    `json:"status"`
	CreatedAt   time.Time `json:"created_at"`
}

// GetRecentOrders 获取最近订单
func (s *DashboardService) GetRecentOrders(ctx context.Context, limit int) ([]RecentOrder, error) {
	if limit <= 0 {
		limit = 10
	}

	var orders []models.Order
	err := s.db.WithContext(ctx).Model(&models.Order{}).
		Preload("User").
		Order("created_at DESC").
		Limit(limit).
		Find(&orders).Error

	if err != nil {
		return nil, err
	}

	results := make([]RecentOrder, len(orders))
	for i, order := range orders {
		results[i] = RecentOrder{
			ID:        order.ID,
			OrderNo:   order.OrderNo,
			Type:      order.Type,
			UserID:    order.UserID,
			Amount:    order.ActualAmount,
			Status:    order.Status,
			CreatedAt: order.CreatedAt,
		}
		if order.User != nil && order.User.Phone != nil {
			results[i].UserPhone = *order.User.Phone
		}
	}

	return results, nil
}

// AlertInfo 告警信息
type AlertInfo struct {
	Type      string    `json:"type"`
	Title     string    `json:"title"`
	Message   string    `json:"message"`
	Level     string    `json:"level"` // warning, error, info
	TargetID  int64     `json:"target_id,omitempty"`
	CreatedAt time.Time `json:"created_at"`
}

// GetAlerts 获取告警信息
func (s *DashboardService) GetAlerts(ctx context.Context, limit int) ([]AlertInfo, error) {
	if limit <= 0 {
		limit = 10
	}

	var alerts []AlertInfo

	// 查询离线设备告警
	var offlineDevices []models.Device
	err := s.db.WithContext(ctx).Model(&models.Device{}).
		Where("online_status = ? AND status = ?", models.DeviceOffline, models.DeviceStatusActive).
		Where("last_heartbeat_at < ?", time.Now().Add(-10*time.Minute)).
		Limit(5).
		Find(&offlineDevices).Error

	if err == nil {
		for _, device := range offlineDevices {
			alerts = append(alerts, AlertInfo{
				Type:      "device_offline",
				Title:     "设备离线",
				Message:   "设备 " + device.DeviceNo + " 已离线超过10分钟",
				Level:     "warning",
				TargetID:  device.ID,
				CreatedAt: time.Now(),
			})
		}
	}

	// 查询故障设备告警
	var faultyDevices []models.Device
	err = s.db.WithContext(ctx).Model(&models.Device{}).
		Where("status = ?", models.DeviceStatusFault).
		Limit(5).
		Find(&faultyDevices).Error

	if err == nil {
		for _, device := range faultyDevices {
			alerts = append(alerts, AlertInfo{
				Type:      "device_faulty",
				Title:     "设备故障",
				Message:   "设备 " + device.DeviceNo + " 报告故障",
				Level:     "error",
				TargetID:  device.ID,
				CreatedAt: time.Now(),
			})
		}
	}

	// 查询超时租借告警
	var overtimeRentals []models.Rental
	err = s.db.WithContext(ctx).Model(&models.Rental{}).
		Where("status = ?", models.RentalStatusOverdue).
		Limit(5).
		Find(&overtimeRentals).Error

	if err == nil {
		for _, rental := range overtimeRentals {
			alerts = append(alerts, AlertInfo{
				Type:      "rental_overtime",
				Title:     "租借超时",
				Message:   "租借订单超时未归还",
				Level:     "warning",
				TargetID:  rental.ID,
				CreatedAt: time.Now(),
			})
		}
	}

	// 限制返回数量
	if len(alerts) > limit {
		alerts = alerts[:limit]
	}

	return alerts, nil
}
