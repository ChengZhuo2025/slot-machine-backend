// Package repository 提现仓储单元测试
package repository

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"

	"github.com/dumeirei/smart-locker-backend/internal/models"
)

func setupWithdrawalTestDB(t *testing.T) *gorm.DB {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Silent),
	})
	require.NoError(t, err)

	err = db.AutoMigrate(&models.Withdrawal{}, &models.User{}, &models.Admin{})
	require.NoError(t, err)

	return db
}

func TestWithdrawalRepository_Create(t *testing.T) {
	db := setupWithdrawalTestDB(t)
	repo := NewWithdrawalRepository(db)
	ctx := context.Background()

	withdrawal := &models.Withdrawal{
		WithdrawalNo:         "WD202601001",
		UserID:               1,
		Type:                 "wallet",
		Amount:               100.0,
		Fee:                  5.0,
		ActualAmount:         95.0,
		WithdrawTo:           "wechat",
		AccountInfoEncrypted: "encrypted_info",
		Status:               models.WithdrawalStatusPending,
	}

	err := repo.Create(ctx, withdrawal)
	require.NoError(t, err)
	assert.NotZero(t, withdrawal.ID)
}

func TestWithdrawalRepository_GetByID(t *testing.T) {
	db := setupWithdrawalTestDB(t)
	repo := NewWithdrawalRepository(db)
	ctx := context.Background()

	withdrawal := &models.Withdrawal{
		WithdrawalNo:         "WD202601001",
		UserID:               1,
		Type:                 "wallet",
		Amount:               100.0,
		Fee:                  5.0,
		ActualAmount:         95.0,
		WithdrawTo:           "wechat",
		AccountInfoEncrypted: "encrypted_info",
		Status:               models.WithdrawalStatusPending,
	}
	db.Create(withdrawal)

	found, err := repo.GetByID(ctx, withdrawal.ID)
	require.NoError(t, err)
	assert.Equal(t, withdrawal.ID, found.ID)
}

func TestWithdrawalRepository_GetByIDWithRelations(t *testing.T) {
	db := setupWithdrawalTestDB(t)
	repo := NewWithdrawalRepository(db)
	ctx := context.Background()

	phone := "13800138000"
	user := &models.User{
		Phone: &phone,
	}
	db.Create(user)

	withdrawal := &models.Withdrawal{
		WithdrawalNo:         "WD202601001",
		UserID:               user.ID,
		Type:                 "wallet",
		Amount:               100.0,
		Fee:                  5.0,
		ActualAmount:         95.0,
		WithdrawTo:           "wechat",
		AccountInfoEncrypted: "encrypted_info",
		Status:               models.WithdrawalStatusPending,
	}
	db.Create(withdrawal)

	found, err := repo.GetByIDWithRelations(ctx, withdrawal.ID)
	require.NoError(t, err)
	assert.Equal(t, withdrawal.ID, found.ID)
	assert.NotNil(t, found.User)
	assert.Equal(t, user.ID, found.User.ID)
}

func TestWithdrawalRepository_GetByWithdrawalNo(t *testing.T) {
	db := setupWithdrawalTestDB(t)
	repo := NewWithdrawalRepository(db)
	ctx := context.Background()

	db.Create(&models.Withdrawal{
		WithdrawalNo:         "WD202601001",
		UserID:               1,
		Type:                 "wallet",
		Amount:               100.0,
		Fee:                  5.0,
		ActualAmount:         95.0,
		WithdrawTo:           "wechat",
		AccountInfoEncrypted: "encrypted_info",
		Status:               models.WithdrawalStatusPending,
	})

	found, err := repo.GetByWithdrawalNo(ctx, "WD202601001")
	require.NoError(t, err)
	assert.Equal(t, "WD202601001", found.WithdrawalNo)
}

func TestWithdrawalRepository_GetByUserID(t *testing.T) {
	db := setupWithdrawalTestDB(t)
	repo := NewWithdrawalRepository(db)
	ctx := context.Background()

	db.Create(&models.Withdrawal{
		WithdrawalNo: "WD001", UserID: 1, Type: "wallet", Amount: 100.0, Fee: 5.0, ActualAmount: 95.0,
		WithdrawTo: "wechat", AccountInfoEncrypted: "info1", Status: models.WithdrawalStatusPending,
	})

	db.Create(&models.Withdrawal{
		WithdrawalNo: "WD002", UserID: 1, Type: "wallet", Amount: 200.0, Fee: 10.0, ActualAmount: 190.0,
		WithdrawTo: "alipay", AccountInfoEncrypted: "info2", Status: models.WithdrawalStatusSuccess,
	})

	db.Create(&models.Withdrawal{
		WithdrawalNo: "WD003", UserID: 2, Type: "wallet", Amount: 150.0, Fee: 7.5, ActualAmount: 142.5,
		WithdrawTo: "wechat", AccountInfoEncrypted: "info3", Status: models.WithdrawalStatusPending,
	})

	_, total, err := repo.GetByUserID(ctx, 1, 0, 10)
	require.NoError(t, err)
	assert.Equal(t, int64(2), total)
}

