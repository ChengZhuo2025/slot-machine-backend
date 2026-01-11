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

	t.Run("无效token", func(t *testing.T) {
		_, err := service.RefreshToken(ctx, "invalid-token")
		assert.Error(t, err)
	})

	t.Run("过期token", func(t *testing.T) {
		// 创建一个过期的 JWT manager
		expiredJWTManager := jwt.NewManager(&jwt.Config{
			Secret:            "test-secret-key",
			AccessExpireTime:  -time.Hour,  // 负数表示已过期
			RefreshExpireTime: -time.Hour,  // 负数表示已过期
			Issuer:            "test",
		})

		// 生成一个已过期的 token
		tokenPair, err := expiredJWTManager.GenerateTokenPair(1, jwt.UserTypeUser, "")
		require.NoError(t, err)

		// 尝试用已过期的 token 刷新
		_, err = service.RefreshToken(ctx, tokenPair.RefreshToken)
		assert.Error(t, err)
	})
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

	t.Run("正常手机号", func(t *testing.T) {
		nickname := service.generateNickname("13800138000")
		assert.Equal(t, "用户8000", nickname)
	})

	t.Run("手机号长度不足4位使用时间戳", func(t *testing.T) {
		nickname := service.generateNickname("123")
		assert.Contains(t, nickname, "用户")
		// 验证格式为 "用户" + 数字，数字范围 0-9999
		assert.Regexp(t, `^用户\d{1,4}$`, nickname)
	})
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

func TestAuthService_SendSmsCode_Real(t *testing.T) {
	service, _ := setupTestAuthService(t)
	ctx := context.Background()

	// Set up code service for the real auth service
	redisClient, _ := newTestRedisClient(t)
	smsSender := &stubSMSSender{}
	service.codeService = NewCodeService(redisClient, smsSender, nil)

	// Test successful code send
	err := service.SendSmsCode(ctx, &SendSmsCodeRequest{
		Phone:    "13800138000",
		CodeType: CodeTypeLogin,
	})
	assert.NoError(t, err)

	// Test invalid phone
	err = service.SendSmsCode(ctx, &SendSmsCodeRequest{
		Phone:    "123",
		CodeType: CodeTypeLogin,
	})
	assert.Error(t, err)
}

func TestAuthService_SmsLogin_Real(t *testing.T) {
	service, _ := setupTestAuthService(t)
	ctx := context.Background()

	// Set up code service
	redisClient, _ := newTestRedisClient(t)
	smsSender := &stubSMSSender{}
	service.codeService = NewCodeService(redisClient, smsSender, nil)

	phone := "13800138999"

	// Send code first
	err := service.SendSmsCode(ctx, &SendSmsCodeRequest{
		Phone:    phone,
		CodeType: CodeTypeLogin,
	})
	require.NoError(t, err)

	// Get the code from Redis
	code, err := redisClient.Get(ctx, service.codeService.codeKey(phone, CodeTypeLogin)).Result()
	require.NoError(t, err)

	// Test successful login
	resp, err := service.SmsLogin(ctx, &SmsLoginRequest{
		Phone: phone,
		Code:  code,
	})
	require.NoError(t, err)
	assert.NotNil(t, resp)
	assert.True(t, resp.IsNewUser)
	assert.Equal(t, phone, *resp.User.Phone)

	// Test login with wrong code
	_, err = service.SmsLogin(ctx, &SmsLoginRequest{
		Phone: phone,
		Code:  "wrong_code",
	})
	assert.Error(t, err)
}

func TestAuthService_FindOrCreateUser_WithReferrer(t *testing.T) {
	service, db := setupTestAuthService(t)
	ctx := context.Background()

	// Create referrer
	refPhone := "13800138001"
	referrer := &models.User{
		Phone:         &refPhone,
		Nickname:      "推荐人",
		MemberLevelID: 1,
		Status:        models.UserStatusActive,
	}
	require.NoError(t, db.Create(referrer).Error)

	// Create distributor
	require.NoError(t, db.Create(&models.Distributor{
		UserID:     referrer.ID,
		InviteCode: "TEST123",
		Status:     models.DistributorStatusApproved,
	}).Error)

	// Create new user with invite code
	newPhone := "13800138002"
	inviteCode := "TEST123"
	user, isNew, err := service.findOrCreateUser(ctx, newPhone, &inviteCode)
	require.NoError(t, err)
	assert.True(t, isNew)
	assert.NotNil(t, user.ReferrerID)
	assert.Equal(t, referrer.ID, *user.ReferrerID)

	// Try to create same user again (should return existing)
	user2, isNew2, err := service.findOrCreateUser(ctx, newPhone, nil)
	require.NoError(t, err)
	assert.False(t, isNew2)
	assert.Equal(t, user.ID, user2.ID)
}

