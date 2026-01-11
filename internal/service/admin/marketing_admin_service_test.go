package admin

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

func setupMarketingAdminTestDB(t *testing.T) *gorm.DB {
	t.Helper()

	dsn := fmt.Sprintf("file:%s?mode=memory&cache=shared", strings.ReplaceAll(t.Name(), "/", "_"))
	db, err := gorm.Open(sqlite.Open(dsn), &gorm.Config{Logger: logger.Default.LogMode(logger.Silent)})
	require.NoError(t, err)

	sqlDB, err := db.DB()
	require.NoError(t, err)
	sqlDB.SetMaxOpenConns(1)
	sqlDB.SetMaxIdleConns(1)

	require.NoError(t, db.AutoMigrate(&models.Coupon{}, &models.Campaign{}))
	return db
}

func TestMarketingAdminService_CreateAndListCoupon(t *testing.T) {
	db := setupMarketingAdminTestDB(t)
	svc := NewMarketingAdminService(db, repository.NewCouponRepository(db), repository.NewCampaignRepository(db))
	ctx := context.Background()

	now := time.Now()
	start := now.Add(-time.Hour).Format("2006-01-02 15:04:05")
	end := now.Add(time.Hour).Format("2006-01-02 15:04:05")

	coupon, err := svc.CreateCoupon(ctx, &CreateCouponRequest{
		Name:            "满10减5",
		Type:            models.CouponTypeFixed,
		Value:           5,
		MinAmount:       10,
		TotalCount:      100,
		PerUserLimit:    1,
		ApplicableScope: models.CouponScopeProduct,
		ApplicableIDs:   []int64{1, 2},
		StartTime:       start,
		EndTime:         end,
	})
	require.NoError(t, err)
	require.NotNil(t, coupon)

	resp, err := svc.GetCouponList(ctx, &AdminCouponListRequest{Page: 1, PageSize: 10})
	require.NoError(t, err)
	require.NotNil(t, resp)
	require.Equal(t, int64(1), resp.Total)
	require.Len(t, resp.List, 1)
	assert.Equal(t, coupon.ID, resp.List[0].ID)
	assert.Equal(t, "满10减5", resp.List[0].Name)
	assert.Equal(t, "固定金额", resp.List[0].TypeText)
	assert.ElementsMatch(t, []int64{1, 2}, resp.List[0].ApplicableIDs)
}

