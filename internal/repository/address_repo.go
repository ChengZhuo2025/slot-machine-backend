// Package repository 提供数据访问层
package repository

import (
	"context"

	"gorm.io/gorm"

	"github.com/dumeirei/smart-locker-backend/internal/models"
)

// AddressRepository 地址仓储
type AddressRepository struct {
	db *gorm.DB
}

// NewAddressRepository 创建地址仓储
func NewAddressRepository(db *gorm.DB) *AddressRepository {
	return &AddressRepository{db: db}
}

// Create 创建地址
func (r *AddressRepository) Create(ctx context.Context, address *models.Address) error {
	return r.db.WithContext(ctx).Create(address).Error
}

// GetByID 根据 ID 获取地址
func (r *AddressRepository) GetByID(ctx context.Context, id int64) (*models.Address, error) {
	var address models.Address
	err := r.db.WithContext(ctx).First(&address, id).Error
	if err != nil {
		return nil, err
	}
	return &address, nil
}

// GetByIDAndUser 根据 ID 和用户 ID 获取地址
func (r *AddressRepository) GetByIDAndUser(ctx context.Context, id, userID int64) (*models.Address, error) {
	var address models.Address
	err := r.db.WithContext(ctx).
		Where("id = ? AND user_id = ?", id, userID).
		First(&address).Error
	if err != nil {
		return nil, err
	}
	return &address, nil
}

// Update 更新地址
func (r *AddressRepository) Update(ctx context.Context, address *models.Address) error {
	return r.db.WithContext(ctx).Save(address).Error
}

// Delete 删除地址
func (r *AddressRepository) Delete(ctx context.Context, id int64) error {
	return r.db.WithContext(ctx).Delete(&models.Address{}, id).Error
}

// DeleteByIDAndUser 根据 ID 和用户 ID 删除地址
func (r *AddressRepository) DeleteByIDAndUser(ctx context.Context, id, userID int64) error {
	result := r.db.WithContext(ctx).
		Where("id = ? AND user_id = ?", id, userID).
		Delete(&models.Address{})
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return gorm.ErrRecordNotFound
	}
	return nil
}

// ListByUser 获取用户的地址列表
func (r *AddressRepository) ListByUser(ctx context.Context, userID int64) ([]*models.Address, error) {
	var addresses []*models.Address
	err := r.db.WithContext(ctx).
		Where("user_id = ?", userID).
		Order("is_default DESC, id DESC").
		Find(&addresses).Error
	return addresses, err
}

// GetDefaultByUser 获取用户的默认地址
func (r *AddressRepository) GetDefaultByUser(ctx context.Context, userID int64) (*models.Address, error) {
	var address models.Address
	err := r.db.WithContext(ctx).
		Where("user_id = ? AND is_default = ?", userID, true).
		First(&address).Error
	if err != nil {
		return nil, err
	}
	return &address, nil
}

// SetDefault 设置默认地址
func (r *AddressRepository) SetDefault(ctx context.Context, userID, addressID int64) error {
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		// 取消原默认地址
		if err := tx.Model(&models.Address{}).
			Where("user_id = ? AND is_default = ?", userID, true).
			Update("is_default", false).Error; err != nil {
			return err
		}

		// 设置新默认地址
		result := tx.Model(&models.Address{}).
			Where("id = ? AND user_id = ?", addressID, userID).
			Update("is_default", true)
		if result.Error != nil {
			return result.Error
		}
		if result.RowsAffected == 0 {
			return gorm.ErrRecordNotFound
		}

		return nil
	})
}

// ClearDefault 清除用户的默认地址
func (r *AddressRepository) ClearDefault(ctx context.Context, userID int64) error {
	return r.db.WithContext(ctx).Model(&models.Address{}).
		Where("user_id = ? AND is_default = ?", userID, true).
		Update("is_default", false).Error
}

// CountByUser 统计用户地址数量
func (r *AddressRepository) CountByUser(ctx context.Context, userID int64) (int64, error) {
	var count int64
	err := r.db.WithContext(ctx).Model(&models.Address{}).
		Where("user_id = ?", userID).
		Count(&count).Error
	return count, err
}
