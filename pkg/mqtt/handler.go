// Package mqtt 提供 MQTT 设备通信服务
package mqtt

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"strings"
	"time"
)

// Topic 定义
const (
	// 设备上报主题
	TopicDeviceHeartbeat = "device/+/heartbeat" // 心跳上报
	TopicDeviceStatus    = "device/+/status"    // 状态上报
	TopicDeviceEvent     = "device/+/event"     // 事件上报
	TopicDeviceAck       = "device/+/ack"       // 命令响应

	// 服务端下发主题
	TopicDeviceCommand = "device/%s/command" // 命令下发
	TopicDeviceConfig  = "device/%s/config"  // 配置下发
)

// MessageType 消息类型
const (
	MsgTypeHeartbeat  = "heartbeat"  // 心跳
	MsgTypeStatus     = "status"     // 状态
	MsgTypeEvent      = "event"      // 事件
	MsgTypeAck        = "ack"        // 响应
	MsgTypeUnlock     = "unlock"     // 开锁
	MsgTypeLock       = "lock"       // 锁定
	MsgTypeReboot     = "reboot"     // 重启
	MsgTypeUpgrade    = "upgrade"    // 升级
	MsgTypeConfig     = "config"     // 配置
)

// EventType 事件类型
const (
	EventUnlocked   = "unlocked"    // 已开锁
	EventLocked     = "locked"      // 已锁定
	EventDoorOpened = "door_opened" // 门已打开
	EventDoorClosed = "door_closed" // 门已关闭
	EventError      = "error"       // 错误
	EventAlarm      = "alarm"       // 告警
)

// HeartbeatPayload 心跳数据
type HeartbeatPayload struct {
	DeviceNo        string   `json:"device_no"`
	SignalStrength  int      `json:"signal_strength,omitempty"`
	BatteryLevel    int      `json:"battery_level,omitempty"`
	Temperature     float64  `json:"temperature,omitempty"`
	Humidity        float64  `json:"humidity,omitempty"`
	FirmwareVersion string   `json:"firmware_version,omitempty"`
	LockStatus      int8     `json:"lock_status"`
	AvailableSlots  int      `json:"available_slots"`
	Timestamp       int64    `json:"timestamp"`
}

// StatusPayload 状态数据
type StatusPayload struct {
	DeviceNo       string `json:"device_no"`
	OnlineStatus   int8   `json:"online_status"`
	LockStatus     int8   `json:"lock_status"`
	RentalStatus   int8   `json:"rental_status"`
	AvailableSlots int    `json:"available_slots"`
	Timestamp      int64  `json:"timestamp"`
}

// EventPayload 事件数据
type EventPayload struct {
	DeviceNo  string                 `json:"device_no"`
	EventType string                 `json:"event_type"`
	Data      map[string]interface{} `json:"data,omitempty"`
	Timestamp int64                  `json:"timestamp"`
}

// AckPayload 响应数据
type AckPayload struct {
	DeviceNo  string `json:"device_no"`
	CommandID string `json:"command_id"`
	Success   bool   `json:"success"`
	Message   string `json:"message,omitempty"`
	Timestamp int64  `json:"timestamp"`
}

// CommandPayload 命令数据
type CommandPayload struct {
	CommandID   string                 `json:"command_id"`
	CommandType string                 `json:"command_type"`
	Data        map[string]interface{} `json:"data,omitempty"`
	Timestamp   int64                  `json:"timestamp"`
}

// DeviceHandler 设备消息处理器接口
type DeviceHandler interface {
	OnHeartbeat(ctx context.Context, deviceNo string, payload *HeartbeatPayload) error
	OnStatus(ctx context.Context, deviceNo string, payload *StatusPayload) error
	OnEvent(ctx context.Context, deviceNo string, payload *EventPayload) error
	OnAck(ctx context.Context, deviceNo string, payload *AckPayload) error
}

// DeviceMessageHandler 设备消息处理器
type DeviceMessageHandler struct {
	client  *Client
	handler DeviceHandler
}

