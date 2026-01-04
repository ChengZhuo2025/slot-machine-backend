// Package mall 提供商城服务
package mall

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"gorm.io/gorm"

	"github.com/dumeirei/smart-locker-backend/internal/common/errors"
	"github.com/dumeirei/smart-locker-backend/internal/common/utils"
	"github.com/dumeirei/smart-locker-backend/internal/models"
	"github.com/dumeirei/smart-locker-backend/internal/repository"
)

// MallOrderService 商城订单服务
type MallOrderService struct {
	db             *gorm.DB
	orderRepo      *repository.OrderRepository
	cartRepo       *repository.CartRepository
	productRepo    *repository.ProductRepository
	skuRepo        *repository.ProductSkuRepository
	productService *ProductService
}

// NewMallOrderService 创建商城订单服务
func NewMallOrderService(
	db *gorm.DB,
	orderRepo *repository.OrderRepository,
	cartRepo *repository.CartRepository,
	productRepo *repository.ProductRepository,
	skuRepo *repository.ProductSkuRepository,
	productService *ProductService,
) *MallOrderService {
	return &MallOrderService{
		db:             db,
		orderRepo:      orderRepo,
		cartRepo:       cartRepo,
		productRepo:    productRepo,
		skuRepo:        skuRepo,
		productService: productService,
	}
}

// OrderItemRequest 订单项请求
type OrderItemRequest struct {
	ProductID int64  `json:"product_id" binding:"required"`
	SkuID     *int64 `json:"sku_id"`
	Quantity  int    `json:"quantity" binding:"required,min=1"`
}

// CreateMallOrderRequest 创建商城订单请求
type CreateMallOrderRequest struct {
	Items     []OrderItemRequest `json:"items" binding:"required,min=1"`
	AddressID int64              `json:"address_id" binding:"required"`
	CouponID  *int64             `json:"coupon_id"`
	Remark    string             `json:"remark"`
}

// CreateFromCartRequest 从购物车创建订单请求
type CreateFromCartRequest struct {
	AddressID int64  `json:"address_id" binding:"required"`
	CouponID  *int64 `json:"coupon_id"`
	Remark    string `json:"remark"`
}

// MallOrderInfo 商城订单信息
type MallOrderInfo struct {
	ID             int64              `json:"id"`
	OrderNo        string             `json:"order_no"`
	Status         string             `json:"status"`
	StatusName     string             `json:"status_name"`
	OriginalAmount float64            `json:"original_amount"`
	DiscountAmount float64            `json:"discount_amount"`
	ActualAmount   float64            `json:"actual_amount"`
	Items          []*MallOrderItem   `json:"items"`
	Address        *AddressSnapshot   `json:"address,omitempty"`
	ExpressCompany string             `json:"express_company,omitempty"`
	ExpressNo      string             `json:"express_no,omitempty"`
	Remark         string             `json:"remark,omitempty"`
	CreatedAt      string             `json:"created_at"`
	PaidAt         string             `json:"paid_at,omitempty"`
	ShippedAt      string             `json:"shipped_at,omitempty"`
	ReceivedAt     string             `json:"received_at,omitempty"`
}

// MallOrderItem 订单项
type MallOrderItem struct {
	ProductID    int64             `json:"product_id"`
	ProductName  string            `json:"product_name"`
	ProductImage string            `json:"product_image"`
	SkuInfo      string            `json:"sku_info,omitempty"`
	Price        float64           `json:"price"`
	Quantity     int               `json:"quantity"`
	Subtotal     float64           `json:"subtotal"`
}

// AddressSnapshot 地址快照
type AddressSnapshot struct {
	ReceiverName  string `json:"receiver_name"`
	ReceiverPhone string `json:"receiver_phone"`
	Province      string `json:"province"`
	City          string `json:"city"`
	District      string `json:"district"`
	Detail        string `json:"detail"`
}

