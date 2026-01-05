// Package distribution 分销服务
package distribution

import (
	"context"
	"errors"
	"time"

	"gorm.io/gorm"

	"github.com/dumeirei/smart-locker-backend/internal/models"
	"github.com/dumeirei/smart-locker-backend/internal/repository"
)

// 默认佣金比例
const (
	DefaultDirectRate   = 0.10 // 直推佣金比例 10%
	DefaultIndirectRate = 0.05 // 间推佣金比例 5%
	DefaultSettleDelay  = 7    // 默认结算延迟天数
)

// CommissionService 佣金服务
type CommissionService struct {
	commissionRepo  *repository.CommissionRepository
	distributorRepo *repository.DistributorRepository
	userRepo        *repository.UserRepository
	db              *gorm.DB
	directRate      float64 // 直推佣金比例
	indirectRate    float64 // 间推佣金比例
	settleDelay     int     // 结算延迟天数
}

// NewCommissionService 创建佣金服务
func NewCommissionService(
	commissionRepo *repository.CommissionRepository,
	distributorRepo *repository.DistributorRepository,
	userRepo *repository.UserRepository,
	db *gorm.DB,
) *CommissionService {
	return &CommissionService{
		commissionRepo:  commissionRepo,
		distributorRepo: distributorRepo,
		userRepo:        userRepo,
		db:              db,
		directRate:      DefaultDirectRate,
		indirectRate:    DefaultIndirectRate,
		settleDelay:     DefaultSettleDelay,
	}
}

// SetRates 设置佣金比例
func (s *CommissionService) SetRates(directRate, indirectRate float64, settleDelay int) {
	s.directRate = directRate
	s.indirectRate = indirectRate
	s.settleDelay = settleDelay
}

// CalculateRequest 计算佣金请求
type CalculateRequest struct {
	OrderID     int64   `json:"order_id"`
	UserID      int64   `json:"user_id"`      // 消费用户ID
	OrderAmount float64 `json:"order_amount"` // 订单实付金额
}

// CalculateResponse 计算佣金响应
type CalculateResponse struct {
	DirectCommission   *models.Commission `json:"direct_commission,omitempty"`
	IndirectCommission *models.Commission `json:"indirect_commission,omitempty"`
	TotalAmount        float64            `json:"total_amount"`
}

// Calculate 计算订单佣金
// 当订单完成时调用此方法计算并记录佣金
func (s *CommissionService) Calculate(ctx context.Context, req *CalculateRequest) (*CalculateResponse, error) {
	if req.OrderAmount <= 0 {
		return nil, errors.New("订单金额无效")
	}

	// 获取消费用户信息
	user, err := s.userRepo.GetByID(ctx, req.UserID)
	if err != nil {
		return nil, err
	}

	// 如果用户没有推荐人，无需计算佣金
	if user.ReferrerID == nil {
		return &CalculateResponse{TotalAmount: 0}, nil
	}

	response := &CalculateResponse{TotalAmount: 0}
	var commissions []*models.Commission

	// 使用事务处理
	err = s.db.Transaction(func(tx *gorm.DB) error {
		// 查找直推分销商（推荐人）
		directDistributor, err := s.findDistributorByUserID(ctx, tx, *user.ReferrerID)
		if err != nil || directDistributor == nil {
			// 推荐人不是分销商或未审核通过，不计算佣金
			return nil
		}

		// 计算直推佣金
		directAmount := req.OrderAmount * s.directRate
		if directAmount > 0 {
			directCommission := &models.Commission{
				DistributorID: directDistributor.ID,
				OrderID:       req.OrderID,
				FromUserID:    req.UserID,
				Type:          models.CommissionTypeDirect,
				OrderAmount:   req.OrderAmount,
				Rate:          s.directRate,
				Amount:        directAmount,
				Status:        models.CommissionStatusPending,
			}
			commissions = append(commissions, directCommission)
			response.DirectCommission = directCommission
			response.TotalAmount += directAmount
		}

		// 查找间推分销商（推荐人的推荐人）
		if directDistributor.ParentID != nil {
			indirectDistributor, err := s.findDistributorByID(ctx, tx, *directDistributor.ParentID)
			if err == nil && indirectDistributor != nil {
				// 计算间推佣金
				indirectAmount := req.OrderAmount * s.indirectRate
				if indirectAmount > 0 {
					indirectCommission := &models.Commission{
						DistributorID: indirectDistributor.ID,
						OrderID:       req.OrderID,
						FromUserID:    req.UserID,
						Type:          models.CommissionTypeIndirect,
						OrderAmount:   req.OrderAmount,
						Rate:          s.indirectRate,
						Amount:        indirectAmount,
						Status:        models.CommissionStatusPending,
					}
					commissions = append(commissions, indirectCommission)
					response.IndirectCommission = indirectCommission
					response.TotalAmount += indirectAmount
				}
			}
		}

		// 批量创建佣金记录
		if len(commissions) > 0 {
			if err := tx.Create(&commissions).Error; err != nil {
				return err
			}
		}

		return nil
	})

	if err != nil {
		return nil, err
	}

	return response, nil
}

