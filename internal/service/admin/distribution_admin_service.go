// Package admin 提供管理端相关服务
package admin

import (
	"context"
	"time"

	"gorm.io/gorm"

	"github.com/dumeirei/smart-locker-backend/internal/models"
	"github.com/dumeirei/smart-locker-backend/internal/repository"
)

// DistributionAdminService 分销管理服务
type DistributionAdminService struct {
	distributorRepo *repository.DistributorRepository
	commissionRepo  *repository.CommissionRepository
	withdrawalRepo  *repository.WithdrawalRepository
	db              *gorm.DB
}

// NewDistributionAdminService 创建分销管理服务
func NewDistributionAdminService(
	distributorRepo *repository.DistributorRepository,
	commissionRepo *repository.CommissionRepository,
	withdrawalRepo *repository.WithdrawalRepository,
	db *gorm.DB,
) *DistributionAdminService {
	return &DistributionAdminService{
		distributorRepo: distributorRepo,
		commissionRepo:  commissionRepo,
		withdrawalRepo:  withdrawalRepo,
		db:              db,
	}
}

// DistributorListFilter 分销商列表过滤条件
type DistributorListFilter struct {
	Status    *int    `json:"status"`     // 状态
	Level     *int    `json:"level"`      // 层级
	Keyword   string  `json:"keyword"`    // 关键词（用户名/手机号）
	StartTime *string `json:"start_time"` // 开始时间
	EndTime   *string `json:"end_time"`   // 结束时间
}

// ListDistributors 获取分销商列表
func (s *DistributionAdminService) ListDistributors(ctx context.Context, offset, limit int, filter *DistributorListFilter) ([]*models.Distributor, int64, error) {
	filters := make(map[string]interface{})
	if filter != nil {
		if filter.Status != nil {
			filters["status"] = *filter.Status
		}
		if filter.Level != nil {
			filters["level"] = *filter.Level
		}
	}
	return s.distributorRepo.List(ctx, offset, limit, filters)
}

// GetDistributor 获取分销商详情
func (s *DistributionAdminService) GetDistributor(ctx context.Context, id int64) (*models.Distributor, error) {
	return s.distributorRepo.GetByIDWithUser(ctx, id)
}

// ApproveDistributor 审核通过分销商
func (s *DistributionAdminService) ApproveDistributor(ctx context.Context, distributorID, operatorID int64) error {
	now := time.Now()
	return s.db.Transaction(func(tx *gorm.DB) error {
		// 获取分销商信息
		var distributor models.Distributor
		if err := tx.First(&distributor, distributorID).Error; err != nil {
			return err
		}

		if distributor.Status != models.DistributorStatusPending {
			return nil // 已处理
		}

		// 更新状态
		if err := tx.Model(&models.Distributor{}).
			Where("id = ?", distributorID).
			Updates(map[string]interface{}{
				"status":      models.DistributorStatusApproved,
				"approved_at": now,
				"approved_by": operatorID,
			}).Error; err != nil {
			return err
		}

		// 更新上级的团队人数
		if distributor.ParentID != nil {
			// 更新直接上级的直推人数和团队人数
			if err := tx.Model(&models.Distributor{}).
				Where("id = ?", *distributor.ParentID).
				Updates(map[string]interface{}{
					"direct_count": gorm.Expr("direct_count + 1"),
					"team_count":   gorm.Expr("team_count + 1"),
				}).Error; err != nil {
				return err
			}

			// 递归更新所有上级的团队人数
			parentID := distributor.ParentID
			for {
				var parent models.Distributor
				if err := tx.First(&parent, *parentID).Error; err != nil {
					break
				}
				if parent.ParentID == nil {
					break
				}
				if err := tx.Model(&models.Distributor{}).
					Where("id = ?", *parent.ParentID).
					UpdateColumn("team_count", gorm.Expr("team_count + 1")).Error; err != nil {
					return err
				}
				parentID = parent.ParentID
			}
		}

		return nil
	})
}

// RejectDistributor 拒绝分销商申请
func (s *DistributionAdminService) RejectDistributor(ctx context.Context, distributorID, operatorID int64, reason string) error {
	now := time.Now()
	return s.db.Model(&models.Distributor{}).
		Where("id = ? AND status = ?", distributorID, models.DistributorStatusPending).
		Updates(map[string]interface{}{
			"status":      models.DistributorStatusRejected,
			"approved_at": now,
			"approved_by": operatorID,
		}).Error
}

