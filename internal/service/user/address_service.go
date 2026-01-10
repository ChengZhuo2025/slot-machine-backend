// Package user 用户服务
package user

import (
	"context"
	"fmt"

	"github.com/dumeirei/smart-locker-backend/internal/models"
	"github.com/dumeirei/smart-locker-backend/internal/repository"
)

// AddressService 地址服务
type AddressService struct {
	addressRepo *repository.AddressRepository
}

// NewAddressService 创建地址服务
func NewAddressService(addressRepo *repository.AddressRepository) *AddressService {
	return &AddressService{addressRepo: addressRepo}
}

// MaxAddressCount 每个用户最大地址数量
const MaxAddressCount = 20

// CreateAddressRequest 创建地址请求
type CreateAddressRequest struct {
	ReceiverName  string  `json:"receiver_name" binding:"required,max=50"`
	ReceiverPhone string  `json:"receiver_phone" binding:"required,max=20"`
	Province      string  `json:"province" binding:"required,max=50"`
	City          string  `json:"city" binding:"required,max=50"`
	District      string  `json:"district" binding:"required,max=50"`
	Detail        string  `json:"detail" binding:"required,max=255"`
	PostalCode    *string `json:"postal_code"`
	Tag           *string `json:"tag"`
	IsDefault     bool    `json:"is_default"`
}

// UpdateAddressRequest 更新地址请求
type UpdateAddressRequest struct {
	ReceiverName  *string `json:"receiver_name"`
	ReceiverPhone *string `json:"receiver_phone"`
	Province      *string `json:"province"`
	City          *string `json:"city"`
	District      *string `json:"district"`
	Detail        *string `json:"detail"`
	PostalCode    *string `json:"postal_code"`
	Tag           *string `json:"tag"`
	IsDefault     *bool   `json:"is_default"`
}

// Create 创建地址
func (s *AddressService) Create(ctx context.Context, userID int64, req *CreateAddressRequest) (*models.Address, error) {
	// 检查地址数量限制
	count, err := s.addressRepo.CountByUser(ctx, userID)
	if err != nil {
		return nil, err
	}
	if count >= MaxAddressCount {
		return nil, fmt.Errorf("地址数量已达上限（%d个）", MaxAddressCount)
	}

	// 如果设置为默认，先清除原默认地址
	if req.IsDefault {
		if err := s.addressRepo.ClearDefault(ctx, userID); err != nil {
			return nil, err
		}
	}

	// 如果是第一个地址，自动设为默认
	if count == 0 {
		req.IsDefault = true
	}

	address := &models.Address{
		UserID:        userID,
		ReceiverName:  req.ReceiverName,
		ReceiverPhone: req.ReceiverPhone,
		Province:      req.Province,
		City:          req.City,
		District:      req.District,
		Detail:        req.Detail,
		PostalCode:    req.PostalCode,
		Tag:           req.Tag,
		IsDefault:     req.IsDefault,
	}

	if err := s.addressRepo.Create(ctx, address); err != nil {
		return nil, err
	}

	return address, nil
}

// GetByID 根据 ID 获取地址
func (s *AddressService) GetByID(ctx context.Context, id, userID int64) (*models.Address, error) {
	return s.addressRepo.GetByIDAndUser(ctx, id, userID)
}

// Update 更新地址
func (s *AddressService) Update(ctx context.Context, id, userID int64, req *UpdateAddressRequest) (*models.Address, error) {
	address, err := s.addressRepo.GetByIDAndUser(ctx, id, userID)
	if err != nil {
		return nil, err
	}

	// 如果要设置为默认地址
	if req.IsDefault != nil && *req.IsDefault && !address.IsDefault {
		if err := s.addressRepo.ClearDefault(ctx, userID); err != nil {
			return nil, err
		}
	}

	if req.ReceiverName != nil {
		address.ReceiverName = *req.ReceiverName
	}
	if req.ReceiverPhone != nil {
		address.ReceiverPhone = *req.ReceiverPhone
	}
	if req.Province != nil {
		address.Province = *req.Province
	}
	if req.City != nil {
		address.City = *req.City
	}
	if req.District != nil {
		address.District = *req.District
	}
	if req.Detail != nil {
		address.Detail = *req.Detail
	}
	if req.PostalCode != nil {
		address.PostalCode = req.PostalCode
	}
	if req.Tag != nil {
		address.Tag = req.Tag
	}
	if req.IsDefault != nil {
		address.IsDefault = *req.IsDefault
	}

	if err := s.addressRepo.Update(ctx, address); err != nil {
		return nil, err
	}

	return address, nil
}

// Delete 删除地址
func (s *AddressService) Delete(ctx context.Context, id, userID int64) error {
	address, err := s.addressRepo.GetByIDAndUser(ctx, id, userID)
	if err != nil {
		return err
	}

	// 如果删除的是默认地址，需要设置新的默认地址
	wasDefault := address.IsDefault

	if err := s.addressRepo.DeleteByIDAndUser(ctx, id, userID); err != nil {
		return err
	}

	// 如果删除的是默认地址，自动设置第一个地址为默认
	if wasDefault {
		addresses, err := s.addressRepo.ListByUser(ctx, userID)
		if err != nil {
			return nil // 忽略错误，删除已成功
		}
		if len(addresses) > 0 {
			_ = s.addressRepo.SetDefault(ctx, userID, addresses[0].ID)
		}
	}

	return nil
}

// List 获取用户的地址列表
func (s *AddressService) List(ctx context.Context, userID int64) ([]*models.Address, error) {
	return s.addressRepo.ListByUser(ctx, userID)
}

// GetDefault 获取用户的默认地址
func (s *AddressService) GetDefault(ctx context.Context, userID int64) (*models.Address, error) {
	return s.addressRepo.GetDefaultByUser(ctx, userID)
}

// SetDefault 设置默认地址
func (s *AddressService) SetDefault(ctx context.Context, id, userID int64) error {
	return s.addressRepo.SetDefault(ctx, userID, id)
}

// GetFullAddress 获取完整地址字符串
func (s *AddressService) GetFullAddress(address *models.Address) string {
	return address.Province + address.City + address.District + address.Detail
}

// CreateSnapshot 创建地址快照（用于订单）
func (s *AddressService) CreateSnapshot(address *models.Address) map[string]interface{} {
	return map[string]interface{}{
		"id":             address.ID,
		"receiver_name":  address.ReceiverName,
		"receiver_phone": address.ReceiverPhone,
		"province":       address.Province,
		"city":           address.City,
		"district":       address.District,
		"detail":         address.Detail,
		"postal_code":    address.PostalCode,
		"full_address":   s.GetFullAddress(address),
	}
}
