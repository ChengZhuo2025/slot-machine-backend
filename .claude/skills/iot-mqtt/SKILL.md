---
name: IoT Device Communication
description: This skill should be used when the user asks to "send device command", "handle device status", "implement MQTT", "device heartbeat", "remote unlock", "device monitoring", "IoT integration", or needs guidance on MQTT communication, device control, status synchronization, and IoT patterns for the smart locker system.
version: 1.0.0
---

# IoT Device Communication Skill

This skill provides guidance for MQTT-based IoT communication with smart locker devices.

## Technology Stack

| Component | Version | Purpose |
|-----------|---------|---------|
| EMQX | 5.0+ | MQTT Broker |
| paho.mqtt.golang | - | MQTT Client |
| Protocol | MQTT 3.1.1/5.0 | Communication |
| QoS | Level 1 | At least once delivery |

## MQTT Topic Design

### Topic Hierarchy

```
smart-locker/
├── devices/
│   ├── {device_no}/
│   │   ├── status          # Device status reports (device → server)
│   │   ├── heartbeat       # Heartbeat signals (device → server)
│   │   ├── command         # Commands (server → device)
│   │   └── response        # Command responses (device → server)
│   └── +/status            # Subscribe all device status
└── system/
    ├── broadcast           # System-wide broadcasts
    └── alerts              # Alert notifications
```

### Topic Examples

| Topic | Direction | Purpose |
|-------|-----------|---------|
| `smart-locker/devices/D001/status` | Device → Server | Status updates |
| `smart-locker/devices/D001/heartbeat` | Device → Server | Heartbeat |
| `smart-locker/devices/D001/command` | Server → Device | Control commands |
| `smart-locker/devices/D001/response` | Device → Server | Command response |

## MQTT Client Implementation

### Client Setup

```go
package mqtt

import (
    "fmt"
    "time"

    mqtt "github.com/eclipse/paho.mqtt.golang"
    "backend/internal/common/config"
    "backend/internal/common/logger"
)

type Client struct {
    client mqtt.Client
    config *config.MQTTConfig
}

func NewClient(cfg *config.MQTTConfig) (*Client, error) {
    opts := mqtt.NewClientOptions().
        AddBroker(cfg.Broker).
        SetClientID(cfg.ClientID).
        SetUsername(cfg.Username).
        SetPassword(cfg.Password).
        SetCleanSession(false).
        SetAutoReconnect(true).
        SetConnectRetry(true).
        SetConnectRetryInterval(5 * time.Second).
        SetKeepAlive(60 * time.Second).
        SetPingTimeout(10 * time.Second).
        SetConnectionLostHandler(onConnectionLost).
        SetOnConnectHandler(onConnect)

    client := mqtt.NewClient(opts)
    token := client.Connect()
    if token.Wait() && token.Error() != nil {
        return nil, token.Error()
    }

    return &Client{client: client, config: cfg}, nil
}

func onConnectionLost(client mqtt.Client, err error) {
    logger.Log.With("error", err).Warn("MQTT connection lost")
}

func onConnect(client mqtt.Client) {
    logger.Log.Info("MQTT connected")
    // Re-subscribe to topics after reconnect
}
```

### Subscribe to Device Messages

```go
func (c *Client) SubscribeDeviceStatus(handler DeviceStatusHandler) error {
    topic := "smart-locker/devices/+/status"
    token := c.client.Subscribe(topic, 1, func(client mqtt.Client, msg mqtt.Message) {
        var status DeviceStatus
        if err := json.Unmarshal(msg.Payload(), &status); err != nil {
            logger.Log.With("error", err).Error("Failed to parse device status")
            return
        }

        // Extract device_no from topic
        parts := strings.Split(msg.Topic(), "/")
        deviceNo := parts[2]

        handler.HandleStatus(deviceNo, &status)
    })

    return token.Error()
}

func (c *Client) SubscribeHeartbeat(handler HeartbeatHandler) error {
    topic := "smart-locker/devices/+/heartbeat"
    token := c.client.Subscribe(topic, 1, func(client mqtt.Client, msg mqtt.Message) {
        parts := strings.Split(msg.Topic(), "/")
        deviceNo := parts[2]

        var heartbeat DeviceHeartbeat
        json.Unmarshal(msg.Payload(), &heartbeat)

        handler.HandleHeartbeat(deviceNo, &heartbeat)
    })

    return token.Error()
}
```

