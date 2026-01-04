// Package mall 提供商城服务
package mall

import (
	"context"
	"encoding/json"

	"gorm.io/gorm"

	"github.com/dumeirei/smart-locker-backend/internal/common/errors"
	"github.com/dumeirei/smart-locker-backend/internal/models"
	"github.com/dumeirei/smart-locker-backend/internal/repository"
)

// CartService 购物车服务
type CartService struct {
	db          *gorm.DB
	cartRepo    *repository.CartRepository
	productRepo *repository.ProductRepository
	skuRepo     *repository.ProductSkuRepository
}

// NewCartService 创建购物车服务
func NewCartService(
	db *gorm.DB,
	cartRepo *repository.CartRepository,
	productRepo *repository.ProductRepository,
	skuRepo *repository.ProductSkuRepository,
) *CartService {
	return &CartService{
		db:          db,
		cartRepo:    cartRepo,
		productRepo: productRepo,
		skuRepo:     skuRepo,
	}
}

// CartItemInfo 购物车项信息
type CartItemInfo struct {
	ID           int64             `json:"id"`
	ProductID    int64             `json:"product_id"`
	ProductName  string            `json:"product_name"`
	ProductImage string            `json:"product_image"`
	SkuID        *int64            `json:"sku_id,omitempty"`
	SkuCode      string            `json:"sku_code,omitempty"`
	Attributes   map[string]string `json:"attributes,omitempty"`
	Price        float64           `json:"price"`
	Quantity     int               `json:"quantity"`
	Subtotal     float64           `json:"subtotal"`
	Selected     bool              `json:"selected"`
	Stock        int               `json:"stock"`
	IsOnSale     bool              `json:"is_on_sale"`
}

// CartInfo 购物车信息
type CartInfo struct {
	Items         []*CartItemInfo `json:"items"`
	TotalCount    int             `json:"total_count"`
	SelectedCount int             `json:"selected_count"`
	TotalAmount   float64         `json:"total_amount"`
}

// AddCartItemRequest 添加购物车请求
type AddCartItemRequest struct {
	ProductID int64  `json:"product_id" binding:"required"`
	SkuID     *int64 `json:"sku_id"`
	Quantity  int    `json:"quantity" binding:"required,min=1"`
}

// UpdateCartItemRequest 更新购物车请求
type UpdateCartItemRequest struct {
	Quantity int  `json:"quantity" binding:"required,min=1"`
	Selected *bool `json:"selected"`
}

// GetCart 获取购物车
func (s *CartService) GetCart(ctx context.Context, userID int64) (*CartInfo, error) {
	items, err := s.cartRepo.ListByUserID(ctx, userID)
	if err != nil {
		return nil, errors.ErrDatabaseError.WithError(err)
	}

	cartInfo := &CartInfo{
		Items: make([]*CartItemInfo, 0),
	}

	for _, item := range items {
		itemInfo := s.toCartItemInfo(item)
		cartInfo.Items = append(cartInfo.Items, itemInfo)
		cartInfo.TotalCount += item.Quantity

		if item.Selected {
			cartInfo.SelectedCount += item.Quantity
			cartInfo.TotalAmount += itemInfo.Subtotal
		}
	}

	return cartInfo, nil
}

// AddItem 添加商品到购物车
func (s *CartService) AddItem(ctx context.Context, userID int64, req *AddCartItemRequest) (*CartItemInfo, error) {
	// 检查商品是否存在且上架
	product, err := s.productRepo.GetByID(ctx, req.ProductID)
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, errors.ErrProductNotFound
		}
		return nil, errors.ErrDatabaseError.WithError(err)
	}

	if !product.IsOnSale {
		return nil, errors.ErrProductOffShelf
	}

	// 如果有 SKU，检查 SKU 是否存在
	if req.SkuID != nil && *req.SkuID > 0 {
		sku, err := s.skuRepo.GetByID(ctx, *req.SkuID)
		if err != nil {
			if err == gorm.ErrRecordNotFound {
				return nil, errors.ErrProductNotFound
			}
			return nil, errors.ErrDatabaseError.WithError(err)
		}
		if !sku.IsActive {
			return nil, errors.ErrProductOffShelf
		}
		if sku.ProductID != req.ProductID {
			return nil, errors.ErrInvalidParams.WithMessage("规格不属于该商品")
		}
	}

	// 检查购物车是否已有该商品
	existingItem, err := s.cartRepo.GetByUserIDAndProductSku(ctx, userID, req.ProductID, req.SkuID)
	if err != nil && err != gorm.ErrRecordNotFound {
		return nil, errors.ErrDatabaseError.WithError(err)
	}

	if existingItem != nil {
		// 更新数量
		existingItem.Quantity += req.Quantity
		if err := s.cartRepo.Update(ctx, existingItem); err != nil {
			return nil, errors.ErrDatabaseError.WithError(err)
		}
		return s.getCartItemInfo(ctx, existingItem.ID)
	}

	// 创建新购物车项
	item := &models.CartItem{
		UserID:    userID,
		ProductID: req.ProductID,
		SkuID:     req.SkuID,
		Quantity:  req.Quantity,
		Selected:  true,
	}

	if err := s.cartRepo.Create(ctx, item); err != nil {
		return nil, errors.ErrDatabaseError.WithError(err)
	}

	return s.getCartItemInfo(ctx, item.ID)
}

