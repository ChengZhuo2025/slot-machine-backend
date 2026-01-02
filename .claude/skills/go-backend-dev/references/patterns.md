# Advanced Go Patterns for Smart Locker Backend

## Dependency Injection

### Wire-based DI

Use Google Wire for compile-time dependency injection:

```go
// wire.go
//+build wireinject

package main

import (
    "github.com/google/wire"
    "backend/internal/handler/user"
    "backend/internal/service/user"
    "backend/internal/repository"
)

func InitializeApp() (*App, error) {
    wire.Build(
        // Database
        provideDB,

        // Repositories
        repository.NewUserRepository,
        repository.NewOrderRepository,

        // Services
        user.NewService,

        // Handlers
        user.NewHandler,

        // App
        NewApp,
    )
    return nil, nil
}
```

## Transaction Management

### Unit of Work Pattern

```go
package repository

import (
    "context"
    "gorm.io/gorm"
)

type UnitOfWork struct {
    db *gorm.DB
}

func NewUnitOfWork(db *gorm.DB) *UnitOfWork {
    return &UnitOfWork{db: db}
}

func (u *UnitOfWork) Transaction(ctx context.Context, fn func(tx *gorm.DB) error) error {
    return u.db.WithContext(ctx).Transaction(fn)
}

// Usage in service
func (s *OrderService) CreateOrder(ctx context.Context, req *CreateOrderRequest) error {
    return s.uow.Transaction(ctx, func(tx *gorm.DB) error {
        // Create order
        order := &models.Order{...}
        if err := tx.Create(order).Error; err != nil {
            return err
        }

        // Create order items
        for _, item := range req.Items {
            orderItem := &models.OrderItem{OrderID: order.ID, ...}
            if err := tx.Create(orderItem).Error; err != nil {
                return err
            }
        }

        // Deduct inventory
        if err := tx.Model(&models.Product{}).
            Where("id IN ?", productIDs).
            Update("stock", gorm.Expr("stock - ?", 1)).Error; err != nil {
            return err
        }

        return nil
    })
}
```

## Concurrency Patterns

### Worker Pool

```go
package worker

import (
    "context"
    "sync"
)

type Job func(ctx context.Context) error

type Pool struct {
    workers int
    jobs    chan Job
    wg      sync.WaitGroup
}

func NewPool(workers int) *Pool {
    return &Pool{
        workers: workers,
        jobs:    make(chan Job, 100),
    }
}

func (p *Pool) Start(ctx context.Context) {
    for i := 0; i < p.workers; i++ {
        p.wg.Add(1)
        go func() {
            defer p.wg.Done()
            for {
                select {
                case job, ok := <-p.jobs:
                    if !ok {
                        return
                    }
                    _ = job(ctx)
                case <-ctx.Done():
                    return
                }
            }
        }()
    }
}

func (p *Pool) Submit(job Job) {
    p.jobs <- job
}

func (p *Pool) Stop() {
    close(p.jobs)
    p.wg.Wait()
}
```

### Rate Limiting

```go
package middleware

import (
    "github.com/gin-gonic/gin"
    "golang.org/x/time/rate"
    "sync"
)

type RateLimiter struct {
    limiters map[string]*rate.Limiter
    mu       sync.RWMutex
    rate     rate.Limit
    burst    int
}

func NewRateLimiter(r rate.Limit, b int) *RateLimiter {
    return &RateLimiter{
        limiters: make(map[string]*rate.Limiter),
        rate:     r,
        burst:    b,
    }
}

func (rl *RateLimiter) getLimiter(key string) *rate.Limiter {
    rl.mu.Lock()
    defer rl.mu.Unlock()

    limiter, exists := rl.limiters[key]
    if !exists {
        limiter = rate.NewLimiter(rl.rate, rl.burst)
        rl.limiters[key] = limiter
    }

    return limiter
}

func (rl *RateLimiter) Middleware() gin.HandlerFunc {
    return func(c *gin.Context) {
        key := c.ClientIP()
        limiter := rl.getLimiter(key)

        if !limiter.Allow() {
            c.AbortWithStatusJSON(429, gin.H{
                "code":    429,
                "message": "too many requests",
            })
            return
        }

        c.Next()
    }
}
```

## Caching Patterns

### Cache-Aside Pattern

