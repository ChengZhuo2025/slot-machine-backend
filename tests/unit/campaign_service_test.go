//go:build unit
// +build unit

// Package unit 活动服务单元测试
package unit

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
	"github.com/dumeirei/smart-locker-backend/internal/service/marketing"
)

// setupCampaignServiceTestDB 创建测试数据库
func setupCampaignServiceTestDB(t *testing.T) *gorm.DB {
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
		&models.Campaign{},
	)
	require.NoError(t, err)

	return db
}

// createCampaignTestService 创建测试服务
func createCampaignTestService(db *gorm.DB) *marketing.CampaignService {
	campaignRepo := repository.NewCampaignRepository(db)
	return marketing.NewCampaignService(campaignRepo)
}

// createTestCampaign 创建测试活动
func createTestCampaign(db *gorm.DB, opts ...func(*models.Campaign)) *models.Campaign {
	campaign := &models.Campaign{
		Name:      "测试活动",
		Type:      models.CampaignTypeDiscount,
		StartTime: time.Now().Add(-time.Hour),
		EndTime:   time.Now().Add(24 * time.Hour),
		Status:    models.CampaignStatusActive,
	}

	for _, opt := range opts {
		opt(campaign)
	}

	// 保存原始状态值（GORM 会跳过零值）
	originalStatus := campaign.Status

	db.Create(campaign)

	// 如果状态是禁用(0)，需要显式更新，因为 GORM 会使用数据库默认值
	if originalStatus == models.CampaignStatusDisabled {
		db.Model(campaign).Update("status", originalStatus)
	}

	return campaign
}

// TestCampaignService_GetCampaignList 测试获取活动列表
func TestCampaignService_GetCampaignList(t *testing.T) {
	t.Run("正常获取活动列表", func(t *testing.T) {
		db := setupCampaignServiceTestDB(t)
		svc := createCampaignTestService(db)

		// 创建多个活动
		createTestCampaign(db, func(c *models.Campaign) { c.Name = "活动1" })
		createTestCampaign(db, func(c *models.Campaign) { c.Name = "活动2" })
		createTestCampaign(db, func(c *models.Campaign) { c.Name = "活动3" })

		req := &marketing.CampaignListRequest{
			Page:     1,
			PageSize: 10,
		}

		result, err := svc.GetCampaignList(context.Background(), req)
		require.NoError(t, err)
		assert.Equal(t, int64(3), result.Total)
		assert.Len(t, result.List, 3)
	})

	t.Run("分页获取活动列表", func(t *testing.T) {
		db := setupCampaignServiceTestDB(t)
		svc := createCampaignTestService(db)

		// 创建5个活动
		for i := 0; i < 5; i++ {
			createTestCampaign(db, func(c *models.Campaign) {
				c.Name = fmt.Sprintf("活动%d", i+1)
			})
		}

		// 获取第一页（每页2个）
		req := &marketing.CampaignListRequest{
			Page:     1,
			PageSize: 2,
		}

		result, err := svc.GetCampaignList(context.Background(), req)
		require.NoError(t, err)
		assert.Equal(t, int64(5), result.Total)
		assert.Len(t, result.List, 2)

		// 获取第二页
		req.Page = 2
		result, err = svc.GetCampaignList(context.Background(), req)
		require.NoError(t, err)
		assert.Len(t, result.List, 2)
	})

	t.Run("只返回有效活动", func(t *testing.T) {
		db := setupCampaignServiceTestDB(t)
		svc := createCampaignTestService(db)

		// 创建有效活动
		createTestCampaign(db, func(c *models.Campaign) { c.Name = "有效活动" })

		// 创建未开始活动
		createTestCampaign(db, func(c *models.Campaign) {
			c.Name = "未开始"
			c.StartTime = time.Now().Add(24 * time.Hour)
			c.EndTime = time.Now().Add(48 * time.Hour)
		})

		// 创建已结束活动
		createTestCampaign(db, func(c *models.Campaign) {
			c.Name = "已结束"
			c.StartTime = time.Now().Add(-48 * time.Hour)
			c.EndTime = time.Now().Add(-24 * time.Hour)
		})

		// 创建禁用活动
		createTestCampaign(db, func(c *models.Campaign) {
			c.Name = "禁用"
			c.Status = models.CampaignStatusDisabled
		})

		req := &marketing.CampaignListRequest{
			Page:     1,
			PageSize: 10,
		}

		result, err := svc.GetCampaignList(context.Background(), req)
		require.NoError(t, err)
		assert.Equal(t, int64(1), result.Total)
		assert.Equal(t, "有效活动", result.List[0].Name)
	})
}

