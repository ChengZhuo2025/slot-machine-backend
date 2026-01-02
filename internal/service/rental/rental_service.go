// Package rental 提供租借服务
package rental

import (
	"context"
	"fmt"
	"time"

	"gorm.io/gorm"

	"smart-locker-backend/internal/common/errors"
	"smart-locker-backend/internal/common/utils"
	"smart-locker-backend/internal/models"
	"smart-locker-backend/internal/repository"
	deviceService "smart-locker-backend/internal/service/device"
	userService "smart-locker-backend/internal/service/user"
)

// RentalService 租借服务
type RentalService struct {
	db            *gorm.DB
	rentalRepo    *repository.RentalRepository
	deviceRepo    *repository.DeviceRepository
	deviceService *deviceService.DeviceService
	walletService *userService.WalletService
	mqttService   *deviceService.MQTTService
}

// NewRentalService 创建租借服务
func NewRentalService(
	db *gorm.DB,
	rentalRepo *repository.RentalRepository,
	deviceRepo *repository.DeviceRepository,
	deviceSvc *deviceService.DeviceService,
	walletSvc *userService.WalletService,
	mqttSvc *deviceService.MQTTService,
) *RentalService {
	return &RentalService{
		db:            db,
		rentalRepo:    rentalRepo,
		deviceRepo:    deviceRepo,
		deviceService: deviceSvc,
		walletService: walletSvc,
		mqttService:   mqttSvc,
	}
}

// CreateRentalRequest 创建租借请求
type CreateRentalRequest struct {
	DeviceID  int64 `json:"device_id" binding:"required"`
	PricingID int64 `json:"pricing_id" binding:"required"`
}

// RentalInfo 租借信息
type RentalInfo struct {
	ID             int64                    `json:"id"`
	RentalNo       string                   `json:"rental_no"`
	Status         int8                     `json:"status"`
	StatusName     string                   `json:"status_name"`
	Device         *deviceService.DeviceInfo `json:"device,omitempty"`
	Pricing        *deviceService.PricingInfo `json:"pricing,omitempty"`
	StartTime      *time.Time               `json:"start_time,omitempty"`
	EndTime        *time.Time               `json:"end_time,omitempty"`
	Duration       *int                     `json:"duration,omitempty"`
	UnitPrice      float64                  `json:"unit_price"`
	DepositAmount  float64                  `json:"deposit_amount"`
	RentalAmount   float64                  `json:"rental_amount"`
	DiscountAmount float64                  `json:"discount_amount"`
	ActualAmount   float64                  `json:"actual_amount"`
	RefundAmount   float64                  `json:"refund_amount"`
	CreatedAt      time.Time                `json:"created_at"`
}

// CreateRental 创建租借订单
func (s *RentalService) CreateRental(ctx context.Context, userID int64, req *CreateRentalRequest) (*RentalInfo, error) {
	// 检查用户是否有进行中的租借
	hasActive, err := s.rentalRepo.HasActiveRental(ctx, userID)
	if err != nil {
		return nil, errors.ErrDatabaseError.WithError(err)
	}
	if hasActive {
		return nil, errors.ErrRentalInProgress
	}

	// 检查设备是否可用
	if err := s.deviceService.CheckDeviceAvailable(ctx, req.DeviceID); err != nil {
		return nil, err
	}

	// 获取定价信息
	pricing, err := s.deviceRepo.GetPricingByID(ctx, req.PricingID)
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, errors.ErrPricingNotFound
		}
		return nil, errors.ErrDatabaseError.WithError(err)
	}

	if pricing.DeviceID != req.DeviceID {
		return nil, errors.ErrInvalidParams.WithMessage("定价方案不属于该设备")
	}

	if pricing.Status != models.RentalPricingStatusActive {
		return nil, errors.ErrPricingNotFound
	}

	// 检查用户余额是否足够支付押金
	totalAmount := pricing.Price + pricing.Deposit
	sufficient, err := s.walletService.CheckBalance(ctx, userID, totalAmount)
	if err != nil {
		return nil, errors.ErrDatabaseError.WithError(err)
	}
	if !sufficient {
		return nil, errors.ErrBalanceInsufficient
	}

	// 创建租借订单
	rentalNo := utils.GenerateOrderNo("R")
	rental := &models.Rental{
		RentalNo:      rentalNo,
		UserID:        userID,
		DeviceID:      req.DeviceID,
		PricingID:     req.PricingID,
		Status:        models.RentalStatusPending,
		UnitPrice:     pricing.Price,
		DepositAmount: pricing.Deposit,
		RentalAmount:  pricing.Price,
		ActualAmount:  totalAmount,
	}

	// 使用事务
	err = s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		// 创建租借订单
		if err := tx.Create(rental).Error; err != nil {
			return err
		}

		// 减少设备可用槽位（预占）
		result := tx.Model(&models.Device{}).
			Where("id = ? AND available_slots > 0", req.DeviceID).
			UpdateColumn("available_slots", gorm.Expr("available_slots - 1"))
		if result.Error != nil {
			return result.Error
		}
		if result.RowsAffected == 0 {
			return errors.ErrDeviceNoSlot
		}

		return nil
	})

	if err != nil {
		if appErr, ok := err.(*errors.AppError); ok {
			return nil, appErr
		}
		return nil, errors.ErrDatabaseError.WithError(err)
	}

	return s.toRentalInfo(rental, nil, nil), nil
}

