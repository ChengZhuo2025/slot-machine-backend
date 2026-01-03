// Package repository 提供数据访问层
package repository

import (
	"context"
	"time"

	"gorm.io/gorm"

	"github.com/dumeirei/smart-locker-backend/internal/models"
)

// AdminRepository 管理员仓储
type AdminRepository struct {
	db *gorm.DB
}

// NewAdminRepository 创建管理员仓储
func NewAdminRepository(db *gorm.DB) *AdminRepository {
	return &AdminRepository{db: db}
}

// Create 创建管理员
func (r *AdminRepository) Create(ctx context.Context, admin *models.Admin) error {
	return r.db.WithContext(ctx).Create(admin).Error
}

// GetByID 根据 ID 获取管理员
func (r *AdminRepository) GetByID(ctx context.Context, id int64) (*models.Admin, error) {
	var admin models.Admin
	err := r.db.WithContext(ctx).First(&admin, id).Error
	if err != nil {
		return nil, err
	}
	return &admin, nil
}

// GetByIDWithRole 根据 ID 获取管理员（包含角色）
func (r *AdminRepository) GetByIDWithRole(ctx context.Context, id int64) (*models.Admin, error) {
	var admin models.Admin
	err := r.db.WithContext(ctx).Preload("Role").First(&admin, id).Error
	if err != nil {
		return nil, err
	}
	return &admin, nil
}

// GetByIDWithRoleAndPermissions 根据 ID 获取管理员（包含角色和权限）
func (r *AdminRepository) GetByIDWithRoleAndPermissions(ctx context.Context, id int64) (*models.Admin, error) {
	var admin models.Admin
	err := r.db.WithContext(ctx).Preload("Role.Permissions").First(&admin, id).Error
	if err != nil {
		return nil, err
	}
	return &admin, nil
}

// GetByUsername 根据用户名获取管理员
func (r *AdminRepository) GetByUsername(ctx context.Context, username string) (*models.Admin, error) {
	var admin models.Admin
	err := r.db.WithContext(ctx).Where("username = ?", username).First(&admin).Error
	if err != nil {
		return nil, err
	}
	return &admin, nil
}

// GetByUsernameWithRole 根据用户名获取管理员（包含角色）
func (r *AdminRepository) GetByUsernameWithRole(ctx context.Context, username string) (*models.Admin, error) {
	var admin models.Admin
	err := r.db.WithContext(ctx).Preload("Role").Where("username = ?", username).First(&admin).Error
	if err != nil {
		return nil, err
	}
	return &admin, nil
}

// GetByUsernameWithRoleAndPermissions 根据用户名获取管理员（包含角色和权限）
func (r *AdminRepository) GetByUsernameWithRoleAndPermissions(ctx context.Context, username string) (*models.Admin, error) {
	var admin models.Admin
	err := r.db.WithContext(ctx).Preload("Role.Permissions").Where("username = ?", username).First(&admin).Error
	if err != nil {
		return nil, err
	}
	return &admin, nil
}

// Update 更新管理员
func (r *AdminRepository) Update(ctx context.Context, admin *models.Admin) error {
	return r.db.WithContext(ctx).Save(admin).Error
}

// UpdateFields 更新指定字段
func (r *AdminRepository) UpdateFields(ctx context.Context, id int64, fields map[string]interface{}) error {
	return r.db.WithContext(ctx).Model(&models.Admin{}).Where("id = ?", id).Updates(fields).Error
}

// UpdateStatus 更新管理员状态
func (r *AdminRepository) UpdateStatus(ctx context.Context, id int64, status int8) error {
	return r.db.WithContext(ctx).Model(&models.Admin{}).Where("id = ?", id).Update("status", status).Error
}

// UpdatePassword 更新密码
func (r *AdminRepository) UpdatePassword(ctx context.Context, id int64, passwordHash string) error {
	return r.db.WithContext(ctx).Model(&models.Admin{}).Where("id = ?", id).Update("password_hash", passwordHash).Error
}

// UpdateLoginInfo 更新登录信息
func (r *AdminRepository) UpdateLoginInfo(ctx context.Context, id int64, ip string) error {
	now := time.Now()
	return r.db.WithContext(ctx).Model(&models.Admin{}).Where("id = ?", id).Updates(map[string]interface{}{
		"last_login_at": now,
		"last_login_ip": ip,
	}).Error
}

// Delete 删除管理员（软删除）
func (r *AdminRepository) Delete(ctx context.Context, id int64) error {
	return r.db.WithContext(ctx).Delete(&models.Admin{}, id).Error
}

// List 获取管理员列表
func (r *AdminRepository) List(ctx context.Context, offset, limit int, filters map[string]interface{}) ([]*models.Admin, int64, error) {
	var admins []*models.Admin
	var total int64

	query := r.db.WithContext(ctx).Model(&models.Admin{})

	// 应用过滤条件
	if username, ok := filters["username"].(string); ok && username != "" {
		query = query.Where("username LIKE ?", "%"+username+"%")
	}
	if name, ok := filters["name"].(string); ok && name != "" {
		query = query.Where("name LIKE ?", "%"+name+"%")
	}
	if status, ok := filters["status"].(int8); ok {
		query = query.Where("status = ?", status)
	}
	if roleID, ok := filters["role_id"].(int64); ok && roleID > 0 {
		query = query.Where("role_id = ?", roleID)
	}
	if merchantID, ok := filters["merchant_id"].(int64); ok && merchantID > 0 {
		query = query.Where("merchant_id = ?", merchantID)
	}

	// 统计总数
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	// 查询列表（预加载角色）
	if err := query.Preload("Role").Order("id DESC").Offset(offset).Limit(limit).Find(&admins).Error; err != nil {
		return nil, 0, err
	}

	return admins, total, nil
}

// ExistsByUsername 检查用户名是否存在
func (r *AdminRepository) ExistsByUsername(ctx context.Context, username string) (bool, error) {
	var count int64
	err := r.db.WithContext(ctx).Model(&models.Admin{}).Where("username = ?", username).Count(&count).Error
	return count > 0, err
}

// ExistsByUsernameExcludeID 检查用户名是否存在（排除指定 ID）
func (r *AdminRepository) ExistsByUsernameExcludeID(ctx context.Context, username string, excludeID int64) (bool, error) {
	var count int64
	err := r.db.WithContext(ctx).Model(&models.Admin{}).Where("username = ? AND id != ?", username, excludeID).Count(&count).Error
	return count > 0, err
}

// GetByMerchantID 根据商户 ID 获取管理员列表
func (r *AdminRepository) GetByMerchantID(ctx context.Context, merchantID int64) ([]*models.Admin, error) {
	var admins []*models.Admin
	err := r.db.WithContext(ctx).Preload("Role").Where("merchant_id = ?", merchantID).Find(&admins).Error
	if err != nil {
		return nil, err
	}
	return admins, nil
}

// CountByRoleID 统计指定角色的管理员数量
func (r *AdminRepository) CountByRoleID(ctx context.Context, roleID int64) (int64, error) {
	var count int64
	err := r.db.WithContext(ctx).Model(&models.Admin{}).Where("role_id = ?", roleID).Count(&count).Error
	return count, err
}