## Message Formats

### Device Status Message

```json
{
    "device_no": "D001",
    "online_status": 1,
    "lock_status": 0,
    "rental_status": 0,
    "signal_strength": 85,
    "battery_level": 100,
    "temperature": 25.5,
    "humidity": 60.2,
    "firmware_version": "1.2.0",
    "timestamp": "2026-01-02T12:00:00Z"
}
```

### Heartbeat Message

```json
{
    "device_no": "D001",
    "uptime": 86400,
    "memory_usage": 45,
    "cpu_usage": 12,
    "timestamp": "2026-01-02T12:00:00Z"
}
```

### Command Message

```json
{
    "command_id": "cmd_123456",
    "command": "unlock",
    "params": {
        "rental_id": 1001,
        "timeout": 30
    },
    "timestamp": "2026-01-02T12:00:00Z"
}
```

### Command Response

```json
{
    "command_id": "cmd_123456",
    "status": "success",
    "result": {
        "unlock_time": "2026-01-02T12:00:05Z"
    },
    "timestamp": "2026-01-02T12:00:05Z"
}
```

## Device Commands

### Unlock Command

```go
type UnlockCommand struct {
    CommandID string `json:"command_id"`
    Command   string `json:"command"`
    Params    struct {
        RentalID int64 `json:"rental_id"`
        Timeout  int   `json:"timeout"`
    } `json:"params"`
    Timestamp time.Time `json:"timestamp"`
}

func (c *Client) SendUnlockCommand(deviceNo string, rentalID int64) (*CommandResponse, error) {
    commandID := generateCommandID()

    cmd := UnlockCommand{
        CommandID: commandID,
        Command:   "unlock",
        Timestamp: time.Now(),
    }
    cmd.Params.RentalID = rentalID
    cmd.Params.Timeout = 30

    topic := fmt.Sprintf("smart-locker/devices/%s/command", deviceNo)
    payload, _ := json.Marshal(cmd)

    // Publish command
    token := c.client.Publish(topic, 1, false, payload)
    if token.Wait() && token.Error() != nil {
        return nil, token.Error()
    }

    // Wait for response with timeout
    return c.waitForResponse(deviceNo, commandID, 5*time.Second)
}
```

### Command Response Handling

```go
func (c *Client) waitForResponse(deviceNo, commandID string, timeout time.Duration) (*CommandResponse, error) {
    responseChan := make(chan *CommandResponse, 1)
    errorChan := make(chan error, 1)

    topic := fmt.Sprintf("smart-locker/devices/%s/response", deviceNo)

    // Subscribe to response topic
    token := c.client.Subscribe(topic, 1, func(client mqtt.Client, msg mqtt.Message) {
        var resp CommandResponse
        if err := json.Unmarshal(msg.Payload(), &resp); err != nil {
            return
        }

        if resp.CommandID == commandID {
            responseChan <- &resp
        }
    })

    if token.Wait() && token.Error() != nil {
        return nil, token.Error()
    }

    defer c.client.Unsubscribe(topic)

    select {
    case resp := <-responseChan:
        return resp, nil
    case err := <-errorChan:
        return nil, err
    case <-time.After(timeout):
        return nil, ErrCommandTimeout
    }
}
```

## Retry Strategy

### Exponential Backoff

