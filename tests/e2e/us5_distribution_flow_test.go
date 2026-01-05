// Package e2e 分销推广与佣金管理完整流程 E2E 测试
package e2e

import (
	"context"
	"fmt"
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

// distributionE2ETestContext E2E测试上下文
type distributionE2ETestContext struct {
	db                *gorm.DB
	distributorSvc    *distributionService.DistributorService
	commissionSvc     *distributionService.CommissionService
	withdrawSvc       *distributionService.WithdrawService
	inviteSvc         *distributionService.InviteService
	orderCompleteHook *orderService.OrderCompleteHook
}

// setupDistributionE2ETestDB 创建E2E测试数据库
func setupDistributionE2ETestDB(t *testing.T) *gorm.DB {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Silent),
	})
	require.NoError(t, err)

	err = db.AutoMigrate(
		&models.User{},
		&models.UserWallet{},
		&models.MemberLevel{},
		&models.Admin{},
		&models.Role{},
		&models.Order{},
		&models.Distributor{},
		&models.Commission{},
		&models.Withdrawal{},
	)
	require.NoError(t, err)

	// 初始化基础数据
	level := &models.MemberLevel{ID: 1, Name: "普通会员", Level: 1, MinPoints: 0, Discount: 1.0}
	db.Create(level)

	return db
}

// setupDistributionE2ETestContext 创建E2E测试上下文
func setupDistributionE2ETestContext(t *testing.T) *distributionE2ETestContext {
	db := setupDistributionE2ETestDB(t)

	commissionRepo := repository.NewCommissionRepository(db)
	distributorRepo := repository.NewDistributorRepository(db)
	withdrawalRepo := repository.NewWithdrawalRepository(db)
	userRepo := repository.NewUserRepository(db)

	distributorSvc := distributionService.NewDistributorService(distributorRepo, userRepo, db)
	commissionSvc := distributionService.NewCommissionService(commissionRepo, distributorRepo, userRepo, db)
	withdrawSvc := distributionService.NewWithdrawService(withdrawalRepo, distributorRepo, userRepo, db)
	inviteSvc := distributionService.NewInviteService(distributorRepo, "https://example.com")
	orderHook := orderService.NewOrderCompleteHook(commissionSvc)

	return &distributionE2ETestContext{
		db:                db,
		distributorSvc:    distributorSvc,
		commissionSvc:     commissionSvc,
		withdrawSvc:       withdrawSvc,
		inviteSvc:         inviteSvc,
		orderCompleteHook: orderHook,
	}
}

// createE2EDistributionUser 创建分销E2E测试用户
func createE2EDistributionUser(t *testing.T, db *gorm.DB, phone string, balance float64, referrerID *int64) *models.User {
	user := &models.User{
		Phone:         &phone,
		Nickname:      "E2E测试用户",
		MemberLevelID: 1,
		ReferrerID:    referrerID,
		Status:        models.UserStatusActive,
	}
	db.Create(user)

	wallet := &models.UserWallet{
		UserID:  user.ID,
		Balance: balance,
	}
	db.Create(wallet)

	return user
}

// createE2EDistributionAdmin 创建测试管理员
func createE2EDistributionAdmin(t *testing.T, db *gorm.DB) *models.Admin {
	description := "分销管理员"
	role := &models.Role{Code: "distribution_admin", Name: "分销管理员", Description: &description}
	db.Create(role)

	admin := &models.Admin{
		Username:     "dist_admin",
		PasswordHash: "hashedpassword",
		Name:         "分销管理员",
		RoleID:       role.ID,
		Status:       models.AdminStatusActive,
	}
	db.Create(admin)

	return admin
}

// createE2EOrder 创建测试订单
func createE2EOrder(t *testing.T, db *gorm.DB, userID int64, amount float64, status string) *models.Order {
	now := time.Now()
	order := &models.Order{
		OrderNo:        fmt.Sprintf("E2E%d", time.Now().UnixNano()),
		UserID:         userID,
		Type:           models.OrderTypeMall,
		OriginalAmount: amount,
		ActualAmount:   amount,
		Status:         status,
		PaidAt:         &now,
	}
	if status == models.OrderStatusCompleted {
		order.CompletedAt = &now
	}
	db.Create(order)
	return order
}

