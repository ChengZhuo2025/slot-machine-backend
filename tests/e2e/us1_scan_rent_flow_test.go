//go:build e2e
// +build e2e

package e2e

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"

	"github.com/dumeirei/smart-locker-backend/internal/common/jwt"
	deviceHandler "github.com/dumeirei/smart-locker-backend/internal/handler/device"
	paymentHandler "github.com/dumeirei/smart-locker-backend/internal/handler/payment"
	rentalHandler "github.com/dumeirei/smart-locker-backend/internal/handler/rental"
	userMiddleware "github.com/dumeirei/smart-locker-backend/internal/middleware"
	"github.com/dumeirei/smart-locker-backend/internal/models"
	"github.com/dumeirei/smart-locker-backend/internal/repository"
	deviceService "github.com/dumeirei/smart-locker-backend/internal/service/device"
	paymentService "github.com/dumeirei/smart-locker-backend/internal/service/payment"
	rentalService "github.com/dumeirei/smart-locker-backend/internal/service/rental"
	userService "github.com/dumeirei/smart-locker-backend/internal/service/user"
)

type apiResp struct {
	Code    int             `json:"code"`
	Message string          `json:"message"`
	Data    json.RawMessage `json:"data"`
}

func setupUS1E2E(t *testing.T) (*gin.Engine, *gorm.DB, *jwt.Manager, *rentalService.RentalService) {
	t.Helper()

	gin.SetMode(gin.TestMode)
	engine := gin.New()

	dsn := fmt.Sprintf("file:%s?mode=memory&cache=shared", strings.ReplaceAll(t.Name(), "/", "_"))
	db, err := gorm.Open(sqlite.Open(dsn), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Silent),
	})
	require.NoError(t, err)

	sqlDB, err := db.DB()
	require.NoError(t, err)
	sqlDB.SetMaxOpenConns(1)
	sqlDB.SetMaxIdleConns(1)

	require.NoError(t, db.AutoMigrate(
		&models.User{},
		&models.UserWallet{},
		&models.MemberLevel{},
		&models.WalletTransaction{},
		&models.Merchant{},
		&models.Venue{},
		&models.Device{},
		&models.RentalPricing{},
		&models.Order{},
		&models.Rental{},
		&models.Payment{},
		&models.Refund{},
	))

	db.Create(&models.MemberLevel{ID: 1, Name: "普通会员", Level: 1, MinPoints: 0, Discount: 1.0})

	jwtManager := jwt.NewManager(&jwt.Config{
		Secret:            "test-secret-key-us1-e2e",
		AccessExpireTime:  time.Hour,
		RefreshExpireTime: 2 * time.Hour,
		Issuer:            "test",
	})

	userRepo := repository.NewUserRepository(db)
	deviceRepo := repository.NewDeviceRepository(db)
	venueRepo := repository.NewVenueRepository(db)
	rentalRepo := repository.NewRentalRepository(db)
	paymentRepo := repository.NewPaymentRepository(db)
	refundRepo := repository.NewRefundRepository(db)

	deviceSvc := deviceService.NewDeviceService(db, deviceRepo, venueRepo)
	venueSvc := deviceService.NewVenueService(db, venueRepo, deviceRepo)
	walletSvc := userService.NewWalletService(db, userRepo)
	rentalSvc := rentalService.NewRentalService(db, rentalRepo, deviceRepo, deviceSvc, walletSvc, nil)
	paymentSvc := paymentService.NewPaymentService(db, paymentRepo, refundRepo, rentalRepo, nil)

	deviceH := deviceHandler.NewHandler(deviceSvc, venueSvc)
	rentalH := rentalHandler.NewHandler(rentalSvc)
	paymentH := paymentHandler.NewHandler(paymentSvc)

	v1 := engine.Group("/api/v1")
	{
		public := v1.Group("")
		deviceH.RegisterRoutes(public)

		user := v1.Group("")
		user.Use(userMiddleware.UserAuth(jwtManager))
		rentalH.RegisterRoutes(user)
		paymentH.RegisterRoutes(user)
	}

	return engine, db, jwtManager, rentalSvc
}

