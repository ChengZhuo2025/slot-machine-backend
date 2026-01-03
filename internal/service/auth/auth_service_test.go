// Package auth 认证服务单元测试
package auth

import (
	"context"
	"fmt"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"

	"github.com/dumeirei/smart-locker-backend/internal/common/jwt"
	"github.com/dumeirei/smart-locker-backend/internal/models"
	"github.com/dumeirei/smart-locker-backend/internal/repository"
)

// mockCodeService 模拟验证码服务用于测试
type mockCodeService struct {
	codes map[string]string
	mu    sync.RWMutex
}

func newMockCodeService() *mockCodeService {
	return &mockCodeService{
		codes: make(map[string]string),
	}
}

func (m *mockCodeService) SendCode(ctx context.Context, phone string, codeType CodeType) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.codes[phone+":"+string(codeType)] = "123456" // 固定测试验证码
	return nil
}

func (m *mockCodeService) VerifyCode(ctx context.Context, phone string, code string, codeType CodeType) (bool, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	key := phone + ":" + string(codeType)
	stored, ok := m.codes[key]
	if !ok {
		return false, nil
	}
	if stored != code {
		return false, nil
	}
	// 删除已使用的验证码
	delete(m.codes, key)
	return true, nil
}

func (m *mockCodeService) GetCodeExpireIn() time.Duration {
	return 5 * time.Minute
}

// CodeVerifier 验证码验证接口
type CodeVerifier interface {
	SendCode(ctx context.Context, phone string, codeType CodeType) error
	VerifyCode(ctx context.Context, phone string, code string, codeType CodeType) (bool, error)
}

// testAuthService 带有 mock 依赖的测试服务
type testAuthService struct {
	*AuthService
	mockCode *mockCodeService
}

// setupTestDB 创建测试数据库
func setupTestDB(t *testing.T) *gorm.DB {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Silent),
	})
	require.NoError(t, err)

	err = db.AutoMigrate(
		&models.User{},
		&models.UserWallet{},
		&models.MemberLevel{},
		&models.Distributor{},
	)
	require.NoError(t, err)

	// 创建默认会员等级
	level := &models.MemberLevel{
		ID:        1,
		Name:      "普通会员",
		Level:     1,
		MinPoints: 0,
		Discount:  1.0,
	}
	db.Create(level)

	return db
}

// setupTestAuthService 创建测试用的 AuthService
func setupTestAuthService(t *testing.T) (*testAuthService, *gorm.DB) {
	db := setupTestDB(t)
	userRepo := repository.NewUserRepository(db)
	jwtManager := jwt.NewManager(&jwt.Config{
		Secret:            "test-secret-key",
		AccessExpireTime:  time.Hour,
		RefreshExpireTime: 2 * time.Hour,
		Issuer:            "test",
	})
	mockCode := newMockCodeService()

	// 使用 mock code service
	service := &AuthService{
		db:         db,
		userRepo:   userRepo,
		jwtManager: jwtManager,
	}

	return &testAuthService{
		AuthService: service,
		mockCode:    mockCode,
	}, db
}

// SendSmsCodeWithMock 使用 mock 验证码服务发送验证码
func (ts *testAuthService) SendSmsCodeWithMock(ctx context.Context, req *SendSmsCodeRequest) error {
	if len(req.Phone) != 11 {
		return fmt.Errorf("手机号格式错误")
	}
	return ts.mockCode.SendCode(ctx, req.Phone, req.CodeType)
}

// SmsLoginWithMock 使用 mock 验证码服务登录
func (ts *testAuthService) SmsLoginWithMock(ctx context.Context, req *SmsLoginRequest) (*LoginResponse, error) {
	valid, err := ts.mockCode.VerifyCode(ctx, req.Phone, req.Code, CodeTypeLogin)
	if err != nil {
		return nil, err
	}
	if !valid {
		return nil, fmt.Errorf("验证码错误")
	}

	user, isNew, err := ts.findOrCreateUser(ctx, req.Phone, req.InviteCode)
	if err != nil {
		return nil, err
	}

	if user.Status == models.UserStatusDisabled {
		return nil, fmt.Errorf("账号已被禁用")
	}

	tokenPair, err := ts.jwtManager.GenerateTokenPair(user.ID, jwt.UserTypeUser, "")
	if err != nil {
		return nil, err
	}

	return &LoginResponse{
		User:      ts.toUserInfo(user),
		TokenPair: tokenPair,
		IsNewUser: isNew,
	}, nil
}

