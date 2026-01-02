// Package auth 提供认证服务
package auth

import (
	"context"
	"fmt"
	"time"

	"gorm.io/gorm"

	"github.com/dumeirei/smart-locker-backend/internal/common/errors"
	"github.com/dumeirei/smart-locker-backend/internal/common/jwt"
	"github.com/dumeirei/smart-locker-backend/internal/models"
	"github.com/dumeirei/smart-locker-backend/internal/repository"
)

// AuthService 认证服务
type AuthService struct {
	db          *gorm.DB
	userRepo    *repository.UserRepository
	jwtManager  *jwt.Manager
	codeService *CodeService
}

// NewAuthService 创建认证服务
func NewAuthService(
	db *gorm.DB,
	userRepo *repository.UserRepository,
	jwtManager *jwt.Manager,
	codeService *CodeService,
) *AuthService {
	return &AuthService{
		db:          db,
		userRepo:    userRepo,
		jwtManager:  jwtManager,
		codeService: codeService,
	}
}

// SendSmsCodeRequest 发送短信验证码请求
type SendSmsCodeRequest struct {
	Phone    string   `json:"phone" binding:"required"`
	CodeType CodeType `json:"code_type" binding:"required"`
}

// SmsLoginRequest 短信验证码登录请求
type SmsLoginRequest struct {
	Phone    string  `json:"phone" binding:"required"`
	Code     string  `json:"code" binding:"required"`
	InviteCode *string `json:"invite_code,omitempty"`
}

// LoginResponse 登录响应
type LoginResponse struct {
	User        *UserInfo       `json:"user"`
	TokenPair   *jwt.TokenPair  `json:"token"`
	IsNewUser   bool            `json:"is_new_user"`
}

// UserInfo 用户信息
type UserInfo struct {
	ID            int64   `json:"id"`
	Phone         *string `json:"phone,omitempty"`
	Nickname      string  `json:"nickname"`
	Avatar        *string `json:"avatar,omitempty"`
	Gender        int8    `json:"gender"`
	MemberLevelID int64   `json:"member_level_id"`
	Points        int     `json:"points"`
	IsVerified    bool    `json:"is_verified"`
}

// SendSmsCode 发送短信验证码
func (s *AuthService) SendSmsCode(ctx context.Context, req *SendSmsCodeRequest) error {
	// 验证手机号格式
	if len(req.Phone) != 11 {
		return errors.ErrPhoneInvalid
	}

	// 发送验证码
	if err := s.codeService.SendCode(ctx, req.Phone, req.CodeType); err != nil {
		return errors.Wrap(errors.ErrSmsSendFail.Code, err.Error(), err)
	}

	return nil
}

// SmsLogin 短信验证码登录（自动注册）
func (s *AuthService) SmsLogin(ctx context.Context, req *SmsLoginRequest) (*LoginResponse, error) {
	// 验证验证码
	valid, err := s.codeService.VerifyCode(ctx, req.Phone, req.Code, CodeTypeLogin)
	if err != nil {
		return nil, errors.ErrInternalError.WithError(err)
	}
	if !valid {
		return nil, errors.ErrSmsCodeError
	}

	// 查找或创建用户
	user, isNew, err := s.findOrCreateUser(ctx, req.Phone, req.InviteCode)
	if err != nil {
		return nil, err
	}

	// 检查用户状态
	if user.Status == models.UserStatusDisabled {
		return nil, errors.ErrAccountDisabled
	}

	// 生成 Token
	tokenPair, err := s.jwtManager.GenerateTokenPair(user.ID, jwt.UserTypeUser, "")
	if err != nil {
		return nil, errors.ErrInternalError.WithError(err)
	}

	return &LoginResponse{
		User:      s.toUserInfo(user),
		TokenPair: tokenPair,
		IsNewUser: isNew,
	}, nil
}

// findOrCreateUser 查找或创建用户
func (s *AuthService) findOrCreateUser(ctx context.Context, phone string, inviteCode *string) (*models.User, bool, error) {
	// 先查找用户
	user, err := s.userRepo.GetByPhone(ctx, phone)
	if err == nil {
		return user, false, nil
	}
	if err != gorm.ErrRecordNotFound {
		return nil, false, errors.ErrDatabaseError.WithError(err)
	}

	// 用户不存在，创建新用户
	var referrerID *int64
	if inviteCode != nil && *inviteCode != "" {
		// 查找邀请人
		referrer, err := s.userRepo.GetByInviteCode(ctx, *inviteCode)
		if err == nil {
			referrerID = &referrer.ID
		}
	}

	user = &models.User{
		Phone:         &phone,
		Nickname:      s.generateNickname(phone),
		MemberLevelID: 1, // 默认会员等级
		Status:        models.UserStatusActive,
		ReferrerID:    referrerID,
	}

	if err := s.userRepo.Create(ctx, user); err != nil {
		return nil, false, errors.ErrDatabaseError.WithError(err)
	}

	// 创建用户钱包
	wallet := &models.UserWallet{
		UserID: user.ID,
	}
	if err := s.db.WithContext(ctx).Create(wallet).Error; err != nil {
		return nil, false, errors.ErrDatabaseError.WithError(err)
	}

	return user, true, nil
}

// generateNickname 生成默认昵称
func (s *AuthService) generateNickname(phone string) string {
	if len(phone) >= 4 {
		return fmt.Sprintf("用户%s", phone[len(phone)-4:])
	}
	return fmt.Sprintf("用户%d", time.Now().UnixNano()%10000)
}

// toUserInfo 转换为用户信息
func (s *AuthService) toUserInfo(user *models.User) *UserInfo {
	return &UserInfo{
		ID:            user.ID,
		Phone:         user.Phone,
		Nickname:      user.Nickname,
		Avatar:        user.Avatar,
		Gender:        user.Gender,
		MemberLevelID: user.MemberLevelID,
		Points:        user.Points,
		IsVerified:    user.IsVerified,
	}
}

// RefreshToken 刷新 Token
func (s *AuthService) RefreshToken(ctx context.Context, refreshToken string) (*jwt.TokenPair, error) {
	tokenPair, err := s.jwtManager.RefreshToken(refreshToken)
	if err != nil {
		if err == jwt.ErrTokenExpired {
			return nil, errors.ErrTokenExpired
		}
		return nil, errors.ErrTokenInvalid
	}
	return tokenPair, nil
}

// GetUserByID 根据 ID 获取用户
func (s *AuthService) GetUserByID(ctx context.Context, userID int64) (*models.User, error) {
	user, err := s.userRepo.GetByID(ctx, userID)
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, errors.ErrUserNotFound
		}
		return nil, errors.ErrDatabaseError.WithError(err)
	}
	return user, nil
}
