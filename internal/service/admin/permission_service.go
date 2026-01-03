// Package admin 提供管理员相关服务
package admin

import (
	"context"
	"errors"

	"gorm.io/gorm"

	"github.com/dumeirei/smart-locker-backend/internal/models"
	"github.com/dumeirei/smart-locker-backend/internal/repository"
)

// PermissionService 权限服务
type PermissionService struct {
	roleRepo       *repository.RoleRepository
	permissionRepo *repository.PermissionRepository
	adminRepo      *repository.AdminRepository
}

// NewPermissionService 创建权限服务
func NewPermissionService(
	roleRepo *repository.RoleRepository,
	permissionRepo *repository.PermissionRepository,
	adminRepo *repository.AdminRepository,
) *PermissionService {
	return &PermissionService{
		roleRepo:       roleRepo,
		permissionRepo: permissionRepo,
		adminRepo:      adminRepo,
	}
}

// 预定义错误
var (
	ErrRoleNotFound         = errors.New("角色不存在")
	ErrRoleCodeExists       = errors.New("角色编码已存在")
	ErrRoleIsSystem         = errors.New("系统角色不能删除或修改")
	ErrRoleHasAdmins        = errors.New("角色下有管理员，无法删除")
	ErrPermissionNotFound   = errors.New("权限不存在")
	ErrPermissionCodeExists = errors.New("权限编码已存在")
	ErrPermissionHasChildren = errors.New("权限下有子权限，无法删除")
)

// RoleInfo 角色信息
type RoleInfo struct {
	ID          int64    `json:"id"`
	Code        string   `json:"code"`
	Name        string   `json:"name"`
	Description *string  `json:"description,omitempty"`
	IsSystem    bool     `json:"is_system"`
	Permissions []string `json:"permissions,omitempty"`
}

// CreateRoleRequest 创建角色请求
type CreateRoleRequest struct {
	Code          string  `json:"code" binding:"required,max=50"`
	Name          string  `json:"name" binding:"required,max=50"`
	Description   *string `json:"description"`
	PermissionIDs []int64 `json:"permission_ids"`
}

// CreateRole 创建角色
func (s *PermissionService) CreateRole(ctx context.Context, req *CreateRoleRequest) (*models.Role, error) {
	// 检查编码是否存在
	exists, err := s.roleRepo.ExistsByCode(ctx, req.Code)
	if err != nil {
		return nil, err
	}
	if exists {
		return nil, ErrRoleCodeExists
	}

	role := &models.Role{
		Code:        req.Code,
		Name:        req.Name,
		Description: req.Description,
		IsSystem:    false,
	}

	if err := s.roleRepo.Create(ctx, role); err != nil {
		return nil, err
	}

	// 设置权限
	if len(req.PermissionIDs) > 0 {
		if err := s.roleRepo.SetPermissions(ctx, role.ID, req.PermissionIDs); err != nil {
			return nil, err
		}
	}

	return role, nil
}

// UpdateRoleRequest 更新角色请求
type UpdateRoleRequest struct {
	Name          string  `json:"name" binding:"required,max=50"`
	Description   *string `json:"description"`
	PermissionIDs []int64 `json:"permission_ids"`
}

// UpdateRole 更新角色
func (s *PermissionService) UpdateRole(ctx context.Context, id int64, req *UpdateRoleRequest) error {
	role, err := s.roleRepo.GetByID(ctx, id)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return ErrRoleNotFound
		}
		return err
	}

	// 系统角色只能修改权限，不能修改其他信息
	if !role.IsSystem {
		role.Name = req.Name
		role.Description = req.Description
		if err := s.roleRepo.Update(ctx, role); err != nil {
			return err
		}
	}

	// 更新权限
	return s.roleRepo.SetPermissions(ctx, id, req.PermissionIDs)
}

