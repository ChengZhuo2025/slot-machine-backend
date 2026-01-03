// Package wechatpay 提供微信支付 SDK 封装
package wechatpay

import (
	"context"
	"crypto/rsa"
	"encoding/json"
	"fmt"
	"time"
)

// Config 微信支付配置
type Config struct {
	AppID          string `mapstructure:"app_id"`
	MchID          string `mapstructure:"mch_id"`
	APIKey         string `mapstructure:"api_key"`
	APIKeyV3       string `mapstructure:"api_key_v3"`
	SerialNo       string `mapstructure:"serial_no"`
	PrivateKeyPath string `mapstructure:"private_key_path"`
	NotifyURL      string `mapstructure:"notify_url"`
	IsSandbox      bool   `mapstructure:"is_sandbox"`
}

// Client 微信支付客户端
type Client struct {
	config     *Config
	privateKey *rsa.PrivateKey
}

// NewClient 创建微信支付客户端
func NewClient(config *Config) (*Client, error) {
	client := &Client{
		config: config,
	}

	// TODO: 加载私钥
	// privateKey, err := loadPrivateKey(config.PrivateKeyPath)
	// if err != nil {
	//     return nil, err
	// }
	// client.privateKey = privateKey

	return client, nil
}

// UnifiedOrderRequest 统一下单请求
type UnifiedOrderRequest struct {
	OutTradeNo  string  `json:"out_trade_no"`
	Description string  `json:"description"`
	Amount      int64   `json:"amount"` // 单位：分
	OpenID      string  `json:"openid,omitempty"`
	Attach      string  `json:"attach,omitempty"`
	ExpireTime  string  `json:"expire_time,omitempty"`
}

// UnifiedOrderResponse 统一下单响应
type UnifiedOrderResponse struct {
	PrepayID  string `json:"prepay_id"`
	CodeURL   string `json:"code_url,omitempty"`
	H5URL     string `json:"h5_url,omitempty"`
	TimeStamp string `json:"timestamp"`
	NonceStr  string `json:"nonce_str"`
	Package   string `json:"package"`
	SignType  string `json:"sign_type"`
	PaySign   string `json:"pay_sign"`
}

// CreateOrder 创建支付订单（小程序支付）
func (c *Client) CreateOrder(ctx context.Context, req *UnifiedOrderRequest) (*UnifiedOrderResponse, error) {
	// TODO: 实现微信支付统一下单
	// 这里返回模拟数据，实际需要调用微信支付 API

	now := time.Now()
	resp := &UnifiedOrderResponse{
		PrepayID:  fmt.Sprintf("wx%d_prepay", now.UnixNano()),
		TimeStamp: fmt.Sprintf("%d", now.Unix()),
		NonceStr:  generateNonceStr(),
		Package:   "prepay_id=wx_mock_prepay_id",
		SignType:  "RSA",
		PaySign:   "mock_sign",
	}

	return resp, nil
}

// CreateNativeOrder 创建扫码支付订单
func (c *Client) CreateNativeOrder(ctx context.Context, req *UnifiedOrderRequest) (*UnifiedOrderResponse, error) {
	// TODO: 实现微信支付 Native 下单
	resp := &UnifiedOrderResponse{
		CodeURL: fmt.Sprintf("weixin://wxpay/bizpayurl?pr=%s", req.OutTradeNo),
	}

	return resp, nil
}

// CreateH5Order 创建 H5 支付订单
func (c *Client) CreateH5Order(ctx context.Context, req *UnifiedOrderRequest) (*UnifiedOrderResponse, error) {
	// TODO: 实现微信支付 H5 下单
	resp := &UnifiedOrderResponse{
		H5URL: fmt.Sprintf("https://wx.tenpay.com/cgi-bin/mmpayweb-bin/checkmweb?prepay_id=%s", req.OutTradeNo),
	}

	return resp, nil
}

// QueryOrderRequest 查询订单请求
type QueryOrderRequest struct {
	OutTradeNo    string `json:"out_trade_no,omitempty"`
	TransactionID string `json:"transaction_id,omitempty"`
}

// QueryOrderResponse 查询订单响应
type QueryOrderResponse struct {
	TradeState     string     `json:"trade_state"`
	TradeStateDesc string     `json:"trade_state_desc"`
	TransactionID  string     `json:"transaction_id"`
	Amount         int64      `json:"amount"`
	PayerOpenID    string     `json:"payer_openid"`
	SuccessTime    *time.Time `json:"success_time,omitempty"`
}

