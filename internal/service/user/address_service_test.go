// Package user 地址服务单元测试
package user

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"

	"github.com/dumeirei/smart-locker-backend/internal/models"
	"github.com/dumeirei/smart-locker-backend/internal/repository"
)

// setupAddressTestDB 创建地址测试数据库
func setupAddressTestDB(t *testing.T) *gorm.DB {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Silent),
	})
	require.NoError(t, err)

	err = db.AutoMigrate(&models.User{}, &models.Address{})
	require.NoError(t, err)

	return db
}

// setupAddressService 创建测试用的 AddressService
func setupAddressService(t *testing.T) (*AddressService, *gorm.DB) {
	db := setupAddressTestDB(t)
	addressRepo := repository.NewAddressRepository(db)
	service := NewAddressService(addressRepo)
	return service, db
}

// createTestUserForAddress 创建测试用户
func createTestUserForAddress(t *testing.T, db *gorm.DB) *models.User {
	phone := "13800138000"
	user := &models.User{
		Phone:    &phone,
		Nickname: "测试用户",
		Status:   1,
	}
	err := db.Create(user).Error
	require.NoError(t, err)
	return user
}

func TestAddressService_Create(t *testing.T) {
	service, db := setupAddressService(t)
	ctx := context.Background()

	user := createTestUserForAddress(t, db)

	t.Run("创建第一个地址自动设为默认", func(t *testing.T) {
		req := &CreateAddressRequest{
			ReceiverName:  "张三",
			ReceiverPhone: "13800138000",
			Province:      "广东省",
			City:          "深圳市",
			District:      "南山区",
			Detail:        "科技园路1号",
			IsDefault:     false, // 即使设为 false，第一个地址也会自动设为默认
		}

		address, err := service.Create(ctx, user.ID, req)
		require.NoError(t, err)
		assert.NotZero(t, address.ID)
		assert.Equal(t, user.ID, address.UserID)
		assert.Equal(t, "张三", address.ReceiverName)
		assert.True(t, address.IsDefault) // 第一个地址自动设为默认
	})

	t.Run("创建地址带邮编和标签", func(t *testing.T) {
		postalCode := "518000"
		tag := "公司"
		req := &CreateAddressRequest{
			ReceiverName:  "李四",
			ReceiverPhone: "13900139000",
			Province:      "广东省",
			City:          "深圳市",
			District:      "福田区",
			Detail:        "福田CBD大厦",
			PostalCode:    &postalCode,
			Tag:           &tag,
			IsDefault:     false,
		}

		address, err := service.Create(ctx, user.ID, req)
		require.NoError(t, err)
		assert.NotNil(t, address.PostalCode)
		assert.Equal(t, "518000", *address.PostalCode)
		assert.NotNil(t, address.Tag)
		assert.Equal(t, "公司", *address.Tag)
		assert.False(t, address.IsDefault) // 不是第一个，保持原设置
	})

	t.Run("设置新地址为默认会清除原默认", func(t *testing.T) {
		req := &CreateAddressRequest{
			ReceiverName:  "王五",
			ReceiverPhone: "13700137000",
			Province:      "北京市",
			City:          "北京市",
			District:      "朝阳区",
			Detail:        "国贸大厦",
			IsDefault:     true,
		}

		newDefault, err := service.Create(ctx, user.ID, req)
		require.NoError(t, err)
		assert.True(t, newDefault.IsDefault)

		// 验证旧的默认地址已被取消
		addresses, _ := service.List(ctx, user.ID)
		defaultCount := 0
		for _, addr := range addresses {
			if addr.IsDefault {
				defaultCount++
				assert.Equal(t, newDefault.ID, addr.ID)
			}
		}
		assert.Equal(t, 1, defaultCount)
	})
}

