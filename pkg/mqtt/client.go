// Package mqtt 提供 MQTT 客户端封装
package mqtt

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"sync"
	"time"

	mqtt "github.com/eclipse/paho.mqtt.golang"
)

// Config MQTT 配置
type Config struct {
	Broker       string `mapstructure:"broker"`
	Port         int    `mapstructure:"port"`
	ClientID     string `mapstructure:"client_id"`
	Username     string `mapstructure:"username"`
	Password     string `mapstructure:"password"`
	CleanSession bool   `mapstructure:"clean_session"`
	QoS          byte   `mapstructure:"qos"`
	KeepAlive    int    `mapstructure:"keep_alive"`
	AutoReconnect bool  `mapstructure:"auto_reconnect"`
}

// Client MQTT 客户端
type Client struct {
	config   *Config
	client   mqtt.Client
	handlers map[string]MessageHandler
	mu       sync.RWMutex
}

// MessageHandler 消息处理器
type MessageHandler func(topic string, payload []byte)

// Message MQTT 消息
type Message struct {
	DeviceNo  string          `json:"device_no"`
	Type      string          `json:"type"`
	Timestamp int64           `json:"timestamp"`
	Data      json.RawMessage `json:"data,omitempty"`
}

// NewClient 创建 MQTT 客户端
func NewClient(config *Config) *Client {
	return &Client{
		config:   config,
		handlers: make(map[string]MessageHandler),
	}
}

// Connect 连接 MQTT Broker
func (c *Client) Connect() error {
	opts := mqtt.NewClientOptions()
	opts.AddBroker(fmt.Sprintf("tcp://%s:%d", c.config.Broker, c.config.Port))
	opts.SetClientID(c.config.ClientID)
	opts.SetUsername(c.config.Username)
	opts.SetPassword(c.config.Password)
	opts.SetCleanSession(c.config.CleanSession)
	opts.SetKeepAlive(time.Duration(c.config.KeepAlive) * time.Second)
	opts.SetAutoReconnect(c.config.AutoReconnect)
	opts.SetConnectionLostHandler(c.onConnectionLost)
	opts.SetOnConnectHandler(c.onConnect)
	opts.SetReconnectingHandler(c.onReconnecting)

	c.client = mqtt.NewClient(opts)

	if token := c.client.Connect(); token.Wait() && token.Error() != nil {
		return fmt.Errorf("mqtt connect error: %w", token.Error())
	}

	log.Printf("[MQTT] Connected to broker: %s:%d", c.config.Broker, c.config.Port)
	return nil
}

// Disconnect 断开连接
func (c *Client) Disconnect() {
	if c.client != nil && c.client.IsConnected() {
		c.client.Disconnect(250)
		log.Println("[MQTT] Disconnected from broker")
	}
}

// IsConnected 检查是否已连接
func (c *Client) IsConnected() bool {
	return c.client != nil && c.client.IsConnected()
}

// Subscribe 订阅主题
func (c *Client) Subscribe(topic string, handler MessageHandler) error {
	c.mu.Lock()
	c.handlers[topic] = handler
	c.mu.Unlock()

	token := c.client.Subscribe(topic, c.config.QoS, func(client mqtt.Client, msg mqtt.Message) {
		c.mu.RLock()
		if h, ok := c.handlers[msg.Topic()]; ok {
			h(msg.Topic(), msg.Payload())
		}
		c.mu.RUnlock()
	})

	if token.Wait() && token.Error() != nil {
		return fmt.Errorf("mqtt subscribe error: %w", token.Error())
	}

	log.Printf("[MQTT] Subscribed to topic: %s", topic)
	return nil
}

