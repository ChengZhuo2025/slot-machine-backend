// Package rental 提供租借服务
package rental

import (
	"context"
	"fmt"
	"time"

	"gorm.io/gorm"

	"github.com/dumeirei/smart-locker-backend/internal/common/errors"
	"github.com/dumeirei/smart-locker-backend/internal/common/utils"
	"github.com/dumeirei/smart-locker-backend/internal/models"
	"github.com/dumeirei/smart-locker-backend/internal/repository"
	deviceService "github.com/dumeirei/smart-locker-backend/internal/service/device"
	userService "github.com/dumeirei/smart-locker-backend/internal/service/user"
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
	ID               int64                     `json:"id"`
	OrderID          int64                     `json:"order_id"`
	OrderNo          string                    `json:"order_no"`
	Status           string                    `json:"status"`
	StatusName       string                    `json:"status_name"`
	Device           *deviceService.DeviceInfo  `json:"device,omitempty"`
	DurationHours    int                       `json:"duration_hours"`
	RentalFee        float64                   `json:"rental_fee"`
	Deposit          float64                   `json:"deposit"`
	OvertimeRate     float64                   `json:"overtime_rate"`
	OvertimeFee      float64                   `json:"overtime_fee"`
	UnlockedAt       *time.Time                `json:"unlocked_at,omitempty"`
	ExpectedReturnAt *time.Time                `json:"expected_return_at,omitempty"`
	ReturnedAt       *time.Time                `json:"returned_at,omitempty"`
	IsPurchased      bool                      `json:"is_purchased"`
	CreatedAt        time.Time                 `json:"created_at"`
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

	if !pricing.IsActive {
		return nil, errors.ErrPricingNotFound
	}

	// 计算总金额
	totalAmount := pricing.Price + pricing.Deposit

	// 使用事务创建Order和Rental
	var rental *models.Rental
	var order *models.Order

	err = s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		// 1. 创建Order记录
		orderNo := utils.GenerateOrderNo("O")
		order = &models.Order{
			OrderNo:        orderNo,
			UserID:         userID,
			Type:           "rental",
			OriginalAmount: totalAmount,
			DiscountAmount: 0,
			ActualAmount:   totalAmount,
			Status:         models.OrderStatusPending,
		}

		if err := tx.Create(order).Error; err != nil {
			return err
		}

		// 2. 创建Rental记录
		expectedReturn := time.Now().Add(time.Duration(pricing.DurationHours) * time.Hour)
		rental = &models.Rental{
			OrderID:          order.ID,
			UserID:           userID,
			DeviceID:         req.DeviceID,
			DurationHours:    pricing.DurationHours,
			RentalFee:        pricing.Price,
			Deposit:          pricing.Deposit,
			OvertimeRate:     pricing.OvertimeRate,
			OvertimeFee:      0,
			Status:           models.RentalStatusPending,
			ExpectedReturnAt: &expectedReturn,
		}

		if err := tx.Create(rental).Error; err != nil {
			return err
		}

		// 3. 减少设备可用槽位（预占）
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

		// TODO: 对接钱包服务 - 冻结押金 + 扣除租金
		// 临时注释,等钱包服务适配新字段后再启用
		/*
		if rental.Deposit > 0 {
			orderNo := fmt.Sprintf("R%d", rental.OrderID)
			if err := s.walletService.FreezeDeposit(ctx, userID, rental.Deposit, orderNo); err != nil {
				return err
			}
		}

		if rental.RentalFee > 0 {
			orderNo := fmt.Sprintf("R%d", rental.OrderID)
			if err := s.walletService.Consume(ctx, userID, rental.RentalFee, orderNo); err != nil {
				return err
			}
		}
		*/

		// 更新订单状态
		updates := map[string]interface{}{
			"status": models.RentalStatusPaid,
		}
		if err := tx.Model(rental).Updates(updates).Error; err != nil {
			return errors.ErrDatabaseError.WithError(err)
		}

		// 同时更新Order状态
		if err := tx.Model(&models.Order{}).Where("id = ?", rental.OrderID).
			Update("status", models.OrderStatusPaid).Error; err != nil {
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

		// 获取设备信息(用于后续MQTT命令)
		_, err = s.deviceRepo.GetByID(ctx, rental.DeviceID)
		if err != nil {
			return errors.ErrDeviceNotFound
		}

		// TODO: 发送开锁命令 (MQTT服务集成)
		// 临时注释,等MQTT服务完善后启用
		/*
		if s.mqttService != nil {
			_, err := s.mqttService.SendUnlockCommand(ctx, device.DeviceNo, nil)
			if err != nil {
				return errors.ErrUnlockFailed.WithError(err)
			}
		}
		*/

		now := time.Now()

		// 更新租借状态
		updates := map[string]interface{}{
			"status":      models.RentalStatusInUse,
			"unlocked_at": now,
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

		// TODO: MQTT开锁命令(归还时)
		now := time.Now()

		// 计算超时费用
		var overtimeFee float64
		if rental.ExpectedReturnAt != nil && now.After(*rental.ExpectedReturnAt) {
			// 超时,计算超时费用
			overtimeHours := int(now.Sub(*rental.ExpectedReturnAt).Hours()) + 1
			overtimeFee = float64(overtimeHours) * rental.OvertimeRate
			// 超时费用不能超过押金
			if overtimeFee > rental.Deposit {
				overtimeFee = rental.Deposit
			}
		}

		// 更新租借状态
		updates := map[string]interface{}{
			"status":       models.RentalStatusReturned,
			"returned_at":  now,
			"overtime_fee": overtimeFee,
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

		// TODO: 钱包服务 - 退还押金或扣除超时费

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

		// TODO: 结算逻辑 - 钱包服务退还押金或扣除超时费

		// 更新订单状态
		updates := map[string]interface{}{
			"status": models.RentalStatusCompleted,
		}
		if err := tx.Model(rental).Updates(updates).Error; err != nil {
			return errors.ErrDatabaseError.WithError(err)
		}

		// 更新Order状态
		if err := tx.Model(&models.Order{}).Where("id = ?", rental.OrderID).
			Update("status", models.OrderStatusCompleted).Error; err != nil {
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

		// 更新租借状态
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

	return s.toRentalInfo(rental, rental.Device, nil), nil
}

// ListRentals 获取用户租借列表
func (s *RentalService) ListRentals(ctx context.Context, userID int64, offset, limit int, status *string) ([]*RentalInfo, int64, error) {
	rentals, total, err := s.rentalRepo.ListByUser(ctx, userID, offset, limit, status)
	if err != nil {
		return nil, 0, errors.ErrDatabaseError.WithError(err)
	}

	result := make([]*RentalInfo, len(rentals))
	for i, r := range rentals {
		result[i] = s.toRentalInfo(r, r.Device, nil)
	}

	return result, total, nil
}

// toRentalInfo 转换为租借信息
func (s *RentalService) toRentalInfo(rental *models.Rental, device *models.Device, _ *models.RentalPricing) *RentalInfo {
	// 获取Order信息
	var order models.Order
	if rental.OrderID > 0 {
		s.db.Where("id = ?", rental.OrderID).First(&order)
	}

	info := &RentalInfo{
		ID:               rental.ID,
		OrderID:          rental.OrderID,
		Status:           rental.Status,
		StatusName:       s.getStatusName(rental.Status),
		DurationHours:    rental.DurationHours,
		RentalFee:        rental.RentalFee,
		Deposit:          rental.Deposit,
		OvertimeRate:     rental.OvertimeRate,
		OvertimeFee:      rental.OvertimeFee,
		UnlockedAt:       rental.UnlockedAt,
		ExpectedReturnAt: rental.ExpectedReturnAt,
		ReturnedAt:       rental.ReturnedAt,
		IsPurchased:      rental.IsPurchased,
		CreatedAt:        rental.CreatedAt,
	}

	// 如果有Order，添加OrderNo
	if rental.OrderID > 0 {
		info.OrderNo = order.OrderNo
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

	return info
}

// getStatusName 获取状态名称
func (s *RentalService) getStatusName(status string) string {
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
	case models.RentalStatusRefunding:
		return "退款中"
	case models.RentalStatusRefunded:
		return "已退款"
	default:
		return "未知"
	}
}

// GenerateRentalNo 生成租借订单号
func GenerateRentalNo() string {
	return fmt.Sprintf("R%s", time.Now().Format("20060102150405"))
}