func TestUS1_E2E_ScanPayUnlockReturnSettle(t *testing.T) {
	router, db, jwtManager, rentalSvc := setupUS1E2E(t)

	// Seed
	phone := "13800138000"
	user := &models.User{Phone: &phone, Nickname: "测试用户", MemberLevelID: 1, Status: models.UserStatusActive}
	require.NoError(t, db.Create(user).Error)
	require.NoError(t, db.Create(&models.UserWallet{UserID: user.ID, Balance: 200.0}).Error)

	merchant := &models.Merchant{Name: "测试商户", ContactName: "联系人", ContactPhone: "13900139000", CommissionRate: 0.2, SettlementType: "monthly", Status: models.MerchantStatusActive}
	require.NoError(t, db.Create(merchant).Error)
	venue := &models.Venue{MerchantID: merchant.ID, Name: "测试场地", Type: "mall", Province: "广东省", City: "深圳市", District: "南山区", Address: "科技园路1号", Status: models.VenueStatusActive}
	require.NoError(t, db.Create(venue).Error)
	device := &models.Device{
		DeviceNo:       "D20240101001",
		Name:           "测试设备",
		Type:           models.DeviceTypeStandard,
		VenueID:        venue.ID,
		QRCode:         "https://qr.example.com/D20240101001",
		ProductName:    "测试产品",
		SlotCount:      1,
		AvailableSlots: 1,
		OnlineStatus:   models.DeviceOnline,
		LockStatus:     models.DeviceLocked,
		RentalStatus:   models.DeviceRentalFree,
		NetworkType:    "WiFi",
		Status:         models.DeviceStatusActive,
	}
	require.NoError(t, db.Create(device).Error)
	pricing := &models.RentalPricing{VenueID: &venue.ID, DurationHours: 1, Price: 10.0, Deposit: 50.0, OvertimeRate: 1.5, IsActive: true}
	require.NoError(t, db.Create(pricing).Error)

	tokenPair, err := jwtManager.GenerateTokenPair(user.ID, jwt.UserTypeUser, "")
	require.NoError(t, err)
	authz := "Bearer " + tokenPair.AccessToken

	// 1) Scan
	scanReq, _ := http.NewRequest("GET", "/api/v1/device/scan?qr_code="+device.QRCode, nil)
	scanW := httptest.NewRecorder()
	router.ServeHTTP(scanW, scanReq)
	require.Equal(t, http.StatusOK, scanW.Code)

	// 2) Create rental
	createBody, _ := json.Marshal(map[string]interface{}{"device_id": device.ID, "pricing_id": pricing.ID})
	createReq, _ := http.NewRequest("POST", "/api/v1/rental", bytes.NewBuffer(createBody))
	createReq.Header.Set("Content-Type", "application/json")
	createReq.Header.Set("Authorization", authz)
	createW := httptest.NewRecorder()
	router.ServeHTTP(createW, createReq)
	require.Equal(t, http.StatusOK, createW.Code)

	var createAPIResp apiResp
	require.NoError(t, json.NewDecoder(bytes.NewReader(createW.Body.Bytes())).Decode(&createAPIResp))
	require.Equal(t, 0, createAPIResp.Code)
	var rentalData struct {
		ID int64 `json:"id"`
	}
	require.NoError(t, json.Unmarshal(createAPIResp.Data, &rentalData))
	require.NotZero(t, rentalData.ID)
	rentalIDStr := strconv.FormatInt(rentalData.ID, 10)

	// 3) Pay
	payReq, _ := http.NewRequest("POST", "/api/v1/rental/"+rentalIDStr+"/pay", nil)
	payReq.Header.Set("Authorization", authz)
	payW := httptest.NewRecorder()
	router.ServeHTTP(payW, payReq)
	require.Equal(t, http.StatusOK, payW.Code)

	// 4) Start (unlock)
	startReq, _ := http.NewRequest("POST", "/api/v1/rental/"+rentalIDStr+"/start", nil)
	startReq.Header.Set("Authorization", authz)
	startW := httptest.NewRecorder()
	router.ServeHTTP(startW, startReq)
	require.Equal(t, http.StatusOK, startW.Code)

	// 5) Return
	returnReq, _ := http.NewRequest("POST", "/api/v1/rental/"+rentalIDStr+"/return", nil)
	returnReq.Header.Set("Authorization", authz)
	returnW := httptest.NewRecorder()
	router.ServeHTTP(returnW, returnReq)
	require.Equal(t, http.StatusOK, returnW.Code)

	// 6) Settle (simulate scheduler)
	require.NoError(t, rentalSvc.CompleteRental(context.Background(), rentalData.ID))

	// 7) Verify wallet
	var wallet models.UserWallet
	require.NoError(t, db.Where("user_id = ?", user.ID).First(&wallet).Error)
	assert.Equal(t, 200.0-10.0, wallet.Balance)
	assert.Equal(t, float64(0), wallet.FrozenBalance)
}
