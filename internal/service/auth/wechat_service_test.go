package auth

import (
	"bytes"
	"context"
	"io"
	"net/http"
	"net/url"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"

	appErrors "github.com/dumeirei/smart-locker-backend/internal/common/errors"
	"github.com/dumeirei/smart-locker-backend/internal/common/jwt"
	"github.com/dumeirei/smart-locker-backend/internal/models"
	"github.com/dumeirei/smart-locker-backend/internal/repository"
)

type roundTripperFunc func(*http.Request) (*http.Response, error)

func (f roundTripperFunc) RoundTrip(r *http.Request) (*http.Response, error) { return f(r) }

func setupWechatTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Silent),
	})
	require.NoError(t, err)

	require.NoError(t, db.AutoMigrate(
		&models.User{},
		&models.UserWallet{},
		&models.MemberLevel{},
		&models.Distributor{},
	))

	db.Create(&models.MemberLevel{
		ID:        1,
		Name:      "普通会员",
		Level:     1,
		MinPoints: 0,
		Discount:  1.0,
	})

	return db
}

func setupWechatService(t *testing.T) (*WechatService, *gorm.DB) {
	t.Helper()
	db := setupWechatTestDB(t)
	userRepo := repository.NewUserRepository(db)
	jwtManager := jwt.NewManager(&jwt.Config{
		Secret:            "test-secret-key",
		AccessExpireTime:  time.Hour,
		RefreshExpireTime: 2 * time.Hour,
		Issuer:            "test",
	})

	svc := NewWechatService(&WechatConfig{
		AppID:     "wx_test",
		AppSecret: "secret_test",
	}, db, userRepo, jwtManager)

	return svc, db
}

func TestWechatService_WechatLogin_Code2SessionFailed(t *testing.T) {
	svc, _ := setupWechatService(t)

	svc.httpClient = &http.Client{
		Transport: roundTripperFunc(func(r *http.Request) (*http.Response, error) {
			u, _ := url.Parse(r.URL.String())
			code := u.Query().Get("js_code")
			if code == "bad" {
				body := `{"errcode":40029,"errmsg":"invalid code"}`
				return &http.Response{
					StatusCode: 200,
					Body:       io.NopCloser(bytes.NewBufferString(body)),
					Header:     make(http.Header),
				}, nil
			}
			t.Fatalf("unexpected code: %s", code)
			return nil, nil
		}),
	}

	_, err := svc.WechatLogin(context.Background(), &WechatLoginRequest{Code: "bad"})
	require.Error(t, err)
	var appErr *appErrors.AppError
	require.ErrorAs(t, err, &appErr)
	assert.Equal(t, appErrors.ErrExternalService.Code, appErr.Code)
	assert.Contains(t, appErr.Message, "微信登录失败")
}

func TestWechatService_WechatLogin_NewUserWithInviteCode(t *testing.T) {
	svc, db := setupWechatService(t)

	// referrer user + distributor invite code
	refPhone := "13800138099"
	ref := &models.User{Phone: &refPhone, Nickname: "推荐人", MemberLevelID: 1, Status: models.UserStatusActive}
	require.NoError(t, db.Create(ref).Error)
	require.NoError(t, db.Create(&models.Distributor{UserID: ref.ID, InviteCode: "INVITE123", Status: models.DistributorStatusApproved}).Error)

	svc.httpClient = &http.Client{
		Transport: roundTripperFunc(func(r *http.Request) (*http.Response, error) {
			body := `{"openid":"openid_1","session_key":"sk","unionid":"union_1","errcode":0,"errmsg":""}`
			return &http.Response{
				StatusCode: 200,
				Body:       io.NopCloser(bytes.NewBufferString(body)),
				Header:     make(http.Header),
			}, nil
		}),
	}

	nickname := "小明"
	avatar := "https://example.com/a.png"
	gender := int8(models.GenderMale)
	invite := "INVITE123"

	resp, err := svc.WechatLogin(context.Background(), &WechatLoginRequest{
		Code:       "good",
		Nickname:   &nickname,
		Avatar:     &avatar,
		Gender:     &gender,
		InviteCode: &invite,
	})
	require.NoError(t, err)
	require.NotNil(t, resp)
	assert.True(t, resp.IsNewUser)
	assert.NotEmpty(t, resp.TokenPair.AccessToken)
	assert.Equal(t, nickname, resp.User.Nickname)

	var created models.User
	require.NoError(t, db.Where("openid = ?", "openid_1").First(&created).Error)
	require.NotNil(t, created.ReferrerID)
	assert.Equal(t, ref.ID, *created.ReferrerID)
	require.NotNil(t, created.UnionID)
	assert.Equal(t, "union_1", *created.UnionID)

	// wallet should be created
	var wallet models.UserWallet
	require.NoError(t, db.Where("user_id = ?", created.ID).First(&wallet).Error)
}

