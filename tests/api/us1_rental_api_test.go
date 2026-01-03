//go:build api
// +build api

package api

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"strconv"
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

func setupUS1APIRouter(t *testing.T) (*gin.Engine, *gorm.DB, *jwt.Manager) {
	t.Helper()

	gin.SetMode(gin.TestMode)
	r := gin.New()

	dsn := fmt.Sprintf("file:%s?mode=memory&cache=shared", strings.ReplaceAll(t.Name(), "/", "_"))
	db, err := gorm.Open(sqlite.Open(dsn), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Silent),
	})
	require.NoError(t, err)

	sqlDB, err := db.DB()
	require.NoError(t, err)
	sqlDB.SetMaxOpenConns(1)
	sqlDB.SetMaxIdleConns(1)

	err = db.AutoMigrate(
		&models.User{},
		&models.UserWallet{},
		&models.MemberLevel{},
		&models.Merchant{},
		&models.Venue{},
		&models.Device{},
		&models.RentalPricing{},
		&models.Order{},
		&models.Rental{},
		&models.WalletTransaction{},
		&models.Payment{},
		&models.Refund{},
	)
	require.NoError(t, err)

	// 默认会员等级
	db.Create(&models.MemberLevel{
		ID:        1,
		Name:      "普通会员",
		Level:     1,
		MinPoints: 0,
		Discount:  1.0,
	})

	jwtManager := jwt.NewManager(&jwt.Config{
		Secret:            "test-secret-key-us1-api",
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

	v1 := r.Group("/api/v1")
	{
		public := v1.Group("")
		deviceH.RegisterRoutes(public)

		user := v1.Group("")
		user.Use(userMiddleware.UserAuth(jwtManager))
		rentalH.RegisterRoutes(user)
		paymentH.RegisterRoutes(user)
	}

	return r, db, jwtManager
}

func seedUS1DeviceAndUser(t *testing.T, db *gorm.DB) (*models.User, *models.Device, *models.RentalPricing) {
	t.Helper()

	phone := "13800138000"
	user := &models.User{
		Phone:         &phone,
		Nickname:      "测试用户",
		MemberLevelID: 1,
		Status:        models.UserStatusActive,
	}
	require.NoError(t, db.Create(user).Error)
	require.NoError(t, db.Create(&models.UserWallet{UserID: user.ID, Balance: 200.0}).Error)

	merchant := &models.Merchant{
		Name:           "测试商户",
		ContactName:    "联系人",
		ContactPhone:   "13900139000",
		CommissionRate: 0.2,
		SettlementType: "monthly",
		Status:         models.MerchantStatusActive,
	}
	require.NoError(t, db.Create(merchant).Error)

	venue := &models.Venue{
		MerchantID: merchant.ID,
		Name:       "测试场地",
		Type:       "mall",
		Province:   "广东省",
		City:       "深圳市",
		District:   "南山区",
		Address:    "科技园路1号",
		Status:     models.VenueStatusActive,
	}
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

	pricing := &models.RentalPricing{
		VenueID:       &venue.ID,
		DurationHours: 1,
		Price:         10.0,
		Deposit:       50.0,
		OvertimeRate:  1.5,
		IsActive:      true,
	}
	require.NoError(t, db.Create(pricing).Error)

	return user, device, pricing
}

func TestUS1API_DeviceScan_Success(t *testing.T) {
	router, db, _ := setupUS1APIRouter(t)
	_, device, _ := seedUS1DeviceAndUser(t, db)

	req, _ := http.NewRequest("GET", "/api/v1/device/scan?qr_code="+device.QRCode, nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var resp map[string]interface{}
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	assert.Equal(t, float64(0), resp["code"])

	data := resp["data"].(map[string]interface{})
	assert.Equal(t, device.DeviceNo, data["device_no"])
	pricings := data["pricings"].([]interface{})
	assert.Len(t, pricings, 1)
}

func TestUS1API_RentalLifecycle_ScanToReturn(t *testing.T) {
	router, db, jwtManager := setupUS1APIRouter(t)
	user, device, pricing := seedUS1DeviceAndUser(t, db)

	tokenPair, err := jwtManager.GenerateTokenPair(user.ID, jwt.UserTypeUser, "")
	require.NoError(t, err)
	authz := "Bearer " + tokenPair.AccessToken

	// 1) 扫码
	scanReq, _ := http.NewRequest("GET", "/api/v1/device/scan?qr_code="+device.QRCode, nil)
	scanW := httptest.NewRecorder()
	router.ServeHTTP(scanW, scanReq)
	assert.Equal(t, http.StatusOK, scanW.Code)

	// 2) 创建租借
	createBody, _ := json.Marshal(map[string]interface{}{
		"device_id":  device.ID,
		"pricing_id": pricing.ID,
	})
	createReq, _ := http.NewRequest("POST", "/api/v1/rental", bytes.NewBuffer(createBody))
	createReq.Header.Set("Content-Type", "application/json")
	createReq.Header.Set("Authorization", authz)
	createW := httptest.NewRecorder()
	router.ServeHTTP(createW, createReq)
	assert.Equal(t, http.StatusOK, createW.Code)

	var createResp map[string]interface{}
	require.NoError(t, json.Unmarshal(createW.Body.Bytes(), &createResp))
	assert.Equal(t, float64(0), createResp["code"])
	rentalID := int64(createResp["data"].(map[string]interface{})["id"].(float64))

	// 3) 支付
	idStr := strconv.FormatInt(rentalID, 10)
	payReq, _ := http.NewRequest("POST", "/api/v1/rental/"+idStr+"/pay", nil)
	payReq.Header.Set("Authorization", authz)
	payW := httptest.NewRecorder()
	router.ServeHTTP(payW, payReq)
	assert.Equal(t, http.StatusOK, payW.Code)

	// 4) 开始租借
	startReq, _ := http.NewRequest("POST", "/api/v1/rental/"+idStr+"/start", nil)
	startReq.Header.Set("Authorization", authz)
	startW := httptest.NewRecorder()
	router.ServeHTTP(startW, startReq)
	assert.Equal(t, http.StatusOK, startW.Code)

	// 5) 归还
	retReq, _ := http.NewRequest("POST", "/api/v1/rental/"+idStr+"/return", nil)
	retReq.Header.Set("Authorization", authz)
	retW := httptest.NewRecorder()
	router.ServeHTTP(retW, retReq)
	assert.Equal(t, http.StatusOK, retW.Code)

	// 6) 查询详情
	getReq, _ := http.NewRequest("GET", "/api/v1/rental/"+idStr, nil)
	getReq.Header.Set("Authorization", authz)
	getW := httptest.NewRecorder()
	router.ServeHTTP(getW, getReq)
	assert.Equal(t, http.StatusOK, getW.Code)

	var getResp map[string]interface{}
	require.NoError(t, json.Unmarshal(getW.Body.Bytes(), &getResp))
	assert.Equal(t, float64(0), getResp["code"])
	assert.Equal(t, models.RentalStatusReturned, getResp["data"].(map[string]interface{})["status"])
}
