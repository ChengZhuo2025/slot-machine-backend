// Package admin 提供管理员相关服务
package admin

import (
	"context"
	"errors"
	"fmt"
	"time"

	"gorm.io/gorm"

	"github.com/dumeirei/smart-locker-backend/internal/models"
	"github.com/dumeirei/smart-locker-backend/internal/repository"
)

// AlertLevel 告警级别
type AlertLevel string

const (
	AlertLevelInfo     AlertLevel = "info"
	AlertLevelWarning  AlertLevel = "warning"
	AlertLevelCritical AlertLevel = "critical"
)

// AlertType 告警类型
type AlertType string

const (
	AlertTypeOffline          AlertType = "offline"
	AlertTypeLowBattery       AlertType = "low_battery"
	AlertTypeHighTemperature  AlertType = "high_temperature"
	AlertTypeLowSignal        AlertType = "low_signal"
	AlertTypeFault            AlertType = "fault"
	AlertTypeAbnormalUnlock   AlertType = "abnormal_unlock"
	AlertTypeLongTimeNoReturn AlertType = "long_time_no_return"
)

// DeviceAlert 设备告警
type DeviceAlert struct {
	ID         int64      `json:"id"`
	DeviceID   int64      `json:"device_id"`
	DeviceNo   string     `json:"device_no"`
	DeviceName string     `json:"device_name"`
	VenueName  string     `json:"venue_name,omitempty"`
	Type       AlertType  `json:"type"`
	Level      AlertLevel `json:"level"`
	Title      string     `json:"title"`
	Content    string     `json:"content"`
	IsResolved bool       `json:"is_resolved"`
	ResolvedBy *int64     `json:"resolved_by,omitempty"`
	ResolvedAt *time.Time `json:"resolved_at,omitempty"`
	CreatedAt  time.Time  `json:"created_at"`
}

// DeviceAlertService 设备告警服务
type DeviceAlertService struct {
	deviceRepo    *repository.DeviceRepository
	deviceLogRepo *repository.DeviceLogRepository
	alertRepo     *repository.DeviceAlertRepository
}

// NewDeviceAlertService 创建设备告警服务
func NewDeviceAlertService(
	deviceRepo *repository.DeviceRepository,
	deviceLogRepo *repository.DeviceLogRepository,
	alertRepo *repository.DeviceAlertRepository,
) *DeviceAlertService {
	return &DeviceAlertService{
		deviceRepo:    deviceRepo,
		deviceLogRepo: deviceLogRepo,
		alertRepo:     alertRepo,
	}
}

// 预定义错误
var (
	ErrAlertNotFound      = errors.New("告警不存在")
	ErrAlertAlreadyResolved = errors.New("告警已处理")
)

// AlertThresholds 告警阈值配置
type AlertThresholds struct {
	LowBatteryPercent      int     // 低电量阈值
	HighTemperatureCelsius float64 // 高温阈值
	LowSignalStrength      int     // 低信号阈值
	OfflineMinutes         int     // 离线时间阈值（分钟）
	LongRentalHours        int     // 长时间未归还阈值（小时）
}

// DefaultThresholds 默认阈值
var DefaultThresholds = AlertThresholds{
	LowBatteryPercent:      20,
	HighTemperatureCelsius: 45.0,
	LowSignalStrength:      20,
	OfflineMinutes:         30,
	LongRentalHours:        48,
}

