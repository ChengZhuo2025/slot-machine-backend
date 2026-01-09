//go:build api
// +build api

// Package api 会员体系（用户端）API 测试
package api

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
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
	userHandler "github.com/dumeirei/smart-locker-backend/internal/handler/user"
	"github.com/dumeirei/smart-locker-backend/internal/middleware"
	"github.com/dumeirei/smart-locker-backend/internal/models"
	"github.com/dumeirei/smart-locker-backend/internal/repository"
	userService "github.com/dumeirei/smart-locker-backend/internal/service/user"
)

func setupMemberAPITestDB(t *testing.T) *gorm.DB {
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
		&models.MemberPackage{},
		&models.Order{},
		&models.OrderItem{},
		&models.WalletTransaction{},
	))

	db.Create(&models.MemberLevel{
		ID:        1,
		Name:      "普通会员",
		Level:     1,
		MinPoints: 0,
		Discount:  1.0,
		Benefits:  models.JSON{"free_shipping": false},
	})
	db.Create(&models.MemberLevel{
		ID:        2,
		Name:      "黄金会员",
		Level:     2,
		MinPoints: 100,
		Discount:  0.9,
		Benefits:  models.JSON{"free_shipping": true},
	})

	return db
}

func setupMemberAPITestRouter(db *gorm.DB, jwtManager *jwt.Manager) *gin.Engine {
	gin.SetMode(gin.TestMode)
	r := gin.New()

	userRepo := repository.NewUserRepository(db)
	levelRepo := repository.NewMemberLevelRepository(db)
	packageRepo := repository.NewMemberPackageRepository(db)
	orderRepo := repository.NewOrderRepository(db)

	pointsSvc := userService.NewPointsService(db, userRepo, levelRepo)
	memberLevelSvc := userService.NewMemberLevelService(db, userRepo, levelRepo)
	memberPackageSvc := userService.NewMemberPackageService(db, userRepo, packageRepo, levelRepo, orderRepo, pointsSvc)

	memberH := userHandler.NewMemberHandler(memberLevelSvc, memberPackageSvc, pointsSvc)

	api := r.Group("/api/v1")
	api.Use(middleware.UserAuth(jwtManager))
	memberH.RegisterRoutes(api)

	return r
}

func createMemberAPITestJWTManager() *jwt.Manager {
	return jwt.NewManager(&jwt.Config{
		Secret:            "test-secret-key-for-member-api",
		AccessExpireTime:  24 * time.Hour,
		RefreshExpireTime: 7 * 24 * time.Hour,
		Issuer:            "test",
	})
}

func createMemberAPITestUser(db *gorm.DB) *models.User {
	phone := fmt.Sprintf("138%08d", time.Now().UnixNano()%100000000)
	user := &models.User{
		Phone:         &phone,
		Nickname:      "会员API用户",
		MemberLevelID: 1,
		Status:        models.UserStatusActive,
	}
	db.Create(user)
	db.Create(&models.UserWallet{UserID: user.ID, Balance: 1000})
	return user
}

func createMemberAPITestPackage(db *gorm.DB, name string, levelID int64, status int8, giftPoints int) *models.MemberPackage {
	pkg := &models.MemberPackage{
		Name:          name,
		MemberLevelID: levelID,
		Duration:      1,
		DurationUnit:  models.PackageDurationUnitMonth,
		Price:         30,
		GiftPoints:    giftPoints,
		Sort:          1,
		IsRecommend:   true,
		Status:        status,
	}
	db.Create(pkg)
	return pkg
}

func generateMemberTestToken(jwtManager *jwt.Manager, userID int64) string {
	token, _, _ := jwtManager.GenerateAccessToken(userID, jwt.UserTypeUser, "")
	return token
}

func TestMemberAPI_Unauthorized(t *testing.T) {
	db := setupMemberAPITestDB(t)
	jwtManager := createMemberAPITestJWTManager()
	router := setupMemberAPITestRouter(db, jwtManager)

	req, _ := http.NewRequest("GET", "/api/v1/member/info", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusUnauthorized, w.Code)
}