```go
package cache

import (
    "context"
    "encoding/json"
    "time"
    "github.com/redis/go-redis/v9"
)

type CacheAside[T any] struct {
    client *redis.Client
    ttl    time.Duration
}

func NewCacheAside[T any](client *redis.Client, ttl time.Duration) *CacheAside[T] {
    return &CacheAside[T]{client: client, ttl: ttl}
}

func (c *CacheAside[T]) Get(ctx context.Context, key string, fetch func() (T, error)) (T, error) {
    var result T

    // Try cache first
    data, err := c.client.Get(ctx, key).Bytes()
    if err == nil {
        if err := json.Unmarshal(data, &result); err == nil {
            return result, nil
        }
    }

    // Cache miss, fetch from source
    result, err = fetch()
    if err != nil {
        return result, err
    }

    // Store in cache
    data, _ = json.Marshal(result)
    c.client.Set(ctx, key, data, c.ttl)

    return result, nil
}
```

### Distributed Lock

```go
package lock

import (
    "context"
    "time"
    "github.com/redis/go-redis/v9"
    "github.com/google/uuid"
)

type DistributedLock struct {
    client *redis.Client
    key    string
    value  string
    ttl    time.Duration
}

func NewDistributedLock(client *redis.Client, key string, ttl time.Duration) *DistributedLock {
    return &DistributedLock{
        client: client,
        key:    "lock:" + key,
        value:  uuid.New().String(),
        ttl:    ttl,
    }
}

func (l *DistributedLock) Acquire(ctx context.Context) (bool, error) {
    return l.client.SetNX(ctx, l.key, l.value, l.ttl).Result()
}

func (l *DistributedLock) Release(ctx context.Context) error {
    script := `
        if redis.call("get", KEYS[1]) == ARGV[1] then
            return redis.call("del", KEYS[1])
        else
            return 0
        end
    `
    _, err := l.client.Eval(ctx, script, []string{l.key}, l.value).Result()
    return err
}

// Usage for device rental
func (s *RentalService) CreateRental(ctx context.Context, deviceNo string) error {
    lock := lock.NewDistributedLock(s.redis, "device:"+deviceNo, 10*time.Second)

    acquired, err := lock.Acquire(ctx)
    if err != nil {
        return err
    }
    if !acquired {
        return errors.ErrDeviceBusy
    }
    defer lock.Release(ctx)

    // Check device status and create rental
    // ...
}
```

## Event-Driven Patterns

### Domain Events

```go
package events

type EventType string

const (
    OrderCreated    EventType = "order.created"
    OrderPaid       EventType = "order.paid"
    OrderCompleted  EventType = "order.completed"
    DeviceUnlocked  EventType = "device.unlocked"
    DeviceLocked    EventType = "device.locked"
)

type Event struct {
    Type      EventType
    Payload   interface{}
    Timestamp time.Time
}

type EventBus struct {
    handlers map[EventType][]func(Event)
    mu       sync.RWMutex
}

func (b *EventBus) Subscribe(eventType EventType, handler func(Event)) {
    b.mu.Lock()
    defer b.mu.Unlock()
    b.handlers[eventType] = append(b.handlers[eventType], handler)
}

func (b *EventBus) Publish(event Event) {
    b.mu.RLock()
    handlers := b.handlers[event.Type]
    b.mu.RUnlock()

    for _, handler := range handlers {
        go handler(event)
    }
}
```

## Graceful Shutdown

```go
package main

import (
    "context"
    "net/http"
    "os"
    "os/signal"
    "syscall"
    "time"
)

func main() {
    srv := &http.Server{
        Addr:    ":8080",
        Handler: router,
    }

    go func() {
        if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
            log.Fatalf("listen: %s\n", err)
        }
    }()

    // Wait for interrupt signal
    quit := make(chan os.Signal, 1)
    signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
    <-quit

    log.Println("Shutting down server...")

    // Give outstanding requests 30 seconds to complete
    ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
    defer cancel()

    if err := srv.Shutdown(ctx); err != nil {
        log.Fatal("Server forced to shutdown:", err)
    }

    log.Println("Server exiting")
}
```

## Optimistic Locking

For wallet balance updates:

```go
func (r *WalletRepository) UpdateBalance(ctx context.Context, userID int64, amount decimal.Decimal) error {
    return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
        var wallet models.UserWallet
        if err := tx.Where("user_id = ?", userID).First(&wallet).Error; err != nil {
            return err
        }

        newBalance := wallet.Balance.Add(amount)
        if newBalance.LessThan(decimal.Zero) {
            return errors.ErrInsufficientBalance
        }

        result := tx.Model(&wallet).
            Where("version = ?", wallet.Version).
            Updates(map[string]interface{}{
                "balance": newBalance,
                "version": wallet.Version + 1,
            })

        if result.RowsAffected == 0 {
            return errors.ErrConcurrentModification
        }

        return nil
    })
}
```
