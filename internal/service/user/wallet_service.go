// Package user 提供用户服务
package user

import (
	"context"
	"fmt"
	"time"

	"gorm.io/gorm"

	"github.com/dumeirei/smart-locker-backend/internal/common/errors"
	"github.com/dumeirei/smart-locker-backend/internal/common/utils"
	"github.com/dumeirei/smart-locker-backend/internal/models"
	"github.com/dumeirei/smart-locker-backend/internal/repository"
)

// WalletService 钱包服务
type WalletService struct {
	db       *gorm.DB
	userRepo *repository.UserRepository
}

// NewWalletService 创建钱包服务
func NewWalletService(db *gorm.DB, userRepo *repository.UserRepository) *WalletService {
	return &WalletService{
		db:       db,
		userRepo: userRepo,
	}
}

// WalletInfo 钱包信息
type WalletInfo struct {
	Balance        float64 `json:"balance"`
	FrozenBalance  float64 `json:"frozen_balance"`
	TotalRecharged float64 `json:"total_recharged"`
	TotalConsumed  float64 `json:"total_consumed"`
}

// TransactionRecord 交易记录
type TransactionRecord struct {
	ID            int64     `json:"id"`
	Type          string    `json:"type"`
	TypeName      string    `json:"type_name"`
	Amount        float64   `json:"amount"`
	BalanceBefore float64   `json:"balance_before"`
	BalanceAfter  float64   `json:"balance_after"`
	OrderNo       *string   `json:"order_no,omitempty"`
	Remark        *string   `json:"remark,omitempty"`
	CreatedAt     time.Time `json:"created_at"`
}

// GetWallet 获取钱包信息
func (s *WalletService) GetWallet(ctx context.Context, userID int64) (*WalletInfo, error) {
	var wallet models.UserWallet
	err := s.db.WithContext(ctx).Where("user_id = ?", userID).First(&wallet).Error
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			// 钱包不存在，创建一个
			wallet = models.UserWallet{UserID: userID}
			if err := s.db.WithContext(ctx).Create(&wallet).Error; err != nil {
				return nil, errors.ErrDatabaseError.WithError(err)
			}
		} else {
			return nil, errors.ErrDatabaseError.WithError(err)
		}
	}

	return &WalletInfo{
		Balance:        wallet.Balance,
		FrozenBalance:  wallet.FrozenBalance,
		TotalRecharged: wallet.TotalRecharged,
		TotalConsumed:  wallet.TotalConsumed,
	}, nil
}

// GetTransactions 获取交易记录
func (s *WalletService) GetTransactions(ctx context.Context, userID int64, offset, limit int, txType string) ([]*TransactionRecord, int64, error) {
	var transactions []*models.WalletTransaction
	var total int64

	query := s.db.WithContext(ctx).Model(&models.WalletTransaction{}).Where("user_id = ?", userID)

	if txType != "" {
		query = query.Where("type = ?", txType)
	}

	if err := query.Count(&total).Error; err != nil {
		return nil, 0, errors.ErrDatabaseError.WithError(err)
	}

	if err := query.Order("id DESC").Offset(offset).Limit(limit).Find(&transactions).Error; err != nil {
		return nil, 0, errors.ErrDatabaseError.WithError(err)
	}

	records := make([]*TransactionRecord, len(transactions))
	for i, tx := range transactions {
		records[i] = &TransactionRecord{
			ID:            tx.ID,
			Type:          tx.Type,
			TypeName:      s.getTypeName(tx.Type),
			Amount:        tx.Amount,
			BalanceBefore: tx.BalanceBefore,
			BalanceAfter:  tx.BalanceAfter,
			OrderNo:       tx.OrderNo,
			Remark:        tx.Remark,
			CreatedAt:     tx.CreatedAt,
		}
	}

	return records, total, nil
}

// getTypeName 获取交易类型名称
func (s *WalletService) getTypeName(txType string) string {
	switch txType {
	case models.WalletTxTypeRecharge:
		return "充值"
	case models.WalletTxTypeConsume:
		return "消费"
	case models.WalletTxTypeRefund:
		return "退款"
	case models.WalletTxTypeWithdraw:
		return "提现"
	case models.WalletTxTypeDeposit:
		return "押金冻结"
	case models.WalletTxTypeReturnDeposit:
		return "押金退还"
	default:
		return "其他"
	}
}