func TestWithdrawalRepository_Update(t *testing.T) {
	db := setupWithdrawalTestDB(t)
	repo := NewWithdrawalRepository(db)
	ctx := context.Background()

	withdrawal := &models.Withdrawal{
		WithdrawalNo: "WD001", UserID: 1, Type: "wallet", Amount: 100.0, Fee: 5.0, ActualAmount: 95.0,
		WithdrawTo: "wechat", AccountInfoEncrypted: "info1", Status: models.WithdrawalStatusPending,
	}
	db.Create(withdrawal)

	withdrawal.Status = models.WithdrawalStatusApproved
	err := repo.Update(ctx, withdrawal)
	require.NoError(t, err)

	var found models.Withdrawal
	db.First(&found, withdrawal.ID)
	assert.Equal(t, models.WithdrawalStatusApproved, found.Status)
}

func TestWithdrawalRepository_UpdateStatus(t *testing.T) {
	db := setupWithdrawalTestDB(t)
	repo := NewWithdrawalRepository(db)
	ctx := context.Background()

	withdrawal := &models.Withdrawal{
		WithdrawalNo: "WD001", UserID: 1, Type: "wallet", Amount: 100.0, Fee: 5.0, ActualAmount: 95.0,
		WithdrawTo: "wechat", AccountInfoEncrypted: "info1", Status: models.WithdrawalStatusPending,
	}
	db.Create(withdrawal)

	operatorID := int64(100)
	err := repo.UpdateStatus(ctx, withdrawal.ID, models.WithdrawalStatusApproved, &operatorID)
	require.NoError(t, err)

	var found models.Withdrawal
	db.First(&found, withdrawal.ID)
	assert.Equal(t, models.WithdrawalStatusApproved, found.Status)
	assert.NotNil(t, found.OperatorID)
	assert.Equal(t, int64(100), *found.OperatorID)
	assert.NotNil(t, found.ProcessedAt)
}

func TestWithdrawalRepository_Approve(t *testing.T) {
	db := setupWithdrawalTestDB(t)
	repo := NewWithdrawalRepository(db)
	ctx := context.Background()

	withdrawal := &models.Withdrawal{
		WithdrawalNo: "WD001", UserID: 1, Type: "wallet", Amount: 100.0, Fee: 5.0, ActualAmount: 95.0,
		WithdrawTo: "wechat", AccountInfoEncrypted: "info1", Status: models.WithdrawalStatusPending,
	}
	db.Create(withdrawal)

	err := repo.Approve(ctx, withdrawal.ID, 100)
	require.NoError(t, err)

	var found models.Withdrawal
	db.First(&found, withdrawal.ID)
	assert.Equal(t, models.WithdrawalStatusApproved, found.Status)
	assert.NotNil(t, found.OperatorID)
	assert.Equal(t, int64(100), *found.OperatorID)
	assert.NotNil(t, found.ProcessedAt)
}

func TestWithdrawalRepository_Reject(t *testing.T) {
	db := setupWithdrawalTestDB(t)
	repo := NewWithdrawalRepository(db)
	ctx := context.Background()

	withdrawal := &models.Withdrawal{
		WithdrawalNo: "WD001", UserID: 1, Type: "wallet", Amount: 100.0, Fee: 5.0, ActualAmount: 95.0,
		WithdrawTo: "wechat", AccountInfoEncrypted: "info1", Status: models.WithdrawalStatusPending,
	}
	db.Create(withdrawal)

	err := repo.Reject(ctx, withdrawal.ID, 100, "账户信息错误")
	require.NoError(t, err)

	var found models.Withdrawal
	db.First(&found, withdrawal.ID)
	assert.Equal(t, models.WithdrawalStatusRejected, found.Status)
	assert.NotNil(t, found.OperatorID)
	assert.Equal(t, int64(100), *found.OperatorID)
	assert.NotNil(t, found.RejectReason)
	assert.Equal(t, "账户信息错误", *found.RejectReason)
	assert.NotNil(t, found.ProcessedAt)
}

