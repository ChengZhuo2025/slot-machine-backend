// Package hotel 提供酒店预订服务
package hotel

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
)

// BookingService 预订服务
type BookingService struct {
	db               *gorm.DB
	bookingRepo      *repository.BookingRepository
	roomRepo         *repository.RoomRepository
	hotelRepo        *repository.HotelRepository
	orderRepo        *repository.OrderRepository
	timeSlotRepo     *repository.RoomTimeSlotRepository
	codeService      *CodeService
	deviceService    *deviceService.DeviceService
	mqttService      *deviceService.MQTTService
}

// NewBookingService 创建预订服务
func NewBookingService(
	db *gorm.DB,
	bookingRepo *repository.BookingRepository,
	roomRepo *repository.RoomRepository,
	hotelRepo *repository.HotelRepository,
	orderRepo *repository.OrderRepository,
	timeSlotRepo *repository.RoomTimeSlotRepository,
	codeService *CodeService,
	deviceSvc *deviceService.DeviceService,
	mqttSvc *deviceService.MQTTService,
) *BookingService {
	return &BookingService{
		db:            db,
		bookingRepo:   bookingRepo,
		roomRepo:      roomRepo,
		hotelRepo:     hotelRepo,
		orderRepo:     orderRepo,
		timeSlotRepo:  timeSlotRepo,
		codeService:   codeService,
		deviceService: deviceSvc,
		mqttService:   mqttSvc,
	}
}

// CreateBookingRequest 创建预订请求
type CreateBookingRequest struct {
	RoomID        int64     `json:"room_id" binding:"required"`
	DurationHours int       `json:"duration_hours" binding:"required,min=1"`
	CheckInTime   time.Time `json:"check_in_time" binding:"required"`
}

// BookingInfo 预订信息
type BookingInfo struct {
	ID               int64      `json:"id"`
	BookingNo        string     `json:"booking_no"`
	OrderNo          string     `json:"order_no,omitempty"`
	Status           string     `json:"status"`
	StatusName       string     `json:"status_name"`
	Hotel            *HotelInfo `json:"hotel,omitempty"`
	Room             *RoomInfo  `json:"room,omitempty"`
	CheckInTime      time.Time  `json:"check_in_time"`
	CheckOutTime     time.Time  `json:"check_out_time"`
	DurationHours    int        `json:"duration_hours"`
	Amount           float64    `json:"amount"`
	VerificationCode string     `json:"verification_code,omitempty"`
	UnlockCode       string     `json:"unlock_code,omitempty"`
	QRCode           string     `json:"qr_code,omitempty"`
	VerifiedAt       *time.Time `json:"verified_at,omitempty"`
	UnlockedAt       *time.Time `json:"unlocked_at,omitempty"`
	CompletedAt      *time.Time `json:"completed_at,omitempty"`
	CreatedAt        time.Time  `json:"created_at"`
}

