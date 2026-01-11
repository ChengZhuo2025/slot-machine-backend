// Package repository 地址仓储单元测试
package repository

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"

	"github.com/dumeirei/smart-locker-backend/internal/models"
)

func setupAddressTestDB(t *testing.T) *gorm.DB {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Silent),
	})
	require.NoError(t, err)

	err = db.AutoMigrate(&models.Address{}, &models.User{})
	require.NoError(t, err)

	return db
}

func TestAddressRepository_Create(t *testing.T) {
	db := setupAddressTestDB(t)
	repo := NewAddressRepository(db)
	ctx := context.Background()

	address := &models.Address{
		UserID:       1,
		ReceiverName: "张三",
		ReceiverPhone: "13800138000",
		Province:     "广东省",
		City:         "深圳市",
		District:     "南山区",
		Detail:       "科技园",
		IsDefault:    false,
	}

	err := repo.Create(ctx, address)
	require.NoError(t, err)
	assert.NotZero(t, address.ID)
}

func TestAddressRepository_GetByID(t *testing.T) {
	db := setupAddressTestDB(t)
	repo := NewAddressRepository(db)
	ctx := context.Background()

	address := &models.Address{
		UserID:       1,
		ReceiverName: "李四",
		ReceiverPhone: "13800138001",
		Province:     "北京市",
		City:         "北京市",
		District:     "朝阳区",
		Detail:       "三里屯",
		IsDefault:    true,
	}
	db.Create(address)

	found, err := repo.GetByID(ctx, address.ID)
	require.NoError(t, err)
	assert.Equal(t, address.ID, found.ID)
	assert.Equal(t, "李四", found.ReceiverName)
	assert.True(t, found.IsDefault)
}

func TestAddressRepository_GetByIDAndUser(t *testing.T) {
	db := setupAddressTestDB(t)
	repo := NewAddressRepository(db)
	ctx := context.Background()

	address := &models.Address{
		UserID:       100,
		ReceiverName: "王五",
		ReceiverPhone: "13800138002",
		Province:     "上海市",
		City:         "上海市",
		District:     "浦东新区",
		Detail:       "陆家嘴",
		IsDefault:    false,
	}
	db.Create(address)

	// 正确的用户ID
	found, err := repo.GetByIDAndUser(ctx, address.ID, 100)
	require.NoError(t, err)
	assert.Equal(t, address.ID, found.ID)

	// 错误的用户ID
	found, err = repo.GetByIDAndUser(ctx, address.ID, 999)
	assert.Error(t, err)
	assert.Nil(t, found)
}

func TestAddressRepository_Update(t *testing.T) {
	db := setupAddressTestDB(t)
	repo := NewAddressRepository(db)
	ctx := context.Background()

	address := &models.Address{
		UserID:       1,
		ReceiverName: "赵六",
		ReceiverPhone: "13800138003",
		Province:     "浙江省",
		City:         "杭州市",
		District:     "西湖区",
		Detail:       "西溪湿地",
		IsDefault:    false,
	}
	db.Create(address)

	address.ReceiverName = "赵六（更新）"
	address.Detail = "西溪湿地公园"
	err := repo.Update(ctx, address)
	require.NoError(t, err)

	var found models.Address
	db.First(&found, address.ID)
	assert.Equal(t, "赵六（更新）", found.ReceiverName)
	assert.Equal(t, "西溪湿地公园", found.Detail)
}

func TestAddressRepository_Delete(t *testing.T) {
	db := setupAddressTestDB(t)
	repo := NewAddressRepository(db)
	ctx := context.Background()

	address := &models.Address{
		UserID:       1,
		ReceiverName: "测试删除",
		ReceiverPhone: "13800138004",
		Province:     "江苏省",
		City:         "南京市",
		District:     "鼓楼区",
		Detail:       "新街口",
		IsDefault:    false,
	}
	db.Create(address)

	err := repo.Delete(ctx, address.ID)
	require.NoError(t, err)

	var count int64
	db.Model(&models.Address{}).Where("id = ?", address.ID).Count(&count)
	assert.Equal(t, int64(0), count)
}

func TestAddressRepository_DeleteByIDAndUser(t *testing.T) {
	db := setupAddressTestDB(t)
	repo := NewAddressRepository(db)
	ctx := context.Background()

	address := &models.Address{
		UserID:       200,
		ReceiverName: "测试用户删除",
		ReceiverPhone: "13800138005",
		Province:     "四川省",
		City:         "成都市",
		District:     "武侯区",
		Detail:       "天府广场",
		IsDefault:    false,
	}
	db.Create(address)

	// 错误的用户ID，应该返回错误
	err := repo.DeleteByIDAndUser(ctx, address.ID, 999)
	assert.Error(t, err) // 应该返回 record not found 错误

	var count int64
	db.Model(&models.Address{}).Where("id = ?", address.ID).Count(&count)
	assert.Equal(t, int64(1), count) // 仍然存在

	// 正确的用户ID
	err = repo.DeleteByIDAndUser(ctx, address.ID, 200)
	require.NoError(t, err)

	db.Model(&models.Address{}).Where("id = ?", address.ID).Count(&count)
	assert.Equal(t, int64(0), count)
}