func TestWithdrawalRepository_MarkProcessing(t *testing.T) {
	db := setupWithdrawalTestDB(t)
	repo := NewWithdrawalRepository(db)
	ctx := context.Background()

	withdrawal := &models.Withdrawal{
		WithdrawalNo: "WD001", UserID: 1, Type: "wallet", Amount: 100.0, Fee: 5.0, ActualAmount: 95.0,
		WithdrawTo: "wechat", AccountInfoEncrypted: "info1", Status: models.WithdrawalStatusApproved,
	}
	db.Create(withdrawal)

	err := repo.MarkProcessing(ctx, withdrawal.ID)
	require.NoError(t, err)

	var found models.Withdrawal
	db.First(&found, withdrawal.ID)
	assert.Equal(t, models.WithdrawalStatusProcessing, found.Status)
}

func TestWithdrawalRepository_MarkSuccess(t *testing.T) {
	db := setupWithdrawalTestDB(t)
	repo := NewWithdrawalRepository(db)
	ctx := context.Background()

	withdrawal := &models.Withdrawal{
		WithdrawalNo: "WD001", UserID: 1, Type: "wallet", Amount: 100.0, Fee: 5.0, ActualAmount: 95.0,
		WithdrawTo: "wechat", AccountInfoEncrypted: "info1", Status: models.WithdrawalStatusProcessing,
	}
	db.Create(withdrawal)

	err := repo.MarkSuccess(ctx, withdrawal.ID)
	require.NoError(t, err)

	var found models.Withdrawal
	db.First(&found, withdrawal.ID)
	assert.Equal(t, models.WithdrawalStatusSuccess, found.Status)
}

func TestWithdrawalRepository_List(t *testing.T) {
	db := setupWithdrawalTestDB(t)
	repo := NewWithdrawalRepository(db)
	ctx := context.Background()

	db.Create(&models.Withdrawal{
		WithdrawalNo: "WD001", UserID: 1, Type: "wallet", Amount: 100.0, Fee: 5.0, ActualAmount: 95.0,
		WithdrawTo: "wechat", AccountInfoEncrypted: "info1", Status: models.WithdrawalStatusPending,
	})

	db.Create(&models.Withdrawal{
		WithdrawalNo: "WD002", UserID: 1, Type: "commission", Amount: 200.0, Fee: 10.0, ActualAmount: 190.0,
		WithdrawTo: "alipay", AccountInfoEncrypted: "info2", Status: models.WithdrawalStatusSuccess,
	})

	db.Create(&models.Withdrawal{
		WithdrawalNo: "WD003", UserID: 2, Type: "wallet", Amount: 150.0, Fee: 7.5, ActualAmount: 142.5,
		WithdrawTo: "wechat", AccountInfoEncrypted: "info3", Status: models.WithdrawalStatusPending,
	})

	// 获取所有提现
	_, total, err := repo.List(ctx, 0, 10, map[string]interface{}{})
	require.NoError(t, err)
	assert.Equal(t, int64(3), total)

	// 按用户过滤
	_, total, err = repo.List(ctx, 0, 10, map[string]interface{}{
		"user_id": int64(1),
	})
	require.NoError(t, err)
	assert.Equal(t, int64(2), total)

	// 按状态过滤
	_, total, err = repo.List(ctx, 0, 10, map[string]interface{}{
		"status": models.WithdrawalStatusPending,
	})
	require.NoError(t, err)
	assert.Equal(t, int64(2), total)

	// 按类型过滤
	_, total, err = repo.List(ctx, 0, 10, map[string]interface{}{
		"type": "wallet",
	})
	require.NoError(t, err)
	assert.Equal(t, int64(2), total)

	// 按提现方式过滤
	_, total, err = repo.List(ctx, 0, 10, map[string]interface{}{
		"withdraw_to": "wechat",
	})
	require.NoError(t, err)
	assert.Equal(t, int64(2), total)
}

func TestWithdrawalRepository_GetPendingList(t *testing.T) {
	db := setupWithdrawalTestDB(t)
	repo := NewWithdrawalRepository(db)
	ctx := context.Background()

	db.Create(&models.Withdrawal{
		WithdrawalNo: "WD001", UserID: 1, Type: "wallet", Amount: 100.0, Fee: 5.0, ActualAmount: 95.0,
		WithdrawTo: "wechat", AccountInfoEncrypted: "info1", Status: models.WithdrawalStatusPending,
	})

	db.Create(&models.Withdrawal{
		WithdrawalNo: "WD002", UserID: 1, Type: "wallet", Amount: 200.0, Fee: 10.0, ActualAmount: 190.0,
		WithdrawTo: "alipay", AccountInfoEncrypted: "info2", Status: models.WithdrawalStatusSuccess,
	})

	db.Create(&models.Withdrawal{
		WithdrawalNo: "WD003", UserID: 2, Type: "wallet", Amount: 150.0, Fee: 7.5, ActualAmount: 142.5,
		WithdrawTo: "wechat", AccountInfoEncrypted: "info3", Status: models.WithdrawalStatusPending,
	})

	_, total, err := repo.GetPendingList(ctx, 0, 10)
	require.NoError(t, err)
	assert.Equal(t, int64(2), total)
}

