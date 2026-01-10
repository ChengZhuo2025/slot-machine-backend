package auth

import (
	"context"
	"testing"
	"time"

	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/dumeirei/smart-locker-backend/pkg/sms"
)

type stubSMSSender struct {
	sendErr error
	last    struct {
		phone        string
		templateCode string
		params       map[string]string
	}
}

func (s *stubSMSSender) Send(ctx context.Context, phone, templateCode string, params map[string]string) error {
	s.last.phone = phone
	s.last.templateCode = templateCode
	s.last.params = params
	return s.sendErr
}

func (s *stubSMSSender) SendVerifyCode(ctx context.Context, phone, code string) error {
	return s.Send(ctx, phone, "verify_code", map[string]string{"code": code})
}

func (s *stubSMSSender) SendOrderNotify(ctx context.Context, phone, orderNo string) error {
	return s.Send(ctx, phone, "order_notify", map[string]string{"order_no": orderNo})
}

func TestCodeService_SendCodeAndVerifyCode(t *testing.T) {
	redisClient, clock := newTestRedisClient(t)
	smsSender := &stubSMSSender{}
	svc := NewCodeService(redisClient, smsSender, &CodeServiceConfig{
		CodeLength: 6,
		ExpireIn:   5 * time.Minute,
	})

	ctx := context.Background()
	phone := "13800138000"

	require.NoError(t, svc.SendCode(ctx, phone, CodeTypeLogin))
	assert.Equal(t, phone, smsSender.last.phone)
	assert.Equal(t, string(sms.TemplateCodeLogin), smsSender.last.templateCode)
	assert.Contains(t, smsSender.last.params, "code")

	codeKey := svc.codeKey(phone, CodeTypeLogin)
	code, err := redisClient.Get(ctx, codeKey).Result()
	require.NoError(t, err)
	assert.Len(t, code, 6)

	// wrong code does not consume stored code
	ok, err := svc.VerifyCode(ctx, phone, "000000", CodeTypeLogin)
	require.NoError(t, err)
	assert.False(t, ok)
	_, err = redisClient.Get(ctx, codeKey).Result()
	require.NoError(t, err)

	// correct code consumes stored code (one-time)
	ok, err = svc.VerifyCode(ctx, phone, code, CodeTypeLogin)
	require.NoError(t, err)
	assert.True(t, ok)
	_, err = redisClient.Get(ctx, codeKey).Result()
	assert.ErrorIs(t, err, redis.Nil)

	// second verify returns false
	ok, err = svc.VerifyCode(ctx, phone, code, CodeTypeLogin)
	require.NoError(t, err)
	assert.False(t, ok)

	// advance time to prove server clock is controllable (sanity only)
	clock.Advance(6 * time.Minute)
}

func TestCodeService_SendCode_RateLimit(t *testing.T) {
	redisClient, _ := newTestRedisClient(t)
	smsSender := &stubSMSSender{}
	svc := NewCodeService(redisClient, smsSender, nil)
	ctx := context.Background()

	phone := "13800138001"
	require.NoError(t, svc.SendCode(ctx, phone, CodeTypeLogin))

	err := svc.SendCode(ctx, phone, CodeTypeLogin)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "短信发送过于频繁")
}

func TestCodeService_SendCode_DayLimit(t *testing.T) {
	redisClient, clock := newTestRedisClient(t)
	smsSender := &stubSMSSender{}
	svc := NewCodeService(redisClient, smsSender, nil)
	ctx := context.Background()

	phone := "13800138002"

	for i := 0; i < 10; i++ {
		require.NoError(t, svc.SendCode(ctx, phone, CodeTypeLogin))
		clock.Advance(time.Minute + time.Second) // bypass send frequency key TTL
	}

	err := svc.SendCode(ctx, phone, CodeTypeLogin)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "今日短信发送次数已达上限")
}

func TestCodeService_SendCode_SendFailRollbackCode(t *testing.T) {
	redisClient, _ := newTestRedisClient(t)
	smsSender := &stubSMSSender{sendErr: assert.AnError}
	svc := NewCodeService(redisClient, smsSender, nil)
	ctx := context.Background()

	phone := "13800138003"
	err := svc.SendCode(ctx, phone, CodeTypeLogin)
	require.Error(t, err)

	codeKey := svc.codeKey(phone, CodeTypeLogin)
	_, getErr := redisClient.Get(ctx, codeKey).Result()
	assert.ErrorIs(t, getErr, redis.Nil)
}