func TestAddressRepository_ListByUser(t *testing.T) {
	db := setupAddressTestDB(t)
	repo := NewAddressRepository(db)
	ctx := context.Background()

	// 创建多个地址
	userID := int64(300)
	addresses := []*models.Address{
		{UserID: userID, ReceiverName: "地址1", ReceiverPhone: "13800138001", Province: "广东省", City: "广州市", District: "天河区", Detail: "天河路", IsDefault: false},
		{UserID: userID, ReceiverName: "地址2", ReceiverPhone: "13800138002", Province: "广东省", City: "深圳市", District: "福田区", Detail: "华强北", IsDefault: true},
		{UserID: 999, ReceiverName: "其他用户", ReceiverPhone: "13800138003", Province: "北京市", City: "北京市", District: "海淀区", Detail: "中关村", IsDefault: false},
	}
	for _, addr := range addresses {
		db.Create(addr)
	}

	list, err := repo.ListByUser(ctx, userID)
	require.NoError(t, err)
	assert.Equal(t, 2, len(list))
}

func TestAddressRepository_GetDefaultByUser(t *testing.T) {
	db := setupAddressTestDB(t)
	repo := NewAddressRepository(db)
	ctx := context.Background()

	userID := int64(400)
	addresses := []*models.Address{
		{UserID: userID, ReceiverName: "非默认地址", ReceiverPhone: "13800138001", Province: "广东省", City: "广州市", District: "天河区", Detail: "天河路", IsDefault: false},
		{UserID: userID, ReceiverName: "默认地址", ReceiverPhone: "13800138002", Province: "广东省", City: "深圳市", District: "福田区", Detail: "华强北", IsDefault: true},
	}
	for _, addr := range addresses {
		db.Create(addr)
	}

	defaultAddr, err := repo.GetDefaultByUser(ctx, userID)
	require.NoError(t, err)
	assert.Equal(t, "默认地址", defaultAddr.ReceiverName)
	assert.True(t, defaultAddr.IsDefault)
}

func TestAddressRepository_SetDefault(t *testing.T) {
	db := setupAddressTestDB(t)
	repo := NewAddressRepository(db)
	ctx := context.Background()

	userID := int64(500)
	addr1 := &models.Address{UserID: userID, ReceiverName: "地址1", ReceiverPhone: "13800138001", Province: "广东省", City: "广州市", District: "天河区", Detail: "天河路", IsDefault: true}
	addr2 := &models.Address{UserID: userID, ReceiverName: "地址2", ReceiverPhone: "13800138002", Province: "广东省", City: "深圳市", District: "福田区", Detail: "华强北", IsDefault: false}
	db.Create(addr1)
	db.Create(addr2)

	// 设置地址2为默认
	err := repo.SetDefault(ctx, userID, addr2.ID)
	require.NoError(t, err)

	// 验证
	var updatedAddr1, updatedAddr2 models.Address
	db.First(&updatedAddr1, addr1.ID)
	db.First(&updatedAddr2, addr2.ID)

	assert.False(t, updatedAddr1.IsDefault)
	assert.True(t, updatedAddr2.IsDefault)
}

func TestAddressRepository_ClearDefault(t *testing.T) {
	db := setupAddressTestDB(t)
	repo := NewAddressRepository(db)
	ctx := context.Background()

	userID := int64(600)
	addr := &models.Address{UserID: userID, ReceiverName: "默认地址", ReceiverPhone: "13800138001", Province: "广东省", City: "广州市", District: "天河区", Detail: "天河路", IsDefault: true}
	db.Create(addr)

	err := repo.ClearDefault(ctx, userID)
	require.NoError(t, err)

	var updated models.Address
	db.First(&updated, addr.ID)
	assert.False(t, updated.IsDefault)
}

func TestAddressRepository_CountByUser(t *testing.T) {
	db := setupAddressTestDB(t)
	repo := NewAddressRepository(db)
	ctx := context.Background()

	userID := int64(700)
	addresses := []*models.Address{
		{UserID: userID, ReceiverName: "地址1", ReceiverPhone: "13800138001", Province: "广东省", City: "广州市", District: "天河区", Detail: "天河路", IsDefault: false},
		{UserID: userID, ReceiverName: "地址2", ReceiverPhone: "13800138002", Province: "广东省", City: "深圳市", District: "福田区", Detail: "华强北", IsDefault: false},
		{UserID: userID, ReceiverName: "地址3", ReceiverPhone: "13800138003", Province: "广东省", City: "东莞市", District: "南城区", Detail: "南城中心", IsDefault: true},
	}
	for _, addr := range addresses {
		db.Create(addr)
	}

	count, err := repo.CountByUser(ctx, userID)
	require.NoError(t, err)
	assert.Equal(t, int64(3), count)

	// 测试不存在的用户
	count, err = repo.CountByUser(ctx, 999)
	require.NoError(t, err)
	assert.Equal(t, int64(0), count)
}