// TestCampaignService_GetCampaignDetail 测试获取活动详情
func TestCampaignService_GetCampaignDetail(t *testing.T) {
	t.Run("正常获取活动详情", func(t *testing.T) {
		db := setupCampaignServiceTestDB(t)
		svc := createCampaignTestService(db)

		description := "这是活动描述"
		campaign := createTestCampaign(db, func(c *models.Campaign) {
			c.Name = "测试活动"
			c.Description = &description
		})

		detail, err := svc.GetCampaignDetail(context.Background(), campaign.ID)
		require.NoError(t, err)
		assert.Equal(t, campaign.ID, detail.ID)
		assert.Equal(t, campaign.Name, detail.Name)
		assert.Equal(t, &description, detail.Description)
	})

	t.Run("获取不存在的活动", func(t *testing.T) {
		db := setupCampaignServiceTestDB(t)
		svc := createCampaignTestService(db)

		_, err := svc.GetCampaignDetail(context.Background(), 99999)
		assert.Error(t, err)
	})

	t.Run("活动状态文本正确", func(t *testing.T) {
		db := setupCampaignServiceTestDB(t)
		svc := createCampaignTestService(db)

		// 有效活动
		campaign1 := createTestCampaign(db, func(c *models.Campaign) { c.Name = "进行中" })
		detail1, _ := svc.GetCampaignDetail(context.Background(), campaign1.ID)
		assert.Equal(t, "进行中", detail1.StatusText)
		assert.True(t, detail1.IsActive)

		// 未开始活动
		campaign2 := createTestCampaign(db, func(c *models.Campaign) {
			c.Name = "未开始"
			c.StartTime = time.Now().Add(24 * time.Hour)
			c.EndTime = time.Now().Add(48 * time.Hour)
		})
		detail2, _ := svc.GetCampaignDetail(context.Background(), campaign2.ID)
		assert.Equal(t, "未开始", detail2.StatusText)
		assert.False(t, detail2.IsActive)

		// 已结束活动
		campaign3 := createTestCampaign(db, func(c *models.Campaign) {
			c.Name = "已结束"
			c.StartTime = time.Now().Add(-48 * time.Hour)
			c.EndTime = time.Now().Add(-24 * time.Hour)
		})
		detail3, _ := svc.GetCampaignDetail(context.Background(), campaign3.ID)
		assert.Equal(t, "已结束", detail3.StatusText)
		assert.False(t, detail3.IsActive)

		// 禁用活动
		campaign4 := createTestCampaign(db, func(c *models.Campaign) {
			c.Name = "禁用"
			c.Status = models.CampaignStatusDisabled
		})
		detail4, _ := svc.GetCampaignDetail(context.Background(), campaign4.ID)
		assert.Equal(t, "已禁用", detail4.StatusText)
		assert.False(t, detail4.IsActive)
	})

	t.Run("活动类型文本正确", func(t *testing.T) {
		db := setupCampaignServiceTestDB(t)
		svc := createCampaignTestService(db)

		// 满减
		campaign1 := createTestCampaign(db, func(c *models.Campaign) {
			c.Type = models.CampaignTypeDiscount
		})
		detail1, _ := svc.GetCampaignDetail(context.Background(), campaign1.ID)
		assert.Equal(t, "满减", detail1.TypeText)

		// 满赠
		campaign2 := createTestCampaign(db, func(c *models.Campaign) {
			c.Type = models.CampaignTypeGift
		})
		detail2, _ := svc.GetCampaignDetail(context.Background(), campaign2.ID)
		assert.Equal(t, "满赠", detail2.TypeText)

		// 秒杀
		campaign3 := createTestCampaign(db, func(c *models.Campaign) {
			c.Type = models.CampaignTypeFlashSale
		})
		detail3, _ := svc.GetCampaignDetail(context.Background(), campaign3.ID)
		assert.Equal(t, "秒杀", detail3.TypeText)

		// 团购
		campaign4 := createTestCampaign(db, func(c *models.Campaign) {
			c.Type = models.CampaignTypeGroupBuy
		})
		detail4, _ := svc.GetCampaignDetail(context.Background(), campaign4.ID)
		assert.Equal(t, "团购", detail4.TypeText)
	})
}