func TestMemberAPI_GetLevelsAndInfo(t *testing.T) {
	db := setupMemberAPITestDB(t)
	jwtManager := createMemberAPITestJWTManager()
	router := setupMemberAPITestRouter(db, jwtManager)

	user := createMemberAPITestUser(db)
	token := generateMemberTestToken(jwtManager, user.ID)

	req, _ := http.NewRequest("GET", "/api/v1/member/levels", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)
	require.Equal(t, http.StatusOK, w.Code)

	var resp map[string]interface{}
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	assert.Equal(t, float64(0), resp["code"])
	levels := resp["data"].([]interface{})
	assert.GreaterOrEqual(t, len(levels), 2)

	req2, _ := http.NewRequest("GET", "/api/v1/member/info", nil)
	req2.Header.Set("Authorization", "Bearer "+token)
	w2 := httptest.NewRecorder()
	router.ServeHTTP(w2, req2)
	require.Equal(t, http.StatusOK, w2.Code)

	var resp2 map[string]interface{}
	require.NoError(t, json.Unmarshal(w2.Body.Bytes(), &resp2))
	assert.Equal(t, float64(0), resp2["code"])
	data := resp2["data"].(map[string]interface{})
	assert.Equal(t, float64(0), data["points"])
	current := data["current_level"].(map[string]interface{})
	assert.Equal(t, "普通会员", current["name"])
}

func TestMemberAPI_PurchasePackage_UpdatesMemberAndPointsHistory(t *testing.T) {
	db := setupMemberAPITestDB(t)
	jwtManager := createMemberAPITestJWTManager()
	router := setupMemberAPITestRouter(db, jwtManager)

	user := createMemberAPITestUser(db)
	token := generateMemberTestToken(jwtManager, user.ID)

	pkg := createMemberAPITestPackage(db, "黄金会员月卡", 2, models.MemberPackageStatusActive, 100)

	// 列表仅返回启用套餐
	reqList, _ := http.NewRequest("GET", "/api/v1/member/packages", nil)
	reqList.Header.Set("Authorization", "Bearer "+token)
	wList := httptest.NewRecorder()
	router.ServeHTTP(wList, reqList)
	require.Equal(t, http.StatusOK, wList.Code)

	// 购买套餐
	body, _ := json.Marshal(map[string]interface{}{"package_id": pkg.ID})
	req, _ := http.NewRequest("POST", "/api/v1/member/packages/purchase", bytes.NewBuffer(body))
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)
	require.Equal(t, http.StatusOK, w.Code)

	var resp map[string]interface{}
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	assert.Equal(t, float64(0), resp["code"])
	purchase := resp["data"].(map[string]interface{})
	assert.NotNil(t, purchase["order_id"])

	// 会员信息更新（等级/折扣/权益）
	reqInfo, _ := http.NewRequest("GET", "/api/v1/member/benefits", nil)
	reqInfo.Header.Set("Authorization", "Bearer "+token)
	wInfo := httptest.NewRecorder()
	router.ServeHTTP(wInfo, reqInfo)
	require.Equal(t, http.StatusOK, wInfo.Code)

	var infoResp map[string]interface{}
	require.NoError(t, json.Unmarshal(wInfo.Body.Bytes(), &infoResp))
	benefits := infoResp["data"].(map[string]interface{})
	assert.Equal(t, "黄金会员", benefits["level_name"])
	assert.Equal(t, true, benefits["benefits"].(map[string]interface{})["free_shipping"])

	// 积分历史包含套餐赠送积分记录
	reqHis, _ := http.NewRequest("GET", "/api/v1/member/points/history?type=package_purchase", nil)
	reqHis.Header.Set("Authorization", "Bearer "+token)
	wHis := httptest.NewRecorder()
	router.ServeHTTP(wHis, reqHis)
	require.Equal(t, http.StatusOK, wHis.Code)

	var hisResp map[string]interface{}
	require.NoError(t, json.Unmarshal(wHis.Body.Bytes(), &hisResp))
	assert.Equal(t, float64(0), hisResp["code"])
	page := hisResp["data"].(map[string]interface{})
	list := page["list"].([]interface{})
	assert.Len(t, list, 1)
}