// CreateOrder 创建商城订单
func (s *MallOrderService) CreateOrder(ctx context.Context, userID int64, req *CreateMallOrderRequest) (*MallOrderInfo, error) {
	var order *models.Order
	var orderItems []*models.OrderItem

	err := s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		// 计算订单金额
		var originalAmount float64
		orderItems = make([]*models.OrderItem, len(req.Items))

		for i, item := range req.Items {
			// 获取商品信息
			product, err := s.productRepo.GetByID(ctx, item.ProductID)
			if err != nil {
				if err == gorm.ErrRecordNotFound {
					return errors.ErrProductNotFound.WithMessage(fmt.Sprintf("商品 %d 不存在", item.ProductID))
				}
				return err
			}

			if !product.IsOnSale {
				return errors.ErrProductOffShelf.WithMessage(fmt.Sprintf("商品 %s 已下架", product.Name))
			}

			price := product.Price
			var skuInfo string
			var productImage string

			// 解析商品图片
			if product.Images != nil {
				var images []string
				if json.Unmarshal(product.Images, &images) == nil && len(images) > 0 {
					productImage = images[0]
				}
			}

			// 如果有 SKU
			if item.SkuID != nil && *item.SkuID > 0 {
				sku, err := s.skuRepo.GetByID(ctx, *item.SkuID)
				if err != nil {
					if err == gorm.ErrRecordNotFound {
						return errors.ErrProductNotFound.WithMessage("商品规格不存在")
					}
					return err
				}
				if !sku.IsActive {
					return errors.ErrProductOffShelf.WithMessage("商品规格已下架")
				}
				if sku.Stock < item.Quantity {
					return errors.ErrStockInsufficient.WithMessage(fmt.Sprintf("商品 %s 库存不足", product.Name))
				}

				price = sku.Price

				// 解析 SKU 属性
				if sku.Attributes != nil {
					var attrs map[string]string
					if json.Unmarshal(sku.Attributes, &attrs) == nil {
						for k, v := range attrs {
							skuInfo += k + ":" + v + " "
						}
					}
				}

				if sku.Image != nil {
					productImage = *sku.Image
				}

				// 扣减 SKU 库存
				if err := s.skuRepo.DecreaseStock(ctx, *item.SkuID, item.Quantity); err != nil {
					return errors.ErrStockInsufficient.WithMessage(fmt.Sprintf("商品 %s 库存不足", product.Name))
				}
			} else {
				// 检查商品库存
				if product.Stock < item.Quantity {
					return errors.ErrStockInsufficient.WithMessage(fmt.Sprintf("商品 %s 库存不足", product.Name))
				}
			}

			// 扣减商品总库存
			if err := s.productRepo.DecreaseStock(ctx, item.ProductID, item.Quantity); err != nil {
				return errors.ErrStockInsufficient.WithMessage(fmt.Sprintf("商品 %s 库存不足", product.Name))
			}

			subtotal := price * float64(item.Quantity)
			originalAmount += subtotal

			orderItems[i] = &models.OrderItem{
				ProductID:    &item.ProductID,
				ProductName:  product.Name,
				ProductImage: &productImage,
				SkuInfo:      &skuInfo,
				Price:        price,
				Quantity:     item.Quantity,
				Subtotal:     subtotal,
			}
		}

		// TODO: 应用优惠券
		discountAmount := 0.0
		actualAmount := originalAmount - discountAmount

		// 获取地址信息（简化处理，实际应该查询数据库）
		addressSnapshot, _ := json.Marshal(AddressSnapshot{
			ReceiverName:  "收货人",
			ReceiverPhone: "13800138000",
			Province:      "广东省",
			City:          "深圳市",
			District:      "南山区",
			Detail:        "详细地址",
		})

		// 创建订单
		order = &models.Order{
			OrderNo:         utils.GenerateOrderNo("M"),
			UserID:          userID,
			Type:            models.OrderTypeMall,
			OriginalAmount:  originalAmount,
			DiscountAmount:  discountAmount,
			ActualAmount:    actualAmount,
			Status:          models.OrderStatusPending,
			CouponID:        req.CouponID,
			AddressID:       &req.AddressID,
			AddressSnapshot: addressSnapshot,
		}

		if req.Remark != "" {
			order.Remark = &req.Remark
		}

		if err := tx.Create(order).Error; err != nil {
			return err
		}

		// 创建订单项
		for _, item := range orderItems {
			item.OrderID = order.ID
			if err := tx.Create(item).Error; err != nil {
				return err
			}
		}

		return nil
	})

	if err != nil {
		return nil, err
	}

	return s.toMallOrderInfo(order, orderItems), nil
}

