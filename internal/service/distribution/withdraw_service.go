// Package distribution 分销服务
package distribution

import (
	"context"
	"errors"
	"fmt"
	"time"

	"gorm.io/gorm"

	"github.com/dumeirei/smart-locker-backend/internal/models"
	"github.com/dumeirei/smart-locker-backend/internal/repository"
)

// 提现相关常量
const (
	DefaultMinWithdraw  = 10.0  // 默认最低提现金额
	DefaultWithdrawFee  = 0.006 // 默认提现手续费比例 0.6%
	MaxWithdrawPerDay   = 3     // 每日最大提现次数
	MaxPendingWithdraw  = 5     // 最大待处理提现数
)

// WithdrawService 提现服务
type WithdrawService struct {
	withdrawalRepo  *repository.WithdrawalRepository
	distributorRepo *repository.DistributorRepository
	userRepo        *repository.UserRepository
	db              *gorm.DB
	minWithdraw     float64 // 最低提现金额
	withdrawFee     float64 // 提现手续费比例
}

// NewWithdrawService 创建提现服务
func NewWithdrawService(
	withdrawalRepo *repository.WithdrawalRepository,
	distributorRepo *repository.DistributorRepository,
	userRepo *repository.UserRepository,
	db *gorm.DB,
) *WithdrawService {
	return &WithdrawService{
		withdrawalRepo:  withdrawalRepo,
		distributorRepo: distributorRepo,
		userRepo:        userRepo,
		db:              db,
		minWithdraw:     DefaultMinWithdraw,
		withdrawFee:     DefaultWithdrawFee,
	}
}

// SetConfig 设置提现配置
func (s *WithdrawService) SetConfig(minWithdraw, withdrawFee float64) {
	s.minWithdraw = minWithdraw
	s.withdrawFee = withdrawFee
}

// WithdrawRequest 提现请求
type WithdrawRequest struct {
	UserID       int64   `json:"user_id"`
	Type         string  `json:"type"`          // wallet/commission
	Amount       float64 `json:"amount"`        // 提现金额
	WithdrawTo   string  `json:"withdraw_to"`   // wechat/alipay/bank
	AccountInfo  string  `json:"account_info"`  // 账户信息（JSON格式）
}

// WithdrawResponse 提现响应
type WithdrawResponse struct {
	Withdrawal   *models.Withdrawal `json:"withdrawal"`
	Fee          float64            `json:"fee"`           // 手续费
	ActualAmount float64            `json:"actual_amount"` // 实际到账金额
	Message      string             `json:"message"`
}