// TestE2E_DistributionCompleteFlow 测试完整的分销推广业务流程
func TestE2E_DistributionCompleteFlow(t *testing.T) {
	tc := setupDistributionE2ETestContext(t)
	ctx := context.Background()

	// 准备测试数据
	admin := createE2EDistributionAdmin(t, tc.db)

	t.Run("场景1: 用户申请成为分销商并推广获得佣金", func(t *testing.T) {
		// Step 1: 用户注册
		user := createE2EDistributionUser(t, tc.db, "13800138001", 500.0, nil)
		t.Logf("Step 1: 用户注册成功，用户ID: %d", user.ID)

		// Step 2: 检查用户是否是分销商
		isDistributor, err := tc.distributorSvc.CheckIsDistributor(ctx, user.ID)
		require.NoError(t, err)
		assert.False(t, isDistributor)
		t.Logf("Step 2: 用户当前不是分销商")

		// Step 3: 用户申请成为分销商
		applyResp, err := tc.distributorSvc.Apply(ctx, &distributionService.ApplyRequest{
			UserID: user.ID,
		})
		require.NoError(t, err)
		assert.Equal(t, models.DistributorStatusPending, applyResp.Distributor.Status)
		assert.NotEmpty(t, applyResp.Distributor.InviteCode)
		t.Logf("Step 3: 用户提交分销商申请，邀请码: %s，状态: 待审核", applyResp.Distributor.InviteCode)

		// Step 4: 管理员审核通过
		err = tc.distributorSvc.Approve(ctx, &distributionService.ApproveRequest{
			DistributorID: applyResp.Distributor.ID,
			OperatorID:    admin.ID,
			Approved:      true,
		})
		require.NoError(t, err)
		t.Logf("Step 4: 管理员 %s 审核通过分销商申请", admin.Name)

		// Step 5: 验证分销商状态
		distributor, err := tc.distributorSvc.GetByUserID(ctx, user.ID)
		require.NoError(t, err)
		assert.Equal(t, models.DistributorStatusApproved, distributor.Status)
		assert.NotNil(t, distributor.ApprovedAt)
		t.Logf("Step 5: 分销商已激活，邀请码: %s", distributor.InviteCode)

		// Step 6: 获取分销商仪表盘数据
		dashboard, err := tc.distributorSvc.GetDashboard(ctx, distributor.ID)
		require.NoError(t, err)
		assert.Equal(t, 0.0, dashboard.TotalCommission)
		assert.Equal(t, 0, dashboard.TeamCount)
		t.Logf("Step 6: 分销商仪表盘 - 总佣金: %.2f, 团队人数: %d", dashboard.TotalCommission, dashboard.TeamCount)

		// Step 7: 获取邀请信息和邀请链接
		inviteInfo, err := tc.inviteSvc.GenerateInviteInfo(ctx, distributor.ID)
		require.NoError(t, err)
		assert.NotEmpty(t, inviteInfo.InviteCode)
		assert.NotEmpty(t, inviteInfo.InviteLink)
		t.Logf("Step 7: 邀请链接: %s", inviteInfo.InviteLink)

		// Step 8: 新用户通过邀请链接注册并下单
		invitedUser := createE2EDistributionUser(t, tc.db, "13800138002", 1000.0, &user.ID)
		t.Logf("Step 8: 新用户通过邀请注册，用户ID: %d，推荐人ID: %d", invitedUser.ID, user.ID)

		// Step 9: 新用户下单并完成订单
		order := createE2EOrder(t, tc.db, invitedUser.ID, 100.0, models.OrderStatusCompleted)
		t.Logf("Step 9: 新用户下单并完成，订单金额: %.2f", order.ActualAmount)

		// Step 10: 触发佣金计算
		err = tc.orderCompleteHook.OnOrderCompleted(ctx, order)
		require.NoError(t, err)
		t.Logf("Step 10: 订单完成，触发佣金计算")

		// Step 11: 验证佣金记录
		commissions, err := tc.commissionSvc.GetByOrderID(ctx, order.ID)
		require.NoError(t, err)
		assert.Len(t, commissions, 1)
		assert.Equal(t, models.CommissionTypeDirect, commissions[0].Type)
		assert.Equal(t, 10.0, commissions[0].Amount) // 100 * 10%
		t.Logf("Step 11: 佣金记录已创建 - 类型: 直推, 金额: %.2f", commissions[0].Amount)

		// Step 12: 结算佣金（模拟7天后自动结算）
		err = tc.commissionSvc.Settle(ctx, commissions[0].ID)
		require.NoError(t, err)
		t.Logf("Step 12: 佣金已结算")

		// Step 13: 验证分销商余额增加
		updatedDistributor, err := tc.distributorSvc.GetByUserID(ctx, user.ID)
		require.NoError(t, err)
		assert.Equal(t, 10.0, updatedDistributor.TotalCommission)
		assert.Equal(t, 10.0, updatedDistributor.AvailableCommission)
		t.Logf("Step 13: 分销商余额已更新 - 总佣金: %.2f, 可用: %.2f",
			updatedDistributor.TotalCommission, updatedDistributor.AvailableCommission)
	})

	t.Run("场景2: 二级分销佣金计算", func(t *testing.T) {
		// 创建一级分销商
		level1User := createE2EDistributionUser(t, tc.db, "13800138010", 500.0, nil)
		level1Resp, err := tc.distributorSvc.Apply(ctx, &distributionService.ApplyRequest{UserID: level1User.ID})
		require.NoError(t, err)
		err = tc.distributorSvc.Approve(ctx, &distributionService.ApproveRequest{
			DistributorID: level1Resp.Distributor.ID,
			OperatorID:    admin.ID,
			Approved:      true,
		})
		require.NoError(t, err)
		level1Distributor, _ := tc.distributorSvc.GetByUserID(ctx, level1User.ID)
		t.Logf("一级分销商创建成功，邀请码: %s", level1Distributor.InviteCode)

		// 创建二级分销商（通过一级分销商邀请）
		level2User := createE2EDistributionUser(t, tc.db, "13800138011", 500.0, &level1User.ID)
		inviteCode := level1Distributor.InviteCode
		level2Resp, err := tc.distributorSvc.Apply(ctx, &distributionService.ApplyRequest{
			UserID:     level2User.ID,
			InviteCode: &inviteCode,
		})
		require.NoError(t, err)
		err = tc.distributorSvc.Approve(ctx, &distributionService.ApproveRequest{
			DistributorID: level2Resp.Distributor.ID,
			OperatorID:    admin.ID,
			Approved:      true,
		})
		require.NoError(t, err)
		t.Logf("二级分销商创建成功，上级ID: %d", *level2Resp.Distributor.ParentID)

		// 验证一级分销商团队人数增加
		level1Updated, _ := tc.distributorSvc.GetByUserID(ctx, level1User.ID)
		assert.Equal(t, 1, level1Updated.DirectCount)
		assert.Equal(t, 1, level1Updated.TeamCount)
		t.Logf("一级分销商团队更新 - 直推: %d, 团队: %d", level1Updated.DirectCount, level1Updated.TeamCount)

		// 消费用户（由二级分销商推荐）下单
		consumer := createE2EDistributionUser(t, tc.db, "13800138012", 1000.0, &level2User.ID)
		order := createE2EOrder(t, tc.db, consumer.ID, 200.0, models.OrderStatusCompleted)
		err = tc.orderCompleteHook.OnOrderCompleted(ctx, order)
		require.NoError(t, err)
		t.Logf("消费用户下单 %.2f 元", order.ActualAmount)

		// 验证二级佣金记录
		commissions, err := tc.commissionSvc.GetByOrderID(ctx, order.ID)
		require.NoError(t, err)
		assert.Len(t, commissions, 2) // 直推 + 间推

		var directCommission, indirectCommission *models.Commission
		for _, c := range commissions {
			if c.Type == models.CommissionTypeDirect {
				directCommission = c
			} else {
				indirectCommission = c
			}
		}

		assert.NotNil(t, directCommission)
		assert.Equal(t, 20.0, directCommission.Amount) // 200 * 10%
		t.Logf("直推佣金: %.2f (分销商ID: %d)", directCommission.Amount, directCommission.DistributorID)

		assert.NotNil(t, indirectCommission)
		assert.Equal(t, 10.0, indirectCommission.Amount) // 200 * 5%
		t.Logf("间推佣金: %.2f (分销商ID: %d)", indirectCommission.Amount, indirectCommission.DistributorID)
	})

	t.Run("场景3: 分销商提现完整流程", func(t *testing.T) {
		// 创建有余额的分销商
		user := createE2EDistributionUser(t, tc.db, "13800138020", 500.0, nil)
		applyResp, _ := tc.distributorSvc.Apply(ctx, &distributionService.ApplyRequest{UserID: user.ID})
		tc.distributorSvc.Approve(ctx, &distributionService.ApproveRequest{
			DistributorID: applyResp.Distributor.ID,
			OperatorID:    admin.ID,
			Approved:      true,
		})

		// 模拟佣金收入
		distributor, _ := tc.distributorSvc.GetByUserID(ctx, user.ID)
		commission := &models.Commission{
			DistributorID: distributor.ID,
			OrderID:       999,
			FromUserID:    888,
			Type:          models.CommissionTypeDirect,
			OrderAmount:   500.0,
			Rate:          0.10,
			Amount:        50.0,
			Status:        models.CommissionStatusPending,
		}
		tc.db.Create(commission)
		tc.commissionSvc.Settle(ctx, commission.ID)
		t.Logf("分销商已有可用佣金: 50.00 元")

		// Step 1: 获取提现配置
		config := tc.withdrawSvc.GetConfig()
		t.Logf("Step 1: 提现配置 - 最低金额: %.2f, 手续费率: %.4f",
			config["min_withdraw"], config["withdraw_fee"])

		// Step 2: 申请提现
		withdrawReq := &distributionService.WithdrawRequest{
			UserID:      user.ID,
			Type:        models.WithdrawalTypeCommission,
			Amount:      50.0,
			WithdrawTo:  models.WithdrawToWechat,
			AccountInfo: `{"openid":"oXXXX_test_openid"}`,
		}
		withdrawResp, err := tc.withdrawSvc.Apply(ctx, withdrawReq)
		require.NoError(t, err)
		assert.Equal(t, models.WithdrawalStatusPending, withdrawResp.Withdrawal.Status)
		t.Logf("Step 2: 提现申请已提交 - 金额: %.2f, 手续费: %.2f, 实际到账: %.2f",
			withdrawResp.Withdrawal.Amount, withdrawResp.Fee, withdrawResp.ActualAmount)

		// Step 3: 验证余额被冻结
		frozenDistributor, _ := tc.distributorSvc.GetByUserID(ctx, user.ID)
		assert.Equal(t, 0.0, frozenDistributor.AvailableCommission)
		assert.Equal(t, 50.0, frozenDistributor.FrozenCommission)
		t.Logf("Step 3: 余额已冻结 - 可用: %.2f, 冻结: %.2f",
			frozenDistributor.AvailableCommission, frozenDistributor.FrozenCommission)

		// Step 4: 管理员审核通过
		err = tc.withdrawSvc.Approve(ctx, withdrawResp.Withdrawal.ID, admin.ID)
		require.NoError(t, err)
		t.Logf("Step 4: 管理员审核通过")

		// Step 5: 开始处理提现
		err = tc.withdrawSvc.Process(ctx, withdrawResp.Withdrawal.ID)
		require.NoError(t, err)
		t.Logf("Step 5: 提现处理中")

		// Step 6: 完成提现
		err = tc.withdrawSvc.Complete(ctx, withdrawResp.Withdrawal.ID)
		require.NoError(t, err)
		t.Logf("Step 6: 提现完成")

		// Step 7: 验证最终状态
		var finalWithdrawal models.Withdrawal
		tc.db.First(&finalWithdrawal, withdrawResp.Withdrawal.ID)
		assert.Equal(t, models.WithdrawalStatusSuccess, finalWithdrawal.Status)
		assert.NotNil(t, finalWithdrawal.ProcessedAt)

		finalDistributor, _ := tc.distributorSvc.GetByUserID(ctx, user.ID)
		assert.Equal(t, 0.0, finalDistributor.FrozenCommission)
		assert.Equal(t, 50.0, finalDistributor.WithdrawnCommission)
		t.Logf("Step 7: 提现完成 - 累计提现: %.2f", finalDistributor.WithdrawnCommission)
	})

	t.Run("场景4: 订单退款取消佣金", func(t *testing.T) {
		// 创建分销商
		user := createE2EDistributionUser(t, tc.db, "13800138030", 500.0, nil)
		applyResp, _ := tc.distributorSvc.Apply(ctx, &distributionService.ApplyRequest{UserID: user.ID})
		tc.distributorSvc.Approve(ctx, &distributionService.ApproveRequest{
			DistributorID: applyResp.Distributor.ID,
			OperatorID:    admin.ID,
			Approved:      true,
		})

		// 新用户下单
		consumer := createE2EDistributionUser(t, tc.db, "13800138031", 1000.0, &user.ID)
		order := createE2EOrder(t, tc.db, consumer.ID, 100.0, models.OrderStatusCompleted)
		tc.orderCompleteHook.OnOrderCompleted(ctx, order)
		t.Logf("订单完成，佣金已生成")

		// 验证佣金状态
		commissions, _ := tc.commissionSvc.GetByOrderID(ctx, order.ID)
		assert.Equal(t, models.CommissionStatusPending, commissions[0].Status)
		t.Logf("佣金状态: 待结算")

		// 模拟订单退款
		order.Status = models.OrderStatusRefunded
		tc.orderCompleteHook.OnOrderRefunded(ctx, order)
		t.Logf("订单已退款")

		// 验证佣金被取消
		cancelledCommissions, _ := tc.commissionSvc.GetByOrderID(ctx, order.ID)
		assert.Equal(t, models.CommissionStatusCancelled, cancelledCommissions[0].Status)
		t.Logf("佣金已取消")
	})

	t.Run("场景5: 提现被拒绝后余额解冻", func(t *testing.T) {
		// 创建有余额的分销商
		user := createE2EDistributionUser(t, tc.db, "13800138040", 500.0, nil)
		applyResp, _ := tc.distributorSvc.Apply(ctx, &distributionService.ApplyRequest{UserID: user.ID})
		tc.distributorSvc.Approve(ctx, &distributionService.ApproveRequest{
			DistributorID: applyResp.Distributor.ID,
			OperatorID:    admin.ID,
			Approved:      true,
		})

		// 模拟佣金收入
		distributor, _ := tc.distributorSvc.GetByUserID(ctx, user.ID)
		commission := &models.Commission{
			DistributorID: distributor.ID,
			OrderID:       1001,
			FromUserID:    1002,
			Type:          models.CommissionTypeDirect,
			OrderAmount:   300.0,
			Rate:          0.10,
			Amount:        30.0,
			Status:        models.CommissionStatusPending,
		}
		tc.db.Create(commission)
		tc.commissionSvc.Settle(ctx, commission.ID)
		t.Logf("分销商可用佣金: 30.00 元")

		// 申请提现
		withdrawResp, err := tc.withdrawSvc.Apply(ctx, &distributionService.WithdrawRequest{
			UserID:      user.ID,
			Type:        models.WithdrawalTypeCommission,
			Amount:      30.0,
			WithdrawTo:  models.WithdrawToAlipay,
			AccountInfo: `{"account":"test@alipay.com"}`,
		})
		require.NoError(t, err)
		t.Logf("提现申请已提交，余额已冻结")

		// 验证余额被冻结
		frozenDistributor, _ := tc.distributorSvc.GetByUserID(ctx, user.ID)
		assert.Equal(t, 0.0, frozenDistributor.AvailableCommission)
		assert.Equal(t, 30.0, frozenDistributor.FrozenCommission)

		// 管理员拒绝提现
		err = tc.withdrawSvc.Reject(ctx, withdrawResp.Withdrawal.ID, admin.ID, "资料不完整，请补充完整后重新申请")
		require.NoError(t, err)
		t.Logf("管理员拒绝提现，原因: 资料不完整")

		// 验证提现状态和余额解冻
		var rejectedWithdrawal models.Withdrawal
		tc.db.First(&rejectedWithdrawal, withdrawResp.Withdrawal.ID)
		assert.Equal(t, models.WithdrawalStatusRejected, rejectedWithdrawal.Status)
		assert.NotNil(t, rejectedWithdrawal.RejectReason)

		unfrozenDistributor, _ := tc.distributorSvc.GetByUserID(ctx, user.ID)
		assert.Equal(t, 30.0, unfrozenDistributor.AvailableCommission)
		assert.Equal(t, 0.0, unfrozenDistributor.FrozenCommission)
		t.Logf("余额已解冻 - 可用: %.2f, 冻结: %.2f",
			unfrozenDistributor.AvailableCommission, unfrozenDistributor.FrozenCommission)
	})
}

