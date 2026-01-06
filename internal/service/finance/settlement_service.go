// Package finance 提供财务管理服务
package finance

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

// SettlementService 结算服务
type SettlementService struct {
	db              *gorm.DB
	settlementRepo  *repository.SettlementRepository
	orderRepo       *repository.OrderRepository
	merchantRepo    *repository.MerchantRepository
	commissionRepo  *repository.CommissionRepository
	distributorRepo *repository.DistributorRepository
}

// NewSettlementService 创建结算服务
func NewSettlementService(
	db *gorm.DB,
	settlementRepo *repository.SettlementRepository,
	orderRepo *repository.OrderRepository,
	merchantRepo *repository.MerchantRepository,
	commissionRepo *repository.CommissionRepository,
	distributorRepo *repository.DistributorRepository,
) *SettlementService {
	return &SettlementService{
		db:              db,
		settlementRepo:  settlementRepo,
		orderRepo:       orderRepo,
		merchantRepo:    merchantRepo,
		commissionRepo:  commissionRepo,
		distributorRepo: distributorRepo,
	}
}

// CreateSettlementRequest 创建结算请求
type CreateSettlementRequest struct {
	Type        string    `json:"type" binding:"required,oneof=merchant distributor"`
	TargetID    int64     `json:"target_id" binding:"required"`
	PeriodStart time.Time `json:"period_start" binding:"required"`
	PeriodEnd   time.Time `json:"period_end" binding:"required"`
}

// CreateSettlement 创建结算记录
func (s *SettlementService) CreateSettlement(ctx context.Context, req *CreateSettlementRequest, operatorID int64) (*models.Settlement, error) {
	// 检查是否已存在该周期的结算记录
	exists, err := s.settlementRepo.ExistsForPeriod(ctx, req.Type, req.TargetID, req.PeriodStart, req.PeriodEnd)
	if err != nil {
		return nil, errors.ErrDatabaseError.WithError(err)
	}
	if exists {
		return nil, errors.ErrDuplicateRecord.WithMessage("该周期的结算记录已存在")
	}

	// 计算结算金额
	var totalAmount, fee, actualAmount float64
	var orderCount int

	if req.Type == models.SettlementTypeMerchant {
		// 商户结算 - 计算商户的订单收入
		totalAmount, orderCount, err = s.calculateMerchantSettlement(ctx, req.TargetID, req.PeriodStart, req.PeriodEnd)
		if err != nil {
			return nil, err
		}
		// 获取商户分成比例计算手续费
		merchant, err := s.merchantRepo.GetByID(ctx, req.TargetID)
		if err != nil {
			return nil, errors.ErrMerchantNotFound.WithError(err)
		}
		fee = totalAmount * merchant.CommissionRate
		actualAmount = totalAmount - fee
	} else {
		// 分销商结算 - 计算分销商的佣金
		totalAmount, orderCount, err = s.calculateDistributorSettlement(ctx, req.TargetID, req.PeriodStart, req.PeriodEnd)
		if err != nil {
			return nil, err
		}
		// 分销商提现无手续费
		fee = 0
		actualAmount = totalAmount
	}

	settlementNo := utils.GenerateOrderNo("ST")
	settlement := &models.Settlement{
		SettlementNo: settlementNo,
		Type:         req.Type,
		TargetID:     req.TargetID,
		PeriodStart:  req.PeriodStart,
		PeriodEnd:    req.PeriodEnd,
		TotalAmount:  totalAmount,
		Fee:          fee,
		ActualAmount: actualAmount,
		OrderCount:   orderCount,
		Status:       models.SettlementStatusPending,
		OperatorID:   &operatorID,
	}

	if err := s.settlementRepo.Create(ctx, settlement); err != nil {
		return nil, errors.ErrDatabaseError.WithError(err)
	}

	return settlement, nil
}

