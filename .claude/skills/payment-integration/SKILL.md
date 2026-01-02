---
name: Payment Integration
description: This skill should be used when the user asks to "integrate payment", "add WeChat Pay", "add Alipay", "handle payment callback", "process refund", "payment notification", "create payment order", or needs guidance on payment gateway integration, callback handling, and transaction management for the smart locker system.
version: 1.0.0
---

# Payment Integration Skill

This skill provides guidance for integrating WeChat Pay and Alipay in the smart locker backend.

## Supported Payment Channels

| Channel | SDK | Use Case |
|---------|-----|----------|
| WeChat Pay | wechatpay-go | Mini program, JSAPI |
| Alipay | alipay-sdk-go | Mobile web |
| Wallet | Internal | Balance payment |

## WeChat Pay Integration

### SDK Setup

```go
package wechat

import (
    "context"
    "github.com/wechatpay-apiv3/wechatpay-go/core"
    "github.com/wechatpay-apiv3/wechatpay-go/services/payments/jsapi"
    "github.com/wechatpay-apiv3/wechatpay-go/core/option"
    "github.com/wechatpay-apiv3/wechatpay-go/utils"
)

type WechatPay struct {
    client *core.Client
    config *WechatPayConfig
}

type WechatPayConfig struct {
    AppID          string
    MchID          string
    APIKey         string
    CertPath       string
    KeyPath        string
    SerialNo       string
    NotifyURL      string
}

func NewWechatPay(cfg *WechatPayConfig) (*WechatPay, error) {
    privateKey, err := utils.LoadPrivateKeyWithPath(cfg.KeyPath)
    if err != nil {
        return nil, err
    }

    ctx := context.Background()
    client, err := core.NewClient(ctx,
        option.WithWechatPayAutoAuthCipher(cfg.MchID, cfg.SerialNo, privateKey, cfg.APIKey),
    )
    if err != nil {
        return nil, err
    }

    return &WechatPay{client: client, config: cfg}, nil
}
```

### Create JSAPI Payment

```go
func (w *WechatPay) CreateJSAPIPayment(ctx context.Context, req *PaymentRequest) (*JSAPIPayParams, error) {
    svc := jsapi.JsapiApiService{Client: w.client}

    resp, _, err := svc.PrepayWithRequestPayment(ctx, jsapi.PrepayRequest{
        Appid:       core.String(w.config.AppID),
        Mchid:       core.String(w.config.MchID),
        Description: core.String(req.Description),
        OutTradeNo:  core.String(req.OrderNo),
        NotifyUrl:   core.String(w.config.NotifyURL),
        Amount: &jsapi.Amount{
            Total:    core.Int64(req.AmountCents), // Amount in cents
            Currency: core.String("CNY"),
        },
        Payer: &jsapi.Payer{
            Openid: core.String(req.OpenID),
        },
    })

    if err != nil {
        return nil, err
    }

    return &JSAPIPayParams{
        AppID:     *resp.Appid,
        TimeStamp: *resp.TimeStamp,
        NonceStr:  *resp.NonceStr,
        Package:   *resp.Package,
        SignType:  *resp.SignType,
        PaySign:   *resp.PaySign,
    }, nil
}
```

### Payment Callback Handler

```go
func (h *PaymentHandler) WechatCallback(c *gin.Context) {
    // Read request body
    body, err := io.ReadAll(c.Request.Body)
    if err != nil {
        c.String(500, "fail")
        return
    }

    // Verify signature
    if !h.wechatPay.VerifySignature(c.Request.Header, body) {
        c.String(401, "fail")
        return
    }

    // Parse notification
    var notification WechatPayNotification
    if err := json.Unmarshal(body, &notification); err != nil {
        c.String(400, "fail")
        return
    }

    // Process payment
    if notification.EventType == "TRANSACTION.SUCCESS" {
        err = h.paymentService.HandleWechatPaySuccess(c.Request.Context(), &notification)
        if err != nil {
            logger.Log.With("error", err).Error("Failed to process WeChat payment")
            c.String(500, "fail")
            return
        }
    }

    c.JSON(200, gin.H{"code": "SUCCESS", "message": "OK"})
}
```

### Process Payment Success