// TradeState 交易状态
const (
	TradeStateSuccess    = "SUCCESS"
	TradeStateRefund     = "REFUND"
	TradeStateNotPay     = "NOTPAY"
	TradeStateClosed     = "CLOSED"
	TradeStateRevoked    = "REVOKED"
	TradeStateUserPaying = "USERPAYING"
	TradeStatePayError   = "PAYERROR"
)

// QueryOrder 查询订单
func (c *Client) QueryOrder(ctx context.Context, req *QueryOrderRequest) (*QueryOrderResponse, error) {
	// TODO: 实现微信支付订单查询
	resp := &QueryOrderResponse{
		TradeState:     TradeStateNotPay,
		TradeStateDesc: "订单未支付",
	}

	return resp, nil
}

// CloseOrderRequest 关闭订单请求
type CloseOrderRequest struct {
	OutTradeNo string `json:"out_trade_no"`
}

// CloseOrder 关闭订单
func (c *Client) CloseOrder(ctx context.Context, req *CloseOrderRequest) error {
	// TODO: 实现微信支付关闭订单
	return nil
}

// RefundRequest 退款请求
type RefundRequest struct {
	OutTradeNo  string `json:"out_trade_no"`
	OutRefundNo string `json:"out_refund_no"`
	Reason      string `json:"reason"`
	Total       int64  `json:"total"`
	Refund      int64  `json:"refund"`
	NotifyURL   string `json:"notify_url,omitempty"`
}

// RefundResponse 退款响应
type RefundResponse struct {
	RefundID      string     `json:"refund_id"`
	OutRefundNo   string     `json:"out_refund_no"`
	TransactionID string     `json:"transaction_id"`
	Status        string     `json:"status"`
	SuccessTime   *time.Time `json:"success_time,omitempty"`
}

// RefundStatus 退款状态
const (
	RefundStatusSuccess    = "SUCCESS"
	RefundStatusClosed     = "CLOSED"
	RefundStatusProcessing = "PROCESSING"
	RefundStatusAbnormal   = "ABNORMAL"
)

// Refund 申请退款
func (c *Client) Refund(ctx context.Context, req *RefundRequest) (*RefundResponse, error) {
	// TODO: 实现微信支付退款
	resp := &RefundResponse{
		RefundID:    fmt.Sprintf("rf_%d", time.Now().UnixNano()),
		OutRefundNo: req.OutRefundNo,
		Status:      RefundStatusProcessing,
	}

	return resp, nil
}

// NotifyPayload 支付回调数据
type NotifyPayload struct {
	ID           string          `json:"id"`
	CreateTime   string          `json:"create_time"`
	ResourceType string          `json:"resource_type"`
	EventType    string          `json:"event_type"`
	Summary      string          `json:"summary"`
	Resource     json.RawMessage `json:"resource"`
}

// NotifyResource 回调资源数据（解密后）
type NotifyResource struct {
	OutTradeNo    string `json:"out_trade_no"`
	TransactionID string `json:"transaction_id"`
	TradeType     string `json:"trade_type"`
	TradeState    string `json:"trade_state"`
	SuccessTime   string `json:"success_time"`
	Payer         struct {
		OpenID string `json:"openid"`
	} `json:"payer"`
	Amount struct {
		Total    int64 `json:"total"`
		PayerTotal int64 `json:"payer_total"`
		Currency string `json:"currency"`
	} `json:"amount"`
}

// ParseNotify 解析支付回调
func (c *Client) ParseNotify(payload []byte) (*NotifyResource, error) {
	var notify NotifyPayload
	if err := json.Unmarshal(payload, &notify); err != nil {
		return nil, fmt.Errorf("parse notify payload error: %w", err)
	}

	// TODO: 解密 Resource 数据
	// 实际需要使用 APIv3 密钥解密

	var resource NotifyResource
	if err := json.Unmarshal(notify.Resource, &resource); err != nil {
		return nil, fmt.Errorf("parse notify resource error: %w", err)
	}

	return &resource, nil
}

// VerifySignature 验证签名
func (c *Client) VerifySignature(signature, timestamp, nonce string, body []byte) error {
	// TODO: 实现签名验证
	return nil
}

// generateNonceStr 生成随机字符串
func generateNonceStr() string {
	return fmt.Sprintf("%d", time.Now().UnixNano())
}
