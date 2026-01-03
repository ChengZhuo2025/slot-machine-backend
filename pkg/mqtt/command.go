// Package mqtt 提供 MQTT 设备通信服务
package mqtt

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/google/uuid"
)

// CommandSender 命令发送器
type CommandSender struct {
	client     *Client
	pending    map[string]*pendingCommand
	mu         sync.RWMutex
	ackTimeout time.Duration
}

// pendingCommand 待响应命令
type pendingCommand struct {
	CommandID string
	DeviceNo  string
	Created   time.Time
	AckChan   chan *AckPayload
}

// CommandResult 命令执行结果
type CommandResult struct {
	Success bool
	Message string
}

// NewCommandSender 创建命令发送器
func NewCommandSender(client *Client, ackTimeout time.Duration) *CommandSender {
	if ackTimeout == 0 {
		ackTimeout = 30 * time.Second
	}

	return &CommandSender{
		client:     client,
		pending:    make(map[string]*pendingCommand),
		ackTimeout: ackTimeout,
	}
}

// SendUnlock 发送开锁命令
func (s *CommandSender) SendUnlock(ctx context.Context, deviceNo string, slotNo *int) (*CommandResult, error) {
	data := make(map[string]interface{})
	if slotNo != nil {
		data["slot_no"] = *slotNo
	}

	return s.sendCommand(ctx, deviceNo, MsgTypeUnlock, data)
}

// SendLock 发送锁定命令
func (s *CommandSender) SendLock(ctx context.Context, deviceNo string, slotNo *int) (*CommandResult, error) {
	data := make(map[string]interface{})
	if slotNo != nil {
		data["slot_no"] = *slotNo
	}

	return s.sendCommand(ctx, deviceNo, MsgTypeLock, data)
}

// SendReboot 发送重启命令
func (s *CommandSender) SendReboot(ctx context.Context, deviceNo string) (*CommandResult, error) {
	return s.sendCommand(ctx, deviceNo, MsgTypeReboot, nil)
}

// SendUpgrade 发送升级命令
func (s *CommandSender) SendUpgrade(ctx context.Context, deviceNo string, firmwareURL string, version string) (*CommandResult, error) {
	data := map[string]interface{}{
		"firmware_url": firmwareURL,
		"version":      version,
	}

	return s.sendCommand(ctx, deviceNo, MsgTypeUpgrade, data)
}

// SendConfig 发送配置命令
func (s *CommandSender) SendConfig(ctx context.Context, deviceNo string, config map[string]interface{}) (*CommandResult, error) {
	return s.sendCommand(ctx, deviceNo, MsgTypeConfig, config)
}

// sendCommand 发送命令并等待响应
func (s *CommandSender) sendCommand(ctx context.Context, deviceNo string, commandType string, data map[string]interface{}) (*CommandResult, error) {
	commandID := generateCommandID()

	// 创建命令载荷
	payload := &CommandPayload{
		CommandID:   commandID,
		CommandType: commandType,
		Data:        data,
		Timestamp:   time.Now().Unix(),
	}

	// 注册待响应命令
	pending := &pendingCommand{
		CommandID: commandID,
		DeviceNo:  deviceNo,
		Created:   time.Now(),
		AckChan:   make(chan *AckPayload, 1),
	}

	s.mu.Lock()
	s.pending[commandID] = pending
	s.mu.Unlock()

	defer func() {
		s.mu.Lock()
		delete(s.pending, commandID)
		s.mu.Unlock()
		close(pending.AckChan)
	}()

	// 发送命令
	topic := fmt.Sprintf(TopicDeviceCommand, deviceNo)
	if err := s.client.PublishWithContext(ctx, topic, payload); err != nil {
		return nil, fmt.Errorf("publish command error: %w", err)
	}

	// 等待响应或超时
	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	case ack := <-pending.AckChan:
		return &CommandResult{
			Success: ack.Success,
			Message: ack.Message,
		}, nil
	case <-time.After(s.ackTimeout):
		return nil, fmt.Errorf("command timeout: %s", commandID)
	}
}

// SendUnlockAsync 异步发送开锁命令（不等待响应）
func (s *CommandSender) SendUnlockAsync(ctx context.Context, deviceNo string, slotNo *int) (string, error) {
	data := make(map[string]interface{})
	if slotNo != nil {
		data["slot_no"] = *slotNo
	}

	return s.sendCommandAsync(ctx, deviceNo, MsgTypeUnlock, data)
}

// SendLockAsync 异步发送锁定命令（不等待响应）
func (s *CommandSender) SendLockAsync(ctx context.Context, deviceNo string, slotNo *int) (string, error) {
	data := make(map[string]interface{})
	if slotNo != nil {
		data["slot_no"] = *slotNo
	}

	return s.sendCommandAsync(ctx, deviceNo, MsgTypeLock, data)
}

// sendCommandAsync 异步发送命令
func (s *CommandSender) sendCommandAsync(ctx context.Context, deviceNo string, commandType string, data map[string]interface{}) (string, error) {
	commandID := generateCommandID()

	payload := &CommandPayload{
		CommandID:   commandID,
		CommandType: commandType,
		Data:        data,
		Timestamp:   time.Now().Unix(),
	}

	topic := fmt.Sprintf(TopicDeviceCommand, deviceNo)
	if err := s.client.PublishWithContext(ctx, topic, payload); err != nil {
		return "", fmt.Errorf("publish command error: %w", err)
	}

	return commandID, nil
}

// HandleAck 处理响应消息
func (s *CommandSender) HandleAck(ack *AckPayload) {
	s.mu.RLock()
	pending, ok := s.pending[ack.CommandID]
	s.mu.RUnlock()

	if ok {
		select {
		case pending.AckChan <- ack:
		default:
		}
	}
}

// CleanupExpired 清理过期的待响应命令
func (s *CommandSender) CleanupExpired() {
	s.mu.Lock()
	defer s.mu.Unlock()

	now := time.Now()
	for id, pending := range s.pending {
		if now.Sub(pending.Created) > s.ackTimeout*2 {
			delete(s.pending, id)
		}
	}
}

// StartCleanup 启动清理协程
func (s *CommandSender) StartCleanup(ctx context.Context) {
	ticker := time.NewTicker(time.Minute)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			s.CleanupExpired()
		}
	}
}

// generateCommandID 生成命令 ID
func generateCommandID() string {
	return uuid.New().String()
}