```go
func (s *PaymentService) HandleWechatPaySuccess(ctx context.Context, notification *WechatPayNotification) error {
    // Decrypt resource
    resource, err := s.wechatPay.DecryptResource(notification.Resource)
    if err != nil {
        return err
    }

    // Check idempotency - prevent duplicate processing
    processed, err := s.redis.SetNX(ctx, "payment:processed:"+resource.OutTradeNo, 1, 24*time.Hour).Result()
    if err != nil || !processed {
        return nil // Already processed
    }

    // Start transaction
    return s.db.Transaction(func(tx *gorm.DB) error {
        // Get payment record
        var payment models.Payment
        if err := tx.Where("payment_no = ?", resource.OutTradeNo).First(&payment).Error; err != nil {
            return err
        }

        // Verify amount
        if payment.Amount.Mul(decimal.NewFromInt(100)).IntPart() != resource.Amount.Total {
            return ErrAmountMismatch
        }

        // Update payment status
        payment.Status = PaymentStatusSuccess
        payment.TradeNo = &resource.TransactionID
        payment.PayTime = &resource.SuccessTime
        payment.CallbackData = notification
        if err := tx.Save(&payment).Error; err != nil {
            return err
        }

        // Update order status
        if err := tx.Model(&models.Order{}).
            Where("id = ?", payment.OrderID).
            Updates(map[string]interface{}{
                "status":  OrderStatusPaid,
                "paid_at": time.Now(),
            }).Error; err != nil {
            return err
        }

        // Trigger post-payment actions
        return s.triggerPostPaymentActions(ctx, tx, &payment)
    })
}
```

## Alipay Integration

### SDK Setup

```go
package alipay

import (
    "github.com/smartwalle/alipay/v3"
)

type Alipay struct {
    client *alipay.Client
    config *AlipayConfig
}

type AlipayConfig struct {
    AppID            string
    PrivateKey       string
    AlipayPublicKey  string
    NotifyURL        string
    ReturnURL        string
    IsSandbox        bool
}

func NewAlipay(cfg *AlipayConfig) (*Alipay, error) {
    client, err := alipay.New(cfg.AppID, cfg.PrivateKey, !cfg.IsSandbox)
    if err != nil {
        return nil, err
    }

    if err := client.LoadAliPayPublicKey(cfg.AlipayPublicKey); err != nil {
        return nil, err
    }

    return &Alipay{client: client, config: cfg}, nil
}
```

### Create Mobile Payment

```go
func (a *Alipay) CreateWAPPayment(ctx context.Context, req *PaymentRequest) (string, error) {
    p := alipay.TradeWapPay{
        Trade: alipay.Trade{
            Subject:     req.Description,
            OutTradeNo:  req.OrderNo,
            TotalAmount: req.Amount.String(),
            ProductCode: "QUICK_WAP_WAY",
            NotifyURL:   a.config.NotifyURL,
            ReturnURL:   a.config.ReturnURL,
        },
    }

    url, err := a.client.TradeWapPay(p)
    if err != nil {
        return "", err
    }

    return url.String(), nil
}
```

### Alipay Callback Handler

```go
func (h *PaymentHandler) AlipayCallback(c *gin.Context) {
    // Parse form data
    if err := c.Request.ParseForm(); err != nil {
        c.String(400, "fail")
        return
    }

    // Verify signature
    if !h.alipay.VerifySign(c.Request.Form) {
        c.String(401, "fail")
        return
    }

    // Get notification data
    notification := &AlipayNotification{
        OutTradeNo: c.Request.Form.Get("out_trade_no"),
        TradeNo:    c.Request.Form.Get("trade_no"),
        TradeStatus: c.Request.Form.Get("trade_status"),
        TotalAmount: c.Request.Form.Get("total_amount"),
    }

    // Process based on status
    if notification.TradeStatus == "TRADE_SUCCESS" || notification.TradeStatus == "TRADE_FINISHED" {
        err := h.paymentService.HandleAlipaySuccess(c.Request.Context(), notification)
        if err != nil {
            c.String(500, "fail")
            return
        }
    }

    c.String(200, "success")
}
```

## Refund Processing

### Create Refund

```go
func (s *PaymentService) CreateRefund(ctx context.Context, orderID int64, amount decimal.Decimal, reason string) error {
    // Get original payment
    var payment models.Payment
    if err := s.db.Where("order_id = ? AND status = ?", orderID, PaymentStatusSuccess).First(&payment).Error; err != nil {
        return err
    }

    // Create refund record
    refund := &models.Refund{
        RefundNo:  generateRefundNo(),
        OrderID:   orderID,
        PaymentID: payment.ID,
        UserID:    payment.UserID,
        Amount:    amount,
        Reason:    reason,
        Status:    RefundStatusPending,
    }

    if err := s.db.Create(refund).Error; err != nil {
        return err
    }

    // Call payment gateway refund
    var err error
    switch payment.Channel {
    case "wechat":
        err = s.wechatPay.Refund(ctx, &RefundRequest{
            TransactionID: *payment.TradeNo,
            OutRefundNo:   refund.RefundNo,
            RefundAmount:  amount,
            TotalAmount:   payment.Amount,
            Reason:        reason,
        })
    case "alipay":
        err = s.alipay.Refund(ctx, &RefundRequest{
            TradeNo:      *payment.TradeNo,
            OutRequestNo: refund.RefundNo,
            RefundAmount: amount,
            Reason:       reason,
        })
    }

    if err != nil {
        refund.Status = RefundStatusFailed
        s.db.Save(refund)
        return err
    }

    refund.Status = RefundStatusProcessing
    return s.db.Save(refund).Error
}
```

