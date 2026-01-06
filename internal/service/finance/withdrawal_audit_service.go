// Package finance 提供财务管理服务
package finance

import (
	"context"
	"time"

	"gorm.io/gorm"

	"github.com/dumeirei/smart-locker-backend/internal/common/errors"
	"github.com/dumeirei/smart-locker-backend/internal/models"
	"github.com/dumeirei/smart-locker-backend/internal/repository"
)

// WithdrawalAuditService 提现审核服务
type WithdrawalAuditService struct {
	db              *gorm.DB
	withdrawalRepo  *repository.WithdrawalRepository
	distributorRepo *repository.DistributorRepository
}

// NewWithdrawalAuditService 创建提现审核服务
func NewWithdrawalAuditService(
	db *gorm.DB,
	withdrawalRepo *repository.WithdrawalRepository,
	distributorRepo *repository.DistributorRepository,
) *WithdrawalAuditService {
	return &WithdrawalAuditService{
		db:              db,
		withdrawalRepo:  withdrawalRepo,
		distributorRepo: distributorRepo,
	}
}

// WithdrawalListRequest 提现列表请求
type WithdrawalListRequest struct {
	UserID    *int64 `form:"user_id"`
	Type      string `form:"type"` // wallet/commission
	Status    string `form:"status"`
	StartDate string `form:"start_date"`
	EndDate   string `form:"end_date"`
	Page      int    `form:"page,default=1"`
	PageSize  int    `form:"page_size,default=20"`
}

// ListWithdrawals 获取提现列表
func (s *WithdrawalAuditService) ListWithdrawals(ctx context.Context, req *WithdrawalListRequest) ([]*models.Withdrawal, int64, error) {
	filters := make(map[string]interface{})

	if req.UserID != nil {
		filters["user_id"] = *req.UserID
	}
	if req.Type != "" {
		filters["type"] = req.Type
	}
	if req.Status != "" {
		filters["status"] = req.Status
	}
	if req.StartDate != "" {
		if t, err := time.Parse("2006-01-02", req.StartDate); err == nil {
			filters["start_time"] = t
		}
	}
	if req.EndDate != "" {
		if t, err := time.Parse("2006-01-02", req.EndDate); err == nil {
			endOfDay := t.Add(24*time.Hour - time.Second)
			filters["end_time"] = endOfDay
		}
	}

	offset := (req.Page - 1) * req.PageSize
	return s.withdrawalRepo.List(ctx, offset, req.PageSize, filters)
}

// GetWithdrawal 获取提现详情
func (s *WithdrawalAuditService) GetWithdrawal(ctx context.Context, id int64) (*models.Withdrawal, error) {
	withdrawal, err := s.withdrawalRepo.GetByIDWithRelations(ctx, id)
	if err != nil {
		return nil, errors.ErrWithdrawalNotFound.WithError(err)
	}
	return withdrawal, nil
}

// ApproveWithdrawal 审核通过提现
func (s *WithdrawalAuditService) ApproveWithdrawal(ctx context.Context, id int64, operatorID int64) error {
	withdrawal, err := s.withdrawalRepo.GetByID(ctx, id)
	if err != nil {
		return errors.ErrWithdrawalNotFound.WithError(err)
	}

	if withdrawal.Status != models.WithdrawalStatusPending {
		return errors.ErrWithdrawalStatus.WithMessage("只能审核待审核状态的提现申请")
	}

	return s.withdrawalRepo.Approve(ctx, id, operatorID)
}

// RejectWithdrawal 审核拒绝提现
func (s *WithdrawalAuditService) RejectWithdrawal(ctx context.Context, id int64, operatorID int64, reason string) error {
	withdrawal, err := s.withdrawalRepo.GetByID(ctx, id)
	if err != nil {
		return errors.ErrWithdrawalNotFound.WithError(err)
	}

	if withdrawal.Status != models.WithdrawalStatusPending {
		return errors.ErrWithdrawalStatus.WithMessage("只能审核待审核状态的提现申请")
	}

	// 开始事务
	tx := s.db.WithContext(ctx).Begin()
	defer func() {
		if r := recover(); r != nil {
			tx.Rollback()
		}
	}()

	// 更新提现状态为已拒绝
	now := time.Now()
	err = tx.Model(&models.Withdrawal{}).
		Where("id = ?", id).
		Updates(map[string]interface{}{
			"status":        models.WithdrawalStatusRejected,
			"operator_id":   operatorID,
			"processed_at":  &now,
			"reject_reason": reason,
		}).Error
	if err != nil {
		tx.Rollback()
		return errors.ErrDatabaseError.WithError(err)
	}

	// 退还金额到可提现余额
	if withdrawal.Type == models.WithdrawalTypeCommission {
		// 退还分销佣金
		err = tx.Model(&models.Distributor{}).
			Where("user_id = ?", withdrawal.UserID).
			Updates(map[string]interface{}{
				"available_commission": gorm.Expr("available_commission + ?", withdrawal.Amount),
				"frozen_commission":    gorm.Expr("frozen_commission - ?", withdrawal.Amount),
			}).Error
		if err != nil {
			tx.Rollback()
			return errors.ErrDatabaseError.WithError(err)
		}
	} else {
		// 退还钱包余额
		err = tx.Model(&models.UserWallet{}).
			Where("user_id = ?", withdrawal.UserID).
			Updates(map[string]interface{}{
				"balance":        gorm.Expr("balance + ?", withdrawal.Amount),
				"frozen_balance": gorm.Expr("frozen_balance - ?", withdrawal.Amount),
			}).Error
		if err != nil {
			tx.Rollback()
			return errors.ErrDatabaseError.WithError(err)
		}
	}

	return tx.Commit().Error
}