// CreateOrderFromCart 从购物车创建订单
func (s *MallOrderService) CreateOrderFromCart(ctx context.Context, userID int64, req *CreateFromCartRequest) (*MallOrderInfo, error) {
	// 获取选中的购物车项
	cartItems, err := s.cartRepo.ListSelectedByUserID(ctx, userID)
	if err != nil {
		return nil, errors.ErrDatabaseError.WithError(err)
	}

	if len(cartItems) == 0 {
		return nil, errors.ErrCartEmpty
	}

	// 转换为订单项请求
	items := make([]OrderItemRequest, len(cartItems))
	for i, cart := range cartItems {
		items[i] = OrderItemRequest{
			ProductID: cart.ProductID,
			SkuID:     cart.SkuID,
			Quantity:  cart.Quantity,
		}
	}

	// 创建订单
	orderInfo, err := s.CreateOrder(ctx, userID, &CreateMallOrderRequest{
		Items:     items,
		AddressID: req.AddressID,
		CouponID:  req.CouponID,
		Remark:    req.Remark,
	})

	if err != nil {
		return nil, err
	}

	// 清除已选中的购物车项
	if err := s.cartRepo.DeleteSelected(ctx, userID); err != nil {
		// 不影响订单创建，只记录日志
		fmt.Printf("清除购物车失败: %v\n", err)
	}

	return orderInfo, nil
}

// GetOrderDetail 获取订单详情
func (s *MallOrderService) GetOrderDetail(ctx context.Context, userID int64, orderID int64) (*MallOrderInfo, error) {
	order, err := s.orderRepo.GetByIDWithItems(ctx, orderID)
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, errors.ErrResourceNotFound
		}
		return nil, errors.ErrDatabaseError.WithError(err)
	}

	if order.UserID != userID {
		return nil, errors.ErrResourceNotFound
	}

	if order.Type != models.OrderTypeMall {
		return nil, errors.ErrResourceNotFound
	}

	return s.toMallOrderInfo(order, order.Items), nil
}

// GetUserOrders 获取用户商城订单列表
func (s *MallOrderService) GetUserOrders(ctx context.Context, userID int64, status string, page, pageSize int) ([]*MallOrderInfo, int64, error) {
	if page == 0 {
		page = 1
	}
	if pageSize == 0 {
		pageSize = 10
	}

	offset := (page - 1) * pageSize

	filters := map[string]interface{}{
		"type": models.OrderTypeMall,
	}
	if status != "" {
		filters["status"] = status
	}

	orders, total, err := s.orderRepo.ListByUserID(ctx, userID, offset, pageSize, filters)
	if err != nil {
		return nil, 0, errors.ErrDatabaseError.WithError(err)
	}

	result := make([]*MallOrderInfo, len(orders))
	for i, o := range orders {
		// 获取订单项
		items, _ := s.orderRepo.GetOrderItems(ctx, o.ID)
		result[i] = s.toMallOrderInfo(o, items)
	}

	return result, total, nil
}

// CancelOrder 取消订单
func (s *MallOrderService) CancelOrder(ctx context.Context, userID int64, orderID int64, reason string) error {
	order, err := s.orderRepo.GetByID(ctx, orderID)
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return errors.ErrResourceNotFound
		}
		return errors.ErrDatabaseError.WithError(err)
	}

	if order.UserID != userID {
		return errors.ErrResourceNotFound
	}

	if order.Status != models.OrderStatusPending {
		return errors.ErrOrderStatusError.WithMessage("订单状态不允许取消")
	}

	return s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		// 恢复库存
		items, err := s.orderRepo.GetOrderItems(ctx, orderID)
		if err != nil {
			return err
		}

		for _, item := range items {
			if item.ProductID != nil {
				if err := s.productRepo.IncreaseStock(ctx, *item.ProductID, item.Quantity); err != nil {
					return err
				}
			}
		}

		// 更新订单状态
		now := time.Now()
		return s.orderRepo.UpdateFields(ctx, orderID, map[string]interface{}{
			"status":        models.OrderStatusCancelled,
			"cancelled_at":  now,
			"cancel_reason": reason,
		})
	})
}