// CreateBooking 创建预订
func (s *BookingService) CreateBooking(ctx context.Context, userID int64, req *CreateBookingRequest) (*BookingInfo, error) {
	// 1. 获取房间信息
	room, err := s.roomRepo.GetByIDWithHotel(ctx, req.RoomID)
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, errors.ErrRoomNotFound
		}
		return nil, errors.ErrDatabaseError.WithError(err)
	}

	// 检查房间状态
	if room.Status != int8(models.RoomStatusActive) {
		return nil, errors.ErrRoomNotAvailable
	}

	// 检查酒店状态
	if room.Hotel == nil || room.Hotel.Status != int8(models.HotelStatusActive) {
		return nil, errors.ErrHotelNotFound
	}

	// 2. 获取时段价格
	timeSlot, err := s.timeSlotRepo.GetByRoomAndDuration(ctx, req.RoomID, req.DurationHours)
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, errors.ErrTimeSlotNotFound
		}
		return nil, errors.ErrDatabaseError.WithError(err)
	}

	if !timeSlot.IsActive {
		return nil, errors.ErrTimeSlotDisabled
	}

	// 3. 计算入住和退房时间
	checkInTime := req.CheckInTime
	checkOutTime := checkInTime.Add(time.Duration(req.DurationHours) * time.Hour)

	// 验证入住时间不能是过去
	if checkInTime.Before(time.Now().Add(-5 * time.Minute)) { // 允许5分钟的误差
		return nil, errors.ErrInvalidParams.WithMessage("入住时间不能是过去")
	}

	// 4. 检查房间可用性（时段冲突）
	exists, err := s.bookingRepo.ExistsByRoomAndTimeRange(ctx, req.RoomID, checkInTime, checkOutTime)
	if err != nil {
		return nil, errors.ErrDatabaseError.WithError(err)
	}
	if exists {
		return nil, errors.ErrBookingConflict
	}

	// 5. 生成核销码和开锁码
	verificationCode := s.codeService.GenerateVerificationCode()
	unlockCode := s.codeService.GenerateUnlockCode()
	bookingNo := utils.GenerateOrderNo("B")
	qrCode := s.codeService.GenerateQRCodeURL(bookingNo, verificationCode)

	// 6. 使用事务创建订单和预订
	var booking *models.Booking
	var order *models.Order

	err = s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		// 创建订单
		orderNo := utils.GenerateOrderNo("O")
		order = &models.Order{
			OrderNo:        orderNo,
			UserID:         userID,
			Type:           models.OrderTypeHotel,
			OriginalAmount: timeSlot.Price,
			DiscountAmount: 0,
			ActualAmount:   timeSlot.Price,
			DepositAmount:  0,
			Status:         models.OrderStatusPending,
		}
		if err := tx.Create(order).Error; err != nil {
			return err
		}

		// 创建预订
		booking = &models.Booking{
			BookingNo:        bookingNo,
			OrderID:          order.ID,
			UserID:           userID,
			HotelID:          room.HotelID,
			RoomID:           req.RoomID,
			DeviceID:         room.DeviceID,
			CheckInTime:      checkInTime,
			CheckOutTime:     checkOutTime,
			DurationHours:    req.DurationHours,
			Amount:           timeSlot.Price,
			VerificationCode: verificationCode,
			UnlockCode:       unlockCode,
			QRCode:           qrCode,
			Status:           models.BookingStatusPending,
		}
		if err := tx.Create(booking).Error; err != nil {
			return err
		}

		return nil
	})

	if err != nil {
		return nil, errors.ErrDatabaseError.WithError(err)
	}

	// 关联数据
	booking.Hotel = room.Hotel
	booking.Room = room
	booking.Order = order

	return s.convertBookingInfo(booking, true), nil
}

// GetBookingByID 根据ID获取预订
func (s *BookingService) GetBookingByID(ctx context.Context, id int64, userID int64) (*BookingInfo, error) {
	booking, err := s.bookingRepo.GetByIDWithDetails(ctx, id)
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, errors.ErrBookingNotFound
		}
		return nil, errors.ErrDatabaseError.WithError(err)
	}

	// 验证用户权限
	if booking.UserID != userID {
		return nil, errors.ErrPermissionDenied
	}

	// 根据状态决定是否显示敏感信息
	showCodes := booking.Status == models.BookingStatusPaid ||
		booking.Status == models.BookingStatusVerified ||
		booking.Status == models.BookingStatusInUse

	return s.convertBookingInfo(booking, showCodes), nil
}

// GetBookingByNo 根据预订号获取预订
func (s *BookingService) GetBookingByNo(ctx context.Context, bookingNo string, userID int64) (*BookingInfo, error) {
	booking, err := s.bookingRepo.GetByBookingNoWithDetails(ctx, bookingNo)
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, errors.ErrBookingNotFound
		}
		return nil, errors.ErrDatabaseError.WithError(err)
	}

	// 验证用户权限
	if booking.UserID != userID {
		return nil, errors.ErrPermissionDenied
	}

	showCodes := booking.Status == models.BookingStatusPaid ||
		booking.Status == models.BookingStatusVerified ||
		booking.Status == models.BookingStatusInUse

	return s.convertBookingInfo(booking, showCodes), nil
}

// GetUserBookings 获取用户预订列表
func (s *BookingService) GetUserBookings(ctx context.Context, userID int64, page, pageSize int, status *string) ([]*BookingInfo, int64, error) {
	if page <= 0 {
		page = 1
	}
	if pageSize <= 0 {
		pageSize = 10
	}
	offset := (page - 1) * pageSize

	bookings, total, err := s.bookingRepo.ListByUser(ctx, userID, offset, pageSize, status)
	if err != nil {
		return nil, 0, errors.ErrDatabaseError.WithError(err)
	}

	var result []*BookingInfo
	for _, booking := range bookings {
		showCodes := booking.Status == models.BookingStatusPaid ||
			booking.Status == models.BookingStatusVerified ||
			booking.Status == models.BookingStatusInUse
		result = append(result, s.convertBookingInfo(booking, showCodes))
	}

	return result, total, nil
}

