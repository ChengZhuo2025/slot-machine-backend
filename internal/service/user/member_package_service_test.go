// Package user 会员套餐服务单元测试
package user

import (
	"context"
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"

	"github.com/dumeirei/smart-locker-backend/internal/models"
	"github.com/dumeirei/smart-locker-backend/internal/repository"
)

func setupMemberPackageServiceTestDB(t *testing.T) *gorm.DB {
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
		&models.MemberLevel{},
		&models.MemberPackage{},
		&models.Order{},
		&models.OrderItem{},
		&models.WalletTransaction{},
	))

	db.Create(&models.MemberLevel{ID: 1, Name: "普通会员", Level: 1, MinPoints: 0, Discount: 1.0})
	db.Create(&models.MemberLevel{ID: 2, Name: "黄金会员", Level: 2, MinPoints: 100, Discount: 0.9})

	return db
}

func newMemberPackageServiceForTest(db *gorm.DB) (*MemberPackageService, *PointsService) {
	userRepo := repository.NewUserRepository(db)
	levelRepo := repository.NewMemberLevelRepository(db)
	packageRepo := repository.NewMemberPackageRepository(db)
	orderRepo := repository.NewOrderRepository(db)
	pointsSvc := NewPointsService(db, userRepo, levelRepo)
	return NewMemberPackageService(db, userRepo, packageRepo, levelRepo, orderRepo, pointsSvc), pointsSvc
}

func createTestUserForPackage(db *gorm.DB) *models.User {
	phone := fmt.Sprintf("137%08d", time.Now().UnixNano()%100000000)
	user := &models.User{
		Phone:         &phone,
		Nickname:      "套餐用户",
		MemberLevelID: 1,
		Status:        models.UserStatusActive,
	}
	db.Create(user)
	return user
}

func createTestMemberPackage(db *gorm.DB, opts ...func(*models.MemberPackage)) *models.MemberPackage {
	pkg := &models.MemberPackage{
		Name:          "黄金会员月卡",
		MemberLevelID: 2,
		Duration:      1,
		DurationUnit:  models.PackageDurationUnitMonth,
		Price:         30.0,
		GiftPoints:    20,
		Sort:          10,
		IsRecommend:   true,
		Status:        models.MemberPackageStatusActive,
	}
	for _, opt := range opts {
		opt(pkg)
	}

	originalStatus := pkg.Status
	db.Create(pkg)
	if originalStatus == models.MemberPackageStatusDisabled {
		db.Model(pkg).Update("status", originalStatus)
	}

	return pkg
}

func TestMemberPackageService_GetActiveAndRecommendedPackages(t *testing.T) {
	db := setupMemberPackageServiceTestDB(t)
	svc, _ := newMemberPackageServiceForTest(db)
	ctx := context.Background()

	createTestMemberPackage(db, func(p *models.MemberPackage) { p.Name = "启用推荐" })
	createTestMemberPackage(db, func(p *models.MemberPackage) { p.Name = "启用不推荐"; p.IsRecommend = false })
	createTestMemberPackage(db, func(p *models.MemberPackage) { p.Name = "禁用"; p.Status = models.MemberPackageStatusDisabled })

	active, err := svc.GetActivePackages(ctx)
	require.NoError(t, err)
	assert.Len(t, active, 2)

	recommended, err := svc.GetRecommendedPackages(ctx)
	require.NoError(t, err)
	assert.Len(t, recommended, 1)
	assert.Equal(t, "启用推荐", recommended[0].Name)
}

func TestMemberPackageService_GetPackageByID_NotFound(t *testing.T) {
	db := setupMemberPackageServiceTestDB(t)
	svc, _ := newMemberPackageServiceForTest(db)
	ctx := context.Background()

	_, err := svc.GetPackageByID(ctx, 999999)
	require.Error(t, err)
}

func TestMemberPackageService_PurchasePackage_Success(t *testing.T) {
	db := setupMemberPackageServiceTestDB(t)
	svc, _ := newMemberPackageServiceForTest(db)
	ctx := context.Background()

	user := createTestUserForPackage(db)
	pkg := createTestMemberPackage(db, func(p *models.MemberPackage) { p.GiftPoints = 100 })

	result, err := svc.PurchasePackage(ctx, user.ID, pkg.ID)
	require.NoError(t, err)
	require.NotNil(t, result)
	assert.NotZero(t, result.OrderID)
	assert.NotEmpty(t, result.OrderNo)
	assert.Equal(t, 30.0, result.Amount)
	assert.Equal(t, 100, result.GiftPoints)
	assert.Equal(t, int64(2), result.NewLevelID)
	assert.Equal(t, "黄金会员", result.NewLevelName)

	var order models.Order
	require.NoError(t, db.First(&order, result.OrderID).Error)
	assert.Equal(t, "member_package", order.Type)
	assert.Equal(t, models.OrderStatusPaid, order.Status)
	assert.Equal(t, 30.0, order.ActualAmount)

	var itemCount int64
	require.NoError(t, db.Model(&models.OrderItem{}).Where("order_id = ?", result.OrderID).Count(&itemCount).Error)
	assert.Equal(t, int64(1), itemCount)

	var refreshed models.User
	require.NoError(t, db.First(&refreshed, user.ID).Error)
	assert.Equal(t, int64(2), refreshed.MemberLevelID)
	assert.Equal(t, 100, refreshed.Points)

	var txCount int64
	require.NoError(t, db.Model(&models.WalletTransaction{}).Where("user_id = ? AND type = ?", user.ID, "points_package_purchase").Count(&txCount).Error)
	assert.Equal(t, int64(1), txCount)
}

func TestMemberPackageService_PurchasePackage_Disabled(t *testing.T) {
	db := setupMemberPackageServiceTestDB(t)
	svc, _ := newMemberPackageServiceForTest(db)
	ctx := context.Background()

	user := createTestUserForPackage(db)
	pkg := createTestMemberPackage(db, func(p *models.MemberPackage) { p.Status = models.MemberPackageStatusDisabled })

	_, err := svc.PurchasePackage(ctx, user.ID, pkg.ID)
	require.Error(t, err)
}