// GetPendingDistributors 获取待审核分销商列表
func (s *DistributionAdminService) GetPendingDistributors(ctx context.Context, offset, limit int) ([]*models.Distributor, int64, error) {
	return s.distributorRepo.GetPendingList(ctx, offset, limit)
}

// CommissionListFilter 佣金列表过滤条件
type CommissionListFilter struct {
	DistributorID *int64  `json:"distributor_id"`
	Status        *int    `json:"status"`
	Type          *string `json:"type"`
	StartTime     *string `json:"start_time"`
	EndTime       *string `json:"end_time"`
}

// ListCommissions 获取佣金记录列表
func (s *DistributionAdminService) ListCommissions(ctx context.Context, offset, limit int, filter *CommissionListFilter) ([]*models.Commission, int64, error) {
	filters := make(map[string]interface{})
	if filter != nil {
		if filter.DistributorID != nil {
			filters["distributor_id"] = *filter.DistributorID
		}
		if filter.Status != nil {
			filters["status"] = *filter.Status
		}
		if filter.Type != nil {
			filters["type"] = *filter.Type
		}
	}
	return s.commissionRepo.List(ctx, offset, limit, filters)
}

// WithdrawalListFilter 提现列表过滤条件
type WithdrawalListFilter struct {
	UserID     *int64  `json:"user_id"`
	Status     *string `json:"status"`
	Type       *string `json:"type"`
	WithdrawTo *string `json:"withdraw_to"`
	StartTime  *string `json:"start_time"`
	EndTime    *string `json:"end_time"`
}

// ListWithdrawals 获取提现记录列表
func (s *DistributionAdminService) ListWithdrawals(ctx context.Context, offset, limit int, filter *WithdrawalListFilter) ([]*models.Withdrawal, int64, error) {
	filters := make(map[string]interface{})
	if filter != nil {
		if filter.UserID != nil {
			filters["user_id"] = *filter.UserID
		}
		if filter.Status != nil {
			filters["status"] = *filter.Status
		}
		if filter.Type != nil {
			filters["type"] = *filter.Type
		}
		if filter.WithdrawTo != nil {
			filters["withdraw_to"] = *filter.WithdrawTo
		}
	}
	return s.withdrawalRepo.List(ctx, offset, limit, filters)
}

// GetWithdrawal 获取提现详情
func (s *DistributionAdminService) GetWithdrawal(ctx context.Context, id int64) (*models.Withdrawal, error) {
	return s.withdrawalRepo.GetByIDWithRelations(ctx, id)
}

// ApproveWithdrawal 审核通过提现
func (s *DistributionAdminService) ApproveWithdrawal(ctx context.Context, withdrawalID, operatorID int64) error {
	return s.withdrawalRepo.Approve(ctx, withdrawalID, operatorID)
}

// RejectWithdrawal 拒绝提现
func (s *DistributionAdminService) RejectWithdrawal(ctx context.Context, withdrawalID, operatorID int64, reason string) error {
	withdrawal, err := s.withdrawalRepo.GetByID(ctx, withdrawalID)
	if err != nil {
		return err
	}

	return s.db.Transaction(func(tx *gorm.DB) error {
		// 更新提现状态
		now := time.Now()
		if err := tx.Model(&models.Withdrawal{}).
			Where("id = ? AND status = ?", withdrawalID, models.WithdrawalStatusPending).
			Updates(map[string]interface{}{
				"status":        models.WithdrawalStatusRejected,
				"operator_id":   operatorID,
				"processed_at":  now,
				"reject_reason": reason,
			}).Error; err != nil {
			return err
		}

		// 解冻余额
		if withdrawal.Type == models.WithdrawalTypeCommission {
			if err := tx.Model(&models.Distributor{}).
				Where("user_id = ?", withdrawal.UserID).
				Updates(map[string]interface{}{
					"available_commission": gorm.Expr("available_commission + ?", withdrawal.Amount),
					"frozen_commission":    gorm.Expr("frozen_commission - ?", withdrawal.Amount),
				}).Error; err != nil {
				return err
			}
		} else {
			if err := tx.Model(&models.UserWallet{}).
				Where("user_id = ?", withdrawal.UserID).
				Updates(map[string]interface{}{
					"balance":        gorm.Expr("balance + ?", withdrawal.Amount),
					"frozen_balance": gorm.Expr("frozen_balance - ?", withdrawal.Amount),
				}).Error; err != nil {
				return err
			}
		}

		return nil
	})
}