// CancelBooking 取消预订
func (s *BookingService) CancelBooking(ctx context.Context, id int64, userID int64) error {
	booking, err := s.bookingRepo.GetByID(ctx, id)
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return errors.ErrBookingNotFound
		}
		return errors.ErrDatabaseError.WithError(err)
	}

	// 验证用户权限
	if booking.UserID != userID {
		return errors.ErrPermissionDenied
	}

	// 只有待支付状态可以取消
	if booking.Status != models.BookingStatusPending {
		return errors.ErrBookingStatusError.WithMessage("只有待支付的预订可以取消")
	}

	return s.bookingRepo.Cancel(ctx, id)
}

// VerifyBooking 核销预订（酒店前台调用）
func (s *BookingService) VerifyBooking(ctx context.Context, verificationCode string, verifiedBy int64) (*BookingInfo, error) {
	// 根据核销码查找预订
	booking, err := s.bookingRepo.GetByVerificationCode(ctx, verificationCode)
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, errors.ErrVerificationCodeInvalid
		}
		return nil, errors.ErrDatabaseError.WithError(err)
	}

	// 检查状态
	if booking.Status != models.BookingStatusPaid {
		switch booking.Status {
		case models.BookingStatusPending:
			return nil, errors.ErrBookingNotPaid
		case models.BookingStatusVerified, models.BookingStatusInUse:
			return nil, errors.ErrBookingVerified
		case models.BookingStatusCancelled:
			return nil, errors.ErrBookingCancelled
		case models.BookingStatusExpired:
			return nil, errors.ErrBookingExpired
		default:
			return nil, errors.ErrBookingStatusError
		}
	}

	// 检查是否过期（超过入住时间一定时间后不能核销）
	if time.Now().After(booking.CheckOutTime) {
		// 自动标记为过期
		_ = s.bookingRepo.MarkExpired(ctx, booking.ID)
		return nil, errors.ErrBookingExpired
	}

	// 执行核销
	if err := s.bookingRepo.Verify(ctx, booking.ID, verifiedBy); err != nil {
		return nil, errors.ErrDatabaseError.WithError(err)
	}

	// 重新获取更新后的预订信息
	booking, _ = s.bookingRepo.GetByIDWithDetails(ctx, booking.ID)

	return s.convertBookingInfo(booking, true), nil
}

// UnlockByCode 使用开锁码开锁
func (s *BookingService) UnlockByCode(ctx context.Context, deviceID int64, unlockCode string) (*BookingInfo, error) {
	// 验证开锁码格式
	if !s.codeService.ValidateUnlockCode(unlockCode) {
		return nil, errors.ErrUnlockCodeInvalid
	}

	// 根据开锁码和设备ID查找预订
	booking, err := s.bookingRepo.GetByUnlockCode(ctx, unlockCode, deviceID)
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, errors.ErrUnlockCodeInvalid
		}
		return nil, errors.ErrDatabaseError.WithError(err)
	}

	// 检查状态
	if booking.Status != models.BookingStatusVerified {
		if booking.Status == models.BookingStatusInUse {
			return nil, errors.ErrBookingStatusError.WithMessage("已开锁")
		}
		if booking.Status != models.BookingStatusVerified {
			return nil, errors.ErrBookingNotVerified
		}
	}

	// 检查是否在有效时段内
	if !s.codeService.IsUnlockCodeValid(booking.CheckInTime, booking.CheckOutTime) {
		now := time.Now()
		if now.Before(booking.CheckInTime) {
			return nil, errors.ErrBookingTimeNotArrived
		}
		if now.After(booking.CheckOutTime) {
			return nil, errors.ErrUnlockCodeExpired
		}
	}

	// 发送开锁命令
	if s.mqttService != nil && s.deviceService != nil && booking.DeviceID != nil {
		// 获取设备信息
		device, err := s.deviceService.GetDeviceByID(ctx, *booking.DeviceID)
		if err != nil {
			return nil, errors.ErrUnlockFailed.WithError(err)
		}
		// 发送开锁命令
		if _, err := s.mqttService.SendUnlockCommand(ctx, device.DeviceNo, nil); err != nil {
			return nil, errors.ErrUnlockFailed.WithError(err)
		}
	}

	// 更新预订状态
	if err := s.bookingRepo.Unlock(ctx, booking.ID); err != nil {
		return nil, errors.ErrDatabaseError.WithError(err)
	}

	// 获取更新后的预订
	booking, _ = s.bookingRepo.GetByIDWithDetails(ctx, booking.ID)

	return s.convertBookingInfo(booking, true), nil
}