// TestE2E_DistributionTeamManagement 测试团队管理功能
func TestE2E_DistributionTeamManagement(t *testing.T) {
	tc := setupDistributionE2ETestContext(t)
	ctx := context.Background()
	admin := createE2EDistributionAdmin(t, tc.db)

	t.Run("场景: 分销商邀请团队成员并查看统计", func(t *testing.T) {
		// 创建顶级分销商
		topUser := createE2EDistributionUser(t, tc.db, "13900139001", 500.0, nil)
		topResp, _ := tc.distributorSvc.Apply(ctx, &distributionService.ApplyRequest{UserID: topUser.ID})
		tc.distributorSvc.Approve(ctx, &distributionService.ApproveRequest{
			DistributorID: topResp.Distributor.ID,
			OperatorID:    admin.ID,
			Approved:      true,
		})
		topDistributor, _ := tc.distributorSvc.GetByUserID(ctx, topUser.ID)
		t.Logf("顶级分销商创建成功，邀请码: %s", topDistributor.InviteCode)

		// 邀请3个直接下级
		for i := 1; i <= 3; i++ {
			phone := fmt.Sprintf("1390013900%d", i+1)
			memberUser := createE2EDistributionUser(t, tc.db, phone, 500.0, &topUser.ID)
			inviteCode := topDistributor.InviteCode
			memberResp, _ := tc.distributorSvc.Apply(ctx, &distributionService.ApplyRequest{
				UserID:     memberUser.ID,
				InviteCode: &inviteCode,
			})
			tc.distributorSvc.Approve(ctx, &distributionService.ApproveRequest{
				DistributorID: memberResp.Distributor.ID,
				OperatorID:    admin.ID,
				Approved:      true,
			})
			t.Logf("邀请第 %d 个直接下级", i)
		}

		// 验证团队统计
		teamStats, err := tc.distributorSvc.GetTeamStats(ctx, topDistributor.ID)
		require.NoError(t, err)
		assert.Equal(t, 3, teamStats.DirectCount)
		assert.Equal(t, 3, teamStats.TeamCount)
		t.Logf("团队统计 - 直推: %d, 团队总人数: %d", teamStats.DirectCount, teamStats.TeamCount)

		// 获取团队成员列表
		members, total, err := tc.distributorSvc.GetTeamMembers(ctx, topDistributor.ID, 0, 10, "all")
		require.NoError(t, err)
		assert.Equal(t, int64(3), total)
		assert.Len(t, members, 3)
		t.Logf("团队成员数: %d", total)

		for _, m := range members {
			t.Logf("- 成员ID: %d, 直推: %v", m.ID, m.ParentID != nil && *m.ParentID == topDistributor.ID)
		}
	})
}

