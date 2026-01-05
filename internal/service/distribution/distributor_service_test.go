package distribution

import (
	"context"
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"

	"github.com/dumeirei/smart-locker-backend/internal/models"
	"github.com/dumeirei/smart-locker-backend/internal/repository"
)

// setupDistributorTestDB 创建分销商测试数据库
func setupDistributorTestDB(t *testing.T) *gorm.DB {
	dsn := fmt.Sprintf("file:%s?mode=memory&cache=shared", strings.ReplaceAll(t.Name(), "/", "_"))
	db, err := gorm.Open(sqlite.Open(dsn), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Silent),
	})
	require.NoError(t, err)

	sqlDB, err := db.DB()
	require.NoError(t, err)
	sqlDB.SetMaxOpenConns(1)
	sqlDB.SetMaxIdleConns(1)

	err = db.AutoMigrate(
		&models.User{},
		&models.UserWallet{},
		&models.MemberLevel{},
		&models.Distributor{},
		&models.Admin{},
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

// createDistributorTestUser 创建测试用户用于分销商测试
func createDistributorTestUser(db *gorm.DB, referrerID *int64) *models.User {
	phone := fmt.Sprintf("138%08d", time.Now().UnixNano()%100000000)
	user := &models.User{
		Phone:         &phone,
		Nickname:      "测试用户",
		MemberLevelID: 1,
		ReferrerID:    referrerID,
		Status:        models.UserStatusActive,
	}
	db.Create(user)
	return user
}

func TestDistributorService_Apply(t *testing.T) {
	t.Run("用户正常申请成为分销商_无邀请码", func(t *testing.T) {
		db := setupDistributorTestDB(t)
		distributorRepo := repository.NewDistributorRepository(db)
		userRepo := repository.NewUserRepository(db)
		svc := NewDistributorService(distributorRepo, userRepo, db)
		ctx := context.Background()

		user := createDistributorTestUser(db, nil)

		req := &ApplyRequest{
			UserID:     user.ID,
			InviteCode: nil,
		}
		resp, err := svc.Apply(ctx, req)

		require.NoError(t, err)
		assert.NotNil(t, resp)
		assert.NotNil(t, resp.Distributor)
		assert.Equal(t, user.ID, resp.Distributor.UserID)
		assert.Equal(t, models.DistributorStatusPending, resp.Distributor.Status)
		assert.Equal(t, models.DistributorLevelDirect, resp.Distributor.Level)
		assert.NotEmpty(t, resp.Distributor.InviteCode)
		assert.Nil(t, resp.Distributor.ParentID)
		assert.Contains(t, resp.Message, "等待审核")
	})

	t.Run("用户申请成为分销商_有效邀请码", func(t *testing.T) {
		db := setupDistributorTestDB(t)
		distributorRepo := repository.NewDistributorRepository(db)
		userRepo := repository.NewUserRepository(db)
		svc := NewDistributorService(distributorRepo, userRepo, db)
		ctx := context.Background()

		// 创建上级分销商
		parentUser := createDistributorTestUser(db, nil)
		parentDistributor := &models.Distributor{
			UserID:     parentUser.ID,
			Level:      models.DistributorLevelDirect,
			InviteCode: "PARENT01",
			Status:     models.DistributorStatusApproved,
		}
		db.Create(parentDistributor)

		// 创建申请者
		user := createDistributorTestUser(db, nil)
		inviteCode := "PARENT01"

		req := &ApplyRequest{
			UserID:     user.ID,
			InviteCode: &inviteCode,
		}
		resp, err := svc.Apply(ctx, req)

		require.NoError(t, err)
		assert.NotNil(t, resp)
		assert.NotNil(t, resp.Distributor.ParentID)
		assert.Equal(t, parentDistributor.ID, *resp.Distributor.ParentID)
	})

	t.Run("用户不存在_返回错误", func(t *testing.T) {
		db := setupDistributorTestDB(t)
		distributorRepo := repository.NewDistributorRepository(db)
		userRepo := repository.NewUserRepository(db)
		svc := NewDistributorService(distributorRepo, userRepo, db)
		ctx := context.Background()

		req := &ApplyRequest{
			UserID:     999999,
			InviteCode: nil,
		}
		resp, err := svc.Apply(ctx, req)

		assert.Error(t, err)
		assert.Nil(t, resp)
		assert.Contains(t, err.Error(), "用户不存在")
	})

	t.Run("用户已是分销商_返回错误", func(t *testing.T) {
		db := setupDistributorTestDB(t)
		distributorRepo := repository.NewDistributorRepository(db)
		userRepo := repository.NewUserRepository(db)
		svc := NewDistributorService(distributorRepo, userRepo, db)
		ctx := context.Background()

		user := createDistributorTestUser(db, nil)
		// 用户已是分销商
		existingDistributor := &models.Distributor{
			UserID:     user.ID,
			Level:      models.DistributorLevelDirect,
			InviteCode: "EXIST001",
			Status:     models.DistributorStatusPending,
		}
		db.Create(existingDistributor)

		req := &ApplyRequest{
			UserID:     user.ID,
			InviteCode: nil,
		}
		resp, err := svc.Apply(ctx, req)

		assert.Error(t, err)
		assert.Nil(t, resp)
		assert.Contains(t, err.Error(), "您已经是分销商了")
	})

	t.Run("无效邀请码_返回错误", func(t *testing.T) {
		db := setupDistributorTestDB(t)
		distributorRepo := repository.NewDistributorRepository(db)
		userRepo := repository.NewUserRepository(db)
		svc := NewDistributorService(distributorRepo, userRepo, db)
		ctx := context.Background()

		user := createDistributorTestUser(db, nil)
		invalidCode := "INVALID1"

		req := &ApplyRequest{
			UserID:     user.ID,
			InviteCode: &invalidCode,
		}
		resp, err := svc.Apply(ctx, req)

		assert.Error(t, err)
		assert.Nil(t, resp)
		assert.Contains(t, err.Error(), "邀请码无效")
	})

	t.Run("邀请人未通过审核_返回错误", func(t *testing.T) {
		db := setupDistributorTestDB(t)
		distributorRepo := repository.NewDistributorRepository(db)
		userRepo := repository.NewUserRepository(db)
		svc := NewDistributorService(distributorRepo, userRepo, db)
		ctx := context.Background()

		// 创建待审核的上级分销商
		parentUser := createDistributorTestUser(db, nil)
		pendingDistributor := &models.Distributor{
			UserID:     parentUser.ID,
			Level:      models.DistributorLevelDirect,
			InviteCode: "PENDING1",
			Status:     models.DistributorStatusPending,
		}
		db.Create(pendingDistributor)

		user := createDistributorTestUser(db, nil)
		inviteCode := "PENDING1"

		req := &ApplyRequest{
			UserID:     user.ID,
			InviteCode: &inviteCode,
		}
		resp, err := svc.Apply(ctx, req)

		assert.Error(t, err)
		assert.Nil(t, resp)
		assert.Contains(t, err.Error(), "邀请人尚未通过审核")
	})

	t.Run("不能填写自己的邀请码_返回错误", func(t *testing.T) {
		db := setupDistributorTestDB(t)
		distributorRepo := repository.NewDistributorRepository(db)
		userRepo := repository.NewUserRepository(db)
		svc := NewDistributorService(distributorRepo, userRepo, db)
		ctx := context.Background()

		user := createDistributorTestUser(db, nil)
		// 创建分销商（用户自己）
		selfDistributor := &models.Distributor{
			UserID:     user.ID,
			Level:      models.DistributorLevelDirect,
			InviteCode: "SELF0001",
			Status:     models.DistributorStatusApproved,
		}
		db.Create(selfDistributor)

		// 删除分销商记录让用户可以重新申请
		db.Delete(selfDistributor)

		// 但是尝试用自己的邀请码申请
		inviteCode := "SELF0001"

		// 需要重新创建邀请码所属的分销商
		selfDistributor2 := &models.Distributor{
			UserID:     user.ID,
			Level:      models.DistributorLevelDirect,
			InviteCode: "SELF0001",
			Status:     models.DistributorStatusApproved,
		}
		db.Create(selfDistributor2)

		req := &ApplyRequest{
			UserID:     user.ID,
			InviteCode: &inviteCode,
		}
		resp, err := svc.Apply(ctx, req)

		// 因为用户已是分销商，应该返回"您已经是分销商了"错误
		assert.Error(t, err)
		assert.Nil(t, resp)
	})

	t.Run("用户有推荐人_自动关联推荐人为分销商上级", func(t *testing.T) {
		db := setupDistributorTestDB(t)
		distributorRepo := repository.NewDistributorRepository(db)
		userRepo := repository.NewUserRepository(db)
		svc := NewDistributorService(distributorRepo, userRepo, db)
		ctx := context.Background()

		// 创建推荐人并使其成为审核通过的分销商
		referrer := createDistributorTestUser(db, nil)
		referrerDistributor := &models.Distributor{
			UserID:     referrer.ID,
			Level:      models.DistributorLevelDirect,
			InviteCode: "REFERRER",
			Status:     models.DistributorStatusApproved,
		}
		db.Create(referrerDistributor)

		// 创建被推荐的用户
		user := createDistributorTestUser(db, &referrer.ID)

		req := &ApplyRequest{
			UserID:     user.ID,
			InviteCode: nil, // 不提供邀请码
		}
		resp, err := svc.Apply(ctx, req)

		require.NoError(t, err)
		assert.NotNil(t, resp)
		assert.NotNil(t, resp.Distributor.ParentID)
		assert.Equal(t, referrerDistributor.ID, *resp.Distributor.ParentID)
	})
}

func TestDistributorService_Approve(t *testing.T) {
	t.Run("审核通过分销商申请", func(t *testing.T) {
		db := setupDistributorTestDB(t)
		distributorRepo := repository.NewDistributorRepository(db)
		userRepo := repository.NewUserRepository(db)
		svc := NewDistributorService(distributorRepo, userRepo, db)
		ctx := context.Background()

		user := createDistributorTestUser(db, nil)
		distributor := &models.Distributor{
			UserID:     user.ID,
			Level:      models.DistributorLevelDirect,
			InviteCode: "PENDING2",
			Status:     models.DistributorStatusPending,
		}
		db.Create(distributor)

		req := &ApproveRequest{
			DistributorID: distributor.ID,
			OperatorID:    1,
			Approved:      true,
		}
		err := svc.Approve(ctx, req)

		require.NoError(t, err)

		// 验证状态
		var updated models.Distributor
		db.First(&updated, distributor.ID)
		assert.Equal(t, models.DistributorStatusApproved, updated.Status)
		assert.NotNil(t, updated.ApprovedAt)
		assert.NotNil(t, updated.ApprovedBy)
		assert.Equal(t, int64(1), *updated.ApprovedBy)
	})

	t.Run("审核通过_更新上级团队人数", func(t *testing.T) {
		db := setupDistributorTestDB(t)
		distributorRepo := repository.NewDistributorRepository(db)
		userRepo := repository.NewUserRepository(db)
		svc := NewDistributorService(distributorRepo, userRepo, db)
		ctx := context.Background()

		// 创建上级分销商
		parentUser := createDistributorTestUser(db, nil)
		parentDistributor := &models.Distributor{
			UserID:      parentUser.ID,
			Level:       models.DistributorLevelDirect,
			InviteCode:  "PARENT02",
			Status:      models.DistributorStatusApproved,
			TeamCount:   0,
			DirectCount: 0,
		}
		db.Create(parentDistributor)

		// 创建待审核分销商
		user := createDistributorTestUser(db, nil)
		distributor := &models.Distributor{
			UserID:     user.ID,
			ParentID:   &parentDistributor.ID,
			Level:      models.DistributorLevelIndirect,
			InviteCode: "CHILD001",
			Status:     models.DistributorStatusPending,
		}
		db.Create(distributor)

		req := &ApproveRequest{
			DistributorID: distributor.ID,
			OperatorID:    1,
			Approved:      true,
		}
		err := svc.Approve(ctx, req)

		require.NoError(t, err)

		// 验证上级团队人数增加
		var updatedParent models.Distributor
		db.First(&updatedParent, parentDistributor.ID)
		assert.Equal(t, 1, updatedParent.DirectCount)
		assert.Equal(t, 1, updatedParent.TeamCount)
	})

	t.Run("拒绝分销商申请", func(t *testing.T) {
		db := setupDistributorTestDB(t)
		distributorRepo := repository.NewDistributorRepository(db)
		userRepo := repository.NewUserRepository(db)
		svc := NewDistributorService(distributorRepo, userRepo, db)
		ctx := context.Background()

		user := createDistributorTestUser(db, nil)
		distributor := &models.Distributor{
			UserID:     user.ID,
			Level:      models.DistributorLevelDirect,
			InviteCode: "REJECT01",
			Status:     models.DistributorStatusPending,
		}
		db.Create(distributor)

		req := &ApproveRequest{
			DistributorID: distributor.ID,
			OperatorID:    1,
			Approved:      false,
			Reason:        "不符合条件",
		}
		err := svc.Approve(ctx, req)

		require.NoError(t, err)

		var updated models.Distributor
		db.First(&updated, distributor.ID)
		assert.Equal(t, models.DistributorStatusRejected, updated.Status)
	})

	t.Run("审核已处理的申请_返回错误", func(t *testing.T) {
		db := setupDistributorTestDB(t)
		distributorRepo := repository.NewDistributorRepository(db)
		userRepo := repository.NewUserRepository(db)
		svc := NewDistributorService(distributorRepo, userRepo, db)
		ctx := context.Background()

		user := createDistributorTestUser(db, nil)
		now := time.Now()
		opID := int64(1)
		distributor := &models.Distributor{
			UserID:     user.ID,
			Level:      models.DistributorLevelDirect,
			InviteCode: "DONE0001",
			Status:     models.DistributorStatusApproved,
			ApprovedAt: &now,
			ApprovedBy: &opID,
		}
		db.Create(distributor)

		req := &ApproveRequest{
			DistributorID: distributor.ID,
			OperatorID:    2,
			Approved:      true,
		}
		err := svc.Approve(ctx, req)

		assert.Error(t, err)
		assert.Contains(t, err.Error(), "该申请已处理")
	})

	t.Run("分销商不存在_返回错误", func(t *testing.T) {
		db := setupDistributorTestDB(t)
		distributorRepo := repository.NewDistributorRepository(db)
		userRepo := repository.NewUserRepository(db)
		svc := NewDistributorService(distributorRepo, userRepo, db)
		ctx := context.Background()

		req := &ApproveRequest{
			DistributorID: 999999,
			OperatorID:    1,
			Approved:      true,
		}
		err := svc.Approve(ctx, req)

		assert.Error(t, err)
		assert.Contains(t, err.Error(), "分销商不存在")
	})
}

func TestDistributorService_GetByUserID(t *testing.T) {
	t.Run("获取分销商信息", func(t *testing.T) {
		db := setupDistributorTestDB(t)
		distributorRepo := repository.NewDistributorRepository(db)
		userRepo := repository.NewUserRepository(db)
		svc := NewDistributorService(distributorRepo, userRepo, db)
		ctx := context.Background()

		user := createDistributorTestUser(db, nil)
		distributor := &models.Distributor{
			UserID:              user.ID,
			Level:               models.DistributorLevelDirect,
			InviteCode:          "GET00001",
			TotalCommission:     100.0,
			AvailableCommission: 50.0,
			Status:              models.DistributorStatusApproved,
		}
		db.Create(distributor)

		result, err := svc.GetByUserID(ctx, user.ID)

		require.NoError(t, err)
		assert.NotNil(t, result)
		assert.Equal(t, distributor.ID, result.ID)
		assert.Equal(t, user.ID, result.UserID)
	})

	t.Run("用户不是分销商_返回错误", func(t *testing.T) {
		db := setupDistributorTestDB(t)
		distributorRepo := repository.NewDistributorRepository(db)
		userRepo := repository.NewUserRepository(db)
		svc := NewDistributorService(distributorRepo, userRepo, db)
		ctx := context.Background()

		user := createDistributorTestUser(db, nil)

		result, err := svc.GetByUserID(ctx, user.ID)

		assert.Error(t, err)
		assert.Nil(t, result)
		assert.Contains(t, err.Error(), "您还不是分销商")
	})
}

func TestDistributorService_GetTeamStats(t *testing.T) {
	t.Run("获取团队统计", func(t *testing.T) {
		db := setupDistributorTestDB(t)
		distributorRepo := repository.NewDistributorRepository(db)
		userRepo := repository.NewUserRepository(db)
		svc := NewDistributorService(distributorRepo, userRepo, db)
		ctx := context.Background()

		user := createDistributorTestUser(db, nil)
		distributor := &models.Distributor{
			UserID:          user.ID,
			Level:           models.DistributorLevelDirect,
			InviteCode:      "STATS001",
			TotalCommission: 500.0,
			TeamCount:       10,
			DirectCount:     5,
			Status:          models.DistributorStatusApproved,
		}
		db.Create(distributor)

		stats, err := svc.GetTeamStats(ctx, distributor.ID)

		require.NoError(t, err)
		assert.NotNil(t, stats)
		assert.Equal(t, 10, stats.TeamCount)
		assert.Equal(t, 5, stats.DirectCount)
		assert.Equal(t, 5, stats.IndirectCount) // 10 - 5 = 5
		assert.Equal(t, 500.0, stats.TotalCommission)
	})
}

func TestDistributorService_GetDashboard(t *testing.T) {
	t.Run("获取仪表盘数据", func(t *testing.T) {
		db := setupDistributorTestDB(t)
		distributorRepo := repository.NewDistributorRepository(db)
		userRepo := repository.NewUserRepository(db)
		svc := NewDistributorService(distributorRepo, userRepo, db)
		ctx := context.Background()

		user := createDistributorTestUser(db, nil)
		distributor := &models.Distributor{
			UserID:              user.ID,
			Level:               models.DistributorLevelDirect,
			InviteCode:          "DASH0001",
			TotalCommission:     1000.0,
			AvailableCommission: 500.0,
			FrozenCommission:    100.0,
			WithdrawnCommission: 400.0,
			TeamCount:           20,
			DirectCount:         10,
			Status:              models.DistributorStatusApproved,
		}
		db.Create(distributor)

		data, err := svc.GetDashboard(ctx, distributor.ID)

		require.NoError(t, err)
		assert.NotNil(t, data)
		assert.Equal(t, 1000.0, data.TotalCommission)
		assert.Equal(t, 500.0, data.AvailableCommission)
		assert.Equal(t, 100.0, data.FrozenCommission)
		assert.Equal(t, 400.0, data.WithdrawnCommission)
		assert.Equal(t, 20, data.TeamCount)
		assert.Equal(t, 10, data.DirectCount)
		assert.Equal(t, "DASH0001", data.InviteCode)
		assert.Contains(t, data.InviteLink, "DASH0001")
	})
}

func TestDistributorService_CheckIsDistributor(t *testing.T) {
	t.Run("用户是分销商_返回true", func(t *testing.T) {
		db := setupDistributorTestDB(t)
		distributorRepo := repository.NewDistributorRepository(db)
		userRepo := repository.NewUserRepository(db)
		svc := NewDistributorService(distributorRepo, userRepo, db)
		ctx := context.Background()

		user := createDistributorTestUser(db, nil)
		distributor := &models.Distributor{
			UserID:     user.ID,
			Level:      models.DistributorLevelDirect,
			InviteCode: "CHECK001",
			Status:     models.DistributorStatusApproved,
		}
		db.Create(distributor)

		isDistributor, err := svc.CheckIsDistributor(ctx, user.ID)

		require.NoError(t, err)
		assert.True(t, isDistributor)
	})

	t.Run("用户不是分销商_返回false", func(t *testing.T) {
		db := setupDistributorTestDB(t)
		distributorRepo := repository.NewDistributorRepository(db)
		userRepo := repository.NewUserRepository(db)
		svc := NewDistributorService(distributorRepo, userRepo, db)
		ctx := context.Background()

		user := createDistributorTestUser(db, nil)

		isDistributor, err := svc.CheckIsDistributor(ctx, user.ID)

		require.NoError(t, err)
		assert.False(t, isDistributor)
	})
}

func TestDistributorService_GetApprovedDistributorByUserID(t *testing.T) {
	t.Run("获取已审核通过的分销商", func(t *testing.T) {
		db := setupDistributorTestDB(t)
		distributorRepo := repository.NewDistributorRepository(db)
		userRepo := repository.NewUserRepository(db)
		svc := NewDistributorService(distributorRepo, userRepo, db)
		ctx := context.Background()

		user := createDistributorTestUser(db, nil)
		distributor := &models.Distributor{
			UserID:     user.ID,
			Level:      models.DistributorLevelDirect,
			InviteCode: "APPROV01",
			Status:     models.DistributorStatusApproved,
		}
		db.Create(distributor)

		result, err := svc.GetApprovedDistributorByUserID(ctx, user.ID)

		require.NoError(t, err)
		assert.NotNil(t, result)
		assert.Equal(t, models.DistributorStatusApproved, result.Status)
	})

	t.Run("分销商未审核通过_返回错误", func(t *testing.T) {
		db := setupDistributorTestDB(t)
		distributorRepo := repository.NewDistributorRepository(db)
		userRepo := repository.NewUserRepository(db)
		svc := NewDistributorService(distributorRepo, userRepo, db)
		ctx := context.Background()

		user := createDistributorTestUser(db, nil)
		distributor := &models.Distributor{
			UserID:     user.ID,
			Level:      models.DistributorLevelDirect,
			InviteCode: "PEND0001",
			Status:     models.DistributorStatusPending,
		}
		db.Create(distributor)

		result, err := svc.GetApprovedDistributorByUserID(ctx, user.ID)

		assert.Error(t, err)
		assert.Nil(t, result)
		assert.Contains(t, err.Error(), "分销商尚未审核通过")
	})
}