// TestCampaignService_GetCampaignsByType 测试按类型获取活动
func TestCampaignService_GetCampaignsByType(t *testing.T) {
	t.Run("正常按类型获取活动", func(t *testing.T) {
		db := setupCampaignServiceTestDB(t)
		svc := createCampaignTestService(db)

		// 创建满减活动
		createTestCampaign(db, func(c *models.Campaign) {
			c.Name = "满减1"
			c.Type = models.CampaignTypeDiscount
		})
		createTestCampaign(db, func(c *models.Campaign) {
			c.Name = "满减2"
			c.Type = models.CampaignTypeDiscount
		})

		// 创建秒杀活动
		createTestCampaign(db, func(c *models.Campaign) {
			c.Name = "秒杀1"
			c.Type = models.CampaignTypeFlashSale
		})

		// 获取满减活动
		campaigns, err := svc.GetCampaignsByType(context.Background(), models.CampaignTypeDiscount)
		require.NoError(t, err)
		assert.Len(t, campaigns, 2)

		// 获取秒杀活动
		campaigns, err = svc.GetCampaignsByType(context.Background(), models.CampaignTypeFlashSale)
		require.NoError(t, err)
		assert.Len(t, campaigns, 1)
	})

	t.Run("只返回有效活动", func(t *testing.T) {
		db := setupCampaignServiceTestDB(t)
		svc := createCampaignTestService(db)

		// 创建有效满减活动
		createTestCampaign(db, func(c *models.Campaign) {
			c.Name = "有效满减"
			c.Type = models.CampaignTypeDiscount
		})

		// 创建禁用满减活动
		createTestCampaign(db, func(c *models.Campaign) {
			c.Name = "禁用满减"
			c.Type = models.CampaignTypeDiscount
			c.Status = models.CampaignStatusDisabled
		})

		// 创建已结束满减活动
		createTestCampaign(db, func(c *models.Campaign) {
			c.Name = "已结束满减"
			c.Type = models.CampaignTypeDiscount
			c.StartTime = time.Now().Add(-48 * time.Hour)
			c.EndTime = time.Now().Add(-24 * time.Hour)
		})

		campaigns, err := svc.GetCampaignsByType(context.Background(), models.CampaignTypeDiscount)
		require.NoError(t, err)
		assert.Len(t, campaigns, 1)
		assert.Equal(t, "有效满减", campaigns[0].Name)
	})
}

// TestCampaignService_CalculateDiscountCampaign 测试计算满减活动优惠
func TestCampaignService_CalculateDiscountCampaign(t *testing.T) {
	t.Run("无满减活动返回零优惠", func(t *testing.T) {
		db := setupCampaignServiceTestDB(t)
		svc := createCampaignTestService(db)

		discount, campaign, err := svc.CalculateDiscountCampaign(context.Background(), 100.0)
		require.NoError(t, err)
		assert.Equal(t, 0.0, discount)
		assert.Nil(t, campaign)
	})

	t.Run("不使用禁用活动", func(t *testing.T) {
		db := setupCampaignServiceTestDB(t)
		svc := createCampaignTestService(db)

		createTestCampaign(db, func(c *models.Campaign) {
			c.Name = "禁用满减"
			c.Type = models.CampaignTypeDiscount
			c.Status = models.CampaignStatusDisabled
			c.Rules = nil
		})

		discount, campaign, err := svc.CalculateDiscountCampaign(context.Background(), 150.0)
		require.NoError(t, err)
		assert.Equal(t, 0.0, discount)
		assert.Nil(t, campaign)
	})

	t.Run("不使用已结束活动", func(t *testing.T) {
		db := setupCampaignServiceTestDB(t)
		svc := createCampaignTestService(db)

		createTestCampaign(db, func(c *models.Campaign) {
			c.Name = "已结束满减"
			c.Type = models.CampaignTypeDiscount
			c.StartTime = time.Now().Add(-48 * time.Hour)
			c.EndTime = time.Now().Add(-24 * time.Hour)
			c.Rules = nil
		})

		discount, campaign, err := svc.CalculateDiscountCampaign(context.Background(), 150.0)
		require.NoError(t, err)
		assert.Equal(t, 0.0, discount)
		assert.Nil(t, campaign)
	})

	t.Run("活动规则为空时返回零优惠", func(t *testing.T) {
		db := setupCampaignServiceTestDB(t)
		svc := createCampaignTestService(db)

		createTestCampaign(db, func(c *models.Campaign) {
			c.Name = "无规则活动"
			c.Type = models.CampaignTypeDiscount
			c.Rules = nil
		})

		discount, _, err := svc.CalculateDiscountCampaign(context.Background(), 150.0)
		require.NoError(t, err)
		assert.Equal(t, 0.0, discount)
	})
}