// TestE2E_DistributionRanking 测试分销排行榜功能
func TestE2E_DistributionRanking(t *testing.T) {
	tc := setupDistributionE2ETestContext(t)
	ctx := context.Background()
	admin := createE2EDistributionAdmin(t, tc.db)

	t.Run("场景: 获取分销商排行榜", func(t *testing.T) {
		// 创建多个分销商并设置不同佣金
		commissions := []float64{500.0, 300.0, 100.0, 200.0, 400.0}
		for i, commission := range commissions {
			phone := fmt.Sprintf("1380013800%d", i)
			user := createE2EDistributionUser(t, tc.db, phone, 500.0, nil)
			applyResp, _ := tc.distributorSvc.Apply(ctx, &distributionService.ApplyRequest{UserID: user.ID})
			tc.distributorSvc.Approve(ctx, &distributionService.ApproveRequest{
				DistributorID: applyResp.Distributor.ID,
				OperatorID:    admin.ID,
				Approved:      true,
			})

			// 直接设置佣金（模拟）
			tc.db.Model(&models.Distributor{}).Where("id = ?", applyResp.Distributor.ID).Updates(map[string]interface{}{
				"total_commission":     commission,
				"available_commission": commission,
			})
			t.Logf("分销商 %d 佣金: %.2f", i+1, commission)
		}

		// 获取排行榜
		ranking, err := tc.distributorSvc.GetTopDistributors(ctx, 10)
		require.NoError(t, err)
		assert.Len(t, ranking, 5)

		// 验证排序（按佣金降序）
		for i := 1; i < len(ranking); i++ {
			assert.GreaterOrEqual(t, ranking[i-1].TotalCommission, ranking[i].TotalCommission)
		}

		t.Logf("排行榜前 %d 名:", len(ranking))
		for i, r := range ranking {
			t.Logf("%d. 分销商ID: %d, 佣金: %.2f", i+1, r.ID, r.TotalCommission)
		}
	})
}