func TestWithdrawalRepository_GetApprovedList(t *testing.T) {
	db := setupWithdrawalTestDB(t)
	repo := NewWithdrawalRepository(db)
	ctx := context.Background()

	db.Create(&models.Withdrawal{
		WithdrawalNo: "WD001", UserID: 1, Type: "wallet", Amount: 100.0, Fee: 5.0, ActualAmount: 95.0,
		WithdrawTo: "wechat", AccountInfoEncrypted: "info1", Status: models.WithdrawalStatusApproved,
	})

	db.Create(&models.Withdrawal{
		WithdrawalNo: "WD002", UserID: 1, Type: "wallet", Amount: 200.0, Fee: 10.0, ActualAmount: 190.0,
		WithdrawTo: "alipay", AccountInfoEncrypted: "info2", Status: models.WithdrawalStatusSuccess,
	})

	db.Create(&models.Withdrawal{
		WithdrawalNo: "WD003", UserID: 2, Type: "wallet", Amount: 150.0, Fee: 7.5, ActualAmount: 142.5,
		WithdrawTo: "wechat", AccountInfoEncrypted: "info3", Status: models.WithdrawalStatusApproved,
	})

	_, total, err := repo.GetApprovedList(ctx, 0, 10)
	require.NoError(t, err)
	assert.Equal(t, int64(2), total)
}

func TestWithdrawalRepository_SumByUserID(t *testing.T) {
	db := setupWithdrawalTestDB(t)
	repo := NewWithdrawalRepository(db)
	ctx := context.Background()

	db.Create(&models.Withdrawal{
		WithdrawalNo: "WD001", UserID: 1, Type: "wallet", Amount: 100.0, Fee: 5.0, ActualAmount: 95.0,
		WithdrawTo: "wechat", AccountInfoEncrypted: "info1", Status: models.WithdrawalStatusSuccess,
	})

	db.Create(&models.Withdrawal{
		WithdrawalNo: "WD002", UserID: 1, Type: "wallet", Amount: 200.0, Fee: 10.0, ActualAmount: 190.0,
		WithdrawTo: "alipay", AccountInfoEncrypted: "info2", Status: models.WithdrawalStatusSuccess,
	})

	db.Create(&models.Withdrawal{
		WithdrawalNo: "WD003", UserID: 1, Type: "wallet", Amount: 150.0, Fee: 7.5, ActualAmount: 142.5,
		WithdrawTo: "wechat", AccountInfoEncrypted: "info3", Status: models.WithdrawalStatusPending,
	})

	// 所有提现总额
	sum, err := repo.SumByUserID(ctx, 1, nil)
	require.NoError(t, err)
	assert.Equal(t, 427.5, sum) // 95 + 190 + 142.5 = 427.5

	// 成功提现总额
	status := models.WithdrawalStatusSuccess
	sum, err = repo.SumByUserID(ctx, 1, &status)
	require.NoError(t, err)
	assert.Equal(t, 285.0, sum) // 95 + 190 = 285
}

func TestWithdrawalRepository_CountByStatus(t *testing.T) {
	db := setupWithdrawalTestDB(t)
	repo := NewWithdrawalRepository(db)
	ctx := context.Background()

	db.Create(&models.Withdrawal{
		WithdrawalNo: "WD001", UserID: 1, Type: "wallet", Amount: 100.0, Fee: 5.0, ActualAmount: 95.0,
		WithdrawTo: "wechat", AccountInfoEncrypted: "info1", Status: models.WithdrawalStatusPending,
	})

	db.Create(&models.Withdrawal{
		WithdrawalNo: "WD002", UserID: 1, Type: "wallet", Amount: 200.0, Fee: 10.0, ActualAmount: 190.0,
		WithdrawTo: "alipay", AccountInfoEncrypted: "info2", Status: models.WithdrawalStatusPending,
	})

	db.Create(&models.Withdrawal{
		WithdrawalNo: "WD003", UserID: 2, Type: "wallet", Amount: 150.0, Fee: 7.5, ActualAmount: 142.5,
		WithdrawTo: "wechat", AccountInfoEncrypted: "info3", Status: models.WithdrawalStatusSuccess,
	})

	count, err := repo.CountByStatus(ctx, models.WithdrawalStatusPending)
	require.NoError(t, err)
	assert.Equal(t, int64(2), count)

	count, err = repo.CountByStatus(ctx, models.WithdrawalStatusSuccess)
	require.NoError(t, err)
	assert.Equal(t, int64(1), count)
}

