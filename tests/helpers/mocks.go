// Package helpers 提供 mock 实现
package helpers

import (
	"context"

	"github.com/stretchr/testify/mock"
	"gorm.io/gorm"

	"github.com/dumeirei/smart-locker-backend/internal/models"
)

// MockUserRepository 用户仓储 mock
type MockUserRepository struct {
	mock.Mock
}

func (m *MockUserRepository) Create(ctx context.Context, user *models.User) error {
	args := m.Called(ctx, user)
	return args.Error(0)
}

func (m *MockUserRepository) GetByID(ctx context.Context, id int64) (*models.User, error) {
	args := m.Called(ctx, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.User), args.Error(1)
}

func (m *MockUserRepository) GetByPhone(ctx context.Context, phone string) (*models.User, error) {
	args := m.Called(ctx, phone)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.User), args.Error(1)
}

func (m *MockUserRepository) GetByOpenID(ctx context.Context, openID string) (*models.User, error) {
	args := m.Called(ctx, openID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.User), args.Error(1)
}

func (m *MockUserRepository) GetByInviteCode(ctx context.Context, inviteCode string) (*models.User, error) {
	args := m.Called(ctx, inviteCode)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.User), args.Error(1)
}

func (m *MockUserRepository) Update(ctx context.Context, user *models.User) error {
	args := m.Called(ctx, user)
	return args.Error(0)
}

func (m *MockUserRepository) UpdateFields(ctx context.Context, id int64, fields map[string]interface{}) error {
	args := m.Called(ctx, id, fields)
	return args.Error(0)
}

func (m *MockUserRepository) UpdateStatus(ctx context.Context, id int64, status int8) error {
	args := m.Called(ctx, id, status)
	return args.Error(0)
}

func (m *MockUserRepository) List(ctx context.Context, offset, limit int, filters map[string]interface{}) ([]*models.User, int64, error) {
	args := m.Called(ctx, offset, limit, filters)
	if args.Get(0) == nil {
		return nil, args.Get(1).(int64), args.Error(2)
	}
	return args.Get(0).([]*models.User), args.Get(1).(int64), args.Error(2)
}

func (m *MockUserRepository) ExistsByPhone(ctx context.Context, phone string) (bool, error) {
	args := m.Called(ctx, phone)
	return args.Bool(0), args.Error(1)
}

func (m *MockUserRepository) GetByIDWithWallet(ctx context.Context, id int64) (*models.User, error) {
	args := m.Called(ctx, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.User), args.Error(1)
}

// MockRentalRepository 租借仓储 mock
type MockRentalRepository struct {
	mock.Mock
}

func (m *MockRentalRepository) Create(ctx context.Context, rental *models.Rental) error {
	args := m.Called(ctx, rental)
	return args.Error(0)
}

func (m *MockRentalRepository) GetByID(ctx context.Context, id int64) (*models.Rental, error) {
	args := m.Called(ctx, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.Rental), args.Error(1)
}

func (m *MockRentalRepository) GetByIDWithRelations(ctx context.Context, id int64) (*models.Rental, error) {
	args := m.Called(ctx, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.Rental), args.Error(1)
}

func (m *MockRentalRepository) GetByRentalNo(ctx context.Context, rentalNo string) (*models.Rental, error) {
	args := m.Called(ctx, rentalNo)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.Rental), args.Error(1)
}

func (m *MockRentalRepository) Update(ctx context.Context, rental *models.Rental) error {
	args := m.Called(ctx, rental)
	return args.Error(0)
}

func (m *MockRentalRepository) UpdateFields(ctx context.Context, id int64, fields map[string]interface{}) error {
	args := m.Called(ctx, id, fields)
	return args.Error(0)
}

func (m *MockRentalRepository) UpdateStatus(ctx context.Context, id int64, status int8) error {
	args := m.Called(ctx, id, status)
	return args.Error(0)
}

func (m *MockRentalRepository) ListByUser(ctx context.Context, userID int64, offset, limit int, status *int8) ([]*models.Rental, int64, error) {
	args := m.Called(ctx, userID, offset, limit, status)
	if args.Get(0) == nil {
		return nil, args.Get(1).(int64), args.Error(2)
	}
	return args.Get(0).([]*models.Rental), args.Get(1).(int64), args.Error(2)
}

func (m *MockRentalRepository) HasActiveRental(ctx context.Context, userID int64) (bool, error) {
	args := m.Called(ctx, userID)
	return args.Bool(0), args.Error(1)
}

func (m *MockRentalRepository) GetActiveByUser(ctx context.Context, userID int64) (*models.Rental, error) {
	args := m.Called(ctx, userID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.Rental), args.Error(1)
}