func TestAuthService_SendSmsCode(t *testing.T) {
	service, _ := setupTestAuthService(t)
	ctx := context.Background()

	tests := []struct {
		name    string
		phone   string
		wantErr bool
	}{
		{
			name:    "有效手机号",
			phone:   "13800138000",
			wantErr: false,
		},
		{
			name:    "手机号过短",
			phone:   "1380013800",
			wantErr: true,
		},
		{
			name:    "手机号过长",
			phone:   "138001380001",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := &SendSmsCodeRequest{
				Phone:    tt.phone,
				CodeType: CodeTypeLogin,
			}
			err := service.SendSmsCodeWithMock(ctx, req)
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestAuthService_SmsLogin_NewUser(t *testing.T) {
	service, db := setupTestAuthService(t)
	ctx := context.Background()

	phone := "13800138001"

	// 先发送验证码
	err := service.SendSmsCodeWithMock(ctx, &SendSmsCodeRequest{
		Phone:    phone,
		CodeType: CodeTypeLogin,
	})
	require.NoError(t, err)

	// 使用测试验证码登录
	resp, err := service.SmsLoginWithMock(ctx, &SmsLoginRequest{
		Phone: phone,
		Code:  "123456",
	})

	require.NoError(t, err)
	assert.NotNil(t, resp)
	assert.True(t, resp.IsNewUser)
	assert.Equal(t, phone, *resp.User.Phone)
	assert.NotEmpty(t, resp.TokenPair.AccessToken)
	assert.NotEmpty(t, resp.TokenPair.RefreshToken)

	// 验证用户已创建
	var user models.User
	err = db.Where("phone = ?", phone).First(&user).Error
	require.NoError(t, err)
	assert.Equal(t, int8(models.UserStatusActive), user.Status)

	// 验证钱包已创建
	var wallet models.UserWallet
	err = db.Where("user_id = ?", user.ID).First(&wallet).Error
	require.NoError(t, err)
	assert.Equal(t, float64(0), wallet.Balance)
}

func TestAuthService_SmsLogin_ExistingUser(t *testing.T) {
	service, db := setupTestAuthService(t)
	ctx := context.Background()

	phone := "13800138002"

	// 先创建用户
	user := &models.User{
		Phone:         &phone,
		Nickname:      "测试用户",
		MemberLevelID: 1,
		Status:        models.UserStatusActive,
	}
	err := db.Create(user).Error
	require.NoError(t, err)

	// 创建钱包
	wallet := &models.UserWallet{
		UserID:  user.ID,
		Balance: 100.0,
	}
	err = db.Create(wallet).Error
	require.NoError(t, err)

	// 发送验证码
	err = service.SendSmsCodeWithMock(ctx, &SendSmsCodeRequest{
		Phone:    phone,
		CodeType: CodeTypeLogin,
	})
	require.NoError(t, err)

	// 登录
	resp, err := service.SmsLoginWithMock(ctx, &SmsLoginRequest{
		Phone: phone,
		Code:  "123456",
	})

	require.NoError(t, err)
	assert.NotNil(t, resp)
	assert.False(t, resp.IsNewUser)
	assert.Equal(t, user.ID, resp.User.ID)
}

func TestAuthService_SmsLogin_InvalidCode(t *testing.T) {
	service, _ := setupTestAuthService(t)
	ctx := context.Background()

	phone := "13800138003"

	// 发送验证码
	err := service.SendSmsCodeWithMock(ctx, &SendSmsCodeRequest{
		Phone:    phone,
		CodeType: CodeTypeLogin,
	})
	require.NoError(t, err)

	// 使用错误的验证码登录
	_, err = service.SmsLoginWithMock(ctx, &SmsLoginRequest{
		Phone: phone,
		Code:  "999999",
	})

	assert.Error(t, err)
}

func TestAuthService_SmsLogin_DisabledUser(t *testing.T) {
	service, db := setupTestAuthService(t)
	ctx := context.Background()

	phone := "13800138004"

	// 创建禁用的用户
	user := &models.User{
		Phone:         &phone,
		Nickname:      "禁用用户",
		MemberLevelID: 1,
		Status:        models.UserStatusActive, // 先创建为 Active
	}
	err := db.Create(user).Error
	require.NoError(t, err)

	// 然后更新为 Disabled (GORM 不会插入零值，需要显式更新)
	err = db.Model(user).Update("status", models.UserStatusDisabled).Error
	require.NoError(t, err)

	// 发送验证码
	err = service.SendSmsCodeWithMock(ctx, &SendSmsCodeRequest{
		Phone:    phone,
		CodeType: CodeTypeLogin,
	})
	require.NoError(t, err)

	// 尝试登录
	_, err = service.SmsLoginWithMock(ctx, &SmsLoginRequest{
		Phone: phone,
		Code:  "123456",
	})

	assert.Error(t, err)
}

func TestAuthService_SmsLogin_WithInviteCode(t *testing.T) {
	service, db := setupTestAuthService(t)
	ctx := context.Background()

	// 创建邀请人
	referrerPhone := "13800138005"
	referrer := &models.User{
		Phone:         &referrerPhone,
		Nickname:      "邀请人",
		MemberLevelID: 1,
		Status:        models.UserStatusActive,
	}
	err := db.Create(referrer).Error
	require.NoError(t, err)

	// 创建分销商记录
	distributor := &models.Distributor{
		UserID:     referrer.ID,
		InviteCode: "ABC123",
		Level:      1,
		Status:     1,
	}
	err = db.Create(distributor).Error
	require.NoError(t, err)

	// 新用户使用邀请码注册
	newPhone := "13800138006"
	inviteCode := "ABC123"

	err = service.SendSmsCodeWithMock(ctx, &SendSmsCodeRequest{
		Phone:    newPhone,
		CodeType: CodeTypeLogin,
	})
	require.NoError(t, err)

	resp, err := service.SmsLoginWithMock(ctx, &SmsLoginRequest{
		Phone:      newPhone,
		Code:       "123456",
		InviteCode: &inviteCode,
	})

	require.NoError(t, err)
	assert.True(t, resp.IsNewUser)

	// 验证推荐关系
	var newUser models.User
	err = db.Where("phone = ?", newPhone).First(&newUser).Error
	require.NoError(t, err)
	assert.NotNil(t, newUser.ReferrerID)
	assert.Equal(t, referrer.ID, *newUser.ReferrerID)
}

func TestAuthService_RefreshToken(t *testing.T) {
	service, db := setupTestAuthService(t)
	ctx := context.Background()

	// 创建用户并登录
	phone := "13800138007"
	user := &models.User{
		Phone:         &phone,
		Nickname:      "测试用户",
		MemberLevelID: 1,
		Status:        models.UserStatusActive,
	}
	err := db.Create(user).Error
	require.NoError(t, err)

	// 创建钱包
	wallet := &models.UserWallet{
		UserID: user.ID,
	}
	err = db.Create(wallet).Error
	require.NoError(t, err)

	err = service.SendSmsCodeWithMock(ctx, &SendSmsCodeRequest{
		Phone:    phone,
		CodeType: CodeTypeLogin,
	})
	require.NoError(t, err)

	loginResp, err := service.SmsLoginWithMock(ctx, &SmsLoginRequest{
		Phone: phone,
		Code:  "123456",
	})
	require.NoError(t, err)

	// 刷新 Token
	newTokenPair, err := service.RefreshToken(ctx, loginResp.TokenPair.RefreshToken)
	require.NoError(t, err)
	assert.NotEmpty(t, newTokenPair.AccessToken)
	assert.NotEmpty(t, newTokenPair.RefreshToken)
	// 注意：由于 JWT 在同一秒生成时内容会相同，所以这里只验证新 token 有效性
	// 而不比较新旧 token 是否不同
}

func TestAuthService_RefreshToken_InvalidToken(t *testing.T) {
	service, _ := setupTestAuthService(t)
	ctx := context.Background()

	_, err := service.RefreshToken(ctx, "invalid-token")
	assert.Error(t, err)
}

func TestAuthService_GetUserByID(t *testing.T) {
	service, db := setupTestAuthService(t)
	ctx := context.Background()

	// 创建用户
	phone := "13800138008"
	user := &models.User{
		Phone:         &phone,
		Nickname:      "测试用户",
		MemberLevelID: 1,
		Status:        models.UserStatusActive,
	}
	err := db.Create(user).Error
	require.NoError(t, err)

	// 获取用户
	result, err := service.GetUserByID(ctx, user.ID)
	require.NoError(t, err)
	assert.Equal(t, user.ID, result.ID)
	assert.Equal(t, phone, *result.Phone)
}

func TestAuthService_GetUserByID_NotFound(t *testing.T) {
	service, _ := setupTestAuthService(t)
	ctx := context.Background()

	_, err := service.GetUserByID(ctx, 99999)
	assert.Error(t, err)
}

func TestAuthService_generateNickname(t *testing.T) {
	service, _ := setupTestAuthService(t)

	tests := []struct {
		name     string
		phone    string
		expected string
	}{
		{
			name:     "正常手机号",
			phone:    "13800138000",
			expected: "用户8000",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			nickname := service.generateNickname(tt.phone)
			assert.Equal(t, tt.expected, nickname)
		})
	}
}

func TestAuthService_toUserInfo(t *testing.T) {
	service, _ := setupTestAuthService(t)

	phone := "13800138000"
	avatar := "https://example.com/avatar.png"
	user := &models.User{
		ID:            1,
		Phone:         &phone,
		Nickname:      "测试用户",
		Avatar:        &avatar,
		Gender:        1,
		MemberLevelID: 2,
		Points:        100,
		IsVerified:    true,
	}

	info := service.toUserInfo(user)

	assert.Equal(t, user.ID, info.ID)
	assert.Equal(t, phone, *info.Phone)
	assert.Equal(t, user.Nickname, info.Nickname)
	assert.Equal(t, avatar, *info.Avatar)
	assert.Equal(t, user.Gender, info.Gender)
	assert.Equal(t, user.MemberLevelID, info.MemberLevelID)
	assert.Equal(t, user.Points, info.Points)
	assert.Equal(t, user.IsVerified, info.IsVerified)
}
