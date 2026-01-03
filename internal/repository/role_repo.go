// Package repository 提供数据访问层
package repository

import (
	"context"

	"gorm.io/gorm"

	"github.com/dumeirei/smart-locker-backend/internal/models"
)

// RoleRepository 角色仓储
type RoleRepository struct {
	db *gorm.DB
}

// NewRoleRepository 创建角色仓储
func NewRoleRepository(db *gorm.DB) *RoleRepository {
	return &RoleRepository{db: db}
}

// Create 创建角色
func (r *RoleRepository) Create(ctx context.Context, role *models.Role) error {
	return r.db.WithContext(ctx).Create(role).Error
}

// GetByID 根据 ID 获取角色
func (r *RoleRepository) GetByID(ctx context.Context, id int64) (*models.Role, error) {
	var role models.Role
	err := r.db.WithContext(ctx).First(&role, id).Error
	if err != nil {
		return nil, err
	}
	return &role, nil
}

// GetByIDWithPermissions 根据 ID 获取角色（包含权限）
func (r *RoleRepository) GetByIDWithPermissions(ctx context.Context, id int64) (*models.Role, error) {
	var role models.Role
	err := r.db.WithContext(ctx).Preload("Permissions").First(&role, id).Error
	if err != nil {
		return nil, err
	}
	return &role, nil
}

// GetByCode 根据编码获取角色
func (r *RoleRepository) GetByCode(ctx context.Context, code string) (*models.Role, error) {
	var role models.Role
	err := r.db.WithContext(ctx).Where("code = ?", code).First(&role).Error
	if err != nil {
		return nil, err
	}
	return &role, nil
}

// GetByCodeWithPermissions 根据编码获取角色（包含权限）
func (r *RoleRepository) GetByCodeWithPermissions(ctx context.Context, code string) (*models.Role, error) {
	var role models.Role
	err := r.db.WithContext(ctx).Preload("Permissions").Where("code = ?", code).First(&role).Error
	if err != nil {
		return nil, err
	}
	return &role, nil
}

// Update 更新角色
func (r *RoleRepository) Update(ctx context.Context, role *models.Role) error {
	return r.db.WithContext(ctx).Save(role).Error
}

// UpdateFields 更新指定字段
func (r *RoleRepository) UpdateFields(ctx context.Context, id int64, fields map[string]interface{}) error {
	return r.db.WithContext(ctx).Model(&models.Role{}).Where("id = ?", id).Updates(fields).Error
}

// Delete 删除角色
func (r *RoleRepository) Delete(ctx context.Context, id int64) error {
	return r.db.WithContext(ctx).Delete(&models.Role{}, id).Error
}

// List 获取角色列表
func (r *RoleRepository) List(ctx context.Context, offset, limit int, filters map[string]interface{}) ([]*models.Role, int64, error) {
	var roles []*models.Role
	var total int64

	query := r.db.WithContext(ctx).Model(&models.Role{})

	// 应用过滤条件
	if name, ok := filters["name"].(string); ok && name != "" {
		query = query.Where("name LIKE ?", "%"+name+"%")
	}
	if isSystem, ok := filters["is_system"].(bool); ok {
		query = query.Where("is_system = ?", isSystem)
	}

	// 统计总数
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	// 查询列表
	if err := query.Order("id ASC").Offset(offset).Limit(limit).Find(&roles).Error; err != nil {
		return nil, 0, err
	}

	return roles, total, nil
}

// ListAll 获取所有角色
func (r *RoleRepository) ListAll(ctx context.Context) ([]*models.Role, error) {
	var roles []*models.Role
	err := r.db.WithContext(ctx).Order("id ASC").Find(&roles).Error
	return roles, err
}

// ListWithPermissions 获取所有角色（包含权限）
func (r *RoleRepository) ListWithPermissions(ctx context.Context) ([]*models.Role, error) {
	var roles []*models.Role
	err := r.db.WithContext(ctx).Preload("Permissions").Order("id ASC").Find(&roles).Error
	return roles, err
}

// ExistsByCode 检查编码是否存在
func (r *RoleRepository) ExistsByCode(ctx context.Context, code string) (bool, error) {
	var count int64
	err := r.db.WithContext(ctx).Model(&models.Role{}).Where("code = ?", code).Count(&count).Error
	return count > 0, err
}

// ExistsByCodeExcludeID 检查编码是否存在（排除指定 ID）
func (r *RoleRepository) ExistsByCodeExcludeID(ctx context.Context, code string, excludeID int64) (bool, error) {
	var count int64
	err := r.db.WithContext(ctx).Model(&models.Role{}).Where("code = ? AND id != ?", code, excludeID).Count(&count).Error
	return count > 0, err
}

// SetPermissions 设置角色权限
func (r *RoleRepository) SetPermissions(ctx context.Context, roleID int64, permissionIDs []int64) error {
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		// 删除原有权限
		if err := tx.Where("role_id = ?", roleID).Delete(&models.RolePermission{}).Error; err != nil {
			return err
		}

		// 添加新权限
		if len(permissionIDs) > 0 {
			rolePermissions := make([]models.RolePermission, 0, len(permissionIDs))
			for _, permissionID := range permissionIDs {
				rolePermissions = append(rolePermissions, models.RolePermission{
					RoleID:       roleID,
					PermissionID: permissionID,
				})
			}
			if err := tx.Create(&rolePermissions).Error; err != nil {
				return err
			}
		}

		return nil
	})
}