// PayRental 支付租借订单
func (s *RentalService) PayRental(ctx context.Context, userID int64, rentalID int64) error {
	return s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		// 获取并锁定租借订单
		rental, err := s.rentalRepo.GetForUpdate(ctx, tx, rentalID)
		if err != nil {
			if err == gorm.ErrRecordNotFound {
				return errors.ErrRentalNotFound
			}
			return errors.ErrDatabaseError.WithError(err)
		}

		if rental.UserID != userID {
			return errors.ErrPermissionDenied
		}

		if rental.Status != models.RentalStatusPending {
			return errors.ErrRentalStatusError
		}

		// 冻结押金 + 扣除租金
		if rental.DepositAmount > 0 {
			if err := s.walletService.FreezeDeposit(ctx, userID, rental.DepositAmount, rental.RentalNo); err != nil {
				return err
			}
		}

		if rental.RentalAmount > 0 {
			if err := s.walletService.Consume(ctx, userID, rental.RentalAmount, rental.RentalNo); err != nil {
				return err
			}
		}

		// 更新订单状态
		now := time.Now()
		updates := map[string]interface{}{
			"status":  models.RentalStatusPaid,
			"paid_at": now,
		}
		if err := tx.Model(rental).Updates(updates).Error; err != nil {
			return errors.ErrDatabaseError.WithError(err)
		}

		return nil
	})
}

// StartRental 开始租借（开锁取货）
func (s *RentalService) StartRental(ctx context.Context, userID int64, rentalID int64) error {
	return s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		rental, err := s.rentalRepo.GetForUpdate(ctx, tx, rentalID)
		if err != nil {
			if err == gorm.ErrRecordNotFound {
				return errors.ErrRentalNotFound
			}
			return errors.ErrDatabaseError.WithError(err)
		}

		if rental.UserID != userID {
			return errors.ErrPermissionDenied
		}

		if rental.Status != models.RentalStatusPaid {
			return errors.ErrRentalStatusError
		}

		// 获取设备信息
		device, err := s.deviceRepo.GetByID(ctx, rental.DeviceID)
		if err != nil {
			return errors.ErrDeviceNotFound
		}

		// 发送开锁命令
		if s.mqttService != nil {
			_, err := s.mqttService.SendUnlockCommand(ctx, device.DeviceNo, rental.SlotNo)
			if err != nil {
				return errors.ErrUnlockFailed.WithError(err)
			}
		}

		// 获取定价信息计算结束时间
		pricing, _ := s.deviceRepo.GetPricingByID(ctx, rental.PricingID)

		now := time.Now()
		var endTime time.Time
		if pricing != nil {
			switch pricing.DurationUnit {
			case models.DurationUnitMinute:
				endTime = now.Add(time.Duration(pricing.Duration) * time.Minute)
			case models.DurationUnitHour:
				endTime = now.Add(time.Duration(pricing.Duration) * time.Hour)
			case models.DurationUnitDay:
				endTime = now.AddDate(0, 0, pricing.Duration)
			default:
				endTime = now.Add(time.Duration(pricing.Duration) * time.Hour)
			}
		} else {
			endTime = now.Add(24 * time.Hour) // 默认 24 小时
		}

		// 更新订单状态
		updates := map[string]interface{}{
			"status":     models.RentalStatusInUse,
			"start_time": now,
			"end_time":   endTime,
		}
		if err := tx.Model(rental).Updates(updates).Error; err != nil {
			return errors.ErrDatabaseError.WithError(err)
		}

		// 更新设备租借状态
		if err := tx.Model(&models.Device{}).Where("id = ?", rental.DeviceID).Updates(map[string]interface{}{
			"rental_status":     models.DeviceRentalInUse,
			"current_rental_id": rental.ID,
		}).Error; err != nil {
			return errors.ErrDatabaseError.WithError(err)
		}

		return nil
	})
}

