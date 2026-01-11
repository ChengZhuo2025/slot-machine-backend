package admin

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

func setupDistributionAdminTestDB(t *testing.T) *gorm.DB {
	t.Helper()

	dsn := fmt.Sprintf("file:%s?mode=memory&cache=shared", strings.ReplaceAll(t.Name(), "/", "_"))
	db, err := gorm.Open(sqlite.Open(dsn), &gorm.Config{Logger: logger.Default.LogMode(logger.Silent)})
	require.NoError(t, err)

	sqlDB, err := db.DB()
	require.NoError(t, err)
	sqlDB.SetMaxOpenConns(1)
	sqlDB.SetMaxIdleConns(1)

	require.NoError(t, db.AutoMigrate(
		&models.User{},
		&models.UserWallet{},
		&models.MemberLevel{},
		&models.Distributor{},
		&models.Commission{},
		&models.Withdrawal{},
	))

	// 创建默认会员等级
	level := &models.MemberLevel{ID: 1, Name: "普通会员", Level: 1, MinPoints: 0, Discount: 1.0}
	db.Create(level)

	return db
}

func createDistributionAdminTestUser(db *gorm.DB) *models.User {
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

func TestDistributionAdminService_NewService(t *testing.T) {
	db := setupDistributionAdminTestDB(t)
	distributorRepo := repository.NewDistributorRepository(db)
	commissionRepo := repository.NewCommissionRepository(db)
	withdrawalRepo := repository.NewWithdrawalRepository(db)

	svc := NewDistributionAdminService(distributorRepo, commissionRepo, withdrawalRepo, db)
	assert.NotNil(t, svc)
}

func TestDistributionAdminService_ListDistributors(t *testing.T) {
	db := setupDistributionAdminTestDB(t)
	distributorRepo := repository.NewDistributorRepository(db)
	commissionRepo := repository.NewCommissionRepository(db)
	withdrawalRepo := repository.NewWithdrawalRepository(db)
	svc := NewDistributionAdminService(distributorRepo, commissionRepo, withdrawalRepo, db)
	ctx := context.Background()

	// 创建分销商
	for i := 0; i < 3; i++ {
		user := createDistributionAdminTestUser(db)
		distributor := &models.Distributor{
			UserID:     user.ID,
			Level:      models.DistributorLevelDirect,
			InviteCode: fmt.Sprintf("ADMIN%03d", i),
			Status:     models.DistributorStatusApproved,
		}
		db.Create(distributor)
	}

	list, total, err := svc.ListDistributors(ctx, 0, 10, nil)
	require.NoError(t, err)
	assert.Equal(t, int64(3), total)
	assert.Len(t, list, 3)
}

func TestDistributionAdminService_ListDistributors_WithFilter(t *testing.T) {
	db := setupDistributionAdminTestDB(t)
	distributorRepo := repository.NewDistributorRepository(db)
	commissionRepo := repository.NewCommissionRepository(db)
	withdrawalRepo := repository.NewWithdrawalRepository(db)
	svc := NewDistributionAdminService(distributorRepo, commissionRepo, withdrawalRepo, db)
	ctx := context.Background()

	user := createDistributionAdminTestUser(db)
	distributor := &models.Distributor{
		UserID:     user.ID,
		Level:      models.DistributorLevelDirect,
		InviteCode: "FILTER01",
		Status:     models.DistributorStatusPending,
	}
	db.Create(distributor)

	status := int(1) // pending
	filter := &DistributorListFilter{Status: &status}
	list, _, err := svc.ListDistributors(ctx, 0, 10, filter)
	require.NoError(t, err)
	assert.NotNil(t, list)
}

func TestDistributionAdminService_GetDistributor(t *testing.T) {
	db := setupDistributionAdminTestDB(t)
	distributorRepo := repository.NewDistributorRepository(db)
	commissionRepo := repository.NewCommissionRepository(db)
	withdrawalRepo := repository.NewWithdrawalRepository(db)
	svc := NewDistributionAdminService(distributorRepo, commissionRepo, withdrawalRepo, db)
	ctx := context.Background()

	user := createDistributionAdminTestUser(db)
	distributor := &models.Distributor{
		UserID:     user.ID,
		Level:      models.DistributorLevelDirect,
		InviteCode: "GETDIST1",
		Status:     models.DistributorStatusApproved,
	}
	db.Create(distributor)

	result, err := svc.GetDistributor(ctx, distributor.ID)
	require.NoError(t, err)
	assert.NotNil(t, result)
	assert.Equal(t, distributor.ID, result.ID)
}

func TestDistributionAdminService_ApproveDistributor(t *testing.T) {
	db := setupDistributionAdminTestDB(t)
	distributorRepo := repository.NewDistributorRepository(db)
	commissionRepo := repository.NewCommissionRepository(db)
	withdrawalRepo := repository.NewWithdrawalRepository(db)
	svc := NewDistributionAdminService(distributorRepo, commissionRepo, withdrawalRepo, db)
	ctx := context.Background()

	user := createDistributionAdminTestUser(db)
	distributor := &models.Distributor{
		UserID:     user.ID,
		Level:      models.DistributorLevelDirect,
		InviteCode: "APPROVE1",
		Status:     models.DistributorStatusPending,
	}
	db.Create(distributor)

	err := svc.ApproveDistributor(ctx, distributor.ID, 1)
	require.NoError(t, err)

	var updated models.Distributor
	db.First(&updated, distributor.ID)
	assert.Equal(t, models.DistributorStatusApproved, updated.Status)
}

func TestDistributionAdminService_ApproveDistributor_WithParent(t *testing.T) {
	db := setupDistributionAdminTestDB(t)
	distributorRepo := repository.NewDistributorRepository(db)
	commissionRepo := repository.NewCommissionRepository(db)
	withdrawalRepo := repository.NewWithdrawalRepository(db)
	svc := NewDistributionAdminService(distributorRepo, commissionRepo, withdrawalRepo, db)
	ctx := context.Background()

	// 创建父级分销商
	parentUser := createDistributionAdminTestUser(db)
	parentDistributor := &models.Distributor{
		UserID:      parentUser.ID,
		Level:       models.DistributorLevelDirect,
		InviteCode:  "PARENT01",
		Status:      models.DistributorStatusApproved,
		TeamCount:   0,
		DirectCount: 0,
	}
	db.Create(parentDistributor)

	// 创建子分销商
	childUser := createDistributionAdminTestUser(db)
	childDistributor := &models.Distributor{
		UserID:     childUser.ID,
		ParentID:   &parentDistributor.ID,
		Level:      models.DistributorLevelIndirect,
		InviteCode: "CHILD001",
		Status:     models.DistributorStatusPending,
	}
	db.Create(childDistributor)

	err := svc.ApproveDistributor(ctx, childDistributor.ID, 1)
	require.NoError(t, err)

	// 验证父级团队人数增加
	var updatedParent models.Distributor
	db.First(&updatedParent, parentDistributor.ID)
	assert.Equal(t, 1, updatedParent.DirectCount)
	assert.Equal(t, 1, updatedParent.TeamCount)
}

func TestDistributionAdminService_RejectDistributor(t *testing.T) {
	db := setupDistributionAdminTestDB(t)
	distributorRepo := repository.NewDistributorRepository(db)
	commissionRepo := repository.NewCommissionRepository(db)
	withdrawalRepo := repository.NewWithdrawalRepository(db)
	svc := NewDistributionAdminService(distributorRepo, commissionRepo, withdrawalRepo, db)
	ctx := context.Background()

	user := createDistributionAdminTestUser(db)
	distributor := &models.Distributor{
		UserID:     user.ID,
		Level:      models.DistributorLevelDirect,
		InviteCode: "REJECT01",
		Status:     models.DistributorStatusPending,
	}
	db.Create(distributor)

	err := svc.RejectDistributor(ctx, distributor.ID, 1, "不符合条件")
	require.NoError(t, err)

	var updated models.Distributor
	db.First(&updated, distributor.ID)
	assert.Equal(t, models.DistributorStatusRejected, updated.Status)
}

func TestDistributionAdminService_GetPendingDistributors(t *testing.T) {
	db := setupDistributionAdminTestDB(t)
	distributorRepo := repository.NewDistributorRepository(db)
	commissionRepo := repository.NewCommissionRepository(db)
	withdrawalRepo := repository.NewWithdrawalRepository(db)
	svc := NewDistributionAdminService(distributorRepo, commissionRepo, withdrawalRepo, db)
	ctx := context.Background()

	// 创建待审核分销商
	for i := 0; i < 2; i++ {
		user := createDistributionAdminTestUser(db)
		distributor := &models.Distributor{
			UserID:     user.ID,
			Level:      models.DistributorLevelDirect,
			InviteCode: fmt.Sprintf("PEND%04d", i),
			Status:     models.DistributorStatusPending,
		}
		db.Create(distributor)
	}

	list, total, err := svc.GetPendingDistributors(ctx, 0, 10)
	require.NoError(t, err)
	assert.Equal(t, int64(2), total)
	assert.Len(t, list, 2)
}

func TestDistributionAdminService_ListCommissions(t *testing.T) {
	db := setupDistributionAdminTestDB(t)
	distributorRepo := repository.NewDistributorRepository(db)
	commissionRepo := repository.NewCommissionRepository(db)
	withdrawalRepo := repository.NewWithdrawalRepository(db)
	svc := NewDistributionAdminService(distributorRepo, commissionRepo, withdrawalRepo, db)
	ctx := context.Background()

	user := createDistributionAdminTestUser(db)
	distributor := &models.Distributor{
		UserID:     user.ID,
		Level:      models.DistributorLevelDirect,
		InviteCode: "COMM0001",
		Status:     models.DistributorStatusApproved,
	}
	db.Create(distributor)

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

	list, total, err := svc.ListCommissions(ctx, 0, 10, nil)
	require.NoError(t, err)
	assert.Equal(t, int64(1), total)
	assert.Len(t, list, 1)
}

func TestDistributionAdminService_ListWithdrawals(t *testing.T) {
	db := setupDistributionAdminTestDB(t)
	distributorRepo := repository.NewDistributorRepository(db)
	commissionRepo := repository.NewCommissionRepository(db)
	withdrawalRepo := repository.NewWithdrawalRepository(db)
	svc := NewDistributionAdminService(distributorRepo, commissionRepo, withdrawalRepo, db)
	ctx := context.Background()

	user := createDistributionAdminTestUser(db)
	withdrawal := &models.Withdrawal{
		WithdrawalNo: fmt.Sprintf("W%d", time.Now().UnixNano()),
		UserID:       user.ID,
		Type:         models.WithdrawalTypeCommission,
		Amount:       50.0,
		Fee:          0.3,
		ActualAmount: 49.7,
		WithdrawTo:   models.WithdrawToWechat,
		Status:       models.WithdrawalStatusPending,
	}
	db.Create(withdrawal)

	list, total, err := svc.ListWithdrawals(ctx, 0, 10, nil)
	require.NoError(t, err)
	assert.Equal(t, int64(1), total)
	assert.Len(t, list, 1)
}

func TestDistributionAdminService_GetStats(t *testing.T) {
	db := setupDistributionAdminTestDB(t)
	distributorRepo := repository.NewDistributorRepository(db)
	commissionRepo := repository.NewCommissionRepository(db)
	withdrawalRepo := repository.NewWithdrawalRepository(db)
	svc := NewDistributionAdminService(distributorRepo, commissionRepo, withdrawalRepo, db)
	ctx := context.Background()

	user := createDistributionAdminTestUser(db)
	distributor := &models.Distributor{
		UserID:     user.ID,
		Level:      models.DistributorLevelDirect,
		InviteCode: "STATS001",
		Status:     models.DistributorStatusApproved,
	}
	db.Create(distributor)

	stats, err := svc.GetStats(ctx)
	require.NoError(t, err)
	assert.NotNil(t, stats)
	assert.Equal(t, int64(1), stats.TotalDistributors)
}

func TestDistributionAdminService_GetPendingWithdrawals(t *testing.T) {
	db := setupDistributionAdminTestDB(t)
	distributorRepo := repository.NewDistributorRepository(db)
	commissionRepo := repository.NewCommissionRepository(db)
	withdrawalRepo := repository.NewWithdrawalRepository(db)
	svc := NewDistributionAdminService(distributorRepo, commissionRepo, withdrawalRepo, db)
	ctx := context.Background()

	user := createDistributionAdminTestUser(db)
	withdrawal := &models.Withdrawal{
		WithdrawalNo: fmt.Sprintf("WP%d", time.Now().UnixNano()),
		UserID:       user.ID,
		Type:         models.WithdrawalTypeCommission,
		Amount:       50.0,
		Fee:          0.3,
		ActualAmount: 49.7,
		WithdrawTo:   models.WithdrawToWechat,
		Status:       models.WithdrawalStatusPending,
	}
	db.Create(withdrawal)

	list, total, err := svc.GetPendingWithdrawals(ctx, 0, 10)
	require.NoError(t, err)
	assert.Equal(t, int64(1), total)
	assert.Len(t, list, 1)
}

func TestDistributionAdminService_GetApprovedWithdrawals(t *testing.T) {
	db := setupDistributionAdminTestDB(t)
	distributorRepo := repository.NewDistributorRepository(db)
	commissionRepo := repository.NewCommissionRepository(db)
	withdrawalRepo := repository.NewWithdrawalRepository(db)
	svc := NewDistributionAdminService(distributorRepo, commissionRepo, withdrawalRepo, db)
	ctx := context.Background()

	user := createDistributionAdminTestUser(db)
	withdrawal := &models.Withdrawal{
		WithdrawalNo: fmt.Sprintf("WA%d", time.Now().UnixNano()),
		UserID:       user.ID,
		Type:         models.WithdrawalTypeCommission,
		Amount:       50.0,
		Fee:          0.3,
		ActualAmount: 49.7,
		WithdrawTo:   models.WithdrawToWechat,
		Status:       models.WithdrawalStatusApproved,
	}
	db.Create(withdrawal)

	list, total, err := svc.GetApprovedWithdrawals(ctx, 0, 10)
	require.NoError(t, err)
	assert.Equal(t, int64(1), total)
	assert.Len(t, list, 1)
}
