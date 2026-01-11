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
)

// setupDashboardTestDB 创建仪表盘测试数据库
func setupDashboardTestDB(t *testing.T) *gorm.DB {
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
		&models.Commission{},
		&models.Order{},
		&models.Withdrawal{},
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

// createDashboardTestUser 创建仪表盘测试用户
func createDashboardTestUser(db *gorm.DB) *models.User {
	phone := fmt.Sprintf("138%08d", time.Now().UnixNano()%100000000)
	user := &models.User{
		Phone:         &phone,
		Nickname:      "测试用户",
		MemberLevelID: 1,
		Status:        models.UserStatusActive,
	}
	db.Create(user)
	return user
}

// createDashboardTestDistributor 创建仪表盘测试分销商
func createDashboardTestDistributor(db *gorm.DB, userID int64, parentID *int64) *models.Distributor {
	distributor := &models.Distributor{
		UserID:              userID,
		ParentID:            parentID,
		Level:               models.DistributorLevelDirect,
		InviteCode:          fmt.Sprintf("D%d", time.Now().UnixNano()%1000000),
		TotalCommission:     100.0,
		AvailableCommission: 80.0,
		FrozenCommission:    10.0,
		WithdrawnCommission: 10.0,
		TeamCount:           5,
		DirectCount:         3,
		Status:              models.DistributorStatusApproved,
	}
	if parentID != nil {
		distributor.Level = models.DistributorLevelIndirect
	}
	db.Create(distributor)
	return distributor
}

func TestDashboardService_NewDashboardService(t *testing.T) {
	db := setupDashboardTestDB(t)
	svc := NewDashboardService(db)
	assert.NotNil(t, svc)
}

func TestDashboardService_GetDistributorOverview(t *testing.T) {
	t.Run("获取分销商概览数据", func(t *testing.T) {
		db := setupDashboardTestDB(t)
		svc := NewDashboardService(db)
		ctx := context.Background()

		user := createDashboardTestUser(db)
		distributor := createDashboardTestDistributor(db, user.ID, nil)

		// 创建一些佣金记录
		commission := &models.Commission{
			DistributorID: distributor.ID,
			OrderID:       1,
			FromUserID:    2,
			Type:          models.CommissionTypeDirect,
			OrderAmount:   100.0,
			Rate:          0.1,
			Amount:        10.0,
			Status:        models.CommissionStatusSettled,
		}
		db.Create(commission)

		overview, err := svc.GetDistributorOverview(ctx, distributor.ID)
		require.NoError(t, err)
		assert.NotNil(t, overview)
		assert.Equal(t, 100.0, overview.TotalCommission)
		assert.Equal(t, 80.0, overview.AvailableCommission)
		assert.Equal(t, 10.0, overview.FrozenCommission)
		assert.Equal(t, 10.0, overview.WithdrawnCommission)
		assert.Equal(t, int64(5), overview.TeamCount)
		assert.Equal(t, int64(3), overview.DirectCount)
	})

	t.Run("分销商不存在_返回错误", func(t *testing.T) {
		db := setupDashboardTestDB(t)
		svc := NewDashboardService(db)
		ctx := context.Background()

		overview, err := svc.GetDistributorOverview(ctx, 999999)
		assert.Error(t, err)
		assert.Nil(t, overview)
	})
}

func TestDashboardService_GetCommissionTrend(t *testing.T) {
	t.Run("获取7天佣金趋势", func(t *testing.T) {
		db := setupDashboardTestDB(t)
		svc := NewDashboardService(db)
		ctx := context.Background()

		user := createDashboardTestUser(db)
		distributor := createDashboardTestDistributor(db, user.ID, nil)

		// 创建佣金记录
		commission := &models.Commission{
			DistributorID: distributor.ID,
			OrderID:       1,
			FromUserID:    2,
			Type:          models.CommissionTypeDirect,
			OrderAmount:   100.0,
			Rate:          0.1,
			Amount:        10.0,
			Status:        models.CommissionStatusSettled,
		}
		db.Create(commission)

		trends, err := svc.GetCommissionTrend(ctx, distributor.ID, 7)
		require.NoError(t, err)
		assert.Len(t, trends, 7)
		// 验证日期格式
		for _, trend := range trends {
			assert.NotEmpty(t, trend.Date)
		}
	})

	t.Run("days小于等于0时默认7天", func(t *testing.T) {
		db := setupDashboardTestDB(t)
		svc := NewDashboardService(db)
		ctx := context.Background()

		user := createDashboardTestUser(db)
		distributor := createDashboardTestDistributor(db, user.ID, nil)

		trends, err := svc.GetCommissionTrend(ctx, distributor.ID, 0)
		require.NoError(t, err)
		assert.Len(t, trends, 7)
	})

	t.Run("days超过30时限制为30天", func(t *testing.T) {
		db := setupDashboardTestDB(t)
		svc := NewDashboardService(db)
		ctx := context.Background()

		user := createDashboardTestUser(db)
		distributor := createDashboardTestDistributor(db, user.ID, nil)

		trends, err := svc.GetCommissionTrend(ctx, distributor.ID, 100)
		require.NoError(t, err)
		assert.Len(t, trends, 30)
	})
}

func TestDashboardService_GetTeamRank(t *testing.T) {
	t.Run("获取团队成员排行", func(t *testing.T) {
		db := setupDashboardTestDB(t)
		svc := NewDashboardService(db)
		ctx := context.Background()

		// 创建父级分销商
		parentUser := createDashboardTestUser(db)
		parentDistributor := createDashboardTestDistributor(db, parentUser.ID, nil)

		// 创建下级分销商
		for i := 0; i < 3; i++ {
			childUser := createDashboardTestUser(db)
			createDashboardTestDistributor(db, childUser.ID, &parentDistributor.ID)

			// 创建佣金记录
			commission := &models.Commission{
				DistributorID: parentDistributor.ID,
				OrderID:       int64(i + 1),
				FromUserID:    childUser.ID,
				Type:          models.CommissionTypeDirect,
				OrderAmount:   float64((i + 1) * 100),
				Rate:          0.1,
				Amount:        float64((i + 1) * 10),
				Status:        models.CommissionStatusSettled,
			}
			db.Create(commission)
		}

		rank, err := svc.GetTeamRank(ctx, parentDistributor.ID, 10, nil, nil)
		require.NoError(t, err)
		assert.NotNil(t, rank)
	})

	t.Run("limit小于等于0时默认10", func(t *testing.T) {
		db := setupDashboardTestDB(t)
		svc := NewDashboardService(db)
		ctx := context.Background()

		user := createDashboardTestUser(db)
		distributor := createDashboardTestDistributor(db, user.ID, nil)

		rank, err := svc.GetTeamRank(ctx, distributor.ID, 0, nil, nil)
		require.NoError(t, err)
		assert.NotNil(t, rank)
	})

	t.Run("带时间范围筛选", func(t *testing.T) {
		db := setupDashboardTestDB(t)
		svc := NewDashboardService(db)
		ctx := context.Background()

		user := createDashboardTestUser(db)
		distributor := createDashboardTestDistributor(db, user.ID, nil)

		startDate := time.Now().AddDate(0, 0, -7)
		endDate := time.Now()
		rank, err := svc.GetTeamRank(ctx, distributor.ID, 10, &startDate, &endDate)
		require.NoError(t, err)
		assert.NotNil(t, rank)
	})
}

func TestDashboardService_GetRecentCommissions(t *testing.T) {
	t.Run("获取最近佣金记录", func(t *testing.T) {
		db := setupDashboardTestDB(t)
		svc := NewDashboardService(db)
		ctx := context.Background()

		user := createDashboardTestUser(db)
		distributor := createDashboardTestDistributor(db, user.ID, nil)

		// 创建佣金记录
		for i := 0; i < 5; i++ {
			fromUser := createDashboardTestUser(db)
			order := &models.Order{
				OrderNo:        fmt.Sprintf("O%d%d", time.Now().UnixNano(), i),
				UserID:         fromUser.ID,
				Type:           models.OrderTypeMall,
				OriginalAmount: 100.0,
				ActualAmount:   100.0,
				Status:         models.OrderStatusCompleted,
			}
			db.Create(order)

			commission := &models.Commission{
				DistributorID: distributor.ID,
				OrderID:       order.ID,
				FromUserID:    fromUser.ID,
				Type:          models.CommissionTypeDirect,
				OrderAmount:   100.0,
				Rate:          0.1,
				Amount:        10.0,
				Status:        models.CommissionStatusSettled,
			}
			db.Create(commission)
		}

		records, err := svc.GetRecentCommissions(ctx, distributor.ID, 10)
		require.NoError(t, err)
		assert.Len(t, records, 5)
	})

	t.Run("limit小于等于0时默认10", func(t *testing.T) {
		db := setupDashboardTestDB(t)
		svc := NewDashboardService(db)
		ctx := context.Background()

		user := createDashboardTestUser(db)
		distributor := createDashboardTestDistributor(db, user.ID, nil)

		records, err := svc.GetRecentCommissions(ctx, distributor.ID, 0)
		require.NoError(t, err)
		assert.NotNil(t, records)
	})
}

func TestDashboardService_GetRecentWithdrawals(t *testing.T) {
	t.Run("获取最近提现记录", func(t *testing.T) {
		db := setupDashboardTestDB(t)
		svc := NewDashboardService(db)
		ctx := context.Background()

		user := createDashboardTestUser(db)

		// 创建提现记录
		for i := 0; i < 3; i++ {
			withdrawal := &models.Withdrawal{
				WithdrawalNo: fmt.Sprintf("W%d%d", time.Now().UnixNano(), i),
				UserID:       user.ID,
				Type:         models.WithdrawalTypeCommission,
				Amount:       50.0,
				Fee:          0.3,
				ActualAmount: 49.7,
				WithdrawTo:   models.WithdrawToWechat,
				Status:       models.WithdrawalStatusSuccess,
			}
			db.Create(withdrawal)
		}

		records, err := svc.GetRecentWithdrawals(ctx, user.ID, 10)
		require.NoError(t, err)
		assert.Len(t, records, 3)
	})

	t.Run("limit小于等于0时默认10", func(t *testing.T) {
		db := setupDashboardTestDB(t)
		svc := NewDashboardService(db)
		ctx := context.Background()

		user := createDashboardTestUser(db)

		records, err := svc.GetRecentWithdrawals(ctx, user.ID, 0)
		require.NoError(t, err)
		assert.NotNil(t, records)
	})
}

func TestDashboardService_GetCommissionTypeSummary(t *testing.T) {
	t.Run("获取佣金类型汇总", func(t *testing.T) {
		db := setupDashboardTestDB(t)
		svc := NewDashboardService(db)
		ctx := context.Background()

		user := createDashboardTestUser(db)
		distributor := createDashboardTestDistributor(db, user.ID, nil)

		// 创建不同类型的佣金记录
		directCommission := &models.Commission{
			DistributorID: distributor.ID,
			OrderID:       1,
			FromUserID:    2,
			Type:          models.CommissionTypeDirect,
			OrderAmount:   100.0,
			Rate:          0.1,
			Amount:        10.0,
			Status:        models.CommissionStatusSettled,
		}
		db.Create(directCommission)

		indirectCommission := &models.Commission{
			DistributorID: distributor.ID,
			OrderID:       2,
			FromUserID:    3,
			Type:          models.CommissionTypeIndirect,
			OrderAmount:   100.0,
			Rate:          0.05,
			Amount:        5.0,
			Status:        models.CommissionStatusSettled,
		}
		db.Create(indirectCommission)

		summary, err := svc.GetCommissionTypeSummary(ctx, distributor.ID, nil, nil)
		require.NoError(t, err)
		assert.Len(t, summary, 2)
	})

	t.Run("带时间范围筛选", func(t *testing.T) {
		db := setupDashboardTestDB(t)
		svc := NewDashboardService(db)
		ctx := context.Background()

		user := createDashboardTestUser(db)
		distributor := createDashboardTestDistributor(db, user.ID, nil)

		startDate := time.Now().AddDate(0, 0, -7)
		endDate := time.Now()
		summary, err := svc.GetCommissionTypeSummary(ctx, distributor.ID, &startDate, &endDate)
		require.NoError(t, err)
		assert.NotNil(t, summary)
	})
}
