// Package device 提供设备 MQTT 服务
package device

import (
	"context"
	"log"

	"github.com/dumeirei/smart-locker-backend/internal/models"
	"github.com/dumeirei/smart-locker-backend/internal/repository"
	"github.com/dumeirei/smart-locker-backend/pkg/mqtt"
)

// MQTTService 设备 MQTT 服务
type MQTTService struct {
	deviceRepo    *repository.DeviceRepository
	deviceService *DeviceService
	commandSender *mqtt.CommandSender
}

// NewMQTTService 创建设备 MQTT 服务
func NewMQTTService(
	deviceRepo *repository.DeviceRepository,
	deviceService *DeviceService,
	commandSender *mqtt.CommandSender,
) *MQTTService {
	return &MQTTService{
		deviceRepo:    deviceRepo,
		deviceService: deviceService,
		commandSender: commandSender,
	}
}

// OnHeartbeat 处理心跳消息
func (s *MQTTService) OnHeartbeat(ctx context.Context, deviceNo string, payload *mqtt.HeartbeatPayload) error {
	log.Printf("[MQTTService] Received heartbeat from device: %s", deviceNo)

	// 更新设备心跳信息
	data := &HeartbeatData{
		SignalStrength:  intPtr(payload.SignalStrength),
		BatteryLevel:    intPtr(payload.BatteryLevel),
		Temperature:     float64Ptr(payload.Temperature),
		Humidity:        float64Ptr(payload.Humidity),
		FirmwareVersion: stringPtrNonEmpty(payload.FirmwareVersion),
		LockStatus:      int8Ptr(payload.LockStatus),
	}

	if err := s.deviceService.UpdateDeviceHeartbeat(ctx, deviceNo, data); err != nil {
		log.Printf("[MQTTService] Update heartbeat error: %v", err)
		return err
	}

	return nil
}

// OnStatus 处理状态消息
func (s *MQTTService) OnStatus(ctx context.Context, deviceNo string, payload *mqtt.StatusPayload) error {
	log.Printf("[MQTTService] Received status from device: %s", deviceNo)

	device, err := s.deviceRepo.GetByDeviceNo(ctx, deviceNo)
	if err != nil {
		log.Printf("[MQTTService] Device not found: %s", deviceNo)
		return err
	}

	fields := map[string]interface{}{
		"online_status":   payload.OnlineStatus,
		"lock_status":     payload.LockStatus,
		"rental_status":   payload.RentalStatus,
		"available_slots": payload.AvailableSlots,
	}

	if err := s.deviceRepo.UpdateFields(ctx, device.ID, fields); err != nil {
		log.Printf("[MQTTService] Update status error: %v", err)
		return err
	}

	return nil
}

// OnEvent 处理事件消息
func (s *MQTTService) OnEvent(ctx context.Context, deviceNo string, payload *mqtt.EventPayload) error {
	log.Printf("[MQTTService] Received event from device: %s, type: %s", deviceNo, payload.EventType)

	device, err := s.deviceRepo.GetByDeviceNo(ctx, deviceNo)
	if err != nil {
		log.Printf("[MQTTService] Device not found: %s", deviceNo)
		return err
	}

	// 记录设备日志
	logType := eventTypeToLogType(payload.EventType)
	content := ""
	if payload.Data != nil {
		if msg, ok := payload.Data["message"].(string); ok {
			content = msg
		}
	}

	deviceLog := &models.DeviceLog{
		DeviceID:     device.ID,
		Type:         logType,
		Content:      stringPtrNonEmpty(content),
		OperatorType: stringPtr(models.DeviceLogOperatorSystem),
	}

	if err := s.deviceRepo.CreateLog(ctx, deviceLog); err != nil {
		log.Printf("[MQTTService] Create log error: %v", err)
	}

	// 根据事件类型更新设备状态
	switch payload.EventType {
	case mqtt.EventUnlocked:
		_ = s.deviceRepo.UpdateLockStatus(ctx, device.ID, models.DeviceUnlocked)
	case mqtt.EventLocked:
		_ = s.deviceRepo.UpdateLockStatus(ctx, device.ID, models.DeviceLocked)
	case mqtt.EventError:
		_ = s.deviceRepo.UpdateStatus(ctx, device.ID, models.DeviceStatusFault)
	}

	return nil
}

// OnAck 处理响应消息
func (s *MQTTService) OnAck(ctx context.Context, deviceNo string, payload *mqtt.AckPayload) error {
	log.Printf("[MQTTService] Received ack from device: %s, command: %s, success: %v",
		deviceNo, payload.CommandID, payload.Success)

	// 转发给命令发送器处理
	if s.commandSender != nil {
		s.commandSender.HandleAck(payload)
	}

	return nil
}

// SendUnlockCommand 发送开锁命令
func (s *MQTTService) SendUnlockCommand(ctx context.Context, deviceNo string, slotNo *int) (*mqtt.CommandResult, error) {
	if s.commandSender == nil {
		return nil, nil
	}

	result, err := s.commandSender.SendUnlock(ctx, deviceNo, slotNo)
	if err != nil {
		log.Printf("[MQTTService] Send unlock command error: %v", err)
		return nil, err
	}

	return result, nil
}

// SendUnlockCommandAsync 异步发送开锁命令
func (s *MQTTService) SendUnlockCommandAsync(ctx context.Context, deviceNo string, slotNo *int) (string, error) {
	if s.commandSender == nil {
		return "", nil
	}

	commandID, err := s.commandSender.SendUnlockAsync(ctx, deviceNo, slotNo)
	if err != nil {
		log.Printf("[MQTTService] Send unlock command async error: %v", err)
		return "", err
	}

	return commandID, nil
}

// SendLockCommand 发送锁定命令
func (s *MQTTService) SendLockCommand(ctx context.Context, deviceNo string, slotNo *int) (*mqtt.CommandResult, error) {
	if s.commandSender == nil {
		return nil, nil
	}

	result, err := s.commandSender.SendLock(ctx, deviceNo, slotNo)
	if err != nil {
		log.Printf("[MQTTService] Send lock command error: %v", err)
		return nil, err
	}

	return result, nil
}

// eventTypeToLogType 事件类型转日志类型
func eventTypeToLogType(eventType string) string {
	switch eventType {
	case mqtt.EventUnlocked:
		return models.DeviceLogTypeUnlock
	case mqtt.EventLocked:
		return models.DeviceLogTypeLock
	case mqtt.EventError, mqtt.EventAlarm:
		return models.DeviceLogTypeError
	default:
		return eventType
	}
}

// 辅助函数
func intPtr(v int) *int {
	return &v
}

func int8Ptr(v int8) *int8 {
	return &v
}

func float64Ptr(v float64) *float64 {
	if v == 0 {
		return nil
	}
	return &v
}

func stringPtrNonEmpty(v string) *string {
	if v == "" {
		return nil
	}
	return &v
}