func TestAuthService_GetCodeExpireIn(t *testing.T) {
	service, _ := setupTestAuthService(t)

	// Set up code service
	redisClient, _ := newTestRedisClient(t)
	smsSender := &stubSMSSender{}
	service.codeService = NewCodeService(redisClient, smsSender, nil)

	expireTime := service.codeService.GetCodeExpireIn()
	assert.Greater(t, expireTime, time.Duration(0))
}

func TestAuthService_SmsLogin_VerifyCodeError(t *testing.T) {
	service, _ := setupTestAuthService(t)
	ctx := context.Background()

	// Set up code service with a faulty Redis (will cause VerifyCode to fail)
	redisClient, _ := newTestRedisClient(t)
	smsSender := &stubSMSSender{}
	service.codeService = NewCodeService(redisClient, smsSender, nil)

	// Try to login without sending code (Redis will be empty)
	_, err := service.SmsLogin(ctx, &SmsLoginRequest{
		Phone: "13800139999",
		Code:  "123456",
	})
	assert.Error(t, err)
}

func TestCodeService_AllCodeTypes(t *testing.T) {
	redisClient, _ := newTestRedisClient(t)
	smsSender := &stubSMSSender{}
	codeSvc := NewCodeService(redisClient, smsSender, nil)
	ctx := context.Background()

	// Test all code types to cover getTemplateCode switch cases
	codeTypes := []struct {
		codeType CodeType
		phone    string
	}{
		{CodeTypeLogin, "13800138881"},
		{CodeTypeRegister, "13800138882"},
		{CodeTypeBind, "13800138883"},
		{CodeTypeReset, "13800138884"},
	}

	for _, tc := range codeTypes {
		t.Run(string(tc.codeType), func(t *testing.T) {
			err := codeSvc.SendCode(ctx, tc.phone, tc.codeType)
			assert.NoError(t, err)

			// Get and verify the code
			code, err := redisClient.Get(ctx, codeSvc.codeKey(tc.phone, tc.codeType)).Result()
			require.NoError(t, err)

			valid, err := codeSvc.VerifyCode(ctx, tc.phone, code, tc.codeType)
			require.NoError(t, err)
			assert.True(t, valid)
		})
	}
}

func TestAuthService_SendSmsCode_RateLimitError(t *testing.T) {
	service, _ := setupTestAuthService(t)
	ctx := context.Background()

	redisClient, _ := newTestRedisClient(t)
	smsSender := &stubSMSSender{}
	service.codeService = NewCodeService(redisClient, smsSender, nil)

	phone := "13800137777"

	// Send first code successfully
	err := service.SendSmsCode(ctx, &SendSmsCodeRequest{
		Phone:    phone,
		CodeType: CodeTypeLogin,
	})
	require.NoError(t, err)

	// Try to send again immediately (should hit rate limit)
	err = service.SendSmsCode(ctx, &SendSmsCodeRequest{
		Phone:    phone,
		CodeType: CodeTypeLogin,
	})
	assert.Error(t, err)
}

func TestAuthService_SmsLogin_DisabledUserScenario(t *testing.T) {
	service, db := setupTestAuthService(t)
	ctx := context.Background()

	redisClient, _ := newTestRedisClient(t)
	smsSender := &stubSMSSender{}
	service.codeService = NewCodeService(redisClient, smsSender, nil)

	phone := "13900136666" // Use unique phone to avoid rate limit conflicts

	// Send a valid code first
	err := service.SendSmsCode(ctx, &SendSmsCodeRequest{
		Phone:    phone,
		CodeType: CodeTypeLogin,
	})
	require.NoError(t, err)

	// Get the valid code
	validCode, err := redisClient.Get(ctx, service.codeService.codeKey(phone, CodeTypeLogin)).Result()
	require.NoError(t, err)

	// First login to create the user
	resp, err := service.SmsLogin(ctx, &SmsLoginRequest{
		Phone: phone,
		Code:  validCode,
	})
	require.NoError(t, err)
	userID := resp.User.ID

	// Disable the user
	err = db.Model(&models.User{}).Where("id = ?", userID).Update("status", models.UserStatusDisabled).Error
	require.NoError(t, err)

	// Use a different code service instance with different phone to avoid rate limit
	phone2 := "13900136667"
	err = service.SendSmsCode(ctx, &SendSmsCodeRequest{
		Phone:    phone2,
		CodeType: CodeTypeLogin,
	})
	require.NoError(t, err)

	validCode2, err := redisClient.Get(ctx, service.codeService.codeKey(phone2, CodeTypeLogin)).Result()
	require.NoError(t, err)

	// Update the disabled user's phone to phone2
	err = db.Model(&models.User{}).Where("id = ?", userID).Update("phone", phone2).Error
	require.NoError(t, err)

	// Try to login with disabled user (using phone2)
	_, err = service.SmsLogin(ctx, &SmsLoginRequest{
		Phone: phone2,
		Code:  validCode2,
	})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "禁用")
}