// Apply 申请提现
func (s *WithdrawService) Apply(ctx context.Context, req *WithdrawRequest) (*WithdrawResponse, error) {
	// 验证提现类型
	if req.Type != models.WithdrawalTypeWallet && req.Type != models.WithdrawalTypeCommission {
		return nil, errors.New("无效的提现类型")
	}

	// 验证提现方式
	if req.WithdrawTo != models.WithdrawToWechat &&
		req.WithdrawTo != models.WithdrawToAlipay &&
		req.WithdrawTo != models.WithdrawToBank {
		return nil, errors.New("无效的提现方式")
	}

	// 验证最低提现金额
	if req.Amount < s.minWithdraw {
		return nil, fmt.Errorf("最低提现金额为%.2f元", s.minWithdraw)
	}

	// 检查待处理提现数量
	pendingCount, err := s.withdrawalRepo.CountPendingByUserID(ctx, req.UserID)
	if err != nil {
		return nil, err
	}
	if pendingCount >= MaxPendingWithdraw {
		return nil, errors.New("您有太多待处理的提现申请，请等待处理后再申请")
	}

	// 根据提现类型验证余额
	var availableBalance float64
	if req.Type == models.WithdrawalTypeCommission {
		// 佣金提现，检查分销商可用佣金
		distributor, err := s.distributorRepo.GetByUserID(ctx, req.UserID)
		if err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return nil, errors.New("您还不是分销商")
			}
			return nil, err
		}
		if distributor.Status != models.DistributorStatusApproved {
			return nil, errors.New("分销商尚未审核通过")
		}
		availableBalance = distributor.AvailableCommission
	} else {
		// 钱包提现，检查用户钱包余额
		user, err := s.userRepo.GetByIDWithWallet(ctx, req.UserID)
		if err != nil {
			return nil, err
		}
		if user.Wallet != nil {
			availableBalance = user.Wallet.Balance
		}
	}

	if availableBalance < req.Amount {
		return nil, fmt.Errorf("可提现余额不足，当前可提现: %.2f元", availableBalance)
	}

	// 计算手续费和实际到账金额
	fee := req.Amount * s.withdrawFee
	actualAmount := req.Amount - fee

	// 生成提现单号
	withdrawalNo := s.generateWithdrawalNo()

	// 创建提现记录
	withdrawal := &models.Withdrawal{
		WithdrawalNo:         withdrawalNo,
		UserID:               req.UserID,
		Type:                 req.Type,
		Amount:               req.Amount,
		Fee:                  fee,
		ActualAmount:         actualAmount,
		WithdrawTo:           req.WithdrawTo,
		AccountInfoEncrypted: req.AccountInfo, // 实际应该加密存储
		Status:               models.WithdrawalStatusPending,
	}

	// 使用事务处理
	err = s.db.Transaction(func(tx *gorm.DB) error {
		// 冻结余额
		if req.Type == models.WithdrawalTypeCommission {
			// 冻结佣金
			result := tx.Model(&models.Distributor{}).
				Where("user_id = ? AND available_commission >= ?", req.UserID, req.Amount).
				Updates(map[string]interface{}{
					"available_commission": gorm.Expr("available_commission - ?", req.Amount),
					"frozen_commission":    gorm.Expr("frozen_commission + ?", req.Amount),
				})
			if result.Error != nil {
				return result.Error
			}
			if result.RowsAffected == 0 {
				return errors.New("余额不足")
			}
		} else {
			// 冻结钱包余额
			result := tx.Model(&models.UserWallet{}).
				Where("user_id = ? AND balance >= ?", req.UserID, req.Amount).
				Updates(map[string]interface{}{
					"balance":        gorm.Expr("balance - ?", req.Amount),
					"frozen_balance": gorm.Expr("frozen_balance + ?", req.Amount),
				})
			if result.Error != nil {
				return result.Error
			}
			if result.RowsAffected == 0 {
				return errors.New("余额不足")
			}
		}

		// 创建提现记录
		if err := tx.Create(withdrawal).Error; err != nil {
			return err
		}

		return nil
	})

	if err != nil {
		return nil, err
	}

	return &WithdrawResponse{
		Withdrawal:   withdrawal,
		Fee:          fee,
		ActualAmount: actualAmount,
		Message:      "提现申请已提交，请等待审核",
	}, nil
}

// generateWithdrawalNo 生成提现单号
func (s *WithdrawService) generateWithdrawalNo() string {
	return fmt.Sprintf("W%s%06d", time.Now().Format("20060102150405"), time.Now().Nanosecond()/1000%1000000)
}

// Approve 审核通过
func (s *WithdrawService) Approve(ctx context.Context, withdrawalID int64, operatorID int64) error {
	withdrawal, err := s.withdrawalRepo.GetByID(ctx, withdrawalID)
	if err != nil {
		return err
	}

	if withdrawal.Status != models.WithdrawalStatusPending {
		return errors.New("该提现申请已处理")
	}

	return s.withdrawalRepo.Approve(ctx, withdrawalID, operatorID)
}

