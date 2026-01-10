package rental

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	appErrors "github.com/dumeirei/smart-locker-backend/internal/common/errors"
	"github.com/dumeirei/smart-locker-backend/internal/models"
)

func TestRentalService_CancelRental(t *testing.T) {
	svc := setupTestRentalService(t)
	ctx := context.Background()
	user, device, pricing := createTestData(t, svc.db)

	rentalInfo, err := svc.CreateRental(ctx, user.ID, &CreateRentalRequest{
		DeviceID:  device.ID,
		PricingID: pricing.ID,
	})
	require.NoError(t, err)

	require.NoError(t, svc.CancelRental(ctx, user.ID, rentalInfo.ID))

	var updated models.Rental
	require.NoError(t, svc.db.First(&updated, rentalInfo.ID).Error)
	assert.Equal(t, models.RentalStatusCancelled, updated.Status)

	var updatedDevice models.Device
	require.NoError(t, svc.db.First(&updatedDevice, device.ID).Error)
	assert.Equal(t, 1, updatedDevice.AvailableSlots)

	err = svc.CancelRental(ctx, user.ID, rentalInfo.ID)
	require.Error(t, err)
	var appErr *appErrors.AppError
	require.ErrorAs(t, err, &appErr)
	assert.Equal(t, appErrors.ErrRentalStatusError.Code, appErr.Code)
	assert.Contains(t, appErr.Message, "只有待支付")
}

func TestRentalService_CancelRental_PermissionDenied(t *testing.T) {
	svc := setupTestRentalService(t)
	ctx := context.Background()
	user, device, pricing := createTestData(t, svc.db)

	rentalInfo, err := svc.CreateRental(ctx, user.ID, &CreateRentalRequest{
		DeviceID:  device.ID,
		PricingID: pricing.ID,
	})
	require.NoError(t, err)

	otherPhone := "13800138088"
	other := &models.User{Phone: &otherPhone, Nickname: "其他用户", MemberLevelID: 1, Status: models.UserStatusActive}
	require.NoError(t, svc.db.Create(other).Error)
	require.NoError(t, svc.db.Create(&models.UserWallet{UserID: other.ID, Balance: 100}).Error)

	err = svc.CancelRental(ctx, other.ID, rentalInfo.ID)
	require.Error(t, err)
	var appErr *appErrors.AppError
	require.ErrorAs(t, err, &appErr)
	assert.Equal(t, appErrors.ErrPermissionDenied.Code, appErr.Code)
}

func TestRentalService_GetRental_And_ListRentals(t *testing.T) {
	svc := setupTestRentalService(t)
	ctx := context.Background()
	user, device, pricing := createTestData(t, svc.db)

	created, err := svc.CreateRental(ctx, user.ID, &CreateRentalRequest{
		DeviceID:  device.ID,
		PricingID: pricing.ID,
	})
	require.NoError(t, err)

	info, err := svc.GetRental(ctx, user.ID, created.ID)
	require.NoError(t, err)
	assert.Equal(t, created.ID, info.ID)
	assert.NotEmpty(t, info.OrderNo)
	require.NotNil(t, info.Device)
	assert.Equal(t, device.ID, info.Device.ID)

	otherPhone := "13800138077"
	other := &models.User{Phone: &otherPhone, Nickname: "其他用户", MemberLevelID: 1, Status: models.UserStatusActive}
	require.NoError(t, svc.db.Create(other).Error)
	_, err = svc.GetRental(ctx, other.ID, created.ID)
	require.Error(t, err)
	var appErr *appErrors.AppError
	require.ErrorAs(t, err, &appErr)
	assert.Equal(t, appErrors.ErrPermissionDenied.Code, appErr.Code)

	// Insert a few rentals to test list and status filter (bypass CreateRental active-rental constraint)
	for _, st := range []string{models.RentalStatusCancelled, models.RentalStatusCompleted} {
		order := &models.Order{
			OrderNo:        strings.Repeat(st, 1) + time.Now().Format("150405") + st[:1],
			UserID:         user.ID,
			Type:           models.OrderTypeRental,
			OriginalAmount: 0,
			DiscountAmount: 0,
			ActualAmount:   0,
			DepositAmount:  0,
			Status:         models.OrderStatusCompleted,
		}
		require.NoError(t, svc.db.Create(order).Error)
		r := &models.Rental{
			OrderID:       order.ID,
			UserID:        user.ID,
			DeviceID:      device.ID,
			DurationHours: 1,
			RentalFee:     0,
			Deposit:       0,
			OvertimeRate:  0,
			OvertimeFee:   0,
			Status:        st,
		}
		require.NoError(t, svc.db.Create(r).Error)
	}

	status := models.RentalStatusCancelled
	list, total, err := svc.ListRentals(ctx, user.ID, 0, 10, &status)
	require.NoError(t, err)
	assert.GreaterOrEqual(t, total, int64(1))
	for _, item := range list {
		assert.Equal(t, status, item.Status)
	}
}

func TestGenerateRentalNo(t *testing.T) {
	no := GenerateRentalNo()
	assert.True(t, strings.HasPrefix(no, "R"))
	assert.Len(t, no, 15)
	_, err := time.Parse("20060102150405", no[1:])
	assert.NoError(t, err)
}