// NewDeviceMessageHandler 创建设备消息处理器
func NewDeviceMessageHandler(client *Client, handler DeviceHandler) *DeviceMessageHandler {
	return &DeviceMessageHandler{
		client:  client,
		handler: handler,
	}
}

// Start 启动消息处理
func (h *DeviceMessageHandler) Start(ctx context.Context) error {
	// 订阅设备消息主题
	topics := map[string]MessageHandler{
		TopicDeviceHeartbeat: h.handleHeartbeat,
		TopicDeviceStatus:    h.handleStatus,
		TopicDeviceEvent:     h.handleEvent,
		TopicDeviceAck:       h.handleAck,
	}

	if err := h.client.SubscribeMultiple(topics); err != nil {
		return fmt.Errorf("subscribe device topics error: %w", err)
	}

	log.Println("[DeviceHandler] Started listening for device messages")
	return nil
}

// Stop 停止消息处理
func (h *DeviceMessageHandler) Stop() error {
	topics := []string{
		TopicDeviceHeartbeat,
		TopicDeviceStatus,
		TopicDeviceEvent,
		TopicDeviceAck,
	}

	if err := h.client.Unsubscribe(topics...); err != nil {
		return fmt.Errorf("unsubscribe device topics error: %w", err)
	}

	log.Println("[DeviceHandler] Stopped listening for device messages")
	return nil
}

// handleHeartbeat 处理心跳消息
func (h *DeviceMessageHandler) handleHeartbeat(topic string, payload []byte) {
	deviceNo := extractDeviceNo(topic)
	if deviceNo == "" {
		log.Printf("[DeviceHandler] Invalid heartbeat topic: %s", topic)
		return
	}

	var data HeartbeatPayload
	if err := json.Unmarshal(payload, &data); err != nil {
		log.Printf("[DeviceHandler] Parse heartbeat error: %v", err)
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := h.handler.OnHeartbeat(ctx, deviceNo, &data); err != nil {
		log.Printf("[DeviceHandler] Handle heartbeat error: %v", err)
	}
}

// handleStatus 处理状态消息
func (h *DeviceMessageHandler) handleStatus(topic string, payload []byte) {
	deviceNo := extractDeviceNo(topic)
	if deviceNo == "" {
		log.Printf("[DeviceHandler] Invalid status topic: %s", topic)
		return
	}

	var data StatusPayload
	if err := json.Unmarshal(payload, &data); err != nil {
		log.Printf("[DeviceHandler] Parse status error: %v", err)
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := h.handler.OnStatus(ctx, deviceNo, &data); err != nil {
		log.Printf("[DeviceHandler] Handle status error: %v", err)
	}
}

// handleEvent 处理事件消息
func (h *DeviceMessageHandler) handleEvent(topic string, payload []byte) {
	deviceNo := extractDeviceNo(topic)
	if deviceNo == "" {
		log.Printf("[DeviceHandler] Invalid event topic: %s", topic)
		return
	}

	var data EventPayload
	if err := json.Unmarshal(payload, &data); err != nil {
		log.Printf("[DeviceHandler] Parse event error: %v", err)
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := h.handler.OnEvent(ctx, deviceNo, &data); err != nil {
		log.Printf("[DeviceHandler] Handle event error: %v", err)
	}
}

// handleAck 处理响应消息
func (h *DeviceMessageHandler) handleAck(topic string, payload []byte) {
	deviceNo := extractDeviceNo(topic)
	if deviceNo == "" {
		log.Printf("[DeviceHandler] Invalid ack topic: %s", topic)
		return
	}

	var data AckPayload
	if err := json.Unmarshal(payload, &data); err != nil {
		log.Printf("[DeviceHandler] Parse ack error: %v", err)
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := h.handler.OnAck(ctx, deviceNo, &data); err != nil {
		log.Printf("[DeviceHandler] Handle ack error: %v", err)
	}
}

// extractDeviceNo 从主题中提取设备编号
func extractDeviceNo(topic string) string {
	parts := strings.Split(topic, "/")
	if len(parts) >= 2 {
		return parts[1]
	}
	return ""
}