// CheckDeviceHealth 检查设备健康状态并生成告警
func (s *DeviceAlertService) CheckDeviceHealth(ctx context.Context, deviceID int64) ([]*DeviceAlert, error) {
	device, err := s.deviceRepo.GetByIDWithVenue(ctx, deviceID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrDeviceNotFound
		}
		return nil, err
	}

	alerts := make([]*DeviceAlert, 0)
	thresholds := DefaultThresholds

	// 检查设备离线
	if device.OnlineStatus == models.DeviceOffline {
		if device.LastHeartbeatAt != nil {
			offlineDuration := time.Since(*device.LastHeartbeatAt)
			if offlineDuration.Minutes() > float64(thresholds.OfflineMinutes) {
				alert := s.createAlert(device, AlertTypeOffline, AlertLevelCritical,
					"设备离线",
					fmt.Sprintf("设备已离线 %d 分钟", int(offlineDuration.Minutes())))
				alerts = append(alerts, alert)
			}
		}
	}

	// 检查低电量
	if device.BatteryLevel != nil && *device.BatteryLevel < thresholds.LowBatteryPercent {
		level := AlertLevelWarning
		if *device.BatteryLevel < 10 {
			level = AlertLevelCritical
		}
		alert := s.createAlert(device, AlertTypeLowBattery, level,
			"电量过低",
			fmt.Sprintf("当前电量 %d%%", *device.BatteryLevel))
		alerts = append(alerts, alert)
	}

	// 检查高温
	if device.Temperature != nil && *device.Temperature > thresholds.HighTemperatureCelsius {
		level := AlertLevelWarning
		if *device.Temperature > 55.0 {
			level = AlertLevelCritical
		}
		alert := s.createAlert(device, AlertTypeHighTemperature, level,
			"温度过高",
			fmt.Sprintf("当前温度 %.1f°C", *device.Temperature))
		alerts = append(alerts, alert)
	}

	// 检查信号强度
	if device.SignalStrength != nil && *device.SignalStrength < thresholds.LowSignalStrength {
		alert := s.createAlert(device, AlertTypeLowSignal, AlertLevelWarning,
			"信号强度低",
			fmt.Sprintf("当前信号强度 %d%%", *device.SignalStrength))
		alerts = append(alerts, alert)
	}

	// 检查设备故障状态
	if device.Status == models.DeviceStatusFault {
		alert := s.createAlert(device, AlertTypeFault, AlertLevelCritical,
			"设备故障",
			"设备处于故障状态")
		alerts = append(alerts, alert)
	}

	return alerts, nil
}

// CheckAllDevices 检查所有设备健康状态
func (s *DeviceAlertService) CheckAllDevices(ctx context.Context) ([]*DeviceAlert, error) {
	devices, _, err := s.deviceRepo.List(ctx, 0, 10000, map[string]interface{}{
		"status": models.DeviceStatusActive,
	})
	if err != nil {
		return nil, err
	}

	allAlerts := make([]*DeviceAlert, 0)
	for _, d := range devices {
		alerts, err := s.CheckDeviceHealth(ctx, d.ID)
		if err != nil {
			continue
		}
		allAlerts = append(allAlerts, alerts...)
	}

	return allAlerts, nil
}

// CreateAlert 创建告警记录
func (s *DeviceAlertService) CreateAlert(ctx context.Context, deviceID int64, alertType AlertType, level AlertLevel, title, content string) (*DeviceAlert, error) {
	device, err := s.deviceRepo.GetByIDWithVenue(ctx, deviceID)
	if err != nil {
		return nil, err
	}

	alert := s.createAlert(device, alertType, level, title, content)

	// 如果有告警仓储，保存到数据库
	if s.alertRepo != nil {
		dbAlert := &models.DeviceAlert{
			DeviceID: deviceID,
			Type:     string(alertType),
			Level:    string(level),
			Title:    title,
			Content:  content,
		}
		if err := s.alertRepo.Create(ctx, dbAlert); err != nil {
			return nil, err
		}
		alert.ID = dbAlert.ID
	}

	// 记录到设备日志
	s.createDeviceLog(ctx, deviceID, "alert", fmt.Sprintf("[%s] %s: %s", level, title, content))

	return alert, nil
}

// ResolveAlert 解决告警
func (s *DeviceAlertService) ResolveAlert(ctx context.Context, alertID int64, operatorID int64) error {
	if s.alertRepo == nil {
		return errors.New("告警仓储未配置")
	}

	alert, err := s.alertRepo.GetByID(ctx, alertID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return ErrAlertNotFound
		}
		return err
	}

	if alert.IsResolved {
		return ErrAlertAlreadyResolved
	}

	now := time.Now()
	alert.IsResolved = true
	alert.ResolvedBy = &operatorID
	alert.ResolvedAt = &now

	return s.alertRepo.Update(ctx, alert)
}

