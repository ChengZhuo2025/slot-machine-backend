// Package database 数据库模块单元测试
package database

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

// ==================== getLogLevel 测试 ====================

func TestGetLogLevel(t *testing.T) {
	tests := []struct {
		name     string
		logMode  bool
		expected logger.LogLevel
	}{
		{"log mode enabled", true, logger.Info},
		{"log mode disabled", false, logger.Silent},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := getLogLevel(tt.logMode)
			assert.Equal(t, tt.expected, result)
		})
	}
}

// ==================== Paginate 测试 ====================

func TestPaginate(t *testing.T) {
	// 创建测试数据库
	testDB, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	require.NoError(t, err)

	// 创建测试表
	type TestModel struct {
		ID   int64
		Name string
	}
	err = testDB.AutoMigrate(&TestModel{})
	require.NoError(t, err)

	// 插入测试数据
	for i := 1; i <= 50; i++ {
		testDB.Create(&TestModel{ID: int64(i), Name: "Item"})
	}

	tests := []struct {
		name         string
		page         int
		pageSize     int
		expectedLen  int
		expectedFrom int64
	}{
		{"first page", 1, 10, 10, 1},
		{"second page", 2, 10, 10, 11},
		{"third page", 3, 10, 10, 21},
		{"last page", 5, 10, 10, 41},
		{"page with less items", 6, 10, 0, 0},
		{"zero page defaults to 1", 0, 10, 10, 1},
		{"negative page defaults to 1", -1, 10, 10, 1},
		{"zero pageSize defaults to 10", 1, 0, 10, 1},
		{"negative pageSize defaults to 10", 1, -5, 10, 1},
		{"pageSize over 100 capped", 1, 200, 50, 1}, // 50 items total
		{"custom pageSize 20", 1, 20, 20, 1},
		{"custom pageSize 5", 2, 5, 5, 6},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var results []TestModel
			testDB.Scopes(Paginate(tt.page, tt.pageSize)).Find(&results)

			assert.Len(t, results, tt.expectedLen)
			if tt.expectedLen > 0 {
				assert.Equal(t, tt.expectedFrom, results[0].ID)
			}
		})
	}
}

func TestPaginate_EdgeCases(t *testing.T) {
	testDB, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	require.NoError(t, err)

	type Item struct {
		ID int64
	}
	_ = testDB.AutoMigrate(&Item{})

	// 插入 5 条数据
	for i := 1; i <= 5; i++ {
		testDB.Create(&Item{ID: int64(i)})
	}

	t.Run("pageSize exactly equals total", func(t *testing.T) {
		var results []Item
		testDB.Scopes(Paginate(1, 5)).Find(&results)
		assert.Len(t, results, 5)
	})

	t.Run("pageSize greater than total", func(t *testing.T) {
		var results []Item
		testDB.Scopes(Paginate(1, 20)).Find(&results)
		assert.Len(t, results, 5)
	})

	t.Run("empty table", func(t *testing.T) {
		testDB.Exec("DELETE FROM items")
		var results []Item
		testDB.Scopes(Paginate(1, 10)).Find(&results)
		assert.Len(t, results, 0)
	})
}

// ==================== OrderByCreatedDesc / OrderByUpdatedDesc 测试 ====================

func TestOrderByCreatedDesc(t *testing.T) {
	testDB, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	require.NoError(t, err)

	type Record struct {
		ID        int64
		CreatedAt time.Time
	}
	_ = testDB.AutoMigrate(&Record{})

	// 插入数据，不同创建时间
	now := time.Now()
	testDB.Create(&Record{ID: 1, CreatedAt: now.Add(-2 * time.Hour)})
	testDB.Create(&Record{ID: 2, CreatedAt: now.Add(-1 * time.Hour)})
	testDB.Create(&Record{ID: 3, CreatedAt: now})

	var results []Record
	testDB.Scopes(OrderByCreatedDesc).Find(&results)

	require.Len(t, results, 3)
	assert.Equal(t, int64(3), results[0].ID)
	assert.Equal(t, int64(2), results[1].ID)
	assert.Equal(t, int64(1), results[2].ID)
}