### WeChat Refund

```go
func (w *WechatPay) Refund(ctx context.Context, req *RefundRequest) error {
    svc := refunddomestic.RefundsApiService{Client: w.client}

    _, _, err := svc.Create(ctx, refunddomestic.CreateRequest{
        TransactionId: core.String(req.TransactionID),
        OutRefundNo:   core.String(req.OutRefundNo),
        Reason:        core.String(req.Reason),
        NotifyUrl:     core.String(w.config.RefundNotifyURL),
        Amount: &refunddomestic.AmountReq{
            Refund:   core.Int64(req.RefundAmount.Mul(decimal.NewFromInt(100)).IntPart()),
            Total:    core.Int64(req.TotalAmount.Mul(decimal.NewFromInt(100)).IntPart()),
            Currency: core.String("CNY"),
        },
    })

    return err
}
```

## Wallet Payment

### Internal Wallet Payment

```go
func (s *PaymentService) PayWithWallet(ctx context.Context, userID, orderID int64, amount decimal.Decimal) error {
    return s.db.Transaction(func(tx *gorm.DB) error {
        // Lock wallet row
        var wallet models.UserWallet
        if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).
            Where("user_id = ?", userID).First(&wallet).Error; err != nil {
            return err
        }

        // Check balance
        if wallet.Balance.LessThan(amount) {
            return ErrInsufficientBalance
        }

        // Deduct balance
        newBalance := wallet.Balance.Sub(amount)
        if err := tx.Model(&wallet).
            Where("version = ?", wallet.Version).
            Updates(map[string]interface{}{
                "balance":        newBalance,
                "total_consumed": gorm.Expr("total_consumed + ?", amount),
                "version":        wallet.Version + 1,
            }).Error; err != nil {
            return err
        }

        // Create payment record
        payment := &models.Payment{
            PaymentNo: generatePaymentNo(),
            OrderID:   orderID,
            UserID:    userID,
            Channel:   "wallet",
            Amount:    amount,
            Status:    PaymentStatusSuccess,
            PayTime:   &time.Now(),
        }
        if err := tx.Create(payment).Error; err != nil {
            return err
        }

        // Update order
        if err := tx.Model(&models.Order{}).
            Where("id = ?", orderID).
            Updates(map[string]interface{}{
                "status":  OrderStatusPaid,
                "paid_at": time.Now(),
            }).Error; err != nil {
            return err
        }

        // Record transaction
        return tx.Create(&models.WalletTransaction{
            UserID:        userID,
            Type:          "consume",
            Amount:        amount.Neg(),
            BalanceBefore: wallet.Balance,
            BalanceAfter:  newBalance,
            OrderNo:       payment.PaymentNo,
        }).Error
    })
}
```

## Idempotency

### Payment Creation Idempotency

```go
func (s *PaymentService) CreatePayment(ctx context.Context, req *CreatePaymentRequest) (*PaymentResponse, error) {
    // Check idempotency key
    idempotencyKey := req.IdempotencyKey
    if idempotencyKey != "" {
        cached, err := s.redis.Get(ctx, "payment:idempotency:"+idempotencyKey).Result()
        if err == nil {
            var resp PaymentResponse
            json.Unmarshal([]byte(cached), &resp)
            return &resp, nil
        }
    }

    // Create payment
    resp, err := s.doCreatePayment(ctx, req)
    if err != nil {
        return nil, err
    }

    // Cache result
    if idempotencyKey != "" {
        data, _ := json.Marshal(resp)
        s.redis.Set(ctx, "payment:idempotency:"+idempotencyKey, data, 24*time.Hour)
    }

    return resp, nil
}
```

## Security Best Practices

1. **Verify signatures** on all callbacks
2. **Verify amounts** match order totals
3. **Use idempotency keys** for all payment operations
4. **Store certificates securely** (not in code)
5. **Use HTTPS** for all payment endpoints
6. **Log all transactions** for audit
7. **Handle timeouts** gracefully

## Additional Resources

### Reference Files

- **`references/wechat-pay-api.md`** - WeChat Pay API reference
- **`references/alipay-api.md`** - Alipay API reference

### Official Documentation

- [微信支付开发文档](https://pay.weixin.qq.com/wiki/doc/api/index.html)
- [支付宝开放平台文档](https://opendocs.alipay.com/open/)