func TestWechatService_WechatLogin_ExistingUserUpdatesProfile(t *testing.T) {
	svc, db := setupWechatService(t)

	openid := "openid_exist"
	oldNickname := "旧昵称"
	user := &models.User{
		OpenID:        &openid,
		Nickname:      oldNickname,
		MemberLevelID: 1,
		Status:        models.UserStatusActive,
	}
	require.NoError(t, db.Create(user).Error)

	svc.httpClient = &http.Client{
		Transport: roundTripperFunc(func(r *http.Request) (*http.Response, error) {
			body := `{"openid":"openid_exist","session_key":"sk","errcode":0,"errmsg":""}`
			return &http.Response{
				StatusCode: 200,
				Body:       io.NopCloser(bytes.NewBufferString(body)),
				Header:     make(http.Header),
			}, nil
		}),
	}

	newNickname := "新昵称"
	newAvatar := "https://example.com/new.png"
	gender := int8(models.GenderFemale)

	resp, err := svc.WechatLogin(context.Background(), &WechatLoginRequest{
		Code:     "good",
		Nickname: &newNickname,
		Avatar:   &newAvatar,
		Gender:   &gender,
	})
	require.NoError(t, err)
	assert.False(t, resp.IsNewUser)

	var updated models.User
	require.NoError(t, db.First(&updated, user.ID).Error)
	assert.Equal(t, newNickname, updated.Nickname)
	require.NotNil(t, updated.Avatar)
	assert.Equal(t, newAvatar, *updated.Avatar)
	assert.Equal(t, gender, updated.Gender)
}

func TestWechatService_WechatLogin_DisabledUser(t *testing.T) {
	svc, db := setupWechatService(t)

	openid := "openid_disabled"
	user := &models.User{
		OpenID:        &openid,
		Nickname:      "禁用用户",
		MemberLevelID: 1,
		Status:        models.UserStatusDisabled,
	}
	require.NoError(t, db.Create(user).Error)
	// gorm may omit zero-value fields with default tags on create; force-disable explicitly
	require.NoError(t, db.Model(&models.User{}).Where("id = ?", user.ID).Update("status", models.UserStatusDisabled).Error)

	svc.httpClient = &http.Client{
		Transport: roundTripperFunc(func(r *http.Request) (*http.Response, error) {
			body := `{"openid":"openid_disabled","session_key":"sk","errcode":0,"errmsg":""}`
			return &http.Response{
				StatusCode: 200,
				Body:       io.NopCloser(bytes.NewBufferString(body)),
				Header:     make(http.Header),
			}, nil
		}),
	}

	_, err := svc.WechatLogin(context.Background(), &WechatLoginRequest{Code: "good"})
	require.Error(t, err)
	var appErr *appErrors.AppError
	require.ErrorAs(t, err, &appErr)
	assert.Equal(t, appErrors.ErrAccountDisabled.Code, appErr.Code)
}

func TestWechatService_BindPhone(t *testing.T) {
	svc, db := setupWechatService(t)
	ctx := context.Background()

	openid := "openid_bind"
	user := &models.User{
		OpenID:        &openid,
		Nickname:      "待绑定",
		MemberLevelID: 1,
		Status:        models.UserStatusActive,
	}
	require.NoError(t, db.Create(user).Error)

	redisClient, _ := newTestRedisClient(t)
	smsSender := &stubSMSSender{}
	codeSvc := NewCodeService(redisClient, smsSender, nil)

	phone := "13800138111"
	require.NoError(t, codeSvc.SendCode(ctx, phone, CodeTypeBind))
	code, err := redisClient.Get(ctx, codeSvc.codeKey(phone, CodeTypeBind)).Result()
	require.NoError(t, err)

	require.NoError(t, svc.BindPhone(ctx, user.ID, phone, code, codeSvc))

	var updated models.User
	require.NoError(t, db.First(&updated, user.ID).Error)
	require.NotNil(t, updated.Phone)
	assert.Equal(t, phone, *updated.Phone)

	// conflict phone
	otherPhone := "13800138222"
	other := &models.User{Nickname: "其他", MemberLevelID: 1, Status: models.UserStatusActive, Phone: &otherPhone}
	require.NoError(t, db.Create(other).Error)

	require.NoError(t, codeSvc.SendCode(ctx, otherPhone, CodeTypeBind))
	otherCode, err := redisClient.Get(ctx, codeSvc.codeKey(otherPhone, CodeTypeBind)).Result()
	require.NoError(t, err)

	err = svc.BindPhone(ctx, user.ID, otherPhone, otherCode, codeSvc)
	require.Error(t, err)
	var appErr *appErrors.AppError
	require.ErrorAs(t, err, &appErr)
	assert.Equal(t, appErrors.ErrPhoneExists.Code, appErr.Code)
}