// ReturnRental 归还租借
func (s *RentalService) ReturnRental(ctx context.Context, userID int64, rentalID int64) error {
	return s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		rental, err := s.rentalRepo.GetForUpdate(ctx, tx, rentalID)
		if err != nil {
			if err == gorm.ErrRecordNotFound {
				return errors.ErrRentalNotFound
			}
			return errors.ErrDatabaseError.WithError(err)
		}

		if rental.UserID != userID {
			return errors.ErrPermissionDenied
		}

		if rental.Status != models.RentalStatusInUse {
			return errors.ErrRentalStatusError
		}

		// 获取设备信息
		device, err := s.deviceRepo.GetByID(ctx, rental.DeviceID)
		if err != nil {
			return errors.ErrDeviceNotFound
		}

		// 发送开锁命令（用于归还）
		if s.mqttService != nil {
			_, err := s.mqttService.SendUnlockCommand(ctx, device.DeviceNo, rental.SlotNo)
			if err != nil {
				return errors.ErrUnlockFailed.WithError(err)
			}
		}

		now := time.Now()

		// 计算实际使用时长（分钟）
		var duration int
		if rental.StartTime != nil {
			duration = int(now.Sub(*rental.StartTime).Minutes())
		}

		// 更新订单状态
		updates := map[string]interface{}{
			"status":      models.RentalStatusReturned,
			"returned_at": now,
			"duration":    duration,
		}
		if err := tx.Model(rental).Updates(updates).Error; err != nil {
			return errors.ErrDatabaseError.WithError(err)
		}

		// 更新设备状态
		if err := tx.Model(&models.Device{}).Where("id = ?", rental.DeviceID).Updates(map[string]interface{}{
			"rental_status":     models.DeviceRentalFree,
			"current_rental_id": nil,
			"available_slots":   gorm.Expr("available_slots + 1"),
		}).Error; err != nil {
			return errors.ErrDatabaseError.WithError(err)
		}

		return nil
	})
}

// CompleteRental 完成租借（结算）
func (s *RentalService) CompleteRental(ctx context.Context, rentalID int64) error {
	return s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		rental, err := s.rentalRepo.GetForUpdate(ctx, tx, rentalID)
		if err != nil {
			if err == gorm.ErrRecordNotFound {
				return errors.ErrRentalNotFound
			}
			return errors.ErrDatabaseError.WithError(err)
		}

		if rental.Status != models.RentalStatusReturned {
			return errors.ErrRentalStatusError
		}

		// 检查是否超时，计算额外费用
		var overdueAmount float64
		if rental.EndTime != nil && rental.ReturnedAt != nil {
			if rental.ReturnedAt.After(*rental.EndTime) {
				// 超时，从押金中扣除超时费用
				overdueMinutes := rental.ReturnedAt.Sub(*rental.EndTime).Minutes()
				pricing, _ := s.deviceRepo.GetPricingByID(ctx, rental.PricingID)
				if pricing != nil {
					// 按小时计算超时费用
					overdueHours := int(overdueMinutes/60) + 1
					overdueAmount = float64(overdueHours) * pricing.Price
					if overdueAmount > rental.DepositAmount {
						overdueAmount = rental.DepositAmount
					}
				}
			}
		}

		// 结算押金
		refundAmount := rental.DepositAmount - overdueAmount
		if refundAmount > 0 {
			// 退还押金
			if err := s.walletService.UnfreezeDeposit(ctx, rental.UserID, refundAmount, rental.RentalNo); err != nil {
				return err
			}
		}
		if overdueAmount > 0 {
			// 扣除超时费用
			if err := s.walletService.DeductFrozenToConsume(ctx, rental.UserID, overdueAmount, rental.RentalNo, "超时费用"); err != nil {
				return err
			}
		}

		// 更新订单状态
		updates := map[string]interface{}{
			"status":        models.RentalStatusCompleted,
			"refund_amount": refundAmount,
			"actual_amount": rental.RentalAmount + overdueAmount,
		}
		if err := tx.Model(rental).Updates(updates).Error; err != nil {
			return errors.ErrDatabaseError.WithError(err)
		}

		return nil
	})
}