// ConfirmReceive 确认收货
func (s *MallOrderService) ConfirmReceive(ctx context.Context, userID int64, orderID int64) error {
	order, err := s.orderRepo.GetByID(ctx, orderID)
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return errors.ErrResourceNotFound
		}
		return errors.ErrDatabaseError.WithError(err)
	}

	if order.UserID != userID {
		return errors.ErrResourceNotFound
	}

	if order.Status != models.OrderStatusShipped {
		return errors.ErrOrderStatusError.WithMessage("订单状态不允许确认收货")
	}

	now := time.Now()
	return s.orderRepo.UpdateFields(ctx, orderID, map[string]interface{}{
		"status":       models.OrderStatusCompleted,
		"received_at":  now,
		"completed_at": now,
	})
}

// toMallOrderInfo 转换为商城订单信息
func (s *MallOrderService) toMallOrderInfo(order *models.Order, items []*models.OrderItem) *MallOrderInfo {
	info := &MallOrderInfo{
		ID:             order.ID,
		OrderNo:        order.OrderNo,
		Status:         order.Status,
		StatusName:     s.getStatusName(order.Status),
		OriginalAmount: order.OriginalAmount,
		DiscountAmount: order.DiscountAmount,
		ActualAmount:   order.ActualAmount,
		CreatedAt:      order.CreatedAt.Format("2006-01-02 15:04:05"),
	}

	if order.Remark != nil {
		info.Remark = *order.Remark
	}
	if order.ExpressCompany != nil {
		info.ExpressCompany = *order.ExpressCompany
	}
	if order.ExpressNo != nil {
		info.ExpressNo = *order.ExpressNo
	}
	if order.PaidAt != nil {
		info.PaidAt = order.PaidAt.Format("2006-01-02 15:04:05")
	}
	if order.ShippedAt != nil {
		info.ShippedAt = order.ShippedAt.Format("2006-01-02 15:04:05")
	}
	if order.ReceivedAt != nil {
		info.ReceivedAt = order.ReceivedAt.Format("2006-01-02 15:04:05")
	}

	// 解析地址快照
	if order.AddressSnapshot != nil {
		var addr AddressSnapshot
		if json.Unmarshal(order.AddressSnapshot, &addr) == nil {
			info.Address = &addr
		}
	}

	// 订单项
	info.Items = make([]*MallOrderItem, len(items))
	for i, item := range items {
		info.Items[i] = &MallOrderItem{
			ProductName: item.ProductName,
			Price:       item.Price,
			Quantity:    item.Quantity,
			Subtotal:    item.Subtotal,
		}
		if item.ProductID != nil {
			info.Items[i].ProductID = *item.ProductID
		}
		if item.ProductImage != nil {
			info.Items[i].ProductImage = *item.ProductImage
		}
		if item.SkuInfo != nil {
			info.Items[i].SkuInfo = *item.SkuInfo
		}
	}

	return info
}

// getStatusName 获取状态名称
func (s *MallOrderService) getStatusName(status string) string {
	switch status {
	case models.OrderStatusPending:
		return "待付款"
	case models.OrderStatusPaid:
		return "待发货"
	case models.OrderStatusPendingShip:
		return "待发货"
	case models.OrderStatusShipped:
		return "已发货"
	case models.OrderStatusCompleted:
		return "已完成"
	case models.OrderStatusCancelled:
		return "已取消"
	case models.OrderStatusRefunding:
		return "退款中"
	case models.OrderStatusRefunded:
		return "已退款"
	default:
		return status
	}
}