// ListAlerts 获取告警列表
func (s *DeviceAlertService) ListAlerts(ctx context.Context, offset, limit int, filters map[string]interface{}) ([]*DeviceAlert, int64, error) {
	if s.alertRepo == nil {
		return nil, 0, errors.New("告警仓储未配置")
	}

	alerts, total, err := s.alertRepo.List(ctx, offset, limit, filters)
	if err != nil {
		return nil, 0, err
	}

	result := make([]*DeviceAlert, 0, len(alerts))
	for _, a := range alerts {
		device, _ := s.deviceRepo.GetByIDWithVenue(ctx, a.DeviceID)
		info := &DeviceAlert{
			ID:         a.ID,
			DeviceID:   a.DeviceID,
			Type:       AlertType(a.Type),
			Level:      AlertLevel(a.Level),
			Title:      a.Title,
			Content:    a.Content,
			IsResolved: a.IsResolved,
			ResolvedBy: a.ResolvedBy,
			ResolvedAt: a.ResolvedAt,
			CreatedAt:  a.CreatedAt,
		}
		if device != nil {
			info.DeviceNo = device.DeviceNo
			info.DeviceName = device.Name
			if device.Venue != nil {
				info.VenueName = device.Venue.Name
			}
		}
		result = append(result, info)
	}

	return result, total, nil
}

// GetUnresolvedCount 获取未解决告警数量
func (s *DeviceAlertService) GetUnresolvedCount(ctx context.Context) (int64, error) {
	if s.alertRepo == nil {
		return 0, nil
	}
	return s.alertRepo.CountUnresolved(ctx)
}

// GetAlertStatistics 获取告警统计
func (s *DeviceAlertService) GetAlertStatistics(ctx context.Context) (*AlertStatistics, error) {
	if s.alertRepo == nil {
		return &AlertStatistics{}, nil
	}

	stats := &AlertStatistics{}

	// 统计各级别未解决告警
	stats.Unresolved, _ = s.alertRepo.CountUnresolved(ctx)
	stats.Critical, _ = s.alertRepo.CountByLevel(ctx, string(AlertLevelCritical), false)
	stats.Warning, _ = s.alertRepo.CountByLevel(ctx, string(AlertLevelWarning), false)
	stats.Info, _ = s.alertRepo.CountByLevel(ctx, string(AlertLevelInfo), false)

	// 今日新增
	today := time.Now().Truncate(24 * time.Hour)
	stats.TodayNew, _ = s.alertRepo.CountSince(ctx, today)

	return stats, nil
}

// AlertStatistics 告警统计
type AlertStatistics struct {
	Unresolved int64 `json:"unresolved"`
	Critical   int64 `json:"critical"`
	Warning    int64 `json:"warning"`
	Info       int64 `json:"info"`
	TodayNew   int64 `json:"today_new"`
}

// createAlert 创建告警信息
func (s *DeviceAlertService) createAlert(device *models.Device, alertType AlertType, level AlertLevel, title, content string) *DeviceAlert {
	alert := &DeviceAlert{
		DeviceID:   device.ID,
		DeviceNo:   device.DeviceNo,
		DeviceName: device.Name,
		Type:       alertType,
		Level:      level,
		Title:      title,
		Content:    content,
		IsResolved: false,
		CreatedAt:  time.Now(),
	}
	if device.Venue != nil {
		alert.VenueName = device.Venue.Name
	}
	return alert
}

// createDeviceLog 创建设备日志
func (s *DeviceAlertService) createDeviceLog(ctx context.Context, deviceID int64, logType, content string) {
	if s.deviceLogRepo == nil {
		return
	}
	log := &models.DeviceLog{
		DeviceID: deviceID,
		Type:     logType,
		Content:  &content,
	}
	_ = s.deviceLogRepo.Create(ctx, log)
}