// Recharge 充值（增加余额）
func (s *WalletService) Recharge(ctx context.Context, userID int64, amount float64, orderNo string) error {
	if amount <= 0 {
		return errors.ErrInvalidParams.WithMessage("充值金额必须大于0")
	}

	return s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		return s.RechargeTx(ctx, tx, userID, amount, orderNo)
	})
}

// RechargeTx 在已有事务中充值（增加余额）
func (s *WalletService) RechargeTx(ctx context.Context, tx *gorm.DB, userID int64, amount float64, orderNo string) error {
	if amount <= 0 {
		return errors.ErrInvalidParams.WithMessage("充值金额必须大于0")
	}

	var wallet models.UserWallet
	if err := tx.WithContext(ctx).Set("gorm:query_option", "FOR UPDATE").
		Where("user_id = ?", userID).First(&wallet).Error; err != nil {
		return errors.ErrDatabaseError.WithError(err)
	}

	balanceBefore := wallet.Balance
	balanceAfter := balanceBefore + amount

	if err := tx.WithContext(ctx).Model(&wallet).Updates(map[string]interface{}{
		"balance":         balanceAfter,
		"total_recharged": gorm.Expr("total_recharged + ?", amount),
	}).Error; err != nil {
		return errors.ErrDatabaseError.WithError(err)
	}

	transaction := &models.WalletTransaction{
		UserID:        userID,
		Type:          models.WalletTxTypeRecharge,
		Amount:        amount,
		BalanceBefore: balanceBefore,
		BalanceAfter:  balanceAfter,
		OrderNo:       &orderNo,
		Remark:        utils.StringPtr("余额充值"),
	}
	if err := tx.WithContext(ctx).Create(transaction).Error; err != nil {
		return errors.ErrDatabaseError.WithError(err)
	}

	return nil
}

// Consume 消费（扣减余额）
func (s *WalletService) Consume(ctx context.Context, userID int64, amount float64, orderNo string) error {
	if amount <= 0 {
		return errors.ErrInvalidParams.WithMessage("消费金额必须大于0")
	}

	return s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		return s.ConsumeTx(ctx, tx, userID, amount, orderNo)
	})
}

// ConsumeTx 在已有事务中消费（扣减余额）
func (s *WalletService) ConsumeTx(ctx context.Context, tx *gorm.DB, userID int64, amount float64, orderNo string) error {
	if amount <= 0 {
		return errors.ErrInvalidParams.WithMessage("消费金额必须大于0")
	}

	var wallet models.UserWallet
	if err := tx.WithContext(ctx).Set("gorm:query_option", "FOR UPDATE").
		Where("user_id = ?", userID).First(&wallet).Error; err != nil {
		return errors.ErrDatabaseError.WithError(err)
	}

	if wallet.Balance < amount {
		return errors.ErrBalanceInsufficient
	}

	balanceBefore := wallet.Balance
	balanceAfter := balanceBefore - amount

	if err := tx.WithContext(ctx).Model(&wallet).Updates(map[string]interface{}{
		"balance":        balanceAfter,
		"total_consumed": gorm.Expr("total_consumed + ?", amount),
	}).Error; err != nil {
		return errors.ErrDatabaseError.WithError(err)
	}

	transaction := &models.WalletTransaction{
		UserID:        userID,
		Type:          models.WalletTxTypeConsume,
		Amount:        -amount,
		BalanceBefore: balanceBefore,
		BalanceAfter:  balanceAfter,
		OrderNo:       &orderNo,
		Remark:        utils.StringPtr("余额消费"),
	}
	if err := tx.WithContext(ctx).Create(transaction).Error; err != nil {
		return errors.ErrDatabaseError.WithError(err)
	}

	return nil
}

// Refund 退款（增加余额）
func (s *WalletService) Refund(ctx context.Context, userID int64, amount float64, orderNo string) error {
	if amount <= 0 {
		return errors.ErrInvalidParams.WithMessage("退款金额必须大于0")
	}

	return s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		return s.RefundTx(ctx, tx, userID, amount, orderNo)
	})
}

