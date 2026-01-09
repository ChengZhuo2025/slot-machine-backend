//go:build api
// +build api

// Package api 会员体系（管理端）API 测试
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
	adminHandler "github.com/dumeirei/smart-locker-backend/internal/handler/admin"
	"github.com/dumeirei/smart-locker-backend/internal/middleware"
	"github.com/dumeirei/smart-locker-backend/internal/models"
	"github.com/dumeirei/smart-locker-backend/internal/repository"
	adminService "github.com/dumeirei/smart-locker-backend/internal/service/admin"
)

func setupMemberAdminAPITestDB(t *testing.T) *gorm.DB {
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
		&models.Admin{},
		&models.User{},
		&models.MemberLevel{},
		&models.MemberPackage{},
		&models.Order{},
	))

	db.Create(&models.MemberLevel{ID: 1, Name: "普通会员", Level: 1, MinPoints: 0, Discount: 1.0})
	db.Create(&models.MemberLevel{ID: 2, Name: "黄金会员", Level: 2, MinPoints: 100, Discount: 0.9})

	return db
}

func setupMemberAdminAPITestRouter(db *gorm.DB, jwtManager *jwt.Manager) *gin.Engine {
	gin.SetMode(gin.TestMode)
	r := gin.New()

	userRepo := repository.NewUserRepository(db)
	levelRepo := repository.NewMemberLevelRepository(db)
	packageRepo := repository.NewMemberPackageRepository(db)

	memberAdminSvc := adminService.NewMemberAdminService(db, levelRepo, packageRepo, userRepo)
	memberAdminH := adminHandler.NewMemberHandler(memberAdminSvc)

	api := r.Group("/api/v1/admin")
	api.Use(middleware.AdminAuth(jwtManager))
	memberAdminH.RegisterRoutes(api)

	return r
}

func createMemberAdminAPITestJWTManager() *jwt.Manager {
	return jwt.NewManager(&jwt.Config{
		Secret:            "test-secret-key-for-member-admin-api",
		AccessExpireTime:  24 * time.Hour,
		RefreshExpireTime: 7 * 24 * time.Hour,
		Issuer:            "test",
	})
}

func generateMemberAdminTestToken(jwtManager *jwt.Manager, adminID int64) string {
	token, _, _ := jwtManager.GenerateAccessToken(adminID, jwt.UserTypeAdmin, "test_role")
	return token
}

func TestMemberAdminAPI_LevelCRUDAndStats(t *testing.T) {
	db := setupMemberAdminAPITestDB(t)
	jwtManager := createMemberAdminAPITestJWTManager()
	router := setupMemberAdminAPITestRouter(db, jwtManager)
	token := generateMemberAdminTestToken(jwtManager, 1)

	// 创建等级
	body, _ := json.Marshal(map[string]interface{}{
		"name":       "钻石会员",
		"level":      3,
		"min_points": 200,
		"discount":   0.8,
		"benefits":   map[string]interface{}{"vip_support": true},
	})
	req, _ := http.NewRequest("POST", "/api/v1/admin/member/levels", bytes.NewBuffer(body))
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)
	require.Equal(t, http.StatusOK, w.Code)

	// 列表
	reqList, _ := http.NewRequest("GET", "/api/v1/admin/member/levels", nil)
	reqList.Header.Set("Authorization", "Bearer "+token)
	wList := httptest.NewRecorder()
	router.ServeHTTP(wList, reqList)
	require.Equal(t, http.StatusOK, wList.Code)
	var listResp map[string]interface{}
	require.NoError(t, json.Unmarshal(wList.Body.Bytes(), &listResp))
	assert.Equal(t, float64(0), listResp["code"])
	assert.GreaterOrEqual(t, len(listResp["data"].([]interface{})), 3)

	// 统计（包含等级分布）
	db.Create(&models.User{Nickname: "U1", MemberLevelID: 1, Status: models.UserStatusActive})
	db.Create(&models.User{Nickname: "U2", MemberLevelID: 2, Status: models.UserStatusActive})
	db.Create(&models.Order{OrderNo: "MP-1", UserID: 1, Type: "member_package", OriginalAmount: 30, DiscountAmount: 0, ActualAmount: 30, Status: models.OrderStatusCompleted})

	reqStats, _ := http.NewRequest("GET", "/api/v1/admin/member/stats", nil)
	reqStats.Header.Set("Authorization", "Bearer "+token)
	wStats := httptest.NewRecorder()
	router.ServeHTTP(wStats, reqStats)
	require.Equal(t, http.StatusOK, wStats.Code)
	var statsResp map[string]interface{}
	require.NoError(t, json.Unmarshal(wStats.Body.Bytes(), &statsResp))
	assert.Equal(t, float64(0), statsResp["code"])
	stats := statsResp["data"].(map[string]interface{})
	assert.Equal(t, float64(2), stats["total_users"])
}