// CancelRental 取消租借
func (s *RentalService) CancelRental(ctx context.Context, userID int64, rentalID int64) error {
	return s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		rental, err := s.rentalRepo.GetForUpdate(ctx, tx, rentalID)
		if err != nil {
			if err == gorm.ErrRecordNotFound {
				return errors.ErrRentalNotFound
			}
			return errors.ErrDatabaseError.WithError(err)
		}

		if rental.UserID != userID {
			return errors.ErrPermissionDenied
		}

		if rental.Status != models.RentalStatusPending {
			return errors.ErrRentalStatusError.WithMessage("只有待支付的订单可以取消")
		}

		// 更新订单状态
		if err := tx.Model(rental).Update("status", models.RentalStatusCancelled).Error; err != nil {
			return errors.ErrDatabaseError.WithError(err)
		}

		// 恢复设备可用槽位
		if err := tx.Model(&models.Device{}).
			Where("id = ?", rental.DeviceID).
			UpdateColumn("available_slots", gorm.Expr("available_slots + 1")).Error; err != nil {
			return errors.ErrDatabaseError.WithError(err)
		}

		return nil
	})
}

// GetRental 获取租借详情
func (s *RentalService) GetRental(ctx context.Context, userID int64, rentalID int64) (*RentalInfo, error) {
	rental, err := s.rentalRepo.GetByIDWithRelations(ctx, rentalID)
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, errors.ErrRentalNotFound
		}
		return nil, errors.ErrDatabaseError.WithError(err)
	}

	if rental.UserID != userID {
		return nil, errors.ErrPermissionDenied
	}

	return s.toRentalInfo(rental, rental.Device, rental.Pricing), nil
}

// ListRentals 获取用户租借列表
func (s *RentalService) ListRentals(ctx context.Context, userID int64, offset, limit int, status *int8) ([]*RentalInfo, int64, error) {
	rentals, total, err := s.rentalRepo.ListByUser(ctx, userID, offset, limit, status)
	if err != nil {
		return nil, 0, errors.ErrDatabaseError.WithError(err)
	}

	result := make([]*RentalInfo, len(rentals))
	for i, r := range rentals {
		result[i] = s.toRentalInfo(r, r.Device, r.Pricing)
	}

	return result, total, nil
}

// toRentalInfo 转换为租借信息
func (s *RentalService) toRentalInfo(rental *models.Rental, device *models.Device, pricing *models.RentalPricing) *RentalInfo {
	info := &RentalInfo{
		ID:             rental.ID,
		RentalNo:       rental.RentalNo,
		Status:         rental.Status,
		StatusName:     s.getStatusName(rental.Status),
		StartTime:      rental.StartTime,
		EndTime:        rental.EndTime,
		Duration:       rental.Duration,
		UnitPrice:      rental.UnitPrice,
		DepositAmount:  rental.DepositAmount,
		RentalAmount:   rental.RentalAmount,
		DiscountAmount: rental.DiscountAmount,
		ActualAmount:   rental.ActualAmount,
		RefundAmount:   rental.RefundAmount,
		CreatedAt:      rental.CreatedAt,
	}

	if device != nil {
		info.Device = &deviceService.DeviceInfo{
			ID:           device.ID,
			DeviceNo:     device.DeviceNo,
			Name:         device.Name,
			Type:         device.Type,
			ProductName:  device.ProductName,
			ProductImage: device.ProductImage,
		}
	}

	if pricing != nil {
		info.Pricing = &deviceService.PricingInfo{
			ID:           pricing.ID,
			Name:         pricing.Name,
			Duration:     pricing.Duration,
			DurationUnit: pricing.DurationUnit,
			Price:        pricing.Price,
			Deposit:      pricing.Deposit,
		}
	}

	return info
}

// getStatusName 获取状态名称
func (s *RentalService) getStatusName(status int8) string {
	switch status {
	case models.RentalStatusPending:
		return "待支付"
	case models.RentalStatusPaid:
		return "待取货"
	case models.RentalStatusInUse:
		return "使用中"
	case models.RentalStatusReturned:
		return "已归还"
	case models.RentalStatusCompleted:
		return "已完成"
	case models.RentalStatusCancelled:
		return "已取消"
	case models.RentalStatusOverdue:
		return "超时未还"
	default:
		return "未知"
	}
}

// GenerateRentalNo 生成租借订单号
func GenerateRentalNo() string {
	return fmt.Sprintf("R%s", time.Now().Format("20060102150405"))
}
