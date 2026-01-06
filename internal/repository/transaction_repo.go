// Package repository 提供数据访问层
package repository

import (
	"context"
	"time"

	"gorm.io/gorm"

	"github.com/dumeirei/smart-locker-backend/internal/models"
)

// TransactionRepository 钱包交易仓储
type TransactionRepository struct {
	db *gorm.DB
}

// NewTransactionRepository 创建交易仓储
func NewTransactionRepository(db *gorm.DB) *TransactionRepository {
	return &TransactionRepository{db: db}
}

// Create 创建交易记录
func (r *TransactionRepository) Create(ctx context.Context, transaction *models.WalletTransaction) error {
	return r.db.WithContext(ctx).Create(transaction).Error
}

// GetByID 根据 ID 获取交易记录
func (r *TransactionRepository) GetByID(ctx context.Context, id int64) (*models.WalletTransaction, error) {
	var transaction models.WalletTransaction
	err := r.db.WithContext(ctx).First(&transaction, id).Error
	if err != nil {
		return nil, err
	}
	return &transaction, nil
}

// TransactionFilter 交易查询过滤条件
type TransactionFilter struct {
	UserID    *int64
	Type      string
	OrderNo   string
	StartDate *time.Time
	EndDate   *time.Time
}

// List 获取交易列表
func (r *TransactionRepository) List(ctx context.Context, filter *TransactionFilter, offset, limit int) ([]*models.WalletTransaction, int64, error) {
	var transactions []*models.WalletTransaction
	var total int64

	query := r.db.WithContext(ctx).Model(&models.WalletTransaction{})

	if filter != nil {
		if filter.UserID != nil {
			query = query.Where("user_id = ?", *filter.UserID)
		}
		if filter.Type != "" {
			query = query.Where("type = ?", filter.Type)
		}
		if filter.OrderNo != "" {
			query = query.Where("order_no = ?", filter.OrderNo)
		}
		if filter.StartDate != nil {
			query = query.Where("created_at >= ?", *filter.StartDate)
		}
		if filter.EndDate != nil {
			query = query.Where("created_at <= ?", *filter.EndDate)
		}
	}

	// 获取总数
	err := query.Count(&total).Error
	if err != nil {
		return nil, 0, err
	}

	// 获取数据
	err = query.Order("created_at DESC").
		Offset(offset).
		Limit(limit).
		Find(&transactions).Error
	if err != nil {
		return nil, 0, err
	}

	return transactions, total, nil
}

// ListByUser 获取用户交易列表
func (r *TransactionRepository) ListByUser(ctx context.Context, userID int64, offset, limit int) ([]*models.WalletTransaction, int64, error) {
	filter := &TransactionFilter{
		UserID: &userID,
	}
	return r.List(ctx, filter, offset, limit)
}

// GetByOrderNo 根据订单号获取交易记录
func (r *TransactionRepository) GetByOrderNo(ctx context.Context, orderNo string) ([]*models.WalletTransaction, error) {
	var transactions []*models.WalletTransaction
	err := r.db.WithContext(ctx).Where("order_no = ?", orderNo).
		Order("created_at DESC").
		Find(&transactions).Error
	return transactions, err
}

// GetStatistics 获取交易统计
func (r *TransactionRepository) GetStatistics(ctx context.Context, startDate, endDate *time.Time) (*models.TransactionStatistics, error) {
	var stats models.TransactionStatistics

	query := r.db.WithContext(ctx).Model(&models.WalletTransaction{})
	if startDate != nil {
		query = query.Where("created_at >= ?", *startDate)
	}
	if endDate != nil {
		query = query.Where("created_at <= ?", *endDate)
	}

	// 充值
	err := query.Where("type = ?", models.WalletTxTypeRecharge).
		Select("COALESCE(SUM(amount), 0)").
		Row().Scan(&stats.TotalRecharge)
	if err != nil {
		return nil, err
	}

	// 消费
	query = r.db.WithContext(ctx).Model(&models.WalletTransaction{})
	if startDate != nil {
		query = query.Where("created_at >= ?", *startDate)
	}
	if endDate != nil {
		query = query.Where("created_at <= ?", *endDate)
	}
	err = query.Where("type = ?", models.WalletTxTypeConsume).
		Select("COALESCE(SUM(amount), 0)").
		Row().Scan(&stats.TotalConsume)
	if err != nil {
		return nil, err
	}

	// 退款
	query = r.db.WithContext(ctx).Model(&models.WalletTransaction{})
	if startDate != nil {
		query = query.Where("created_at >= ?", *startDate)
	}
	if endDate != nil {
		query = query.Where("created_at <= ?", *endDate)
	}
	err = query.Where("type = ?", models.WalletTxTypeRefund).
		Select("COALESCE(SUM(amount), 0)").
		Row().Scan(&stats.TotalRefund)
	if err != nil {
		return nil, err
	}

	// 提现
	query = r.db.WithContext(ctx).Model(&models.WalletTransaction{})
	if startDate != nil {
		query = query.Where("created_at >= ?", *startDate)
	}
	if endDate != nil {
		query = query.Where("created_at <= ?", *endDate)
	}
	err = query.Where("type = ?", models.WalletTxTypeWithdraw).
		Select("COALESCE(SUM(amount), 0)").
		Row().Scan(&stats.TotalWithdraw)
	if err != nil {
		return nil, err
	}

	// 押金
	query = r.db.WithContext(ctx).Model(&models.WalletTransaction{})
	if startDate != nil {
		query = query.Where("created_at >= ?", *startDate)
	}
	if endDate != nil {
		query = query.Where("created_at <= ?", *endDate)
	}
	err = query.Where("type = ?", models.WalletTxTypeDeposit).
		Select("COALESCE(SUM(amount), 0)").
		Row().Scan(&stats.TotalDeposit)
	if err != nil {
		return nil, err
	}

	// 退还押金
	query = r.db.WithContext(ctx).Model(&models.WalletTransaction{})
	if startDate != nil {
		query = query.Where("created_at >= ?", *startDate)
	}
	if endDate != nil {
		query = query.Where("created_at <= ?", *endDate)
	}
	err = query.Where("type = ?", models.WalletTxTypeReturnDeposit).
		Select("COALESCE(SUM(amount), 0)").
		Row().Scan(&stats.TotalReturnDeposit)
	if err != nil {
		return nil, err
	}

	return &stats, nil
}