// ProcessWithdrawal 处理提现（开始打款）
func (s *DistributionAdminService) ProcessWithdrawal(ctx context.Context, withdrawalID int64) error {
	return s.withdrawalRepo.MarkProcessing(ctx, withdrawalID)
}

// CompleteWithdrawal 完成提现
func (s *DistributionAdminService) CompleteWithdrawal(ctx context.Context, withdrawalID int64) error {
	withdrawal, err := s.withdrawalRepo.GetByID(ctx, withdrawalID)
	if err != nil {
		return err
	}

	return s.db.Transaction(func(tx *gorm.DB) error {
		// 更新提现状态
		if err := tx.Model(&models.Withdrawal{}).
			Where("id = ? AND status = ?", withdrawalID, models.WithdrawalStatusProcessing).
			Update("status", models.WithdrawalStatusSuccess).Error; err != nil {
			return err
		}

		// 扣除冻结余额
		if withdrawal.Type == models.WithdrawalTypeCommission {
			if err := tx.Model(&models.Distributor{}).
				Where("user_id = ?", withdrawal.UserID).
				Updates(map[string]interface{}{
					"frozen_commission":    gorm.Expr("frozen_commission - ?", withdrawal.Amount),
					"withdrawn_commission": gorm.Expr("withdrawn_commission + ?", withdrawal.Amount),
				}).Error; err != nil {
				return err
			}
		} else {
			if err := tx.Model(&models.UserWallet{}).
				Where("user_id = ?", withdrawal.UserID).
				Updates(map[string]interface{}{
					"frozen_balance":  gorm.Expr("frozen_balance - ?", withdrawal.Amount),
					"total_withdrawn": gorm.Expr("total_withdrawn + ?", withdrawal.ActualAmount),
				}).Error; err != nil {
				return err
			}
		}

		return nil
	})
}

// DistributionStats 分销统计
type DistributionStats struct {
	TotalDistributors   int64   `json:"total_distributors"`   // 分销商总数
	PendingDistributors int64   `json:"pending_distributors"` // 待审核分销商
	ActiveDistributors  int64   `json:"active_distributors"`  // 活跃分销商
	TotalCommission     float64 `json:"total_commission"`     // 累计佣金
	PendingWithdrawals  int64   `json:"pending_withdrawals"`  // 待处理提现
	TotalWithdrawn      float64 `json:"total_withdrawn"`      // 已提现总额
}

// GetStats 获取分销统计数据
func (s *DistributionAdminService) GetStats(ctx context.Context) (*DistributionStats, error) {
	stats := &DistributionStats{}

	// 分销商总数
	if err := s.db.Model(&models.Distributor{}).
		Where("status = ?", models.DistributorStatusApproved).
		Count(&stats.TotalDistributors).Error; err != nil {
		return nil, err
	}

	// 待审核分销商
	if err := s.db.Model(&models.Distributor{}).
		Where("status = ?", models.DistributorStatusPending).
		Count(&stats.PendingDistributors).Error; err != nil {
		return nil, err
	}

	// 累计佣金
	if err := s.db.Model(&models.Commission{}).
		Select("COALESCE(SUM(amount), 0)").
		Scan(&stats.TotalCommission).Error; err != nil {
		return nil, err
	}

	// 待处理提现
	if err := s.db.Model(&models.Withdrawal{}).
		Where("status = ?", models.WithdrawalStatusPending).
		Count(&stats.PendingWithdrawals).Error; err != nil {
		return nil, err
	}

	// 已提现总额
	if err := s.db.Model(&models.Withdrawal{}).
		Select("COALESCE(SUM(actual_amount), 0)").
		Where("status = ?", models.WithdrawalStatusSuccess).
		Scan(&stats.TotalWithdrawn).Error; err != nil {
		return nil, err
	}

	return stats, nil
}

// GetPendingWithdrawals 获取待审核提现列表
func (s *DistributionAdminService) GetPendingWithdrawals(ctx context.Context, offset, limit int) ([]*models.Withdrawal, int64, error) {
	return s.withdrawalRepo.GetPendingList(ctx, offset, limit)
}

// GetApprovedWithdrawals 获取待打款提现列表
func (s *DistributionAdminService) GetApprovedWithdrawals(ctx context.Context, offset, limit int) ([]*models.Withdrawal, int64, error) {
	return s.withdrawalRepo.GetApprovedList(ctx, offset, limit)
}
