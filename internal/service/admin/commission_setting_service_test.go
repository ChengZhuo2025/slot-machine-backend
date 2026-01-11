package admin

import (
	"context"
	"fmt"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"

	"github.com/dumeirei/smart-locker-backend/internal/models"
)

func setupCommissionSettingTestDB(t *testing.T) *gorm.DB {
	t.Helper()

	dsn := fmt.Sprintf("file:%s?mode=memory&cache=shared", strings.ReplaceAll(t.Name(), "/", "_"))
	db, err := gorm.Open(sqlite.Open(dsn), &gorm.Config{Logger: logger.Default.LogMode(logger.Silent)})
	require.NoError(t, err)

	sqlDB, err := db.DB()
	require.NoError(t, err)
	sqlDB.SetMaxOpenConns(1)
	sqlDB.SetMaxIdleConns(1)

	require.NoError(t, db.AutoMigrate(&models.CommissionSetting{}))
	return db
}

func TestCommissionSettingService_GetConfig_Default(t *testing.T) {
	db := setupCommissionSettingTestDB(t)
	svc := NewCommissionSettingService(db)
	ctx := context.Background()

	cfg, err := svc.GetConfig(ctx)
	require.NoError(t, err)
	require.NotNil(t, cfg)
	assert.Equal(t, DefaultCommissionConfig.DirectRate, cfg.DirectRate)
	assert.Equal(t, DefaultCommissionConfig.IndirectRate, cfg.IndirectRate)
}

func TestCommissionSettingService_UpdateConfig_ValidationAndHistory(t *testing.T) {
	db := setupCommissionSettingTestDB(t)
	svc := NewCommissionSettingService(db)
	ctx := context.Background()

	err := svc.UpdateConfig(ctx, &CommissionConfig{DirectRate: 0.6, IndirectRate: 0, MinWithdraw: 0, WithdrawFee: 0, SettleDelay: 0})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "不能超过50%")

	cfg := &CommissionConfig{
		DirectRate:   0.1,
		IndirectRate: 0.05,
		MinWithdraw:  10,
		WithdrawFee:  0.006,
		SettleDelay:  7,
	}
	require.NoError(t, svc.UpdateConfig(ctx, cfg))

	got, err := svc.GetConfig(ctx)
	require.NoError(t, err)
	assert.Equal(t, cfg.DirectRate, got.DirectRate)

	history, total, err := svc.GetConfigHistory(ctx, 0, 10)
	require.NoError(t, err)
	assert.Equal(t, int64(1), total)
	require.Len(t, history, 1)
}

func TestCommissionSettingService_InitDefaultConfig(t *testing.T) {
	db := setupCommissionSettingTestDB(t)
	svc := NewCommissionSettingService(db)
	ctx := context.Background()

	t.Run("初始化默认配置成功", func(t *testing.T) {
		err := svc.InitDefaultConfig(ctx)
		require.NoError(t, err)

		var count int64
		db.Model(&models.CommissionSetting{}).Count(&count)
		assert.Equal(t, int64(1), count)
	})

	t.Run("已有配置时不重复初始化", func(t *testing.T) {
		err := svc.InitDefaultConfig(ctx)
		require.NoError(t, err)

		var count int64
		db.Model(&models.CommissionSetting{}).Count(&count)
		assert.Equal(t, int64(1), count)
	})
}