func TestCodeService_GetTemplateCode_DefaultCase(t *testing.T) {
	redisClient, _ := newTestRedisClient(t)
	smsSender := &stubSMSSender{}
	codeSvc := NewCodeService(redisClient, smsSender, nil)
	ctx := context.Background()

	// Test with an invalid code type (will hit default case in getTemplateCode)
	phone := "13800135555"
	invalidCodeType := CodeType("unknown_type")

	// This should use the default template (login)
	err := codeSvc.SendCode(ctx, phone, invalidCodeType)
	assert.NoError(t, err)

	// Verify the code was sent
	code, err := redisClient.Get(ctx, codeSvc.codeKey(phone, invalidCodeType)).Result()
	assert.NoError(t, err)
	assert.NotEmpty(t, code)
}


// TestAuthService_findOrCreateUser_DatabaseError 测试数据库错误路径
func TestAuthService_FindOrCreateUser_GetByPhoneError(t *testing.T) {
	service, db := setupTestAuthService(t)
	ctx := context.Background()

	// 关闭数据库连接以模拟数据库错误
	sqlDB, _ := db.DB()
	sqlDB.Close()

	_, _, err := service.findOrCreateUser(ctx, "13800138888", nil)
	assert.Error(t, err)
}

// TestAuthService_GetUserByID_DatabaseError 测试 GetUserByID 数据库错误
func TestAuthService_GetUserByID_DatabaseError(t *testing.T) {
	service, db := setupTestAuthService(t)
	ctx := context.Background()

	// 创建一个用户
	phone := "13800138777"
	user := &models.User{
		Phone:         &phone,
		Nickname:      "测试",
		MemberLevelID: 1,
		Status:        models.UserStatusActive,
	}
	db.Create(user)

	// 关闭数据库连接以模拟数据库错误
	sqlDB, _ := db.DB()
	sqlDB.Close()

	_, err := service.GetUserByID(ctx, user.ID)
	assert.Error(t, err)
}

// TestNewAuthService 测试 AuthService 构造函数
func TestNewAuthService(t *testing.T) {
	db := setupTestDB(t)
	userRepo := repository.NewUserRepository(db)
	jwtManager := jwt.NewManager(&jwt.Config{
		Secret:            "test-secret-key",
		AccessExpireTime:  time.Hour,
		RefreshExpireTime: 2 * time.Hour,
		Issuer:            "test",
	})
	redisClient, _ := newTestRedisClient(t)
	smsSender := &stubSMSSender{}
	codeService := NewCodeService(redisClient, smsSender, nil)

	service := NewAuthService(db, userRepo, jwtManager, codeService)

	assert.NotNil(t, service)
	assert.NotNil(t, service.db)
	assert.NotNil(t, service.userRepo)
	assert.NotNil(t, service.jwtManager)
	assert.NotNil(t, service.codeService)
}

// TestAuthService_FindOrCreateUser_WalletCreateError 测试钱包创建失败
func TestAuthService_FindOrCreateUser_WalletCreateError(t *testing.T) {
	db := setupTestDB(t)
	userRepo := repository.NewUserRepository(db)
	jwtManager := jwt.NewManager(&jwt.Config{
		Secret:            "test-secret-key",
		AccessExpireTime:  time.Hour,
		RefreshExpireTime: 2 * time.Hour,
		Issuer:            "test",
	})

	service := &AuthService{
		db:         db,
		userRepo:   userRepo,
		jwtManager: jwtManager,
	}

	ctx := context.Background()
	phone := "13800139777"

	// 删除 UserWallet 表来模拟钱包创建失败
	db.Migrator().DropTable(&models.UserWallet{})

	_, _, err := service.findOrCreateUser(ctx, phone, nil)
	assert.Error(t, err)
}

// TestAuthService_FindOrCreateUser_CreateError 测试用户创建失败
func TestAuthService_FindOrCreateUser_CreateError(t *testing.T) {
	db := setupTestDB(t)
	userRepo := repository.NewUserRepository(db)
	jwtManager := jwt.NewManager(&jwt.Config{
		Secret:            "test-secret-key",
		AccessExpireTime:  time.Hour,
		RefreshExpireTime: 2 * time.Hour,
		Issuer:            "test",
	})

	service := &AuthService{
		db:         db,
		userRepo:   userRepo,
		jwtManager: jwtManager,
	}

	ctx := context.Background()
	phone := "13800139666"

	// 删除 User 表来模拟创建失败
	db.Migrator().DropTable(&models.User{})

	_, _, err := service.findOrCreateUser(ctx, phone, nil)
	assert.Error(t, err)
}