func TestAddressService_GetByID(t *testing.T) {
	service, db := setupAddressService(t)
	ctx := context.Background()

	user1 := createTestUserForAddress(t, db)
	phone2 := "13900139000"
	user2 := &models.User{Phone: &phone2, Nickname: "用户2", Status: 1}
	db.Create(user2)

	// 创建测试地址
	address := &models.Address{
		UserID:        user1.ID,
		ReceiverName:  "张三",
		ReceiverPhone: "13800138000",
		Province:      "广东省",
		City:          "深圳市",
		District:      "南山区",
		Detail:        "科技园路1号",
		IsDefault:     true,
	}
	db.Create(address)

	t.Run("获取自己的地址", func(t *testing.T) {
		result, err := service.GetByID(ctx, address.ID, user1.ID)
		require.NoError(t, err)
		assert.Equal(t, address.ID, result.ID)
		assert.Equal(t, "张三", result.ReceiverName)
	})

	t.Run("无法获取他人的地址", func(t *testing.T) {
		_, err := service.GetByID(ctx, address.ID, user2.ID)
		assert.Error(t, err)
	})

	t.Run("获取不存在的地址", func(t *testing.T) {
		_, err := service.GetByID(ctx, 99999, user1.ID)
		assert.Error(t, err)
	})
}

func TestAddressService_Update(t *testing.T) {
	service, db := setupAddressService(t)
	ctx := context.Background()

	user := createTestUserForAddress(t, db)

	// 创建测试地址
	address1 := &models.Address{
		UserID:        user.ID,
		ReceiverName:  "张三",
		ReceiverPhone: "13800138000",
		Province:      "广东省",
		City:          "深圳市",
		District:      "南山区",
		Detail:        "科技园路1号",
		IsDefault:     true,
	}
	db.Create(address1)

	address2 := &models.Address{
		UserID:        user.ID,
		ReceiverName:  "李四",
		ReceiverPhone: "13900139000",
		Province:      "广东省",
		City:          "广州市",
		District:      "天河区",
		Detail:        "珠江新城",
		IsDefault:     false,
	}
	db.Create(address2)

	t.Run("更新地址信息", func(t *testing.T) {
		newName := "张三更新"
		newDetail := "科技园路999号"
		req := &UpdateAddressRequest{
			ReceiverName: &newName,
			Detail:       &newDetail,
		}

		updated, err := service.Update(ctx, address1.ID, user.ID, req)
		require.NoError(t, err)
		assert.Equal(t, "张三更新", updated.ReceiverName)
		assert.Equal(t, "科技园路999号", updated.Detail)
		// 其他字段保持不变
		assert.Equal(t, "广东省", updated.Province)
		assert.True(t, updated.IsDefault)
	})

	t.Run("更新为默认地址", func(t *testing.T) {
		isDefault := true
		req := &UpdateAddressRequest{
			IsDefault: &isDefault,
		}

		updated, err := service.Update(ctx, address2.ID, user.ID, req)
		require.NoError(t, err)
		assert.True(t, updated.IsDefault)

		// 验证原默认地址已被取消
		oldDefault, _ := service.GetByID(ctx, address1.ID, user.ID)
		assert.False(t, oldDefault.IsDefault)
	})

	t.Run("更新不存在的地址", func(t *testing.T) {
		newName := "测试"
		req := &UpdateAddressRequest{ReceiverName: &newName}
		_, err := service.Update(ctx, 99999, user.ID, req)
		assert.Error(t, err)
	})
}