// RefundTx 在已有事务中退款（增加余额）
func (s *WalletService) RefundTx(ctx context.Context, tx *gorm.DB, userID int64, amount float64, orderNo string) error {
	if amount <= 0 {
		return errors.ErrInvalidParams.WithMessage("退款金额必须大于0")
	}

	var wallet models.UserWallet
	if err := tx.WithContext(ctx).Set("gorm:query_option", "FOR UPDATE").
		Where("user_id = ?", userID).First(&wallet).Error; err != nil {
		return errors.ErrDatabaseError.WithError(err)
	}

	balanceBefore := wallet.Balance
	balanceAfter := balanceBefore + amount

	if err := tx.WithContext(ctx).Model(&wallet).Update("balance", balanceAfter).Error; err != nil {
		return errors.ErrDatabaseError.WithError(err)
	}

	transaction := &models.WalletTransaction{
		UserID:        userID,
		Type:          models.WalletTxTypeRefund,
		Amount:        amount,
		BalanceBefore: balanceBefore,
		BalanceAfter:  balanceAfter,
		OrderNo:       &orderNo,
		Remark:        utils.StringPtr("订单退款"),
	}
	if err := tx.WithContext(ctx).Create(transaction).Error; err != nil {
		return errors.ErrDatabaseError.WithError(err)
	}

	return nil
}

// FreezeDeposit 冻结押金
func (s *WalletService) FreezeDeposit(ctx context.Context, userID int64, amount float64, orderNo string) error {
	if amount <= 0 {
		return errors.ErrInvalidParams.WithMessage("押金金额必须大于0")
	}

	return s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		return s.FreezeDepositTx(ctx, tx, userID, amount, orderNo)
	})
}

// FreezeDepositTx 在已有事务中冻结押金
func (s *WalletService) FreezeDepositTx(ctx context.Context, tx *gorm.DB, userID int64, amount float64, orderNo string) error {
	if amount <= 0 {
		return errors.ErrInvalidParams.WithMessage("押金金额必须大于0")
	}

	var wallet models.UserWallet
	if err := tx.WithContext(ctx).Set("gorm:query_option", "FOR UPDATE").
		Where("user_id = ?", userID).First(&wallet).Error; err != nil {
		return errors.ErrDatabaseError.WithError(err)
	}

	if wallet.Balance < amount {
		return errors.ErrBalanceInsufficient
	}

	balanceBefore := wallet.Balance
	balanceAfter := balanceBefore - amount

	if err := tx.WithContext(ctx).Model(&wallet).Updates(map[string]interface{}{
		"balance":        balanceAfter,
		"frozen_balance": gorm.Expr("frozen_balance + ?", amount),
	}).Error; err != nil {
		return errors.ErrDatabaseError.WithError(err)
	}

	transaction := &models.WalletTransaction{
		UserID:        userID,
		Type:          models.WalletTxTypeDeposit,
		Amount:        -amount,
		BalanceBefore: balanceBefore,
		BalanceAfter:  balanceAfter,
		OrderNo:       &orderNo,
		Remark:        utils.StringPtr("押金冻结"),
	}
	if err := tx.WithContext(ctx).Create(transaction).Error; err != nil {
		return errors.ErrDatabaseError.WithError(err)
	}

	return nil
}

// UnfreezeDeposit 解冻押金（退还）
func (s *WalletService) UnfreezeDeposit(ctx context.Context, userID int64, amount float64, orderNo string) error {
	if amount <= 0 {
		return errors.ErrInvalidParams.WithMessage("押金金额必须大于0")
	}

	return s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		return s.UnfreezeDepositTx(ctx, tx, userID, amount, orderNo)
	})
}