// GetPermissionIDs 获取角色的权限 ID 列表
func (r *RoleRepository) GetPermissionIDs(ctx context.Context, roleID int64) ([]int64, error) {
	var ids []int64
	err := r.db.WithContext(ctx).Model(&models.RolePermission{}).
		Where("role_id = ?", roleID).
		Pluck("permission_id", &ids).Error
	return ids, err
}

// PermissionRepository 权限仓储
type PermissionRepository struct {
	db *gorm.DB
}

// NewPermissionRepository 创建权限仓储
func NewPermissionRepository(db *gorm.DB) *PermissionRepository {
	return &PermissionRepository{db: db}
}

// Create 创建权限
func (r *PermissionRepository) Create(ctx context.Context, permission *models.Permission) error {
	return r.db.WithContext(ctx).Create(permission).Error
}

// GetByID 根据 ID 获取权限
func (r *PermissionRepository) GetByID(ctx context.Context, id int64) (*models.Permission, error) {
	var permission models.Permission
	err := r.db.WithContext(ctx).First(&permission, id).Error
	if err != nil {
		return nil, err
	}
	return &permission, nil
}

// GetByCode 根据编码获取权限
func (r *PermissionRepository) GetByCode(ctx context.Context, code string) (*models.Permission, error) {
	var permission models.Permission
	err := r.db.WithContext(ctx).Where("code = ?", code).First(&permission).Error
	if err != nil {
		return nil, err
	}
	return &permission, nil
}

// Update 更新权限
func (r *PermissionRepository) Update(ctx context.Context, permission *models.Permission) error {
	return r.db.WithContext(ctx).Save(permission).Error
}

// Delete 删除权限
func (r *PermissionRepository) Delete(ctx context.Context, id int64) error {
	return r.db.WithContext(ctx).Delete(&models.Permission{}, id).Error
}

// List 获取权限列表
func (r *PermissionRepository) List(ctx context.Context, filters map[string]interface{}) ([]*models.Permission, error) {
	var permissions []*models.Permission
	query := r.db.WithContext(ctx).Model(&models.Permission{})

	// 应用过滤条件
	if permType, ok := filters["type"].(string); ok && permType != "" {
		query = query.Where("type = ?", permType)
	}
	if parentID, ok := filters["parent_id"].(int64); ok {
		query = query.Where("parent_id = ?", parentID)
	}
	if parentID, ok := filters["parent_id"]; ok && parentID == nil {
		query = query.Where("parent_id IS NULL")
	}

	if err := query.Order("sort ASC, id ASC").Find(&permissions).Error; err != nil {
		return nil, err
	}

	return permissions, nil
}

// ListAll 获取所有权限
func (r *PermissionRepository) ListAll(ctx context.Context) ([]*models.Permission, error) {
	var permissions []*models.Permission
	err := r.db.WithContext(ctx).Order("sort ASC, id ASC").Find(&permissions).Error
	return permissions, err
}

// ListTree 获取权限树
func (r *PermissionRepository) ListTree(ctx context.Context) ([]*models.Permission, error) {
	var permissions []*models.Permission
	err := r.db.WithContext(ctx).Where("parent_id IS NULL").
		Preload("Children", func(db *gorm.DB) *gorm.DB {
			return db.Order("sort ASC, id ASC")
		}).
		Order("sort ASC, id ASC").Find(&permissions).Error
	return permissions, err
}

// ListByType 根据类型获取权限
func (r *PermissionRepository) ListByType(ctx context.Context, permType string) ([]*models.Permission, error) {
	var permissions []*models.Permission
	err := r.db.WithContext(ctx).Where("type = ?", permType).Order("sort ASC, id ASC").Find(&permissions).Error
	return permissions, err
}

// ExistsByCode 检查编码是否存在
func (r *PermissionRepository) ExistsByCode(ctx context.Context, code string) (bool, error) {
	var count int64
	err := r.db.WithContext(ctx).Model(&models.Permission{}).Where("code = ?", code).Count(&count).Error
	return count > 0, err
}

// GetByIDs 批量获取权限
func (r *PermissionRepository) GetByIDs(ctx context.Context, ids []int64) ([]*models.Permission, error) {
	var permissions []*models.Permission
	if len(ids) == 0 {
		return permissions, nil
	}
	err := r.db.WithContext(ctx).Where("id IN ?", ids).Find(&permissions).Error
	return permissions, err
}

// ListAPIPermissions 获取 API 权限列表
func (r *PermissionRepository) ListAPIPermissions(ctx context.Context) ([]*models.Permission, error) {
	return r.ListByType(ctx, models.PermissionTypeAPI)
}

// ListMenuPermissions 获取菜单权限列表
func (r *PermissionRepository) ListMenuPermissions(ctx context.Context) ([]*models.Permission, error) {
	return r.ListByType(ctx, models.PermissionTypeMenu)
}

// HasChildren 检查是否有子权限
func (r *PermissionRepository) HasChildren(ctx context.Context, id int64) (bool, error) {
	var count int64
	err := r.db.WithContext(ctx).Model(&models.Permission{}).Where("parent_id = ?", id).Count(&count).Error
	return count > 0, err
}
