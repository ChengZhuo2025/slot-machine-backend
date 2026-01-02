// Package auth 提供认证服务
package auth

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"gorm.io/gorm"

	"github.com/dumeirei/smart-locker-backend/internal/common/errors"
	"github.com/dumeirei/smart-locker-backend/internal/common/jwt"
	"github.com/dumeirei/smart-locker-backend/internal/models"
	"github.com/dumeirei/smart-locker-backend/internal/repository"
)

// WechatService 微信服务
type WechatService struct {
	appID      string
	appSecret  string
	db         *gorm.DB
	userRepo   *repository.UserRepository
	jwtManager *jwt.Manager
	httpClient *http.Client
}

// WechatConfig 微信配置
type WechatConfig struct {
	AppID     string
	AppSecret string
}

// NewWechatService 创建微信服务
func NewWechatService(
	cfg *WechatConfig,
	db *gorm.DB,
	userRepo *repository.UserRepository,
	jwtManager *jwt.Manager,
) *WechatService {
	return &WechatService{
		appID:      cfg.AppID,
		appSecret:  cfg.AppSecret,
		db:         db,
		userRepo:   userRepo,
		jwtManager: jwtManager,
		httpClient: &http.Client{Timeout: 10 * time.Second},
	}
}

// WechatLoginRequest 微信登录请求
type WechatLoginRequest struct {
	Code       string  `json:"code" binding:"required"`
	Nickname   *string `json:"nickname,omitempty"`
	Avatar     *string `json:"avatar,omitempty"`
	Gender     *int8   `json:"gender,omitempty"`
	InviteCode *string `json:"invite_code,omitempty"`
}

// Code2SessionResponse 微信 code2Session 响应
type Code2SessionResponse struct {
	OpenID     string `json:"openid"`
	SessionKey string `json:"session_key"`
	UnionID    string `json:"unionid"`
	ErrCode    int    `json:"errcode"`
	ErrMsg     string `json:"errmsg"`
}

// WechatLogin 微信小程序登录
func (s *WechatService) WechatLogin(ctx context.Context, req *WechatLoginRequest) (*LoginResponse, error) {
	// 调用微信 code2Session 接口
	sessionResp, err := s.code2Session(ctx, req.Code)
	if err != nil {
		return nil, errors.ErrExternalService.WithError(err)
	}

	if sessionResp.ErrCode != 0 {
		return nil, errors.New(errors.ErrExternalService.Code,
			fmt.Sprintf("微信登录失败: %s", sessionResp.ErrMsg))
	}

	// 查找或创建用户
	user, isNew, err := s.findOrCreateWechatUser(ctx, sessionResp, req)
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

// code2Session 调用微信 code2Session 接口
func (s *WechatService) code2Session(ctx context.Context, code string) (*Code2SessionResponse, error) {
	url := fmt.Sprintf(
		"https://api.weixin.qq.com/sns/jscode2session?appid=%s&secret=%s&js_code=%s&grant_type=authorization_code",
		s.appID, s.appSecret, code,
	)

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}

	resp, err := s.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	var result Code2SessionResponse
	if err := json.Unmarshal(body, &result); err != nil {
		return nil, err
	}

	return &result, nil
}

// findOrCreateWechatUser 查找或创建微信用户
func (s *WechatService) findOrCreateWechatUser(
	ctx context.Context,
	session *Code2SessionResponse,
	req *WechatLoginRequest,
) (*models.User, bool, error) {
	// 先根据 OpenID 查找用户
	user, err := s.userRepo.GetByOpenID(ctx, session.OpenID)
	if err == nil {
		// 用户存在，更新信息
		if req.Nickname != nil || req.Avatar != nil {
			updates := make(map[string]interface{})
			if req.Nickname != nil {
				updates["nickname"] = *req.Nickname
			}
			if req.Avatar != nil {
				updates["avatar"] = *req.Avatar
			}
			if req.Gender != nil {
				updates["gender"] = *req.Gender
			}
			s.db.WithContext(ctx).Model(user).Updates(updates)
		}
		return user, false, nil
	}
	if err != gorm.ErrRecordNotFound {
		return nil, false, errors.ErrDatabaseError.WithError(err)
	}

	// 用户不存在，创建新用户
	var referrerID *int64
	if req.InviteCode != nil && *req.InviteCode != "" {
		referrer, err := s.userRepo.GetByInviteCode(ctx, *req.InviteCode)
		if err == nil {
			referrerID = &referrer.ID
		}
	}

	nickname := s.generateNickname()
	if req.Nickname != nil && *req.Nickname != "" {
		nickname = *req.Nickname
	}

	var gender int8 = models.GenderUnknown
	if req.Gender != nil {
		gender = *req.Gender
	}

	user = &models.User{
		OpenID:        &session.OpenID,
		Nickname:      nickname,
		Avatar:        req.Avatar,
		Gender:        gender,
		MemberLevelID: 1,
		Status:        models.UserStatusActive,
		ReferrerID:    referrerID,
	}

	// 如果有 UnionID，也保存
	if session.UnionID != "" {
		user.UnionID = &session.UnionID
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
func (s *WechatService) generateNickname() string {
	return fmt.Sprintf("微信用户%d", time.Now().UnixNano()%100000)
}

// toUserInfo 转换为用户信息
func (s *WechatService) toUserInfo(user *models.User) *UserInfo {
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

// BindPhone 绑定手机号
func (s *WechatService) BindPhone(ctx context.Context, userID int64, phone string, code string, codeService *CodeService) error {
	// 验证验证码
	valid, err := codeService.VerifyCode(ctx, phone, code, CodeTypeBind)
	if err != nil {
		return errors.ErrInternalError.WithError(err)
	}
	if !valid {
		return errors.ErrSmsCodeError
	}

	// 检查手机号是否已被使用
	existUser, err := s.userRepo.GetByPhone(ctx, phone)
	if err == nil && existUser.ID != userID {
		return errors.ErrPhoneExists
	}

	// 更新用户手机号
	if err := s.db.WithContext(ctx).Model(&models.User{}).Where("id = ?", userID).Update("phone", phone).Error; err != nil {
		return errors.ErrDatabaseError.WithError(err)
	}

	return nil
}
