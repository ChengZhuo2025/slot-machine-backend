// Package admin 提供管理员相关服务
package admin

import (
	"context"
	"errors"

	"gorm.io/gorm"

	"github.com/dumeirei/smart-locker-backend/internal/common/crypto"
	"github.com/dumeirei/smart-locker-backend/internal/common/jwt"
	"github.com/dumeirei/smart-locker-backend/internal/models"
	"github.com/dumeirei/smart-locker-backend/internal/repository"
)

// AdminAuthService 管理员认证服务
type AdminAuthService struct {
	adminRepo  *repository.AdminRepository
	jwtManager *jwt.Manager
}

// NewAdminAuthService 创建管理员认证服务
func NewAdminAuthService(adminRepo *repository.AdminRepository, jwtManager *jwt.Manager) *AdminAuthService {
	return &AdminAuthService{
		adminRepo:  adminRepo,
		jwtManager: jwtManager,
	}
}

// 预定义错误
var (
	ErrAdminNotFound      = errors.New("管理员不存在")
	ErrAdminDisabled      = errors.New("管理员已禁用")
	ErrInvalidPassword    = errors.New("密码错误")
	ErrUsernameExists     = errors.New("用户名已存在")
	ErrOldPasswordInvalid = errors.New("原密码错误")
)

// LoginRequest 登录请求
type LoginRequest struct {
	Username string `json:"username" binding:"required"`
	Password string `json:"password" binding:"required"`
	IP       string `json:"-"`
}

// LoginResponse 登录响应
type LoginResponse struct {
	Admin       *AdminInfo      `json:"admin"`
	TokenPair   *jwt.TokenPair  `json:"token"`
	Permissions []string        `json:"permissions"`
}

// AdminInfo 管理员信息（不含敏感字段）
type AdminInfo struct {
	ID         int64   `json:"id"`
	Username   string  `json:"username"`
	Name       string  `json:"name"`
	Phone      *string `json:"phone,omitempty"`
	Email      *string `json:"email,omitempty"`
	RoleID     int64   `json:"role_id"`
	RoleName   string  `json:"role_name"`
	RoleCode   string  `json:"role_code"`
	MerchantID *int64  `json:"merchant_id,omitempty"`
}

// Login 管理员登录
func (s *AdminAuthService) Login(ctx context.Context, req *LoginRequest) (*LoginResponse, error) {
	// 查询管理员（包含角色和权限）
	admin, err := s.adminRepo.GetByUsernameWithRoleAndPermissions(ctx, req.Username)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrAdminNotFound
		}
		return nil, err
	}

	// 检查状态
	if admin.Status != models.AdminStatusActive {
		return nil, ErrAdminDisabled
	}

	// 验证密码
	if !crypto.VerifyPassword(req.Password, admin.PasswordHash) {
		return nil, ErrInvalidPassword
	}

	// 生成 JWT
	roleCode := ""
	if admin.Role != nil {
		roleCode = admin.Role.Code
	}
	tokenPair, err := s.jwtManager.GenerateTokenPair(admin.ID, jwt.UserTypeAdmin, roleCode)
	if err != nil {
		return nil, err
	}

	// 更新登录信息
	if err := s.adminRepo.UpdateLoginInfo(ctx, admin.ID, req.IP); err != nil {
		// 记录日志但不阻塞登录
	}

	// 提取权限列表
	permissions := s.extractPermissions(admin)

	// 构建响应
	response := &LoginResponse{
		Admin:       s.toAdminInfo(admin),
		TokenPair:   tokenPair,
		Permissions: permissions,
	}

	return response, nil
}

// GetAdminInfo 获取管理员信息
func (s *AdminAuthService) GetAdminInfo(ctx context.Context, adminID int64) (*AdminInfo, error) {
	admin, err := s.adminRepo.GetByIDWithRole(ctx, adminID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrAdminNotFound
		}
		return nil, err
	}

	return s.toAdminInfo(admin), nil
}

// GetAdminWithPermissions 获取管理员信息（包含权限）
func (s *AdminAuthService) GetAdminWithPermissions(ctx context.Context, adminID int64) (*LoginResponse, error) {
	admin, err := s.adminRepo.GetByIDWithRoleAndPermissions(ctx, adminID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrAdminNotFound
		}
		return nil, err
	}

	permissions := s.extractPermissions(admin)

	return &LoginResponse{
		Admin:       s.toAdminInfo(admin),
		Permissions: permissions,
	}, nil
}

// ChangePasswordRequest 修改密码请求
type ChangePasswordRequest struct {
	OldPassword string `json:"old_password" binding:"required"`
	NewPassword string `json:"new_password" binding:"required,min=6,max=32"`
}

// ChangePassword 修改密码
func (s *AdminAuthService) ChangePassword(ctx context.Context, adminID int64, req *ChangePasswordRequest) error {
	// 获取管理员
	admin, err := s.adminRepo.GetByID(ctx, adminID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return ErrAdminNotFound
		}
		return err
	}

	// 验证原密码
	if !crypto.VerifyPassword(req.OldPassword, admin.PasswordHash) {
		return ErrOldPasswordInvalid
	}

	// 生成新密码哈希
	passwordHash, err := crypto.HashPassword(req.NewPassword)
	if err != nil {
		return err
	}

	// 更新密码
	return s.adminRepo.UpdatePassword(ctx, adminID, passwordHash)
}

// RefreshToken 刷新令牌
func (s *AdminAuthService) RefreshToken(ctx context.Context, refreshToken string) (*jwt.TokenPair, error) {
	return s.jwtManager.RefreshToken(refreshToken)
}

// toAdminInfo 转换为管理员信息
func (s *AdminAuthService) toAdminInfo(admin *models.Admin) *AdminInfo {
	info := &AdminInfo{
		ID:         admin.ID,
		Username:   admin.Username,
		Name:       admin.Name,
		Phone:      admin.Phone,
		Email:      admin.Email,
		RoleID:     admin.RoleID,
		MerchantID: admin.MerchantID,
	}

	if admin.Role != nil {
		info.RoleName = admin.Role.Name
		info.RoleCode = admin.Role.Code
	}

	return info
}

// extractPermissions 提取权限列表
func (s *AdminAuthService) extractPermissions(admin *models.Admin) []string {
	var permissions []string
	if admin.Role != nil && len(admin.Role.Permissions) > 0 {
		for _, p := range admin.Role.Permissions {
			permissions = append(permissions, p.Code)
		}
	}
	return permissions
}

// ValidateAdminToken 验证管理员令牌
func (s *AdminAuthService) ValidateAdminToken(ctx context.Context, token string) (*jwt.Claims, error) {
	claims, err := s.jwtManager.ParseToken(token)
	if err != nil {
		return nil, err
	}

	// 验证用户类型
	if claims.UserType != jwt.UserTypeAdmin {
		return nil, errors.New("invalid user type")
	}

	// 验证管理员是否存在且状态正常
	admin, err := s.adminRepo.GetByID(ctx, claims.UserID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrAdminNotFound
		}
		return nil, err
	}

	if admin.Status != models.AdminStatusActive {
		return nil, ErrAdminDisabled
	}

	return claims, nil
}