// Reject 审核拒绝
func (s *WithdrawService) Reject(ctx context.Context, withdrawalID int64, operatorID int64, reason string) error {
	withdrawal, err := s.withdrawalRepo.GetByID(ctx, withdrawalID)
	if err != nil {
		return err
	}

	if withdrawal.Status != models.WithdrawalStatusPending {
		return errors.New("该提现申请已处理")
	}

	// 拒绝需要解冻余额
	return s.db.Transaction(func(tx *gorm.DB) error {
		// 更新提现状态
		now := time.Now()
		if err := tx.Model(&models.Withdrawal{}).
			Where("id = ?", withdrawalID).
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

// Process 处理提现（打款）
func (s *WithdrawService) Process(ctx context.Context, withdrawalID int64) error {
	withdrawal, err := s.withdrawalRepo.GetByID(ctx, withdrawalID)
	if err != nil {
		return err
	}

	if withdrawal.Status != models.WithdrawalStatusApproved {
		return errors.New("该提现申请状态不正确")
	}

	// 标记为处理中
	return s.withdrawalRepo.MarkProcessing(ctx, withdrawalID)
}

// Complete 完成提现
func (s *WithdrawService) Complete(ctx context.Context, withdrawalID int64) error {
	withdrawal, err := s.withdrawalRepo.GetByID(ctx, withdrawalID)
	if err != nil {
		return err
	}

	if withdrawal.Status != models.WithdrawalStatusProcessing {
		return errors.New("该提现申请状态不正确")
	}

	// 使用事务处理
	return s.db.Transaction(func(tx *gorm.DB) error {
		// 更新提现状态
		if err := tx.Model(&models.Withdrawal{}).
			Where("id = ?", withdrawalID).
			Update("status", models.WithdrawalStatusSuccess).Error; err != nil {
			return err
		}

		// 扣除冻结余额，增加已提现金额
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
					"frozen_balance":   gorm.Expr("frozen_balance - ?", withdrawal.Amount),
					"total_withdrawn":  gorm.Expr("total_withdrawn + ?", withdrawal.ActualAmount),
				}).Error; err != nil {
				return err
			}
		}

		return nil
	})
}

// GetByUserID 获取用户的提现记录
func (s *WithdrawService) GetByUserID(ctx context.Context, userID int64, offset, limit int) ([]*models.Withdrawal, int64, error) {
	return s.withdrawalRepo.GetByUserID(ctx, userID, offset, limit)
}

// GetByID 获取提现详情
func (s *WithdrawService) GetByID(ctx context.Context, id int64) (*models.Withdrawal, error) {
	return s.withdrawalRepo.GetByIDWithRelations(ctx, id)
}

// List 获取提现列表
func (s *WithdrawService) List(ctx context.Context, offset, limit int, filters map[string]interface{}) ([]*models.Withdrawal, int64, error) {
	return s.withdrawalRepo.List(ctx, offset, limit, filters)
}

// GetPendingList 获取待审核列表
func (s *WithdrawService) GetPendingList(ctx context.Context, offset, limit int) ([]*models.Withdrawal, int64, error) {
	return s.withdrawalRepo.GetPendingList(ctx, offset, limit)
}

// GetApprovedList 获取已审批待打款列表
func (s *WithdrawService) GetApprovedList(ctx context.Context, offset, limit int) ([]*models.Withdrawal, int64, error) {
	return s.withdrawalRepo.GetApprovedList(ctx, offset, limit)
}

// GetStats 获取用户提现统计
func (s *WithdrawService) GetStats(ctx context.Context, userID int64) (map[string]interface{}, error) {
	return s.withdrawalRepo.GetStatsByUserID(ctx, userID)
}

// GetConfig 获取提现配置
func (s *WithdrawService) GetConfig() map[string]interface{} {
	return map[string]interface{}{
		"min_withdraw":       s.minWithdraw,
		"withdraw_fee":       s.withdrawFee,
		"withdraw_fee_desc":  fmt.Sprintf("%.1f%%", s.withdrawFee*100),
		"max_pending":        MaxPendingWithdraw,
		"support_methods":    []string{models.WithdrawToWechat, models.WithdrawToAlipay, models.WithdrawToBank},
	}
}