func (m *MockRentalRepository) GetForUpdate(ctx context.Context, tx *gorm.DB, id int64) (*models.Rental, error) {
	args := m.Called(ctx, tx, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.Rental), args.Error(1)
}

// MockPaymentRepository 支付仓储 mock
type MockPaymentRepository struct {
	mock.Mock
}

func (m *MockPaymentRepository) Create(ctx context.Context, payment *models.Payment) error {
	args := m.Called(ctx, payment)
	return args.Error(0)
}

func (m *MockPaymentRepository) GetByID(ctx context.Context, id int64) (*models.Payment, error) {
	args := m.Called(ctx, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.Payment), args.Error(1)
}

func (m *MockPaymentRepository) GetByPaymentNo(ctx context.Context, paymentNo string) (*models.Payment, error) {
	args := m.Called(ctx, paymentNo)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.Payment), args.Error(1)
}

func (m *MockPaymentRepository) Update(ctx context.Context, payment *models.Payment) error {
	args := m.Called(ctx, payment)
	return args.Error(0)
}

func (m *MockPaymentRepository) UpdateFields(ctx context.Context, id int64, fields map[string]interface{}) error {
	args := m.Called(ctx, id, fields)
	return args.Error(0)
}

func (m *MockPaymentRepository) GetPendingExpired(ctx context.Context, expiredBefore interface{}, limit int) ([]*models.Payment, error) {
	args := m.Called(ctx, expiredBefore, limit)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*models.Payment), args.Error(1)
}

// MockDeviceRepository 设备仓储 mock
type MockDeviceRepository struct {
	mock.Mock
}

func (m *MockDeviceRepository) GetByID(ctx context.Context, id int64) (*models.Device, error) {
	args := m.Called(ctx, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.Device), args.Error(1)
}

func (m *MockDeviceRepository) GetByDeviceNo(ctx context.Context, deviceNo string) (*models.Device, error) {
	args := m.Called(ctx, deviceNo)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.Device), args.Error(1)
}

func (m *MockDeviceRepository) Update(ctx context.Context, device *models.Device) error {
	args := m.Called(ctx, device)
	return args.Error(0)
}

func (m *MockDeviceRepository) GetPricingByID(ctx context.Context, id int64) (*models.RentalPricing, error) {
	args := m.Called(ctx, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.RentalPricing), args.Error(1)
}

// MockCodeService 验证码服务 mock
type MockCodeService struct {
	mock.Mock
}

func (m *MockCodeService) SendCode(ctx context.Context, phone string, codeType string) error {
	args := m.Called(ctx, phone, codeType)
	return args.Error(0)
}

func (m *MockCodeService) VerifyCode(ctx context.Context, phone, code, codeType string) (bool, error) {
	args := m.Called(ctx, phone, code, codeType)
	return args.Bool(0), args.Error(1)
}

// MockWalletService 钱包服务 mock
type MockWalletService struct {
	mock.Mock
}

func (m *MockWalletService) CheckBalance(ctx context.Context, userID int64, amount float64) (bool, error) {
	args := m.Called(ctx, userID, amount)
	return args.Bool(0), args.Error(1)
}

func (m *MockWalletService) Consume(ctx context.Context, userID int64, amount float64, orderNo string) error {
	args := m.Called(ctx, userID, amount, orderNo)
	return args.Error(0)
}

func (m *MockWalletService) FreezeDeposit(ctx context.Context, userID int64, amount float64, orderNo string) error {
	args := m.Called(ctx, userID, amount, orderNo)
	return args.Error(0)
}

func (m *MockWalletService) UnfreezeDeposit(ctx context.Context, userID int64, amount float64, orderNo string) error {
	args := m.Called(ctx, userID, amount, orderNo)
	return args.Error(0)
}

func (m *MockWalletService) DeductFrozenToConsume(ctx context.Context, userID int64, amount float64, orderNo, remark string) error {
	args := m.Called(ctx, userID, amount, orderNo, remark)
	return args.Error(0)
}

// MockMQTTService MQTT 服务 mock
type MockMQTTService struct {
	mock.Mock
}

func (m *MockMQTTService) SendUnlockCommand(ctx context.Context, deviceNo string, slotNo *int) (bool, error) {
	args := m.Called(ctx, deviceNo, slotNo)
	return args.Bool(0), args.Error(1)
}

// MockRefundRepository 退款仓储 mock
type MockRefundRepository struct {
	mock.Mock
}

func (m *MockRefundRepository) Create(ctx context.Context, refund *models.Refund) error {
	args := m.Called(ctx, refund)
	return args.Error(0)
}

func (m *MockRefundRepository) GetTotalRefunded(ctx context.Context, paymentID int64) (float64, error) {
	args := m.Called(ctx, paymentID)
	return args.Get(0).(float64), args.Error(1)
}