// DeleteRole 删除角色
func (s *PermissionService) DeleteRole(ctx context.Context, id int64) error {
	role, err := s.roleRepo.GetByID(ctx, id)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return ErrRoleNotFound
		}
		return err
	}

	// 检查是否为系统角色
	if role.IsSystem {
		return ErrRoleIsSystem
	}

	// 检查是否有管理员使用该角色
	count, err := s.adminRepo.CountByRoleID(ctx, id)
	if err != nil {
		return err
	}
	if count > 0 {
		return ErrRoleHasAdmins
	}

	// 删除角色权限关联
	if err := s.roleRepo.SetPermissions(ctx, id, nil); err != nil {
		return err
	}

	return s.roleRepo.Delete(ctx, id)
}

// GetRole 获取角色详情
func (s *PermissionService) GetRole(ctx context.Context, id int64) (*models.Role, error) {
	role, err := s.roleRepo.GetByIDWithPermissions(ctx, id)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrRoleNotFound
		}
		return nil, err
	}
	return role, nil
}

// ListRoles 获取角色列表
func (s *PermissionService) ListRoles(ctx context.Context, offset, limit int, filters map[string]interface{}) ([]*models.Role, int64, error) {
	return s.roleRepo.List(ctx, offset, limit, filters)
}

// ListAllRoles 获取所有角色
func (s *PermissionService) ListAllRoles(ctx context.Context) ([]*models.Role, error) {
	return s.roleRepo.ListAll(ctx)
}

// SetRolePermissions 设置角色权限
func (s *PermissionService) SetRolePermissions(ctx context.Context, roleID int64, permissionIDs []int64) error {
	// 检查角色是否存在
	_, err := s.roleRepo.GetByID(ctx, roleID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return ErrRoleNotFound
		}
		return err
	}

	return s.roleRepo.SetPermissions(ctx, roleID, permissionIDs)
}

// PermissionInfo 权限信息
type PermissionInfo struct {
	ID       int64             `json:"id"`
	Code     string            `json:"code"`
	Name     string            `json:"name"`
	Type     string            `json:"type"`
	ParentID *int64            `json:"parent_id,omitempty"`
	Path     *string           `json:"path,omitempty"`
	Method   *string           `json:"method,omitempty"`
	Sort     int               `json:"sort"`
	Children []*PermissionInfo `json:"children,omitempty"`
}

// CreatePermissionRequest 创建权限请求
type CreatePermissionRequest struct {
	Code     string  `json:"code" binding:"required,max=100"`
	Name     string  `json:"name" binding:"required,max=100"`
	Type     string  `json:"type" binding:"required,oneof=menu api"`
	ParentID *int64  `json:"parent_id"`
	Path     *string `json:"path"`
	Method   *string `json:"method"`
	Sort     int     `json:"sort"`
}

// CreatePermission 创建权限
func (s *PermissionService) CreatePermission(ctx context.Context, req *CreatePermissionRequest) (*models.Permission, error) {
	// 检查编码是否存在
	exists, err := s.permissionRepo.ExistsByCode(ctx, req.Code)
	if err != nil {
		return nil, err
	}
	if exists {
		return nil, ErrPermissionCodeExists
	}

	// 检查父权限是否存在
	if req.ParentID != nil {
		_, err := s.permissionRepo.GetByID(ctx, *req.ParentID)
		if err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return nil, ErrPermissionNotFound
			}
			return nil, err
		}
	}

	permission := &models.Permission{
		Code:     req.Code,
		Name:     req.Name,
		Type:     req.Type,
		ParentID: req.ParentID,
		Path:     req.Path,
		Method:   req.Method,
		Sort:     req.Sort,
	}

	if err := s.permissionRepo.Create(ctx, permission); err != nil {
		return nil, err
	}

	return permission, nil
}

// UpdatePermissionRequest 更新权限请求
type UpdatePermissionRequest struct {
	Name   string  `json:"name" binding:"required,max=100"`
	Path   *string `json:"path"`
	Method *string `json:"method"`
	Sort   int     `json:"sort"`
}

// UpdatePermission 更新权限
func (s *PermissionService) UpdatePermission(ctx context.Context, id int64, req *UpdatePermissionRequest) error {
	permission, err := s.permissionRepo.GetByID(ctx, id)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return ErrPermissionNotFound
		}
		return err
	}

	permission.Name = req.Name
	permission.Path = req.Path
	permission.Method = req.Method
	permission.Sort = req.Sort

	return s.permissionRepo.Update(ctx, permission)
}