// calculateMerchantSettlement 计算商户结算金额
func (s *SettlementService) calculateMerchantSettlement(ctx context.Context, merchantID int64, periodStart, periodEnd time.Time) (float64, int, error) {
	var totalAmount float64
	var orderCount int64

	// 获取商户下所有场地
	venues, err := s.getVenuesByMerchant(ctx, merchantID)
	if err != nil {
		return 0, 0, err
	}

	if len(venues) == 0 {
		return 0, 0, nil
	}

	venueIDs := make([]int64, len(venues))
	for i, v := range venues {
		venueIDs[i] = v.ID
	}

	// 获取场地下所有设备
	var deviceIDs []int64
	err = s.db.WithContext(ctx).Model(&models.Device{}).
		Where("venue_id IN ?", venueIDs).
		Pluck("id", &deviceIDs).Error
	if err != nil {
		return 0, 0, err
	}

	if len(deviceIDs) == 0 {
		return 0, 0, nil
	}

	// 统计租借订单收入
	err = s.db.WithContext(ctx).Model(&models.Rental{}).
		Joins("JOIN orders ON orders.id = rentals.order_id").
		Where("rentals.device_id IN ?", deviceIDs).
		Where("orders.status = ?", models.OrderStatusCompleted).
		Where("orders.completed_at >= ? AND orders.completed_at <= ?", periodStart, periodEnd).
		Select("COALESCE(SUM(orders.actual_amount), 0)").
		Row().Scan(&totalAmount)
	if err != nil {
		return 0, 0, err
	}

	// 统计订单数
	err = s.db.WithContext(ctx).Model(&models.Rental{}).
		Joins("JOIN orders ON orders.id = rentals.order_id").
		Where("rentals.device_id IN ?", deviceIDs).
		Where("orders.status = ?", models.OrderStatusCompleted).
		Where("orders.completed_at >= ? AND orders.completed_at <= ?", periodStart, periodEnd).
		Count(&orderCount).Error
	if err != nil {
		return 0, 0, err
	}

	return totalAmount, int(orderCount), nil
}

// calculateDistributorSettlement 计算分销商结算金额
func (s *SettlementService) calculateDistributorSettlement(ctx context.Context, distributorID int64, periodStart, periodEnd time.Time) (float64, int, error) {
	var totalAmount float64
	var orderCount int64

	// 统计待结算佣金
	err := s.db.WithContext(ctx).Model(&models.Commission{}).
		Where("distributor_id = ?", distributorID).
		Where("status = ?", models.CommissionStatusPending).
		Where("created_at >= ? AND created_at <= ?", periodStart, periodEnd).
		Select("COALESCE(SUM(amount), 0)").
		Row().Scan(&totalAmount)
	if err != nil {
		return 0, 0, err
	}

	// 统计佣金订单数
	err = s.db.WithContext(ctx).Model(&models.Commission{}).
		Where("distributor_id = ?", distributorID).
		Where("status = ?", models.CommissionStatusPending).
		Where("created_at >= ? AND created_at <= ?", periodStart, periodEnd).
		Count(&orderCount).Error
	if err != nil {
		return 0, 0, err
	}

	return totalAmount, int(orderCount), nil
}

// getVenuesByMerchant 获取商户下的场地
func (s *SettlementService) getVenuesByMerchant(ctx context.Context, merchantID int64) ([]*models.Venue, error) {
	var venues []*models.Venue
	err := s.db.WithContext(ctx).Where("merchant_id = ?", merchantID).Find(&venues).Error
	return venues, err
}

