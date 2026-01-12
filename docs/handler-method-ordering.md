# Handler 方法排序规范

## 目标

统一所有 Handler 的方法排序，提高代码可读性和一致性。

## 推荐排序

Handler 方法应按以下顺序排列：

### 1. 结构体定义和构造函数

```go
type XxxHandler struct {
    xxxService *service.XxxService
}

func NewXxxHandler(xxxSvc *service.XxxService) *XxxHandler {
    return &XxxHandler{
        xxxService: xxxSvc,
    }
}
```

### 2. 标准 CRUD 操作（按 RESTful 顺序）

遵循标准 HTTP 方法顺序：

1. **Create** (POST) - 创建资源
2. **Get** (GET /:id) - 获取单个资源
3. **List** (GET /) - 获取资源列表
4. **Update** (PUT /:id) - 更新资源
5. **Delete** (DELETE /:id) - 删除资源

```go
// Create 创建资源
func (h *XxxHandler) Create(c *gin.Context) { }

// Get 获取资源详情
func (h *XxxHandler) Get(c *gin.Context) { }

// List 获取资源列表
func (h *XxxHandler) List(c *gin.Context) { }

// Update 更新资源
func (h *XxxHandler) Update(c *gin.Context) { }

// Delete 删除资源
func (h *XxxHandler) Delete(c *gin.Context) { }
```

### 3. 状态和属性更新操作

在 CRUD 之后，按字母顺序排列：

```go
// UpdateStatus 更新状态
func (h *XxxHandler) UpdateStatus(c *gin.Context) { }

// UpdateAttribute 更新属性
func (h *XxxHandler) UpdateAttribute(c *gin.Context) { }
```

### 4. 特定资源操作

按功能分组，每组内按字母顺序：

```go
// 操作类方法
func (h *DeviceHandler) RemoteLock(c *gin.Context) { }
func (h *DeviceHandler) RemoteUnlock(c *gin.Context) { }

// 查询类方法
func (h *DeviceHandler) GetLogs(c *gin.Context) { }
func (h *DeviceHandler) GetStatistics(c *gin.Context) { }

// 维护类方法
func (h *DeviceHandler) CompleteMaintenance(c *gin.Context) { }
func (h *DeviceHandler) CreateMaintenance(c *gin.Context) { }
func (h *DeviceHandler) ListMaintenance(c *gin.Context) { }
```

### 5. 关联资源操作

处理关联资源的方法：

```go
// ListByParent 获取父资源下的子资源列表
func (h *VenueHandler) ListByMerchant(c *gin.Context) { }
```

### 6. 路由注册

最后是路由注册方法：

```go
// RegisterRoutes 注册路由
func (h *XxxHandler) RegisterRoutes(r *gin.RouterGroup) { }

// RegisterProtectedRoutes 注册需要认证的路由（如果需要）
func (h *XxxHandler) RegisterProtectedRoutes(r *gin.RouterGroup) { }
```

## 完整示例

```go
// Package admin 提供管理员相关的 HTTP Handler
package admin

// VenueHandler 场地管理处理器
type VenueHandler struct {
    venueService *service.VenueService
}

// NewVenueHandler 创建场地管理处理器
func NewVenueHandler(venueSvc *service.VenueService) *VenueHandler {
    return &VenueHandler{
        venueService: venueSvc,
    }
}

// ============ CRUD 操作 ============

// Create 创建场地
func (h *VenueHandler) Create(c *gin.Context) { }

// Get 获取场地详情
func (h *VenueHandler) Get(c *gin.Context) { }

// List 获取场地列表
func (h *VenueHandler) List(c *gin.Context) { }

// Update 更新场地
func (h *VenueHandler) Update(c *gin.Context) { }

// Delete 删除场地
func (h *VenueHandler) Delete(c *gin.Context) { }

// ============ 状态更新 ============

// UpdateStatus 更新场地状态
func (h *VenueHandler) UpdateStatus(c *gin.Context) { }

// ============ 关联资源 ============

// ListByMerchant 获取商户下的场地列表
func (h *VenueHandler) ListByMerchant(c *gin.Context) { }

// ============ 路由注册 ============

// RegisterRoutes 注册路由
func (h *VenueHandler) RegisterRoutes(r *gin.RouterGroup) { }
```

## 当前状态

### 已规范化的 Handler

- ✅ `venue_handler.go` - 基本符合规范（仅需微调顺序）
- ✅ `device_handler.go` - 基本符合规范

### 需要调整的 Handler

大部分 Handler 已经接近规范，主要需要调整的是：

1. **Get 和 List 顺序**：部分 handler 将 Get 放在了 List 之后，应该将 Get 放在 List 之前
2. **UpdateStatus 位置**：应该放在 Delete 之后，而不是在 Update 和 Delete 之间
3. **方法分组**：可以添加注释分隔不同类型的方法，提高可读性

## 实施建议

1. **新建 Handler**：严格按照此规范排序
2. **现有 Handler**：在重大重构时调整，或者逐步调整
3. **代码审查**：在 PR 审查时检查方法排序是否符合规范

## 注意事项

1. 保持一致性比完全符合规范更重要
2. 如果某个 Handler 有特殊需求，可以适当调整，但应该在代码注释中说明原因
3. 使用注释分隔不同类型的操作，提高代码可读性

## 工具支持

可以考虑编写 lint 工具来自动检查 Handler 方法顺序，但目前暂不强制执行。