// CompleteBooking 完成预订
func (s *BookingService) CompleteBooking(ctx context.Context, id int64) error {
	booking, err := s.bookingRepo.GetByID(ctx, id)
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return errors.ErrBookingNotFound
		}
		return errors.ErrDatabaseError.WithError(err)
	}

	// 只有使用中的预订可以完成
	if booking.Status != models.BookingStatusInUse && booking.Status != models.BookingStatusVerified {
		return errors.ErrBookingStatusError
	}

	return s.bookingRepo.Complete(ctx, id)
}

// OnPaymentSuccess 支付成功回调
func (s *BookingService) OnPaymentSuccess(ctx context.Context, orderID int64) error {
	booking, err := s.bookingRepo.GetByOrderID(ctx, orderID)
	if err != nil {
		return errors.ErrDatabaseError.WithError(err)
	}

	if booking.Status != models.BookingStatusPending {
		return nil // 已经处理过
	}

	return s.bookingRepo.UpdateStatus(ctx, booking.ID, models.BookingStatusPaid)
}

// ProcessExpiredBookings 处理过期预订（定时任务调用）
func (s *BookingService) ProcessExpiredBookings(ctx context.Context) error {
	// 获取过期的预订（已支付但超过入住时间）
	bookings, err := s.bookingRepo.ListExpiredBookings(ctx, 100)
	if err != nil {
		return errors.ErrDatabaseError.WithError(err)
	}

	for _, booking := range bookings {
		if err := s.bookingRepo.MarkExpired(ctx, booking.ID); err != nil {
			// 记录日志但继续处理
			fmt.Printf("标记预订过期失败: booking_id=%d, err=%v\n", booking.ID, err)
		}
	}

	return nil
}

// ProcessCompletedBookings 处理需要自动完成的预订（定时任务调用）
func (s *BookingService) ProcessCompletedBookings(ctx context.Context) error {
	// 获取需要完成的预订（已核销/使用中但超过退房时间）
	bookings, err := s.bookingRepo.ListToComplete(ctx, 100)
	if err != nil {
		return errors.ErrDatabaseError.WithError(err)
	}

	for _, booking := range bookings {
		if err := s.bookingRepo.Complete(ctx, booking.ID); err != nil {
			fmt.Printf("自动完成预订失败: booking_id=%d, err=%v\n", booking.ID, err)
		}
	}

	return nil
}

// convertBookingInfo 转换预订信息
func (s *BookingService) convertBookingInfo(booking *models.Booking, showCodes bool) *BookingInfo {
	info := &BookingInfo{
		ID:            booking.ID,
		BookingNo:     booking.BookingNo,
		Status:        booking.Status,
		StatusName:    s.getStatusName(booking.Status),
		CheckInTime:   booking.CheckInTime,
		CheckOutTime:  booking.CheckOutTime,
		DurationHours: booking.DurationHours,
		Amount:        booking.Amount,
		VerifiedAt:    booking.VerifiedAt,
		UnlockedAt:    booking.UnlockedAt,
		CompletedAt:   booking.CompletedAt,
		CreatedAt:     booking.CreatedAt,
	}

	// 敏感信息（核销码、开锁码）仅在特定状态下显示
	if showCodes {
		info.VerificationCode = booking.VerificationCode
		info.UnlockCode = booking.UnlockCode
		info.QRCode = booking.QRCode
	}

	// 订单号
	if booking.Order != nil {
		info.OrderNo = booking.Order.OrderNo
	}

	// 酒店信息
	if booking.Hotel != nil {
		hotelService := &HotelService{}
		info.Hotel = hotelService.convertHotelInfo(booking.Hotel)
	}

	// 房间信息
	if booking.Room != nil {
		hotelService := &HotelService{}
		info.Room = hotelService.convertRoomInfo(booking.Room)
	}

	return info
}

// getStatusName 获取状态名称
func (s *BookingService) getStatusName(status string) string {
	switch status {
	case models.BookingStatusPending:
		return "待支付"
	case models.BookingStatusPaid:
		return "待核销"
	case models.BookingStatusVerified:
		return "已核销"
	case models.BookingStatusInUse:
		return "使用中"
	case models.BookingStatusCompleted:
		return "已完成"
	case models.BookingStatusCancelled:
		return "已取消"
	case models.BookingStatusRefunded:
		return "已退款"
	case models.BookingStatusExpired:
		return "已过期"
	default:
		return "未知"
	}
}