// ProcessWithdrawal 处理提现（打款中）
func (s *WithdrawalAuditService) ProcessWithdrawal(ctx context.Context, id int64, operatorID int64) error {
	withdrawal, err := s.withdrawalRepo.GetByID(ctx, id)
	if err != nil {
		return errors.ErrWithdrawalNotFound.WithError(err)
	}

	if withdrawal.Status != models.WithdrawalStatusApproved {
		return errors.ErrWithdrawalStatus.WithMessage("只能处理已审核通过的提现申请")
	}

	return s.withdrawalRepo.MarkProcessing(ctx, id)
}

// CompleteWithdrawal 完成提现
func (s *WithdrawalAuditService) CompleteWithdrawal(ctx context.Context, id int64, operatorID int64) error {
	withdrawal, err := s.withdrawalRepo.GetByID(ctx, id)
	if err != nil {
		return errors.ErrWithdrawalNotFound.WithError(err)
	}

	if withdrawal.Status != models.WithdrawalStatusProcessing && withdrawal.Status != models.WithdrawalStatusApproved {
		return errors.ErrWithdrawalStatus.WithMessage("只能完成打款中或已审核的提现申请")
	}

	// 开始事务
	tx := s.db.WithContext(ctx).Begin()
	defer func() {
		if r := recover(); r != nil {
			tx.Rollback()
		}
	}()

	// 更新状态为已完成
	now := time.Now()
	err = tx.Model(&models.Withdrawal{}).
		Where("id = ?", id).
		Updates(map[string]interface{}{
			"status":       models.WithdrawalStatusSuccess,
			"operator_id":  operatorID,
			"processed_at": &now,
		}).Error
	if err != nil {
		tx.Rollback()
		return errors.ErrDatabaseError.WithError(err)
	}

	// 扣除冻结金额
	if withdrawal.Type == models.WithdrawalTypeCommission {
		// 扣除分销冻结佣金，更新已提现金额
		err = tx.Model(&models.Distributor{}).
			Where("user_id = ?", withdrawal.UserID).
			Updates(map[string]interface{}{
				"frozen_commission":    gorm.Expr("frozen_commission - ?", withdrawal.Amount),
				"withdrawn_commission": gorm.Expr("withdrawn_commission + ?", withdrawal.ActualAmount),
			}).Error
		if err != nil {
			tx.Rollback()
			return errors.ErrDatabaseError.WithError(err)
		}
	} else {
		// 扣除钱包冻结余额，更新已提现金额
		err = tx.Model(&models.UserWallet{}).
			Where("user_id = ?", withdrawal.UserID).
			Updates(map[string]interface{}{
				"frozen_balance":  gorm.Expr("frozen_balance - ?", withdrawal.Amount),
				"total_withdrawn": gorm.Expr("total_withdrawn + ?", withdrawal.ActualAmount),
			}).Error
		if err != nil {
			tx.Rollback()
			return errors.ErrDatabaseError.WithError(err)
		}
	}

	// 创建钱包交易记录
	remark := "提现到账"
	transaction := &models.WalletTransaction{
		UserID:        withdrawal.UserID,
		Type:          models.WalletTxTypeWithdraw,
		Amount:        withdrawal.ActualAmount,
		BalanceBefore: 0, // 这里简化处理
		BalanceAfter:  0,
		OrderNo:       &withdrawal.WithdrawalNo,
		Remark:        &remark,
	}
	if err := tx.Create(transaction).Error; err != nil {
		tx.Rollback()
		return errors.ErrDatabaseError.WithError(err)
	}

	return tx.Commit().Error
}