func TestAddressService_Delete(t *testing.T) {
	service, db := setupAddressService(t)
	ctx := context.Background()

	user := createTestUserForAddress(t, db)

	t.Run("删除非默认地址", func(t *testing.T) {
		// 创建两个地址
		address1 := &models.Address{
			UserID:        user.ID,
			ReceiverName:  "张三",
			ReceiverPhone: "13800138000",
			Province:      "广东省",
			City:          "深圳市",
			District:      "南山区",
			Detail:        "科技园路1号",
			IsDefault:     true,
		}
		db.Create(address1)

		address2 := &models.Address{
			UserID:        user.ID,
			ReceiverName:  "李四",
			ReceiverPhone: "13900139000",
			Province:      "广东省",
			City:          "广州市",
			District:      "天河区",
			Detail:        "珠江新城",
			IsDefault:     false,
		}
		db.Create(address2)

		err := service.Delete(ctx, address2.ID, user.ID)
		require.NoError(t, err)

		// 验证已删除
		_, err = service.GetByID(ctx, address2.ID, user.ID)
		assert.Error(t, err)

		// 验证默认地址不变
		defaultAddr, _ := service.GetDefault(ctx, user.ID)
		assert.Equal(t, address1.ID, defaultAddr.ID)
	})

	t.Run("删除默认地址会自动设置新默认", func(t *testing.T) {
		// 清理并重新创建
		db.Exec("DELETE FROM addresses")

		address1 := &models.Address{
			UserID:        user.ID,
			ReceiverName:  "张三",
			ReceiverPhone: "13800138000",
			Province:      "广东省",
			City:          "深圳市",
			District:      "南山区",
			Detail:        "科技园路1号",
			IsDefault:     true,
		}
		db.Create(address1)

		address2 := &models.Address{
			UserID:        user.ID,
			ReceiverName:  "李四",
			ReceiverPhone: "13900139000",
			Province:      "广东省",
			City:          "广州市",
			District:      "天河区",
			Detail:        "珠江新城",
			IsDefault:     false,
		}
		db.Create(address2)

		// 删除默认地址
		err := service.Delete(ctx, address1.ID, user.ID)
		require.NoError(t, err)

		// 验证新默认地址已设置
		addresses, _ := service.List(ctx, user.ID)
		assert.Len(t, addresses, 1)
		assert.True(t, addresses[0].IsDefault)
	})
}

func TestAddressService_List(t *testing.T) {
	service, db := setupAddressService(t)
	ctx := context.Background()

	user := createTestUserForAddress(t, db)

	// 创建多个地址
	addresses := []*models.Address{
		{UserID: user.ID, ReceiverName: "张三", ReceiverPhone: "13800138000", Province: "广东省", City: "深圳市", District: "南山区", Detail: "地址1", IsDefault: true},
		{UserID: user.ID, ReceiverName: "李四", ReceiverPhone: "13900139000", Province: "广东省", City: "广州市", District: "天河区", Detail: "地址2", IsDefault: false},
		{UserID: user.ID, ReceiverName: "王五", ReceiverPhone: "13700137000", Province: "北京市", City: "北京市", District: "朝阳区", Detail: "地址3", IsDefault: false},
	}
	for _, addr := range addresses {
		db.Create(addr)
	}

	results, err := service.List(ctx, user.ID)
	require.NoError(t, err)
	assert.Len(t, results, 3)
	// 默认地址应该排在最前
	assert.True(t, results[0].IsDefault)
}

func TestAddressService_GetDefault(t *testing.T) {
	service, db := setupAddressService(t)
	ctx := context.Background()

	user := createTestUserForAddress(t, db)

	t.Run("有默认地址", func(t *testing.T) {
		address := &models.Address{
			UserID:        user.ID,
			ReceiverName:  "张三",
			ReceiverPhone: "13800138000",
			Province:      "广东省",
			City:          "深圳市",
			District:      "南山区",
			Detail:        "科技园路1号",
			IsDefault:     true,
		}
		db.Create(address)

		result, err := service.GetDefault(ctx, user.ID)
		require.NoError(t, err)
		assert.Equal(t, address.ID, result.ID)
		assert.True(t, result.IsDefault)
	})

	t.Run("无默认地址", func(t *testing.T) {
		phone2 := "13900139000"
		user2 := &models.User{Phone: &phone2, Nickname: "用户2", Status: 1}
		db.Create(user2)

		_, err := service.GetDefault(ctx, user2.ID)
		assert.Error(t, err)
	})
}

