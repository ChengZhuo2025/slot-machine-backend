# 模块依赖关系

## 模块交互图

```
                              ┌─────────────┐
                              │   Gateway   │
                              │  (API入口)   │
                              └──────┬──────┘
                                     │
         ┌───────────────────────────┼───────────────────────────┐
         │                           │                           │
         ▼                           ▼                           ▼
┌─────────────────┐        ┌─────────────────┐        ┌─────────────────┐
│      Auth       │        │      User       │        │     Device      │
│   认证授权模块    │◄───────│    用户模块     │        │    设备模块     │
└─────────────────┘        └────────┬────────┘        └────────┬────────┘
                                    │                          │
                           ┌────────┴────────┐                 │
                           ▼                 ▼                 │
                  ┌─────────────┐    ┌─────────────┐           │
                  │  Marketing  │    │Distribution │           │
                  │   营销模块   │    │   分销模块   │           │
                  └──────┬──────┘    └──────┬──────┘           │
                         │                  │                  │
         ┌───────────────┴──────────────────┴──────────────────┤
         │                                                     │
         ▼                                                     ▼
┌─────────────────┐        ┌─────────────────┐        ┌─────────────────┐
│     Order       │◄───────│     Rental      │◄───────│      Hotel      │
│    订单模块     │        │    租借模块     │        │    酒店模块     │
└────────┬────────┘        └─────────────────┘        └─────────────────┘
         │                          │                          │
         │                          │                          │
         ▼                          ▼                          ▼
┌─────────────────┐        ┌─────────────────┐        ┌─────────────────┐
│    Payment      │        │      Mall       │        │    Finance      │
│    支付模块     │        │    商城模块     │        │    财务模块     │
└─────────────────┘        └─────────────────┘        └─────────────────┘
```

## 模块职责与依赖

### Auth (认证授权)

**职责**: 用户登录、JWT 管理、权限验证

**被依赖**: 所有需要认证的模块

**依赖**:
- User (获取用户信息)
- Redis (Token 存储)

```go
// 调用示例
authService.ValidateToken(token) → user_id
authService.CheckPermission(user_id, permission)
```

---

### User (用户)

**职责**: 用户信息、钱包、会员等级、地址

**被依赖**: Order, Rental, Hotel, Distribution, Marketing

**依赖**:
- Auth (认证)
- Marketing (会员权益)

```go
// 调用示例
userService.GetProfile(userID)
userService.UpdateWallet(userID, amount, txType)
userService.GetMemberLevel(userID)
```

---

### Device (设备)

**职责**: 设备管理、场地、商户、MQTT 通信

**被依赖**: Rental, Hotel

**依赖**:
- MQTT (设备通信)
- Redis (状态缓存)

```go
// 调用示例
deviceService.GetByDeviceNo(deviceNo)
deviceService.SendUnlockCommand(deviceNo, rentalID)
deviceService.UpdateStatus(deviceNo, status)
```

---

### Rental (租借)

**职责**: 租借订单、定价、超时处理

**被依赖**: Order

**依赖**:
- Device (设备控制)
- Order (订单创建)
- User (钱包操作)
- Payment (支付处理)

```go
// 调用示例
rentalService.Create(userID, deviceNo, durationHours)
rentalService.Return(rentalID)
rentalService.HandleOvertime(rentalID)
```

---

### Hotel (酒店)

**职责**: 酒店管理、房间、预订、核销

**被依赖**: Order

**依赖**:
- Device (智能柜控制)
- Order (订单创建)
- Payment (支付处理)

```go
// 调用示例
hotelService.CreateBooking(userID, roomID, checkIn, checkOut)
hotelService.Verify(bookingID, verificationCode)
hotelService.UnlockByCode(bookingID, unlockCode)
```

---

### Mall (商城)

**职责**: 商品、分类、购物车、评价

**被依赖**: Order

**依赖**:
- Order (订单创建)
- User (用户信息)

```go
// 调用示例
mallService.GetProducts(categoryID, page, pageSize)
mallService.AddToCart(userID, productID, quantity)
mallService.CreateOrder(userID, cartItemIDs, addressID)
```

---

### Order (订单)

**职责**: 统一订单管理、状态流转

**被依赖**: Payment, Finance, Distribution

**依赖**:
- User (用户信息)
- Payment (支付处理)
- Marketing (优惠券核销)

```go
// 调用示例
orderService.Create(userID, orderType, items)
orderService.UpdateStatus(orderID, status)
orderService.Cancel(orderID, reason)
```

---

### Payment (支付)

**职责**: 支付创建、回调处理、退款

**被依赖**: Order, Rental, Hotel, Mall

**依赖**:
- Order (订单状态)
- User (钱包支付)
- WeChat/Alipay SDK

```go
// 调用示例
paymentService.Create(orderID, channel)
paymentService.HandleCallback(notification)
paymentService.Refund(orderID, amount, reason)
```

---

### Distribution (分销)

**职责**: 分销商、佣金计算、团队管理

**依赖**:
- User (用户信息)
- Order (订单数据)
- Finance (提现处理)

```go
// 调用示例
distributionService.CalculateCommission(orderID)
distributionService.GetTeamMembers(distributorID)
distributionService.Withdraw(distributorID, amount)
```

---

### Marketing (营销)

**职责**: 优惠券、活动、会员套餐

**被依赖**: Order, User

**依赖**:
- User (用户信息)

```go
// 调用示例
marketingService.GetAvailableCoupons(userID)
marketingService.UseCoupon(userCouponID, orderID)
marketingService.PurchaseMemberPackage(userID, packageID)
```

---

### Finance (财务)

**职责**: 结算、提现审核、报表

**依赖**:
- Order (订单数据)
- Distribution (分销结算)
- User (提现账户)

```go
// 调用示例
financeService.CreateSettlement(merchantID, period)
financeService.ApproveWithdrawal(withdrawalID)
financeService.GenerateReport(startDate, endDate)
```

---

## 数据流向

### 租借业务数据流

```
User Request
    │
    ▼
┌─────────┐    ┌─────────┐    ┌─────────┐
│ Rental  │───►│  Order  │───►│ Payment │
└────┬────┘    └─────────┘    └────┬────┘
     │                              │
     ▼                              ▼
┌─────────┐                   ┌─────────┐
│ Device  │◄──────────────────│ Finance │
└─────────┘   (支付成功后开锁)  └─────────┘
```

### 分销佣金数据流

```
Order Completed
    │
    ▼
┌──────────────┐    ┌──────────────┐
│ Distribution │───►│   Finance    │
│ (计算佣金)    │    │  (结算/提现) │
└──────────────┘    └──────────────┘
```

## 事件驱动通信

模块间异步通信使用 Redis Stream：

| Event | Publisher | Subscribers |
|-------|-----------|-------------|
| `order.paid` | Payment | Rental, Hotel, Mall, Distribution |
| `order.completed` | Order | Distribution, Finance |
| `device.status_changed` | Device | Rental, Notification |
| `user.level_upgraded` | User | Marketing, Notification |

```go
// 发布事件
eventBus.Publish(events.Event{
    Type:    events.OrderPaid,
    Payload: &OrderPaidPayload{OrderID: orderID},
})

// 订阅事件
eventBus.Subscribe(events.OrderPaid, func(e events.Event) {
    payload := e.Payload.(*OrderPaidPayload)
    distributionService.CalculateCommission(payload.OrderID)
})
```