// findDistributorByUserID 根据用户ID查找已审核通过的分销商
func (s *CommissionService) findDistributorByUserID(ctx context.Context, tx *gorm.DB, userID int64) (*models.Distributor, error) {
	var distributor models.Distributor
	err := tx.WithContext(ctx).
		Where("user_id = ? AND status = ?", userID, models.DistributorStatusApproved).
		First(&distributor).Error
	if err != nil {
		return nil, err
	}
	return &distributor, nil
}

// findDistributorByID 根据分销商ID查找已审核通过的分销商
func (s *CommissionService) findDistributorByID(ctx context.Context, tx *gorm.DB, distributorID int64) (*models.Distributor, error) {
	var distributor models.Distributor
	err := tx.WithContext(ctx).
		Where("id = ? AND status = ?", distributorID, models.DistributorStatusApproved).
		First(&distributor).Error
	if err != nil {
		return nil, err
	}
	return &distributor, nil
}

// Settle 结算佣金（将待结算佣金转为可提现）
func (s *CommissionService) Settle(ctx context.Context, commissionID int64) error {
	commission, err := s.commissionRepo.GetByID(ctx, commissionID)
	if err != nil {
		return err
	}

	if commission.Status != models.CommissionStatusPending {
		return errors.New("该佣金已处理")
	}

	return s.db.Transaction(func(tx *gorm.DB) error {
		// 更新佣金状态
		now := time.Now()
		if err := tx.Model(&models.Commission{}).
			Where("id = ?", commissionID).
			Updates(map[string]interface{}{
				"status":     models.CommissionStatusSettled,
				"settled_at": now,
			}).Error; err != nil {
			return err
		}

		// 增加分销商的可用佣金
		if err := tx.Model(&models.Distributor{}).
			Where("id = ?", commission.DistributorID).
			Updates(map[string]interface{}{
				"total_commission":     gorm.Expr("total_commission + ?", commission.Amount),
				"available_commission": gorm.Expr("available_commission + ?", commission.Amount),
			}).Error; err != nil {
			return err
		}

		return nil
	})
}

// SettlePendingCommissions 结算超过延迟期的待结算佣金
func (s *CommissionService) SettlePendingCommissions(ctx context.Context) (int64, error) {
	// 计算结算截止时间
	settleTime := time.Now().AddDate(0, 0, -s.settleDelay)

	// 获取需要结算的佣金
	var commissions []*models.Commission
	if err := s.db.WithContext(ctx).
		Where("status = ? AND created_at < ?", models.CommissionStatusPending, settleTime).
		Find(&commissions).Error; err != nil {
		return 0, err
	}

	if len(commissions) == 0 {
		return 0, nil
	}

	var settledCount int64
	for _, commission := range commissions {
		if err := s.Settle(ctx, commission.ID); err == nil {
			settledCount++
		}
	}

	return settledCount, nil
}

// CancelByOrderID 取消订单相关的佣金（退款时调用）
func (s *CommissionService) CancelByOrderID(ctx context.Context, orderID int64) error {
	// 获取订单相关的佣金记录
	commissions, err := s.commissionRepo.GetByOrderID(ctx, orderID)
	if err != nil {
		return err
	}

	if len(commissions) == 0 {
		return nil
	}

	return s.db.Transaction(func(tx *gorm.DB) error {
		for _, commission := range commissions {
			if commission.Status == models.CommissionStatusSettled {
				// 已结算的佣金，需要从分销商账户扣除
				if err := tx.Model(&models.Distributor{}).
					Where("id = ? AND available_commission >= ?", commission.DistributorID, commission.Amount).
					Updates(map[string]interface{}{
						"total_commission":     gorm.Expr("total_commission - ?", commission.Amount),
						"available_commission": gorm.Expr("available_commission - ?", commission.Amount),
					}).Error; err != nil {
					return err
				}
			}

			// 更新佣金状态为已取消
			if err := tx.Model(&models.Commission{}).
				Where("id = ?", commission.ID).
				Update("status", models.CommissionStatusCancelled).Error; err != nil {
				return err
			}
		}
		return nil
	})
}

// GetByDistributorID 获取分销商的佣金记录
func (s *CommissionService) GetByDistributorID(ctx context.Context, distributorID int64, offset, limit int) ([]*models.Commission, int64, error) {
	return s.commissionRepo.GetByDistributorID(ctx, distributorID, offset, limit)
}

// GetStats 获取佣金统计
func (s *CommissionService) GetStats(ctx context.Context, distributorID int64) (map[string]interface{}, error) {
	return s.commissionRepo.GetStatsByDistributorID(ctx, distributorID)
}

// List 获取佣金列表
func (s *CommissionService) List(ctx context.Context, offset, limit int, filters map[string]interface{}) ([]*models.Commission, int64, error) {
	return s.commissionRepo.List(ctx, offset, limit, filters)
}

// GetByOrderID 获取订单相关的佣金记录
func (s *CommissionService) GetByOrderID(ctx context.Context, orderID int64) ([]*models.Commission, error) {
	return s.commissionRepo.GetByOrderID(ctx, orderID)
}