func TestAddressService_SetDefault(t *testing.T) {
	service, db := setupAddressService(t)
	ctx := context.Background()

	user := createTestUserForAddress(t, db)

	// 创建两个地址
	address1 := &models.Address{
		UserID:        user.ID,
		ReceiverName:  "张三",
		ReceiverPhone: "13800138000",
		Province:      "广东省",
		City:          "深圳市",
		District:      "南山区",
		Detail:        "地址1",
		IsDefault:     true,
	}
	db.Create(address1)

	address2 := &models.Address{
		UserID:        user.ID,
		ReceiverName:  "李四",
		ReceiverPhone: "13900139000",
		Province:      "广东省",
		City:          "广州市",
		District:      "天河区",
		Detail:        "地址2",
		IsDefault:     false,
	}
	db.Create(address2)

	t.Run("设置新默认地址", func(t *testing.T) {
		err := service.SetDefault(ctx, address2.ID, user.ID)
		require.NoError(t, err)

		// 验证新默认地址
		newDefault, _ := service.GetDefault(ctx, user.ID)
		assert.Equal(t, address2.ID, newDefault.ID)

		// 验证旧默认地址已取消
		var oldAddr models.Address
		db.First(&oldAddr, address1.ID)
		assert.False(t, oldAddr.IsDefault)
	})

	t.Run("设置不存在的地址为默认", func(t *testing.T) {
		err := service.SetDefault(ctx, 99999, user.ID)
		assert.Error(t, err)
	})
}

func TestAddressService_GetFullAddress(t *testing.T) {
	service, _ := setupAddressService(t)

	address := &models.Address{
		Province: "广东省",
		City:     "深圳市",
		District: "南山区",
		Detail:   "科技园路1号",
	}

	fullAddress := service.GetFullAddress(address)
	assert.Equal(t, "广东省深圳市南山区科技园路1号", fullAddress)
}

func TestAddressService_CreateSnapshot(t *testing.T) {
	service, _ := setupAddressService(t)

	postalCode := "518000"
	address := &models.Address{
		ID:            1,
		ReceiverName:  "张三",
		ReceiverPhone: "13800138000",
		Province:      "广东省",
		City:          "深圳市",
		District:      "南山区",
		Detail:        "科技园路1号",
		PostalCode:    &postalCode,
	}

	snapshot := service.CreateSnapshot(address)

	assert.Equal(t, int64(1), snapshot["id"])
	assert.Equal(t, "张三", snapshot["receiver_name"])
	assert.Equal(t, "13800138000", snapshot["receiver_phone"])
	assert.Equal(t, "广东省", snapshot["province"])
	assert.Equal(t, "深圳市", snapshot["city"])
	assert.Equal(t, "南山区", snapshot["district"])
	assert.Equal(t, "科技园路1号", snapshot["detail"])
	assert.Equal(t, &postalCode, snapshot["postal_code"])
	assert.Equal(t, "广东省深圳市南山区科技园路1号", snapshot["full_address"])
}

func TestAddressService_MaxAddressCount(t *testing.T) {
	service, db := setupAddressService(t)
	ctx := context.Background()

	user := createTestUserForAddress(t, db)

	// 创建20个地址（达到上限）
	for i := 0; i < MaxAddressCount; i++ {
		address := &models.Address{
			UserID:        user.ID,
			ReceiverName:  "用户",
			ReceiverPhone: "13800138000",
			Province:      "广东省",
			City:          "深圳市",
			District:      "南山区",
			Detail:        "地址",
			IsDefault:     i == 0,
		}
		db.Create(address)
	}

	// 尝试创建第21个地址
	req := &CreateAddressRequest{
		ReceiverName:  "超限地址",
		ReceiverPhone: "13900139000",
		Province:      "北京市",
		City:          "北京市",
		District:      "朝阳区",
		Detail:        "国贸大厦",
	}

	_, err := service.Create(ctx, user.ID, req)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "地址数量已达上限")
}