// GetPendingWithdrawalsCount 获取待审核提现数量
func (s *WithdrawalAuditService) GetPendingWithdrawalsCount(ctx context.Context) (int64, error) {
	return s.withdrawalRepo.CountByStatus(ctx, models.WithdrawalStatusPending)
}

// GetWithdrawalSummary 获取提现汇总统计
func (s *WithdrawalAuditService) GetWithdrawalSummary(ctx context.Context, startDate, endDate *time.Time) (*models.WithdrawalSummary, error) {
	var summary models.WithdrawalSummary

	query := s.db.WithContext(ctx).Model(&models.Withdrawal{})
	if startDate != nil {
		query = query.Where("created_at >= ?", *startDate)
	}
	if endDate != nil {
		query = query.Where("created_at <= ?", *endDate)
	}

	// 总数
	var totalCount int64
	err := query.Count(&totalCount).Error
	if err != nil {
		return nil, err
	}
	summary.TotalWithdrawals = int(totalCount)

	// 总金额
	err = s.db.WithContext(ctx).Model(&models.Withdrawal{}).
		Select("COALESCE(SUM(amount), 0)").
		Row().Scan(&summary.TotalAmount)
	if err != nil {
		return nil, err
	}

	// 待审核
	var pendingCount int64
	err = s.db.WithContext(ctx).Model(&models.Withdrawal{}).
		Where("status = ?", models.WithdrawalStatusPending).
		Count(&pendingCount).Error
	if err != nil {
		return nil, err
	}
	summary.PendingCount = int(pendingCount)

	err = s.db.WithContext(ctx).Model(&models.Withdrawal{}).
		Where("status = ?", models.WithdrawalStatusPending).
		Select("COALESCE(SUM(amount), 0)").
		Row().Scan(&summary.PendingAmount)
	if err != nil {
		return nil, err
	}

	// 已通过
	var approvedCount int64
	err = s.db.WithContext(ctx).Model(&models.Withdrawal{}).
		Where("status = ?", models.WithdrawalStatusSuccess).
		Count(&approvedCount).Error
	if err != nil {
		return nil, err
	}
	summary.ApprovedCount = int(approvedCount)

	err = s.db.WithContext(ctx).Model(&models.Withdrawal{}).
		Where("status = ?", models.WithdrawalStatusSuccess).
		Select("COALESCE(SUM(actual_amount), 0)").
		Row().Scan(&summary.ApprovedAmount)
	if err != nil {
		return nil, err
	}

	// 已拒绝
	var rejectedCount int64
	err = s.db.WithContext(ctx).Model(&models.Withdrawal{}).
		Where("status = ?", models.WithdrawalStatusRejected).
		Count(&rejectedCount).Error
	if err != nil {
		return nil, err
	}
	summary.RejectedCount = int(rejectedCount)

	return &summary, nil
}

// GetPendingWithdrawals 获取待审核提现列表
func (s *WithdrawalAuditService) GetPendingWithdrawals(ctx context.Context, page, pageSize int) ([]*models.Withdrawal, int64, error) {
	offset := (page - 1) * pageSize
	return s.withdrawalRepo.GetPendingList(ctx, offset, pageSize)
}

// GetApprovedWithdrawals 获取待打款提现列表
func (s *WithdrawalAuditService) GetApprovedWithdrawals(ctx context.Context, page, pageSize int) ([]*models.Withdrawal, int64, error) {
	offset := (page - 1) * pageSize
	return s.withdrawalRepo.GetApprovedList(ctx, offset, pageSize)
}

// BatchApprove 批量审核通过
func (s *WithdrawalAuditService) BatchApprove(ctx context.Context, ids []int64, operatorID int64) error {
	for _, id := range ids {
		if err := s.ApproveWithdrawal(ctx, id, operatorID); err != nil {
			// 记录错误但继续处理其他
			continue
		}
	}
	return nil
}

// BatchReject 批量审核拒绝
func (s *WithdrawalAuditService) BatchReject(ctx context.Context, ids []int64, operatorID int64, reason string) error {
	for _, id := range ids {
		if err := s.RejectWithdrawal(ctx, id, operatorID, reason); err != nil {
			// 记录错误但继续处理其他
			continue
		}
	}
	return nil
}

// BatchComplete 批量完成提现
func (s *WithdrawalAuditService) BatchComplete(ctx context.Context, ids []int64, operatorID int64) error {
	for _, id := range ids {
		if err := s.CompleteWithdrawal(ctx, id, operatorID); err != nil {
			// 记录错误但继续处理其他
			continue
		}
	}
	return nil
}