func TestWithdrawalRepository_CountPendingByUserID(t *testing.T) {
	db := setupWithdrawalTestDB(t)
	repo := NewWithdrawalRepository(db)
	ctx := context.Background()

	db.Create(&models.Withdrawal{
		WithdrawalNo: "WD001", UserID: 1, Type: "wallet", Amount: 100.0, Fee: 5.0, ActualAmount: 95.0,
		WithdrawTo: "wechat", AccountInfoEncrypted: "info1", Status: models.WithdrawalStatusPending,
	})

	db.Create(&models.Withdrawal{
		WithdrawalNo: "WD002", UserID: 1, Type: "wallet", Amount: 200.0, Fee: 10.0, ActualAmount: 190.0,
		WithdrawTo: "alipay", AccountInfoEncrypted: "info2", Status: models.WithdrawalStatusApproved,
	})

	db.Create(&models.Withdrawal{
		WithdrawalNo: "WD003", UserID: 1, Type: "wallet", Amount: 150.0, Fee: 7.5, ActualAmount: 142.5,
		WithdrawTo: "wechat", AccountInfoEncrypted: "info3", Status: models.WithdrawalStatusSuccess,
	})

	count, err := repo.CountPendingByUserID(ctx, 1)
	require.NoError(t, err)
	assert.Equal(t, int64(2), count) // pending + approved
}

func TestWithdrawalRepository_GetStatsByUserID(t *testing.T) {
	db := setupWithdrawalTestDB(t)
	repo := NewWithdrawalRepository(db)
	ctx := context.Background()

	db.Create(&models.Withdrawal{
		WithdrawalNo: "WD001", UserID: 1, Type: "wallet", Amount: 100.0, Fee: 5.0, ActualAmount: 95.0,
		WithdrawTo: "wechat", AccountInfoEncrypted: "info1", Status: models.WithdrawalStatusSuccess,
	})

	db.Create(&models.Withdrawal{
		WithdrawalNo: "WD002", UserID: 1, Type: "wallet", Amount: 200.0, Fee: 10.0, ActualAmount: 190.0,
		WithdrawTo: "alipay", AccountInfoEncrypted: "info2", Status: models.WithdrawalStatusSuccess,
	})

	db.Create(&models.Withdrawal{
		WithdrawalNo: "WD003", UserID: 1, Type: "wallet", Amount: 150.0, Fee: 7.5, ActualAmount: 142.5,
		WithdrawTo: "wechat", AccountInfoEncrypted: "info3", Status: models.WithdrawalStatusPending,
	})

	stats, err := repo.GetStatsByUserID(ctx, 1)
	require.NoError(t, err)
	assert.Equal(t, 450.0, stats["total_amount"])    // 100 + 200 + 150
	assert.Equal(t, 285.0, stats["success_amount"])  // 95 + 190
	assert.Equal(t, 150.0, stats["pending_amount"])  // 150
	assert.Equal(t, 15.0, stats["total_fee"])        // 5 + 10
	assert.Equal(t, int64(3), stats["total_count"])  // 总共3条
	assert.Equal(t, int64(2), stats["success_count"]) // 成功2条
}

func TestWithdrawalRepository_ExistsWithdrawalNo(t *testing.T) {
	db := setupWithdrawalTestDB(t)
	repo := NewWithdrawalRepository(db)
	ctx := context.Background()

	db.Create(&models.Withdrawal{
		WithdrawalNo: "WD202601001", UserID: 1, Type: "wallet", Amount: 100.0, Fee: 5.0, ActualAmount: 95.0,
		WithdrawTo: "wechat", AccountInfoEncrypted: "info1", Status: models.WithdrawalStatusPending,
	})

	exists, err := repo.ExistsWithdrawalNo(ctx, "WD202601001")
	require.NoError(t, err)
	assert.True(t, exists)

	exists, err = repo.ExistsWithdrawalNo(ctx, "WD999999")
	require.NoError(t, err)
	assert.False(t, exists)
}