// GetUserStatistics 获取用户交易统计
func (r *TransactionRepository) GetUserStatistics(ctx context.Context, userID int64) (*models.TransactionStatistics, error) {
	var stats models.TransactionStatistics

	// 充值
	err := r.db.WithContext(ctx).Model(&models.WalletTransaction{}).
		Where("user_id = ? AND type = ?", userID, models.WalletTxTypeRecharge).
		Select("COALESCE(SUM(amount), 0)").
		Row().Scan(&stats.TotalRecharge)
	if err != nil {
		return nil, err
	}

	// 消费
	err = r.db.WithContext(ctx).Model(&models.WalletTransaction{}).
		Where("user_id = ? AND type = ?", userID, models.WalletTxTypeConsume).
		Select("COALESCE(SUM(amount), 0)").
		Row().Scan(&stats.TotalConsume)
	if err != nil {
		return nil, err
	}

	// 退款
	err = r.db.WithContext(ctx).Model(&models.WalletTransaction{}).
		Where("user_id = ? AND type = ?", userID, models.WalletTxTypeRefund).
		Select("COALESCE(SUM(amount), 0)").
		Row().Scan(&stats.TotalRefund)
	if err != nil {
		return nil, err
	}

	// 提现
	err = r.db.WithContext(ctx).Model(&models.WalletTransaction{}).
		Where("user_id = ? AND type = ?", userID, models.WalletTxTypeWithdraw).
		Select("COALESCE(SUM(amount), 0)").
		Row().Scan(&stats.TotalWithdraw)
	if err != nil {
		return nil, err
	}

	return &stats, nil
}

// CountByType 按类型统计交易数量
func (r *TransactionRepository) CountByType(ctx context.Context, txType string, startDate, endDate *time.Time) (int64, error) {
	var count int64
	query := r.db.WithContext(ctx).Model(&models.WalletTransaction{}).Where("type = ?", txType)
	if startDate != nil {
		query = query.Where("created_at >= ?", *startDate)
	}
	if endDate != nil {
		query = query.Where("created_at <= ?", *endDate)
	}
	err := query.Count(&count).Error
	return count, err
}

// SumByType 按类型汇总交易金额
func (r *TransactionRepository) SumByType(ctx context.Context, txType string, startDate, endDate *time.Time) (float64, error) {
	var sum float64
	query := r.db.WithContext(ctx).Model(&models.WalletTransaction{}).
		Where("type = ?", txType)
	if startDate != nil {
		query = query.Where("created_at >= ?", *startDate)
	}
	if endDate != nil {
		query = query.Where("created_at <= ?", *endDate)
	}
	err := query.Select("COALESCE(SUM(amount), 0)").Row().Scan(&sum)
	return sum, err
}

// GetDailyStatistics 获取每日交易统计
func (r *TransactionRepository) GetDailyStatistics(ctx context.Context, startDate, endDate time.Time) ([]map[string]interface{}, error) {
	var results []map[string]interface{}
	err := r.db.WithContext(ctx).Model(&models.WalletTransaction{}).
		Select(
			"DATE(created_at) as date",
			"type",
			"COUNT(*) as count",
			"SUM(amount) as total_amount",
		).
		Where("created_at >= ? AND created_at <= ?", startDate, endDate).
		Group("DATE(created_at), type").
		Order("date ASC").
		Find(&results).Error
	return results, err
}

// BatchCreate 批量创建交易记录
func (r *TransactionRepository) BatchCreate(ctx context.Context, transactions []*models.WalletTransaction) error {
	if len(transactions) == 0 {
		return nil
	}
	return r.db.WithContext(ctx).CreateInBatches(transactions, 100).Error
}