// TestE2E_DistributionInviteValidation 测试邀请码验证
func TestE2E_DistributionInviteValidation(t *testing.T) {
	tc := setupDistributionE2ETestContext(t)
	ctx := context.Background()
	admin := createE2EDistributionAdmin(t, tc.db)

	t.Run("场景: 验证邀请码有效性", func(t *testing.T) {
		// 创建分销商
		user := createE2EDistributionUser(t, tc.db, "13700137001", 500.0, nil)
		applyResp, _ := tc.distributorSvc.Apply(ctx, &distributionService.ApplyRequest{UserID: user.ID})
		tc.distributorSvc.Approve(ctx, &distributionService.ApproveRequest{
			DistributorID: applyResp.Distributor.ID,
			OperatorID:    admin.ID,
			Approved:      true,
		})
		distributor, _ := tc.distributorSvc.GetByUserID(ctx, user.ID)
		t.Logf("分销商邀请码: %s", distributor.InviteCode)

		// 验证有效邀请码
		result, err := tc.inviteSvc.ValidateInviteCode(ctx, distributor.InviteCode)
		require.NoError(t, err)
		assert.NotNil(t, result)
		assert.Equal(t, distributor.ID, result.ID)
		t.Logf("邀请码 %s 有效，邀请人ID: %d", distributor.InviteCode, result.ID)

		// 验证无效邀请码
		invalidResult, err := tc.inviteSvc.ValidateInviteCode(ctx, "INVALID123")
		assert.Error(t, err)
		assert.Nil(t, invalidResult)
		t.Logf("邀请码 INVALID123 无效")
	})
}