```go
func (s *DeviceService) SendCommandWithRetry(deviceNo string, cmd Command) error {
    maxRetries := 3
    baseDelay := 1 * time.Second

    for i := 0; i < maxRetries; i++ {
        resp, err := s.mqttClient.SendCommand(deviceNo, cmd)
        if err == nil && resp.Status == "success" {
            return nil
        }

        if i < maxRetries-1 {
            delay := baseDelay * time.Duration(1<<uint(i)) // 1s, 2s, 4s
            time.Sleep(delay)
        }
    }

    // All retries failed
    return s.handleCommandFailure(deviceNo, cmd)
}

func (s *DeviceService) handleCommandFailure(deviceNo string, cmd Command) error {
    // Update device status to communication error
    s.deviceRepo.UpdateStatus(context.Background(), deviceNo, StatusCommError)

    // Create alert
    s.alertService.CreateAlert(AlertDeviceCommError, deviceNo)

    // Notify operations
    s.notifyService.NotifyOps(fmt.Sprintf("Device %s communication failed", deviceNo))

    return ErrDeviceCommFailed
}
```

## Heartbeat Monitoring

### Heartbeat Handler

```go
type HeartbeatMonitor struct {
    deviceRepo *repository.DeviceRepository
    redis      *redis.Client
    timeout    time.Duration
}

func (m *HeartbeatMonitor) HandleHeartbeat(deviceNo string, hb *DeviceHeartbeat) {
    ctx := context.Background()

    // Update last heartbeat time in Redis
    key := fmt.Sprintf("device:heartbeat:%s", deviceNo)
    m.redis.Set(ctx, key, time.Now().Unix(), m.timeout)

    // Update device info if changed
    m.deviceRepo.UpdateHeartbeat(ctx, deviceNo, hb)
}

func (m *HeartbeatMonitor) StartMonitoring(ctx context.Context) {
    ticker := time.NewTicker(30 * time.Second)
    defer ticker.Stop()

    for {
        select {
        case <-ticker.C:
            m.checkOfflineDevices(ctx)
        case <-ctx.Done():
            return
        }
    }
}

func (m *HeartbeatMonitor) checkOfflineDevices(ctx context.Context) {
    devices, _ := m.deviceRepo.GetOnlineDevices(ctx)

    for _, device := range devices {
        key := fmt.Sprintf("device:heartbeat:%s", device.DeviceNo)
        exists, _ := m.redis.Exists(ctx, key).Result()

        if exists == 0 {
            // Device offline
            m.deviceRepo.UpdateOnlineStatus(ctx, device.DeviceNo, StatusOffline)
            m.createOfflineAlert(device.DeviceNo)
        }
    }
}
```

## Status Synchronization

### Status Update Handler

```go
type StatusHandler struct {
    deviceRepo  *repository.DeviceRepository
    deviceCache *cache.DeviceCache
    eventBus    *events.EventBus
}

func (h *StatusHandler) HandleStatus(deviceNo string, status *DeviceStatus) {
    ctx := context.Background()

    // Update database
    err := h.deviceRepo.UpdateStatus(ctx, deviceNo, status)
    if err != nil {
        logger.Log.With("error", err, "device", deviceNo).Error("Failed to update device status")
        return
    }

    // Update cache
    h.deviceCache.Set(deviceNo, status)

    // Publish event
    h.eventBus.Publish(events.Event{
        Type:      events.DeviceStatusChanged,
        Payload:   status,
        Timestamp: time.Now(),
    })

    // Handle specific status changes
    if status.LockStatus == 1 {
        h.handleUnlocked(deviceNo, status)
    } else if status.LockStatus == 0 && status.RentalStatus == 0 {
        h.handleReturned(deviceNo, status)
    }
}
```

## Best Practices

1. **Use QoS 1** for critical commands (unlock, lock)
2. **Implement command acknowledgment** with timeout
3. **Use exponential backoff** for retries
4. **Monitor heartbeat** with 30-60 second intervals
5. **Cache device status** in Redis for quick access
6. **Log all commands** for audit trail
7. **Handle reconnection** gracefully

## Additional Resources

### Reference Files

- **`references/mqtt-protocol.md`** - MQTT protocol details
- **`references/device-commands.md`** - Complete command reference

### Project Documentation

- **`specs/001-smart-locker-backend/spec.md`** - Device requirements (FR-011 to FR-017)