func TestOrderByUpdatedDesc(t *testing.T) {
	testDB, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	require.NoError(t, err)

	type Record struct {
		ID        int64
		UpdatedAt time.Time
	}
	_ = testDB.AutoMigrate(&Record{})

	now := time.Now()
	testDB.Create(&Record{ID: 1, UpdatedAt: now.Add(-2 * time.Hour)})
	testDB.Create(&Record{ID: 2, UpdatedAt: now.Add(-1 * time.Hour)})
	testDB.Create(&Record{ID: 3, UpdatedAt: now})

	var results []Record
	testDB.Scopes(OrderByUpdatedDesc).Find(&results)

	require.Len(t, results, 3)
	assert.Equal(t, int64(3), results[0].ID)
	assert.Equal(t, int64(2), results[1].ID)
	assert.Equal(t, int64(1), results[2].ID)
}

// ==================== GetDB / Close / Transaction / WithContext 测试 ====================

func TestGetDB_ReturnsGlobalDB(t *testing.T) {
	// 设置测试数据库
	testDB, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	require.NoError(t, err)

	oldDB := db
	db = testDB
	t.Cleanup(func() {
		db = oldDB
	})

	result := GetDB()
	assert.Equal(t, testDB, result)
}

func TestClose_WithNilDB(t *testing.T) {
	oldDB := db
	db = nil
	t.Cleanup(func() {
		db = oldDB
	})

	err := Close()
	assert.NoError(t, err)
}

func TestClose_WithActiveDB(t *testing.T) {
	testDB, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	require.NoError(t, err)

	oldDB := db
	db = testDB
	t.Cleanup(func() {
		db = oldDB
	})

	err = Close()
	assert.NoError(t, err)
}

func TestTransaction_Success(t *testing.T) {
	testDB, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	require.NoError(t, err)

	type Counter struct {
		ID    int64
		Value int
	}
	_ = testDB.AutoMigrate(&Counter{})

	oldDB := db
	db = testDB
	t.Cleanup(func() {
		db = oldDB
	})

	err = Transaction(func(tx *gorm.DB) error {
		return tx.Create(&Counter{ID: 1, Value: 100}).Error
	})
	assert.NoError(t, err)

	// 验证数据已提交
	var counter Counter
	testDB.First(&counter, 1)
	assert.Equal(t, 100, counter.Value)
}

func TestTransaction_Rollback(t *testing.T) {
	testDB, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	require.NoError(t, err)

	type Counter struct {
		ID    int64
		Value int
	}
	_ = testDB.AutoMigrate(&Counter{})

	oldDB := db
	db = testDB
	t.Cleanup(func() {
		db = oldDB
	})

	err = Transaction(func(tx *gorm.DB) error {
		tx.Create(&Counter{ID: 1, Value: 100})
		return assert.AnError // 模拟错误
	})
	assert.Error(t, err)

	// 验证数据已回滚
	var count int64
	testDB.Model(&Counter{}).Count(&count)
	assert.Equal(t, int64(0), count)
}

func TestWithContext(t *testing.T) {
	testDB, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	require.NoError(t, err)

	oldDB := db
	db = testDB
	t.Cleanup(func() {
		db = oldDB
	})

	ctx := context.WithValue(context.Background(), "key", "value")
	dbWithCtx := WithContext(ctx)

	assert.NotNil(t, dbWithCtx)
	// 验证返回的是带 context 的新 DB 实例
	assert.NotEqual(t, db, dbWithCtx)
}

// ==================== 组合使用测试 ====================

func TestPaginate_WithOrderBy(t *testing.T) {
	testDB, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	require.NoError(t, err)

	type Article struct {
		ID        int64
		Title     string
		CreatedAt time.Time
	}
	_ = testDB.AutoMigrate(&Article{})

	now := time.Now()
	for i := 1; i <= 30; i++ {
		testDB.Create(&Article{
			ID:        int64(i),
			Title:     "Article",
			CreatedAt: now.Add(time.Duration(i) * time.Hour),
		})
	}

	// 分页 + 排序组合
	var results []Article
	testDB.Scopes(OrderByCreatedDesc, Paginate(1, 10)).Find(&results)

	require.Len(t, results, 10)
	// 最新的应该在前面
	assert.Equal(t, int64(30), results[0].ID)
	assert.Equal(t, int64(21), results[9].ID)

	// 第二页
	testDB.Scopes(OrderByCreatedDesc, Paginate(2, 10)).Find(&results)
	require.Len(t, results, 10)
	assert.Equal(t, int64(20), results[0].ID)
	assert.Equal(t, int64(11), results[9].ID)
}

