// Package tests 提供测试框架配置
package tests

import (
	"context"
	"os"
	"testing"

	"github.com/stretchr/testify/require"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"

	"github.com/dumeirei/smart-locker-backend/internal/models"
)

// SetupTestDB 返回一个用于测试的 SQLite 内存数据库
func SetupTestDB(t *testing.T) *gorm.DB {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Silent),
	})
	require.NoError(t, err, "failed to connect to test database")

	// 自动迁移所有模型
	err = db.AutoMigrate(
		&models.User{},
		&models.UserWallet{},
		&models.MemberLevel{},
		&models.WalletTransaction{},
		&models.Address{},
		&models.UserFeedback{},
		&models.Admin{},
		&models.Role{},
		&models.Permission{},
		&models.RolePermission{},
		&models.Merchant{},
		&models.Venue{},
		&models.Device{},
		&models.DeviceLog{},
		&models.DeviceMaintenance{},
		&models.RentalPricing{},
		&models.Rental{},
		&models.Order{},
		&models.OrderItem{},
		&models.Payment{},
		&models.Refund{},
	)
	require.NoError(t, err, "failed to migrate test database")

	return db
}

// NewTestContext 返回一个用于测试的 Context
func NewTestContext() context.Context {
	return context.Background()
}

// TestMain 设置测试环境
func TestMain(m *testing.M) {
	// 设置测试环境变量
	os.Setenv("GO_ENV", "test")

	// 运行测试
	code := m.Run()

	// 退出
	os.Exit(code)
}

// CleanupDB 清理数据库中的所有数据
func CleanupDB(t *testing.T, db *gorm.DB) {
	tables := []string{
		"wallet_transactions",
		"refunds",
		"payments",
		"order_items",
		"orders",
		"rentals",
		"rental_pricings",
		"device_logs",
		"device_maintenances",
		"devices",
		"venues",
		"merchants",
		"role_permissions",
		"permissions",
		"admins",
		"roles",
		"user_feedbacks",
		"addresses",
		"user_wallets",
		"users",
		"member_levels",
	}

	for _, table := range tables {
		db.Exec("DELETE FROM " + table)
	}
}