// TestE2E_DistributionCommissionHistory 测试佣金历史记录
func TestE2E_DistributionCommissionHistory(t *testing.T) {
	tc := setupDistributionE2ETestContext(t)
	ctx := context.Background()
	admin := createE2EDistributionAdmin(t, tc.db)

	t.Run("场景: 查看佣金收入历史", func(t *testing.T) {
		// 创建分销商
		user := createE2EDistributionUser(t, tc.db, "13600136001", 500.0, nil)
		applyResp, _ := tc.distributorSvc.Apply(ctx, &distributionService.ApplyRequest{UserID: user.ID})
		tc.distributorSvc.Approve(ctx, &distributionService.ApproveRequest{
			DistributorID: applyResp.Distributor.ID,
			OperatorID:    admin.ID,
			Approved:      true,
		})

		// 邀请用户并产生多笔订单
		for i := 1; i <= 5; i++ {
			phone := fmt.Sprintf("1360013600%d", i+1)
			consumer := createE2EDistributionUser(t, tc.db, phone, 1000.0, &user.ID)
			order := createE2EOrder(t, tc.db, consumer.ID, float64(100*i), models.OrderStatusCompleted)
			tc.orderCompleteHook.OnOrderCompleted(ctx, order)
			t.Logf("订单 %d: %.2f 元", i, order.ActualAmount)
		}

		// 获取佣金记录
		distributor, _ := tc.distributorSvc.GetByUserID(ctx, user.ID)
		commissions, total, err := tc.commissionSvc.GetByDistributorID(ctx, distributor.ID, 0, 10)
		require.NoError(t, err)
		assert.Equal(t, int64(5), total)
		t.Logf("佣金记录数: %d", total)

		var totalAmount float64
		for _, c := range commissions {
			totalAmount += c.Amount
			t.Logf("- 订单ID: %d, 类型: %s, 金额: %.2f, 状态: %s",
				c.OrderID, c.Type, c.Amount, getCommissionStatusName(c.Status))
		}
		t.Logf("总待结算佣金: %.2f", totalAmount)

		// 获取佣金统计
		stats, err := tc.commissionSvc.GetStats(ctx, distributor.ID)
		require.NoError(t, err)
		t.Logf("佣金统计 - 待结算: %.2f, 已结算: %.2f, 总金额: %.2f",
			stats["pending_amount"], stats["settled_amount"], stats["total_amount"])
	})
}

// getCommissionStatusName 获取佣金状态名称
func getCommissionStatusName(status int) string {
	switch status {
	case models.CommissionStatusPending:
		return "待结算"
	case models.CommissionStatusSettled:
		return "已结算"
	case models.CommissionStatusCancelled:
		return "已取消"
	default:
		return "未知"
	}
}