// ProcessSettlement 处理结算
func (s *SettlementService) ProcessSettlement(ctx context.Context, settlementID int64, operatorID int64) error {
	settlement, err := s.settlementRepo.GetByID(ctx, settlementID)
	if err != nil {
		return errors.ErrSettlementNotFound.WithError(err)
	}

	if settlement.Status != models.SettlementStatusPending {
		return errors.ErrInvalidOperation.WithMessage("只能处理待结算状态的记录")
	}

	// 开始事务
	tx := s.db.WithContext(ctx).Begin()
	defer func() {
		if r := recover(); r != nil {
			tx.Rollback()
		}
	}()

	// 更新结算状态为处理中
	err = tx.Model(&models.Settlement{}).
		Where("id = ?", settlementID).
		Updates(map[string]interface{}{
			"status":      models.SettlementStatusProcessing,
			"operator_id": operatorID,
		}).Error
	if err != nil {
		tx.Rollback()
		return errors.ErrDatabaseError.WithError(err)
	}

	// 如果是分销商结算，更新佣金状态
	if settlement.Type == models.SettlementTypeDistributor {
		err = tx.Model(&models.Commission{}).
			Where("distributor_id = ?", settlement.TargetID).
			Where("status = ?", models.CommissionStatusPending).
			Where("created_at >= ? AND created_at <= ?", settlement.PeriodStart, settlement.PeriodEnd).
			Updates(map[string]interface{}{
				"status":     models.CommissionStatusSettled,
				"settled_at": time.Now(),
			}).Error
		if err != nil {
			tx.Rollback()
			return errors.ErrDatabaseError.WithError(err)
		}

		// 更新分销商佣金余额
		err = tx.Model(&models.Distributor{}).
			Where("id = ?", settlement.TargetID).
			Updates(map[string]interface{}{
				"available_commission": gorm.Expr("available_commission + ?", settlement.ActualAmount),
			}).Error
		if err != nil {
			tx.Rollback()
			return errors.ErrDatabaseError.WithError(err)
		}
	}

	// 更新结算状态为已完成
	now := time.Now()
	err = tx.Model(&models.Settlement{}).
		Where("id = ?", settlementID).
		Updates(map[string]interface{}{
			"status":     models.SettlementStatusCompleted,
			"settled_at": &now,
		}).Error
	if err != nil {
		tx.Rollback()
		return errors.ErrDatabaseError.WithError(err)
	}

	return tx.Commit().Error
}

// GetSettlement 获取结算详情
func (s *SettlementService) GetSettlement(ctx context.Context, id int64) (*models.Settlement, error) {
	settlement, err := s.settlementRepo.GetByID(ctx, id)
	if err != nil {
		return nil, errors.ErrSettlementNotFound.WithError(err)
	}
	return settlement, nil
}

// SettlementListRequest 结算列表请求
type SettlementListRequest struct {
	Type        string `form:"type"`
	TargetID    *int64 `form:"target_id"`
	Status      string `form:"status"`
	PeriodStart string `form:"period_start"`
	PeriodEnd   string `form:"period_end"`
	Page        int    `form:"page,default=1"`
	PageSize    int    `form:"page_size,default=20"`
}

// ListSettlements 获取结算列表
func (s *SettlementService) ListSettlements(ctx context.Context, req *SettlementListRequest) ([]*models.Settlement, int64, error) {
	filter := &repository.SettlementFilter{
		Type:     req.Type,
		TargetID: req.TargetID,
		Status:   req.Status,
	}

	if req.PeriodStart != "" {
		t, err := time.Parse("2006-01-02", req.PeriodStart)
		if err == nil {
			filter.PeriodStart = &t
		}
	}
	if req.PeriodEnd != "" {
		t, err := time.Parse("2006-01-02", req.PeriodEnd)
		if err == nil {
			filter.PeriodEnd = &t
		}
	}

	offset := (req.Page - 1) * req.PageSize
	return s.settlementRepo.List(ctx, filter, offset, req.PageSize)
}

// GenerateMerchantSettlements 生成商户结算记录
func (s *SettlementService) GenerateMerchantSettlements(ctx context.Context, periodStart, periodEnd time.Time, operatorID int64) ([]*models.Settlement, error) {
	// 获取所有活跃商户
	var merchants []*models.Merchant
	err := s.db.WithContext(ctx).Where("status = ?", 1).Find(&merchants).Error
	if err != nil {
		return nil, errors.ErrDatabaseError.WithError(err)
	}

	var settlements []*models.Settlement
	for _, merchant := range merchants {
		// 检查是否已存在结算记录
		exists, err := s.settlementRepo.ExistsForPeriod(ctx, models.SettlementTypeMerchant, merchant.ID, periodStart, periodEnd)
		if err != nil {
			continue
		}
		if exists {
			continue
		}

		// 计算结算金额
		totalAmount, orderCount, err := s.calculateMerchantSettlement(ctx, merchant.ID, periodStart, periodEnd)
		if err != nil {
			continue
		}
		if totalAmount == 0 {
			continue
		}

		fee := totalAmount * merchant.CommissionRate
		actualAmount := totalAmount - fee

		settlement := &models.Settlement{
			SettlementNo: utils.GenerateOrderNo("ST"),
			Type:         models.SettlementTypeMerchant,
			TargetID:     merchant.ID,
			PeriodStart:  periodStart,
			PeriodEnd:    periodEnd,
			TotalAmount:  totalAmount,
			Fee:          fee,
			ActualAmount: actualAmount,
			OrderCount:   orderCount,
			Status:       models.SettlementStatusPending,
			OperatorID:   &operatorID,
		}

		if err := s.settlementRepo.Create(ctx, settlement); err != nil {
			continue
		}

		settlements = append(settlements, settlement)
	}

	return settlements, nil
}