// UnfreezeDepositTx 在已有事务中解冻押金（退还）
func (s *WalletService) UnfreezeDepositTx(ctx context.Context, tx *gorm.DB, userID int64, amount float64, orderNo string) error {
	if amount <= 0 {
		return errors.ErrInvalidParams.WithMessage("押金金额必须大于0")
	}

	var wallet models.UserWallet
	if err := tx.WithContext(ctx).Set("gorm:query_option", "FOR UPDATE").
		Where("user_id = ?", userID).First(&wallet).Error; err != nil {
		return errors.ErrDatabaseError.WithError(err)
	}

	if wallet.FrozenBalance < amount {
		return errors.New(errors.ErrOperationFailed.Code, "冻结余额不足")
	}

	balanceBefore := wallet.Balance
	balanceAfter := balanceBefore + amount

	if err := tx.WithContext(ctx).Model(&wallet).Updates(map[string]interface{}{
		"balance":        balanceAfter,
		"frozen_balance": gorm.Expr("frozen_balance - ?", amount),
	}).Error; err != nil {
		return errors.ErrDatabaseError.WithError(err)
	}

	transaction := &models.WalletTransaction{
		UserID:        userID,
		Type:          models.WalletTxTypeReturnDeposit,
		Amount:        amount,
		BalanceBefore: balanceBefore,
		BalanceAfter:  balanceAfter,
		OrderNo:       &orderNo,
		Remark:        utils.StringPtr("押金退还"),
	}
	if err := tx.WithContext(ctx).Create(transaction).Error; err != nil {
		return errors.ErrDatabaseError.WithError(err)
	}

	return nil
}

// CheckBalance 检查余额是否充足
func (s *WalletService) CheckBalance(ctx context.Context, userID int64, amount float64) (bool, error) {
	var wallet models.UserWallet
	err := s.db.WithContext(ctx).Where("user_id = ?", userID).First(&wallet).Error
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return false, nil
		}
		return false, errors.ErrDatabaseError.WithError(err)
	}

	return wallet.Balance >= amount, nil
}

// GetBalance 获取余额
func (s *WalletService) GetBalance(ctx context.Context, userID int64) (float64, error) {
	var wallet models.UserWallet
	err := s.db.WithContext(ctx).Where("user_id = ?", userID).First(&wallet).Error
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return 0, nil
		}
		return 0, errors.ErrDatabaseError.WithError(err)
	}

	return wallet.Balance, nil
}

// DeductFrozenToConsume 从冻结金额中扣款到消费（用于租借结算）
func (s *WalletService) DeductFrozenToConsume(ctx context.Context, userID int64, amount float64, orderNo string, remark string) error {
	if amount <= 0 {
		return nil
	}

	return s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		return s.DeductFrozenToConsumeTx(ctx, tx, userID, amount, orderNo, remark)
	})
}

// DeductFrozenToConsumeTx 在已有事务中从冻结金额中扣款到消费（用于租借结算）
func (s *WalletService) DeductFrozenToConsumeTx(ctx context.Context, tx *gorm.DB, userID int64, amount float64, orderNo string, remark string) error {
	if amount <= 0 {
		return nil
	}

	var wallet models.UserWallet
	if err := tx.WithContext(ctx).Set("gorm:query_option", "FOR UPDATE").
		Where("user_id = ?", userID).First(&wallet).Error; err != nil {
		return errors.ErrDatabaseError.WithError(err)
	}

	if wallet.FrozenBalance < amount {
		return errors.New(errors.ErrOperationFailed.Code, "冻结余额不足")
	}

	if err := tx.WithContext(ctx).Model(&wallet).Updates(map[string]interface{}{
		"frozen_balance": gorm.Expr("frozen_balance - ?", amount),
		"total_consumed": gorm.Expr("total_consumed + ?", amount),
	}).Error; err != nil {
		return errors.ErrDatabaseError.WithError(err)
	}

	transaction := &models.WalletTransaction{
		UserID:        userID,
		Type:          models.WalletTxTypeConsume,
		Amount:        -amount,
		BalanceBefore: wallet.Balance,
		BalanceAfter:  wallet.Balance,
		OrderNo:       &orderNo,
		Remark:        utils.StringPtr(fmt.Sprintf("押金消费: %s", remark)),
	}
	if err := tx.WithContext(ctx).Create(transaction).Error; err != nil {
		return errors.ErrDatabaseError.WithError(err)
	}

	return nil
}