// DeletePermission 删除权限
func (s *PermissionService) DeletePermission(ctx context.Context, id int64) error {
	// 检查是否存在
	_, err := s.permissionRepo.GetByID(ctx, id)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return ErrPermissionNotFound
		}
		return err
	}

	// 检查是否有子权限
	hasChildren, err := s.permissionRepo.HasChildren(ctx, id)
	if err != nil {
		return err
	}
	if hasChildren {
		return ErrPermissionHasChildren
	}

	return s.permissionRepo.Delete(ctx, id)
}

// GetPermission 获取权限详情
func (s *PermissionService) GetPermission(ctx context.Context, id int64) (*models.Permission, error) {
	permission, err := s.permissionRepo.GetByID(ctx, id)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrPermissionNotFound
		}
		return nil, err
	}
	return permission, nil
}

// ListPermissions 获取权限列表
func (s *PermissionService) ListPermissions(ctx context.Context, filters map[string]interface{}) ([]*models.Permission, error) {
	return s.permissionRepo.List(ctx, filters)
}

// ListPermissionTree 获取权限树
func (s *PermissionService) ListPermissionTree(ctx context.Context) ([]*models.Permission, error) {
	return s.permissionRepo.ListTree(ctx)
}

// CheckPermission 检查管理员是否有指定权限
func (s *PermissionService) CheckPermission(ctx context.Context, adminID int64, permissionCode string) (bool, error) {
	admin, err := s.adminRepo.GetByIDWithRoleAndPermissions(ctx, adminID)
	if err != nil {
		return false, err
	}

	// 超级管理员拥有所有权限
	if admin.Role != nil && admin.Role.Code == models.RoleCodeSuperAdmin {
		return true, nil
	}

	// 检查权限列表
	if admin.Role != nil {
		for _, p := range admin.Role.Permissions {
			if p.Code == permissionCode {
				return true, nil
			}
		}
	}

	return false, nil
}

// CheckAPIPermission 检查管理员是否有指定 API 权限
func (s *PermissionService) CheckAPIPermission(ctx context.Context, adminID int64, path, method string) (bool, error) {
	admin, err := s.adminRepo.GetByIDWithRoleAndPermissions(ctx, adminID)
	if err != nil {
		return false, err
	}

	// 超级管理员拥有所有权限
	if admin.Role != nil && admin.Role.Code == models.RoleCodeSuperAdmin {
		return true, nil
	}

	// 检查 API 权限
	if admin.Role != nil {
		for _, p := range admin.Role.Permissions {
			if p.Type == models.PermissionTypeAPI && p.Path != nil && p.Method != nil {
				if *p.Path == path && *p.Method == method {
					return true, nil
				}
			}
		}
	}

	return false, nil
}

// GetAdminPermissions 获取管理员权限列表
func (s *PermissionService) GetAdminPermissions(ctx context.Context, adminID int64) ([]string, error) {
	admin, err := s.adminRepo.GetByIDWithRoleAndPermissions(ctx, adminID)
	if err != nil {
		return nil, err
	}

	var permissions []string
	if admin.Role != nil {
		for _, p := range admin.Role.Permissions {
			permissions = append(permissions, p.Code)
		}
	}

	return permissions, nil
}

// GetAdminMenus 获取管理员菜单权限
func (s *PermissionService) GetAdminMenus(ctx context.Context, adminID int64) ([]*models.Permission, error) {
	admin, err := s.adminRepo.GetByIDWithRoleAndPermissions(ctx, adminID)
	if err != nil {
		return nil, err
	}

	// 超级管理员拥有所有菜单
	if admin.Role != nil && admin.Role.Code == models.RoleCodeSuperAdmin {
		return s.permissionRepo.ListMenuPermissions(ctx)
	}

	var menus []*models.Permission
	if admin.Role != nil {
		for i := range admin.Role.Permissions {
			if admin.Role.Permissions[i].Type == models.PermissionTypeMenu {
				menus = append(menus, &admin.Role.Permissions[i])
			}
		}
	}

	return menus, nil
}