// UpdateItem 更新购物车项
func (s *CartService) UpdateItem(ctx context.Context, userID, itemID int64, req *UpdateCartItemRequest) (*CartItemInfo, error) {
	item, err := s.cartRepo.GetByID(ctx, itemID)
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, errors.ErrResourceNotFound
		}
		return nil, errors.ErrDatabaseError.WithError(err)
	}

	if item.UserID != userID {
		return nil, errors.ErrResourceNotFound
	}

	// 更新数量
	if req.Quantity > 0 {
		item.Quantity = req.Quantity
	}

	// 更新选中状态
	if req.Selected != nil {
		item.Selected = *req.Selected
	}

	if err := s.cartRepo.Update(ctx, item); err != nil {
		return nil, errors.ErrDatabaseError.WithError(err)
	}

	return s.getCartItemInfo(ctx, item.ID)
}

// RemoveItem 移除购物车项
func (s *CartService) RemoveItem(ctx context.Context, userID, itemID int64) error {
	item, err := s.cartRepo.GetByID(ctx, itemID)
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return errors.ErrResourceNotFound
		}
		return errors.ErrDatabaseError.WithError(err)
	}

	if item.UserID != userID {
		return errors.ErrResourceNotFound
	}

	return s.cartRepo.Delete(ctx, itemID)
}

// RemoveItems 批量移除购物车项
func (s *CartService) RemoveItems(ctx context.Context, userID int64, itemIDs []int64) error {
	return s.cartRepo.DeleteByIDs(ctx, userID, itemIDs)
}

// ClearCart 清空购物车
func (s *CartService) ClearCart(ctx context.Context, userID int64) error {
	return s.cartRepo.DeleteByUserID(ctx, userID)
}

// ClearSelected 清空选中项
func (s *CartService) ClearSelected(ctx context.Context, userID int64) error {
	return s.cartRepo.DeleteSelected(ctx, userID)
}

// SelectAll 全选
func (s *CartService) SelectAll(ctx context.Context, userID int64, selected bool) error {
	return s.cartRepo.UpdateAllSelected(ctx, userID, selected)
}

// GetSelectedItems 获取选中的购物车项
func (s *CartService) GetSelectedItems(ctx context.Context, userID int64) ([]*CartItemInfo, error) {
	items, err := s.cartRepo.ListSelectedByUserID(ctx, userID)
	if err != nil {
		return nil, errors.ErrDatabaseError.WithError(err)
	}

	result := make([]*CartItemInfo, len(items))
	for i, item := range items {
		result[i] = s.toCartItemInfo(item)
	}
	return result, nil
}

// GetCartCount 获取购物车商品数量
func (s *CartService) GetCartCount(ctx context.Context, userID int64) (int, error) {
	count, err := s.cartRepo.SumQuantityByUserID(ctx, userID)
	if err != nil {
		return 0, errors.ErrDatabaseError.WithError(err)
	}
	return count, nil
}

// getCartItemInfo 获取购物车项详情
func (s *CartService) getCartItemInfo(ctx context.Context, itemID int64) (*CartItemInfo, error) {
	items, err := s.cartRepo.ListByUserID(ctx, 0) // 这里需要获取单个项
	if err != nil {
		return nil, errors.ErrDatabaseError.WithError(err)
	}

	// 重新获取带关联的数据
	item, err := s.cartRepo.GetByID(ctx, itemID)
	if err != nil {
		return nil, errors.ErrDatabaseError.WithError(err)
	}

	// 预加载关联数据
	var fullItem models.CartItem
	err = s.db.WithContext(ctx).
		Preload("Product").
		Preload("Sku").
		First(&fullItem, itemID).Error
	if err != nil {
		// 如果加载失败，返回基本信息
		return &CartItemInfo{
			ID:        item.ID,
			ProductID: item.ProductID,
			SkuID:     item.SkuID,
			Quantity:  item.Quantity,
			Selected:  item.Selected,
		}, nil
	}

	_ = items // 静默使用

	return s.toCartItemInfo(&fullItem), nil
}

// toCartItemInfo 转换为购物车项信息
func (s *CartService) toCartItemInfo(item *models.CartItem) *CartItemInfo {
	info := &CartItemInfo{
		ID:        item.ID,
		ProductID: item.ProductID,
		SkuID:     item.SkuID,
		Quantity:  item.Quantity,
		Selected:  item.Selected,
	}

	// 商品信息
	if item.Product != nil {
		info.ProductName = item.Product.Name
		info.Price = item.Product.Price
		info.Stock = item.Product.Stock
		info.IsOnSale = item.Product.IsOnSale

		// 解析商品图片
		if item.Product.Images != nil {
			var images []string
			if json.Unmarshal(item.Product.Images, &images) == nil && len(images) > 0 {
				info.ProductImage = images[0]
			}
		}
	}

	// SKU 信息
	if item.Sku != nil {
		info.SkuCode = item.Sku.SkuCode
		info.Price = item.Sku.Price // SKU 价格覆盖商品价格
		info.Stock = item.Sku.Stock

		// 解析 SKU 属性
		if item.Sku.Attributes != nil {
			_ = json.Unmarshal(item.Sku.Attributes, &info.Attributes)
		}

		// SKU 图片
		if item.Sku.Image != nil {
			info.ProductImage = *item.Sku.Image
		}
	}

	// 计算小计
	info.Subtotal = info.Price * float64(info.Quantity)

	return info
}