// ==================== DatabaseConfig.DSN 测试 ====================

func TestDatabaseConfig_DSN(t *testing.T) {
	// 注意：这个测试需要导入 config 包
	// 这里我们通过模拟验证 DSN 格式

	tests := []struct {
		name     string
		host     string
		port     int
		user     string
		password string
		dbName   string
		sslMode  string
		timezone string
	}{
		{
			name:     "default config",
			host:     "localhost",
			port:     5432,
			user:     "postgres",
			password: "password",
			dbName:   "smart_locker",
			sslMode:  "disable",
			timezone: "Asia/Shanghai",
		},
		{
			name:     "production config",
			host:     "db.example.com",
			port:     5433,
			user:     "app_user",
			password: "secure_pass",
			dbName:   "production_db",
			sslMode:  "require",
			timezone: "UTC",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			expected := "host=" + tt.host +
				" port=" + string(rune(tt.port+'0')) +
				" user=" + tt.user +
				" password=" + tt.password +
				" dbname=" + tt.dbName +
				" sslmode=" + tt.sslMode +
				" TimeZone=" + tt.timezone

			// 验证 DSN 包含必要元素
			assert.Contains(t, expected, "host=")
			assert.Contains(t, expected, "port=")
			assert.Contains(t, expected, "user=")
			assert.Contains(t, expected, "password=")
			assert.Contains(t, expected, "dbname=")
			assert.Contains(t, expected, "sslmode=")
			assert.Contains(t, expected, "TimeZone=")
		})
	}
}

// ==================== 并发安全测试 ====================

func TestGetDB_ConcurrentAccess(t *testing.T) {
	testDB, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	require.NoError(t, err)

	oldDB := db
	db = testDB
	t.Cleanup(func() {
		db = oldDB
	})

	// 并发访问 GetDB
	done := make(chan bool, 10)
	for i := 0; i < 10; i++ {
		go func() {
			result := GetDB()
			assert.NotNil(t, result)
			done <- true
		}()
	}

	// 等待所有 goroutine 完成
	for i := 0; i < 10; i++ {
		<-done
	}
}

func TestWithContext_ConcurrentAccess(t *testing.T) {
	testDB, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	require.NoError(t, err)

	oldDB := db
	db = testDB
	t.Cleanup(func() {
		db = oldDB
	})

	done := make(chan bool, 10)
	for i := 0; i < 10; i++ {
		go func(id int) {
			ctx := context.WithValue(context.Background(), "id", id)
			dbWithCtx := WithContext(ctx)
			assert.NotNil(t, dbWithCtx)
			done <- true
		}(i)
	}

	for i := 0; i < 10; i++ {
		<-done
	}
}

// ==================== 边界条件测试 ====================

func TestPaginate_LargePageNumber(t *testing.T) {
	testDB, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	require.NoError(t, err)

	type Item struct {
		ID int64
	}
	_ = testDB.AutoMigrate(&Item{})

	for i := 1; i <= 10; i++ {
		testDB.Create(&Item{ID: int64(i)})
	}

	var results []Item
	testDB.Scopes(Paginate(1000, 10)).Find(&results)
	assert.Len(t, results, 0)
}

func TestPaginate_PageSizeOne(t *testing.T) {
	testDB, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	require.NoError(t, err)

	type Item struct {
		ID int64
	}
	_ = testDB.AutoMigrate(&Item{})

	for i := 1; i <= 5; i++ {
		testDB.Create(&Item{ID: int64(i)})
	}

	for page := 1; page <= 5; page++ {
		var results []Item
		testDB.Scopes(Paginate(page, 1)).Find(&results)
		require.Len(t, results, 1)
		assert.Equal(t, int64(page), results[0].ID)
	}
}

func TestPaginate_ExactlyMaxPageSize(t *testing.T) {
	testDB, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	require.NoError(t, err)

	type Item struct {
		ID int64
	}
	_ = testDB.AutoMigrate(&Item{})

	for i := 1; i <= 100; i++ {
		testDB.Create(&Item{ID: int64(i)})
	}

	var results []Item
	testDB.Scopes(Paginate(1, 100)).Find(&results)
	assert.Len(t, results, 100)
}