// SubscribeMultiple 批量订阅主题
func (c *Client) SubscribeMultiple(topics map[string]MessageHandler) error {
	filters := make(map[string]byte)
	for topic := range topics {
		filters[topic] = c.config.QoS
	}

	c.mu.Lock()
	for topic, handler := range topics {
		c.handlers[topic] = handler
	}
	c.mu.Unlock()

	token := c.client.SubscribeMultiple(filters, func(client mqtt.Client, msg mqtt.Message) {
		c.mu.RLock()
		if h, ok := c.handlers[msg.Topic()]; ok {
			h(msg.Topic(), msg.Payload())
		}
		c.mu.RUnlock()
	})

	if token.Wait() && token.Error() != nil {
		return fmt.Errorf("mqtt subscribe multiple error: %w", token.Error())
	}

	log.Printf("[MQTT] Subscribed to %d topics", len(topics))
	return nil
}

// Unsubscribe 取消订阅
func (c *Client) Unsubscribe(topics ...string) error {
	token := c.client.Unsubscribe(topics...)
	if token.Wait() && token.Error() != nil {
		return fmt.Errorf("mqtt unsubscribe error: %w", token.Error())
	}

	c.mu.Lock()
	for _, topic := range topics {
		delete(c.handlers, topic)
	}
	c.mu.Unlock()

	log.Printf("[MQTT] Unsubscribed from topics: %v", topics)
	return nil
}

// Publish 发布消息
func (c *Client) Publish(topic string, payload interface{}) error {
	var data []byte
	var err error

	switch v := payload.(type) {
	case []byte:
		data = v
	case string:
		data = []byte(v)
	default:
		data, err = json.Marshal(payload)
		if err != nil {
			return fmt.Errorf("mqtt marshal payload error: %w", err)
		}
	}

	token := c.client.Publish(topic, c.config.QoS, false, data)
	if token.Wait() && token.Error() != nil {
		return fmt.Errorf("mqtt publish error: %w", token.Error())
	}

	return nil
}

// PublishRetained 发布保留消息
func (c *Client) PublishRetained(topic string, payload interface{}) error {
	var data []byte
	var err error

	switch v := payload.(type) {
	case []byte:
		data = v
	case string:
		data = []byte(v)
	default:
		data, err = json.Marshal(payload)
		if err != nil {
			return fmt.Errorf("mqtt marshal payload error: %w", err)
		}
	}

	token := c.client.Publish(topic, c.config.QoS, true, data)
	if token.Wait() && token.Error() != nil {
		return fmt.Errorf("mqtt publish retained error: %w", token.Error())
	}

	return nil
}

// PublishWithContext 发布消息（带超时）
func (c *Client) PublishWithContext(ctx context.Context, topic string, payload interface{}) error {
	var data []byte
	var err error

	switch v := payload.(type) {
	case []byte:
		data = v
	case string:
		data = []byte(v)
	default:
		data, err = json.Marshal(payload)
		if err != nil {
			return fmt.Errorf("mqtt marshal payload error: %w", err)
		}
	}

	token := c.client.Publish(topic, c.config.QoS, false, data)

	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-token.Done():
		if token.Error() != nil {
			return fmt.Errorf("mqtt publish error: %w", token.Error())
		}
		return nil
	}
}

// onConnect 连接成功回调
func (c *Client) onConnect(client mqtt.Client) {
	log.Println("[MQTT] Connected to broker")

	// 重新订阅所有主题
	c.mu.RLock()
	defer c.mu.RUnlock()

	for topic, handler := range c.handlers {
		if token := c.client.Subscribe(topic, c.config.QoS, func(_ mqtt.Client, msg mqtt.Message) {
			c.mu.RLock()
			if h, ok := c.handlers[msg.Topic()]; ok {
				h(msg.Topic(), msg.Payload())
			}
			c.mu.RUnlock()
		}); token.Wait() && token.Error() != nil {
			log.Printf("[MQTT] Resubscribe error for topic %s: %v", topic, token.Error())
		} else {
			log.Printf("[MQTT] Resubscribed to topic: %s", topic)
		}
		_ = handler // 避免未使用警告
	}
}

// onConnectionLost 连接断开回调
func (c *Client) onConnectionLost(client mqtt.Client, err error) {
	log.Printf("[MQTT] Connection lost: %v", err)
}

// onReconnecting 重连回调
func (c *Client) onReconnecting(client mqtt.Client, opts *mqtt.ClientOptions) {
	log.Println("[MQTT] Reconnecting to broker...")
}
