//go:build integration
// +build integration

// Package integration 分销模块集成测试
package integration

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
	distributionService "github.com/dumeirei/smart-locker-backend/internal/service/distribution"
	orderService "github.com/dumeirei/smart-locker-backend/internal/service/order"
)

// setupDistributionIntegrationDB 创建分销集成测试数据库
func setupDistributionIntegrationDB(t *testing.T) *gorm.DB {
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
		&models.Order{},
		&models.Distributor{},
		&models.Commission{},
		&models.Withdrawal{},
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

// setupDistributionIntegrationServices 创建分销集成测试服务
func setupDistributionIntegrationServices(db *gorm.DB) (
	*distributionService.DistributorService,
	*distributionService.CommissionService,
	*distributionService.WithdrawService,
	*orderService.OrderCompleteHook,
) {
	commissionRepo := repository.NewCommissionRepository(db)
	distributorRepo := repository.NewDistributorRepository(db)
	withdrawalRepo := repository.NewWithdrawalRepository(db)
	userRepo := repository.NewUserRepository(db)

	distributorSvc := distributionService.NewDistributorService(distributorRepo, userRepo, db)
	commissionSvc := distributionService.NewCommissionService(commissionRepo, distributorRepo, userRepo, db)
	withdrawSvc := distributionService.NewWithdrawService(withdrawalRepo, distributorRepo, userRepo, db)
	orderHook := orderService.NewOrderCompleteHook(commissionSvc)

	return distributorSvc, commissionSvc, withdrawSvc, orderHook
}

// createIntegrationTestUser 创建集成测试用户
func createIntegrationTestUser(db *gorm.DB, referrerID *int64) *models.User {
	phone := fmt.Sprintf("138%08d", time.Now().UnixNano()%100000000)
	user := &models.User{
		Phone:         &phone,
		Nickname:      "测试用户",
		MemberLevelID: 1,
		ReferrerID:    referrerID,
		Status:        models.UserStatusActive,
	}
	db.Create(user)

	// 创建钱包
	wallet := &models.UserWallet{
		UserID:  user.ID,
		Balance: 1000.0,
	}
	db.Create(wallet)

	return user
}

// TestDistributionFlow_ApplyAndApprove 测试分销商申请和审核流程
func TestDistributionFlow_ApplyAndApprove(t *testing.T) {
	db := setupDistributionIntegrationDB(t)
	distributorSvc, _, _, _ := setupDistributionIntegrationServices(db)
	ctx := context.Background()

	t.Run("完整申请审核流程", func(t *testing.T) {
		// 1. 创建用户
		user := createIntegrationTestUser(db, nil)

		// 2. 申请成为分销商
		applyReq := &distributionService.ApplyRequest{
			UserID: user.ID,
		}
		applyResp, err := distributorSvc.Apply(ctx, applyReq)
		require.NoError(t, err)
		assert.NotNil(t, applyResp)
		assert.Equal(t, models.DistributorStatusPending, applyResp.Distributor.Status)

		// 3. 验证分销商状态
		distributor, err := distributorSvc.GetByUserID(ctx, user.ID)
		require.NoError(t, err)
		assert.Equal(t, models.DistributorStatusPending, distributor.Status)

		// 4. 审核通过
		approveReq := &distributionService.ApproveRequest{
			DistributorID: distributor.ID,
			OperatorID:    1,
			Approved:      true,
		}
		err = distributorSvc.Approve(ctx, approveReq)
		require.NoError(t, err)

		// 5. 验证审核后状态
		distributor, err = distributorSvc.GetByUserID(ctx, user.ID)
		require.NoError(t, err)
		assert.Equal(t, models.DistributorStatusApproved, distributor.Status)
		assert.NotNil(t, distributor.ApprovedAt)
	})
}

// TestDistributionFlow_InviteAndTeam 测试邀请和团队关系
func TestDistributionFlow_InviteAndTeam(t *testing.T) {
	db := setupDistributionIntegrationDB(t)
	distributorSvc, _, _, _ := setupDistributionIntegrationServices(db)
	ctx := context.Background()

	t.Run("邀请码邀请成员并建立团队关系", func(t *testing.T) {
		// 1. 创建并审核通过上级分销商
		parentUser := createIntegrationTestUser(db, nil)
		applyReq := &distributionService.ApplyRequest{UserID: parentUser.ID}
		applyResp, err := distributorSvc.Apply(ctx, applyReq)
		require.NoError(t, err)

		err = distributorSvc.Approve(ctx, &distributionService.ApproveRequest{
			DistributorID: applyResp.Distributor.ID,
			OperatorID:    1,
			Approved:      true,
		})
		require.NoError(t, err)

		parentDistributor, _ := distributorSvc.GetByUserID(ctx, parentUser.ID)
		inviteCode := parentDistributor.InviteCode

		// 2. 新用户使用邀请码申请
		childUser := createIntegrationTestUser(db, nil)
		childApplyReq := &distributionService.ApplyRequest{
			UserID:     childUser.ID,
			InviteCode: &inviteCode,
		}
		childApplyResp, err := distributorSvc.Apply(ctx, childApplyReq)
		require.NoError(t, err)
		assert.NotNil(t, childApplyResp.Distributor.ParentID)
		assert.Equal(t, parentDistributor.ID, *childApplyResp.Distributor.ParentID)

		// 3. 审核通过子分销商
		err = distributorSvc.Approve(ctx, &distributionService.ApproveRequest{
			DistributorID: childApplyResp.Distributor.ID,
			OperatorID:    1,
			Approved:      true,
		})
		require.NoError(t, err)

		// 4. 验证上级团队人数增加
		updatedParent, err := distributorSvc.GetByUserID(ctx, parentUser.ID)
		require.NoError(t, err)
		assert.Equal(t, 1, updatedParent.DirectCount)
		assert.Equal(t, 1, updatedParent.TeamCount)
	})
}

// TestDistributionFlow_CommissionCalculation 测试佣金计算完整流程
func TestDistributionFlow_CommissionCalculation(t *testing.T) {
	db := setupDistributionIntegrationDB(t)
	distributorSvc, commissionSvc, _, orderHook := setupDistributionIntegrationServices(db)
	ctx := context.Background()

	t.Run("二级分销佣金计算", func(t *testing.T) {
		// 1. 创建上级分销商（间推）
		grandParentUser := createIntegrationTestUser(db, nil)
		grandParentResp, _ := distributorSvc.Apply(ctx, &distributionService.ApplyRequest{UserID: grandParentUser.ID})
		distributorSvc.Approve(ctx, &distributionService.ApproveRequest{
			DistributorID: grandParentResp.Distributor.ID,
			OperatorID:    1,
			Approved:      true,
		})
		grandParent, _ := distributorSvc.GetByUserID(ctx, grandParentUser.ID)

		// 2. 创建直推分销商
		parentUser := createIntegrationTestUser(db, nil)
		inviteCode := grandParent.InviteCode
		parentResp, _ := distributorSvc.Apply(ctx, &distributionService.ApplyRequest{
			UserID:     parentUser.ID,
			InviteCode: &inviteCode,
		})
		distributorSvc.Approve(ctx, &distributionService.ApproveRequest{
			DistributorID: parentResp.Distributor.ID,
			OperatorID:    1,
			Approved:      true,
		})
		parent, _ := distributorSvc.GetByUserID(ctx, parentUser.ID)

		// 3. 创建消费用户（被直推分销商推荐）
		consumer := createIntegrationTestUser(db, &parentUser.ID)

		// 4. 创建订单
		now := time.Now()
		order := &models.Order{
			OrderNo:        fmt.Sprintf("O%d", time.Now().UnixNano()),
			UserID:         consumer.ID,
			Type:           models.OrderTypeMall,
			OriginalAmount: 100.0,
			ActualAmount:   100.0,
			Status:         models.OrderStatusCompleted,
			PaidAt:         &now,
			CompletedAt:    &now,
		}
		db.Create(order)

		// 5. 调用订单完成钩子触发佣金计算
		err := orderHook.OnOrderCompleted(ctx, order)
		require.NoError(t, err)

		// 6. 验证佣金记录
		directCommissions, err := commissionSvc.GetByOrderID(ctx, order.ID)
		require.NoError(t, err)
		assert.Len(t, directCommissions, 2) // 直推 + 间推

		// 验证直推佣金
		var directCommission, indirectCommission *models.Commission
		for _, c := range directCommissions {
			if c.Type == models.CommissionTypeDirect {
				directCommission = c
			} else {
				indirectCommission = c
			}
		}

		assert.NotNil(t, directCommission)
		assert.Equal(t, parent.ID, directCommission.DistributorID)
		assert.Equal(t, 10.0, directCommission.Amount) // 100 * 10%

		assert.NotNil(t, indirectCommission)
		assert.Equal(t, grandParent.ID, indirectCommission.DistributorID)
		assert.Equal(t, 5.0, indirectCommission.Amount) // 100 * 5%
	})
}

// TestDistributionFlow_SettleAndWithdraw 测试结算和提现完整流程
func TestDistributionFlow_SettleAndWithdraw(t *testing.T) {
	db := setupDistributionIntegrationDB(t)
	distributorSvc, commissionSvc, withdrawSvc, _ := setupDistributionIntegrationServices(db)
	ctx := context.Background()

	t.Run("佣金结算后提现", func(t *testing.T) {
		// 1. 创建已审核分销商
		user := createIntegrationTestUser(db, nil)
		applyResp, _ := distributorSvc.Apply(ctx, &distributionService.ApplyRequest{UserID: user.ID})
		distributorSvc.Approve(ctx, &distributionService.ApproveRequest{
			DistributorID: applyResp.Distributor.ID,
			OperatorID:    1,
			Approved:      true,
		})
		distributor, _ := distributorSvc.GetByUserID(ctx, user.ID)

		// 2. 创建待结算佣金
		commission := &models.Commission{
			DistributorID: distributor.ID,
			OrderID:       1,
			FromUserID:    999,
			Type:          models.CommissionTypeDirect,
			OrderAmount:   100.0,
			Rate:          0.10,
			Amount:        10.0,
			Status:        models.CommissionStatusPending,
		}
		db.Create(commission)

		// 3. 结算佣金
		err := commissionSvc.Settle(ctx, commission.ID)
		require.NoError(t, err)

		// 4. 验证分销商可用佣金增加
		updatedDistributor, _ := distributorSvc.GetByUserID(ctx, user.ID)
		assert.Equal(t, 10.0, updatedDistributor.TotalCommission)
		assert.Equal(t, 10.0, updatedDistributor.AvailableCommission)

		// 5. 申请提现
		withdrawReq := &distributionService.WithdrawRequest{
			UserID:      user.ID,
			Type:        models.WithdrawalTypeCommission,
			Amount:      10.0,
			WithdrawTo:  models.WithdrawToWechat,
			AccountInfo: `{"openid":"test"}`,
		}
		withdrawResp, err := withdrawSvc.Apply(ctx, withdrawReq)
		require.NoError(t, err)
		assert.NotNil(t, withdrawResp)

		// 6. 验证余额被冻结
		updatedDistributor, _ = distributorSvc.GetByUserID(ctx, user.ID)
		assert.Equal(t, 0.0, updatedDistributor.AvailableCommission)
		assert.Equal(t, 10.0, updatedDistributor.FrozenCommission)

		// 7. 审核通过提现
		err = withdrawSvc.Approve(ctx, withdrawResp.Withdrawal.ID, 1)
		require.NoError(t, err)

		// 8. 处理提现
		err = withdrawSvc.Process(ctx, withdrawResp.Withdrawal.ID)
		require.NoError(t, err)

		// 9. 完成提现
		err = withdrawSvc.Complete(ctx, withdrawResp.Withdrawal.ID)
		require.NoError(t, err)

		// 10. 验证最终状态
		finalDistributor, _ := distributorSvc.GetByUserID(ctx, user.ID)
		assert.Equal(t, 0.0, finalDistributor.FrozenCommission)
		assert.Equal(t, 10.0, finalDistributor.WithdrawnCommission)

		var finalWithdrawal models.Withdrawal
		db.First(&finalWithdrawal, withdrawResp.Withdrawal.ID)
		assert.Equal(t, models.WithdrawalStatusSuccess, finalWithdrawal.Status)
	})
}

// TestDistributionFlow_RefundCancelCommission 测试退款取消佣金
func TestDistributionFlow_RefundCancelCommission(t *testing.T) {
	db := setupDistributionIntegrationDB(t)
	distributorSvc, commissionSvc, _, orderHook := setupDistributionIntegrationServices(db)
	ctx := context.Background()

	t.Run("订单退款取消佣金", func(t *testing.T) {
		// 1. 创建分销商
		parentUser := createIntegrationTestUser(db, nil)
		parentResp, _ := distributorSvc.Apply(ctx, &distributionService.ApplyRequest{UserID: parentUser.ID})
		distributorSvc.Approve(ctx, &distributionService.ApproveRequest{
			DistributorID: parentResp.Distributor.ID,
			OperatorID:    1,
			Approved:      true,
		})

		// 2. 创建消费用户
		consumer := createIntegrationTestUser(db, &parentUser.ID)

		// 3. 创建订单
		now := time.Now()
		order := &models.Order{
			OrderNo:        fmt.Sprintf("O%d", time.Now().UnixNano()),
			UserID:         consumer.ID,
			Type:           models.OrderTypeMall,
			OriginalAmount: 100.0,
			ActualAmount:   100.0,
			Status:         models.OrderStatusCompleted,
			PaidAt:         &now,
			CompletedAt:    &now,
		}
		db.Create(order)

		// 4. 触发佣金计算
		orderHook.OnOrderCompleted(ctx, order)

		// 5. 验证佣金已创建
		commissions, _ := commissionSvc.GetByOrderID(ctx, order.ID)
		assert.Len(t, commissions, 1)
		assert.Equal(t, models.CommissionStatusPending, commissions[0].Status)

		// 6. 模拟订单退款
		order.Status = models.OrderStatusRefunded
		orderHook.OnOrderRefunded(ctx, order)

		// 7. 验证佣金被取消
		cancelledCommissions, _ := commissionSvc.GetByOrderID(ctx, order.ID)
		assert.Len(t, cancelledCommissions, 1)
		assert.Equal(t, models.CommissionStatusCancelled, cancelledCommissions[0].Status)
	})

	t.Run("已结算佣金退款扣减余额", func(t *testing.T) {
		// 1. 创建分销商
		parentUser := createIntegrationTestUser(db, nil)
		parentResp, _ := distributorSvc.Apply(ctx, &distributionService.ApplyRequest{UserID: parentUser.ID})
		distributorSvc.Approve(ctx, &distributionService.ApproveRequest{
			DistributorID: parentResp.Distributor.ID,
			OperatorID:    1,
			Approved:      true,
		})
		parent, _ := distributorSvc.GetByUserID(ctx, parentUser.ID)

		// 2. 创建消费用户
		consumer := createIntegrationTestUser(db, &parentUser.ID)

		// 3. 创建订单
		now := time.Now()
		order := &models.Order{
			OrderNo:        fmt.Sprintf("O%d", time.Now().UnixNano()),
			UserID:         consumer.ID,
			Type:           models.OrderTypeMall,
			OriginalAmount: 100.0,
			ActualAmount:   100.0,
			Status:         models.OrderStatusCompleted,
			PaidAt:         &now,
			CompletedAt:    &now,
		}
		db.Create(order)

		// 4. 触发佣金计算
		orderHook.OnOrderCompleted(ctx, order)

		// 5. 结算佣金
		commissions, _ := commissionSvc.GetByOrderID(ctx, order.ID)
		commissionSvc.Settle(ctx, commissions[0].ID)

		// 6. 验证分销商余额
		updatedParent, _ := distributorSvc.GetByUserID(ctx, parentUser.ID)
		assert.Equal(t, 10.0, updatedParent.AvailableCommission)

		// 7. 模拟订单退款
		order.Status = models.OrderStatusRefunded
		commissionSvc.CancelByOrderID(ctx, order.ID)

		// 8. 验证余额被扣减
		finalParent, _ := distributorSvc.GetByID(ctx, parent.ID)
		assert.Equal(t, 0.0, finalParent.AvailableCommission)
	})
}

// TestDistributionFlow_WithdrawRejection 测试提现拒绝流程
func TestDistributionFlow_WithdrawRejection(t *testing.T) {
	db := setupDistributionIntegrationDB(t)
	distributorSvc, commissionSvc, withdrawSvc, _ := setupDistributionIntegrationServices(db)
	ctx := context.Background()

	t.Run("提现被拒绝后余额解冻", func(t *testing.T) {
		// 1. 创建已审核分销商
		user := createIntegrationTestUser(db, nil)
		applyResp, _ := distributorSvc.Apply(ctx, &distributionService.ApplyRequest{UserID: user.ID})
		distributorSvc.Approve(ctx, &distributionService.ApproveRequest{
			DistributorID: applyResp.Distributor.ID,
			OperatorID:    1,
			Approved:      true,
		})

		// 2. 创建并结算佣金
		distributor, _ := distributorSvc.GetByUserID(ctx, user.ID)
		commission := &models.Commission{
			DistributorID: distributor.ID,
			OrderID:       1,
			FromUserID:    999,
			Type:          models.CommissionTypeDirect,
			OrderAmount:   100.0,
			Rate:          0.10,
			Amount:        50.0,
			Status:        models.CommissionStatusPending,
		}
		db.Create(commission)
		commissionSvc.Settle(ctx, commission.ID)

		// 3. 申请提现
		withdrawReq := &distributionService.WithdrawRequest{
			UserID:      user.ID,
			Type:        models.WithdrawalTypeCommission,
			Amount:      50.0,
			WithdrawTo:  models.WithdrawToWechat,
			AccountInfo: `{"openid":"test"}`,
		}
		withdrawResp, _ := withdrawSvc.Apply(ctx, withdrawReq)

		// 4. 验证余额被冻结
		frozenDistributor, _ := distributorSvc.GetByUserID(ctx, user.ID)
		assert.Equal(t, 0.0, frozenDistributor.AvailableCommission)
		assert.Equal(t, 50.0, frozenDistributor.FrozenCommission)

		// 5. 拒绝提现
		err := withdrawSvc.Reject(ctx, withdrawResp.Withdrawal.ID, 1, "资料不完整")
		require.NoError(t, err)

		// 6. 验证余额解冻
		unfrozenDistributor, _ := distributorSvc.GetByUserID(ctx, user.ID)
		assert.Equal(t, 50.0, unfrozenDistributor.AvailableCommission)
		assert.Equal(t, 0.0, unfrozenDistributor.FrozenCommission)
	})
}
