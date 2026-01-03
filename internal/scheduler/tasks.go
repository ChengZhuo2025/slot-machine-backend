// Package scheduler 提供定时任务
package scheduler

import (
	"context"
	"log"
	"time"

	"gorm.io/gorm"

	"github.com/dumeirei/smart-locker-backend/internal/models"
	"github.com/dumeirei/smart-locker-backend/internal/repository"
	paymentService "github.com/dumeirei/smart-locker-backend/internal/service/payment"
	rentalService "github.com/dumeirei/smart-locker-backend/internal/service/rental"
)

// TaskHandler 任务处理器
type TaskHandler struct {
	db             *gorm.DB
	rentalRepo     *repository.RentalRepository
	deviceRepo     *repository.DeviceRepository
	paymentService *paymentService.PaymentService
	rentalService  *rentalService.RentalService
}

// NewTaskHandler 创建任务处理器
func NewTaskHandler(
	db *gorm.DB,
	rentalRepo *repository.RentalRepository,
	deviceRepo *repository.DeviceRepository,
	paymentSvc *paymentService.PaymentService,
	rentalSvc *rentalService.RentalService,
) *TaskHandler {
	return &TaskHandler{
		db:             db,
		rentalRepo:     rentalRepo,
		deviceRepo:     deviceRepo,
		paymentService: paymentSvc,
		rentalService:  rentalSvc,
	}
}

// CloseExpiredRentals 关闭过期的待支付租借
func (h *TaskHandler) CloseExpiredRentals(ctx context.Context) error {
	expiredBefore := time.Now().Add(-30 * time.Minute) // 30分钟未支付自动关闭

	rentals, err := h.rentalRepo.GetExpiredPending(ctx, expiredBefore, 100)
	if err != nil {
		return err
	}

	if len(rentals) == 0 {
		return nil
	}

	log.Printf("[Task] Found %d expired rentals to close", len(rentals))

	for _, rental := range rentals {
		err := h.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
			// 更新租借状态
			if err := tx.Model(rental).Update("status", models.RentalStatusCancelled).Error; err != nil {
				return err
			}

			// 恢复设备可用槽位
			if err := tx.Model(&models.Device{}).
				Where("id = ?", rental.DeviceID).
				UpdateColumn("available_slots", gorm.Expr("available_slots + 1")).Error; err != nil {
				return err
			}

			return nil
		})

		if err != nil {
			log.Printf("[Task] Failed to close rental %d: %v", rental.ID, err)
		}
	}

	return nil
}

// HandleOverdueRentals 处理超时未还的租借
func (h *TaskHandler) HandleOverdueRentals(ctx context.Context) error {
	rentals, err := h.rentalRepo.GetOverdue(ctx, 100)
	if err != nil {
		return err
	}

	if len(rentals) == 0 {
		return nil
	}

	log.Printf("[Task] Found %d overdue rentals", len(rentals))

	for _, rental := range rentals {
		// 更新为超时状态
		if err := h.rentalRepo.UpdateStatus(ctx, rental.ID, models.RentalStatusOverdue); err != nil {
			log.Printf("[Task] Failed to mark rental %d as overdue: %v", rental.ID, err)
		}

		// TODO: 发送超时通知
	}

	return nil
}

// SetOfflineDevices 设置离线设备
func (h *TaskHandler) SetOfflineDevices(ctx context.Context) error {
	offlineThreshold := time.Now().Add(-5 * time.Minute) // 5分钟未心跳视为离线

	result := h.db.WithContext(ctx).Model(&models.Device{}).
		Where("online_status = ?", models.DeviceOnline).
		Where("last_heartbeat_at < ?", offlineThreshold).
		Updates(map[string]interface{}{
			"online_status":   models.DeviceOffline,
			"last_offline_at": time.Now(),
		})

	if result.Error != nil {
		return result.Error
	}

	if result.RowsAffected > 0 {
		log.Printf("[Task] Set %d devices offline", result.RowsAffected)
	}

	return nil
}

// CloseExpiredPayments 关闭过期支付
func (h *TaskHandler) CloseExpiredPayments(ctx context.Context) error {
	if h.paymentService != nil {
		return h.paymentService.CloseExpiredPayments(ctx)
	}
	return nil
}

// CompleteReturnedRentals 完成已归还的租借（自动结算）
func (h *TaskHandler) CompleteReturnedRentals(ctx context.Context) error {
	// 查找已归还超过一定时间的租借进行自动结算
	var rentals []*models.Rental
	settleBefore := time.Now().Add(-5 * time.Minute) // 归还5分钟后自动结算

	err := h.db.WithContext(ctx).
		Where("status = ?", models.RentalStatusReturned).
		Where("returned_at < ?", settleBefore).
		Limit(50).
		Find(&rentals).Error

	if err != nil {
		return err
	}

	if len(rentals) == 0 {
		return nil
	}

	log.Printf("[Task] Found %d returned rentals to complete", len(rentals))

	for _, rental := range rentals {
		if err := h.rentalService.CompleteRental(ctx, rental.ID); err != nil {
			log.Printf("[Task] Failed to complete rental %d: %v", rental.ID, err)
		}
	}

	return nil
}

// SetupTasks 设置所有任务
func SetupTasks(scheduler *Scheduler, handler *TaskHandler) {
	// 每分钟检查过期租借
	scheduler.AddTask("CloseExpiredRentals", 1*time.Minute, handler.CloseExpiredRentals)

	// 每分钟检查超时租借
	scheduler.AddTask("HandleOverdueRentals", 1*time.Minute, handler.HandleOverdueRentals)

	// 每分钟检查设备离线状态
	scheduler.AddTask("SetOfflineDevices", 1*time.Minute, handler.SetOfflineDevices)

	// 每分钟关闭过期支付
	scheduler.AddTask("CloseExpiredPayments", 1*time.Minute, handler.CloseExpiredPayments)

	// 每分钟完成已归还的租借
	scheduler.AddTask("CompleteReturnedRentals", 1*time.Minute, handler.CompleteReturnedRentals)
}
