package distribution

import (
	"context"
	"regexp"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/dumeirei/smart-locker-backend/internal/models"
	"github.com/dumeirei/smart-locker-backend/internal/repository"
)

func TestInviteService_GenerateInviteInfo(t *testing.T) {
	db := setupDistributorTestDB(t)
	distributorRepo := repository.NewDistributorRepository(db)
	svc := NewInviteService(distributorRepo, "https://example.test")
	ctx := context.Background()

	user := createDistributorTestUser(db, nil)
	user.Nickname = "小明"
	require.NoError(t, db.Save(user).Error)

	distributor := &models.Distributor{
		UserID:     user.ID,
		Level:      models.DistributorLevelDirect,
		InviteCode: "INVITE01",
		Status:     models.DistributorStatusApproved,
	}
	require.NoError(t, db.Create(distributor).Error)

	info, err := svc.GenerateInviteInfo(ctx, distributor.ID)
	require.NoError(t, err)
	require.NotNil(t, info)
	assert.Equal(t, "INVITE01", info.InviteCode)
	assert.Equal(t, "https://example.test/invite/INVITE01", info.InviteLink)
	assert.Contains(t, info.QRCodeURL, "https://example.test/api/qrcode?")
	assert.Contains(t, info.QRCodeURL, "size=200")
	assert.Equal(t, "https://example.test/api/poster?code=INVITE01", info.PosterURL)
	assert.Equal(t, distributor.ID, info.DistributorID)
	assert.Equal(t, "小明", info.UserName)

	assert.Regexp(t, regexp.MustCompile(`^https://example\.test/s/[0-9a-f]{8}$`), info.ShortLink)

	t.Run("未审核通过返回错误", func(t *testing.T) {
		pending := &models.Distributor{
			UserID:     createDistributorTestUser(db, nil).ID,
			Level:      models.DistributorLevelDirect,
			InviteCode: "INVITE_PENDING",
			Status:     models.DistributorStatusPending,
		}
		require.NoError(t, db.Create(pending).Error)

		_, err := svc.GenerateInviteInfo(ctx, pending.ID)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "尚未审核通过")
	})
}

func TestInviteService_ValidateInviteCode(t *testing.T) {
	db := setupDistributorTestDB(t)
	distributorRepo := repository.NewDistributorRepository(db)
	svc := NewInviteService(distributorRepo, "https://example.test")
	ctx := context.Background()

	user := createDistributorTestUser(db, nil)
	approved := &models.Distributor{
		UserID:     user.ID,
		Level:      models.DistributorLevelDirect,
		InviteCode: "OK01",
		Status:     models.DistributorStatusApproved,
	}
	require.NoError(t, db.Create(approved).Error)

	t.Run("有效邀请码返回分销商", func(t *testing.T) {
		got, err := svc.ValidateInviteCode(ctx, "OK01")
		require.NoError(t, err)
		require.NotNil(t, got)
		assert.Equal(t, approved.ID, got.ID)
	})

	t.Run("不存在的邀请码返回无效", func(t *testing.T) {
		_, err := svc.ValidateInviteCode(ctx, "NOT_EXISTS")
		require.Error(t, err)
		assert.Contains(t, err.Error(), "邀请码无效")
	})

	t.Run("未审核通过返回错误", func(t *testing.T) {
		pending := &models.Distributor{
			UserID:     createDistributorTestUser(db, nil).ID,
			Level:      models.DistributorLevelDirect,
			InviteCode: "PEND01",
			Status:     models.DistributorStatusPending,
		}
		require.NoError(t, db.Create(pending).Error)

		_, err := svc.ValidateInviteCode(ctx, "PEND01")
		require.Error(t, err)
		assert.Contains(t, err.Error(), "尚未通过审核")
	})
}

func TestInviteService_GetInviteCodeFromLink(t *testing.T) {
	svc := NewInviteService(nil, "https://example.test")

	t.Run("从 /invite/ 提取邀请码", func(t *testing.T) {
		code, err := svc.GetInviteCodeFromLink("https://example.test/invite/ABC123")
		require.NoError(t, err)
		assert.Equal(t, "ABC123", code)
	})

	t.Run("短链需要专门解析", func(t *testing.T) {
		_, err := svc.GetInviteCodeFromLink("https://example.test/s/abcd1234")
		require.Error(t, err)
		assert.Contains(t, err.Error(), "短链接")
	})

	t.Run("无效URL返回错误", func(t *testing.T) {
		_, err := svc.GetInviteCodeFromLink("://bad")
		require.Error(t, err)
		assert.Contains(t, err.Error(), "无效的邀请链接")
	})

	t.Run("路径格式不匹配返回错误", func(t *testing.T) {
		_, err := svc.GetInviteCodeFromLink("https://example.test/other/ABC123")
		require.Error(t, err)
		assert.Contains(t, err.Error(), "无效的邀请链接格式")
	})
}

func TestInviteService_GenerateShareContent(t *testing.T) {
	db := setupDistributorTestDB(t)
	distributorRepo := repository.NewDistributorRepository(db)
	svc := NewInviteService(distributorRepo, "https://example.test")
	ctx := context.Background()

	user := createDistributorTestUser(db, nil)
	user.Nickname = "小明"
	require.NoError(t, db.Save(user).Error)

	distributor := &models.Distributor{
		UserID:     user.ID,
		Level:      models.DistributorLevelDirect,
		InviteCode: "INVITE01",
		Status:     models.DistributorStatusApproved,
	}
	require.NoError(t, db.Create(distributor).Error)

	content, err := svc.GenerateShareContent(ctx, distributor.ID)
	require.NoError(t, err)
	require.NotNil(t, content)
	assert.NotEmpty(t, content.Title)
	assert.Contains(t, content.Description, "小明")
	assert.Equal(t, "https://example.test/invite/INVITE01", content.Link)
	assert.Contains(t, content.ImageURL, "https://example.test/api/poster?code=INVITE01")
	assert.Equal(t, "/pages/register/index?code=INVITE01", content.WechatPath)
}

func TestInviteService_TrackClick(t *testing.T) {
	db := setupDistributorTestDB(t)
	distributorRepo := repository.NewDistributorRepository(db)
	svc := NewInviteService(distributorRepo, "https://example.test")
	ctx := context.Background()

	user := createDistributorTestUser(db, nil)
	distributor := &models.Distributor{
		UserID:     user.ID,
		Level:      models.DistributorLevelDirect,
		InviteCode: "INVITE01",
		Status:     models.DistributorStatusApproved,
	}
	require.NoError(t, db.Create(distributor).Error)

	require.NoError(t, svc.TrackClick(ctx, "INVITE01", "wechat"))

	err := svc.TrackClick(ctx, "NOT_EXISTS", "wechat")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "邀请码无效")
}

func TestInviteService_EncodeDecodeInviteCode(t *testing.T) {
	svc := NewInviteService(nil, "")

	encoded := svc.EncodeInviteCode("ABC123")
	decoded, err := svc.DecodeInviteCode(encoded)
	require.NoError(t, err)
	assert.Equal(t, "ABC123", decoded)

	_, err = svc.DecodeInviteCode("not_base64!!")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "无效的邀请码")
}