// GenerateDistributorSettlements 生成分销商结算记录
func (s *SettlementService) GenerateDistributorSettlements(ctx context.Context, periodStart, periodEnd time.Time, operatorID int64) ([]*models.Settlement, error) {
	// 获取所有有待结算佣金的分销商
	var distributorIDs []int64
	err := s.db.WithContext(ctx).Model(&models.Commission{}).
		Where("status = ?", models.CommissionStatusPending).
		Where("created_at >= ? AND created_at <= ?", periodStart, periodEnd).
		Distinct("distributor_id").
		Pluck("distributor_id", &distributorIDs).Error
	if err != nil {
		return nil, errors.ErrDatabaseError.WithError(err)
	}

	var settlements []*models.Settlement
	for _, distributorID := range distributorIDs {
		// 检查是否已存在结算记录
		exists, err := s.settlementRepo.ExistsForPeriod(ctx, models.SettlementTypeDistributor, distributorID, periodStart, periodEnd)
		if err != nil {
			continue
		}
		if exists {
			continue
		}

		// 计算结算金额
		totalAmount, orderCount, err := s.calculateDistributorSettlement(ctx, distributorID, periodStart, periodEnd)
		if err != nil {
			continue
		}
		if totalAmount == 0 {
			continue
		}

		settlement := &models.Settlement{
			SettlementNo: utils.GenerateOrderNo("ST"),
			Type:         models.SettlementTypeDistributor,
			TargetID:     distributorID,
			PeriodStart:  periodStart,
			PeriodEnd:    periodEnd,
			TotalAmount:  totalAmount,
			Fee:          0,
			ActualAmount: totalAmount,
			OrderCount:   orderCount,
			Status:       models.SettlementStatusPending,
			OperatorID:   &operatorID,
		}

		if err := s.settlementRepo.Create(ctx, settlement); err != nil {
			continue
		}

		settlements = append(settlements, settlement)
	}

	return settlements, nil
}

// SettlementDetail 获取结算详情（包含目标名称）
func (s *SettlementService) GetSettlementDetail(ctx context.Context, id int64) (*models.SettlementDetail, error) {
	settlement, err := s.settlementRepo.GetByID(ctx, id)
	if err != nil {
		return nil, errors.ErrSettlementNotFound.WithError(err)
	}

	detail := &models.SettlementDetail{
		SettlementNo: settlement.SettlementNo,
		Type:         settlement.Type,
		PeriodStart:  settlement.PeriodStart,
		PeriodEnd:    settlement.PeriodEnd,
		TotalAmount:  settlement.TotalAmount,
		Fee:          settlement.Fee,
		ActualAmount: settlement.ActualAmount,
		OrderCount:   settlement.OrderCount,
		Status:       settlement.Status,
		CreatedAt:    settlement.CreatedAt,
	}

	if settlement.SettledAt != nil {
		detail.SettledAt = settlement.SettledAt.Format("2006-01-02 15:04:05")
	}

	// 获取目标名称
	if settlement.Type == models.SettlementTypeMerchant {
		var merchant models.Merchant
		if err := s.db.WithContext(ctx).First(&merchant, settlement.TargetID).Error; err == nil {
			detail.TargetName = merchant.Name
		}
	} else {
		var distributor models.Distributor
		if err := s.db.WithContext(ctx).Preload("User").First(&distributor, settlement.TargetID).Error; err == nil && distributor.User != nil {
			detail.TargetName = fmt.Sprintf("%s (ID: %d)", distributor.User.Nickname, distributor.ID)
		}
	}

	return detail, nil
}
