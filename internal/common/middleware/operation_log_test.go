package middleware

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

	"github.com/dumeirei/smart-locker-backend/internal/models"
	"github.com/dumeirei/smart-locker-backend/internal/repository"
)

func setupOperationLogTestDB(t *testing.T) *gorm.DB {
	t.Helper()
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
		&models.OperationLog{},
	))
	return db
}

func waitForOperationLog(t *testing.T, db *gorm.DB, where string, args ...interface{}) *models.OperationLog {
	t.Helper()

	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		var log models.OperationLog
		err := db.Where(where, args...).Order("id DESC").First(&log).Error
		if err == nil {
			return &log
		}
		time.Sleep(20 * time.Millisecond)
	}
	t.Fatalf("operation log not created: %s", where)
	return nil
}

func TestOperationLogger_LogsAdminWriteOperations_WithActionMapping(t *testing.T) {
	gin.SetMode(gin.TestMode)
	db := setupOperationLogTestDB(t)

	// Seed admin (only for FK-like semantics; the middleware reads admin_id from context)
	require.NoError(t, db.Create(&models.Admin{
		Username:     "oplog_admin",
		PasswordHash: "hash",
		Name:         "管理员",
		RoleID:       1,
		Status:       models.AdminStatusActive,
	}).Error)

	repo := repository.NewOperationLogRepository(db)
	op := NewOperationLogger(repo)

	r := gin.New()
	admin := r.Group("/api/admin")
	admin.Use(func(c *gin.Context) {
		c.Set("user_id", int64(1))
		c.Set("user_type", "admin")
		c.Next()
	})
	admin.Use(op.Log())

	admin.POST("/devices", func(c *gin.Context) { c.JSON(http.StatusOK, gin.H{"code": 0}) })
	admin.PUT("/devices/:id/status", func(c *gin.Context) { c.JSON(http.StatusOK, gin.H{"code": 0}) })

	body, _ := json.Marshal(map[string]interface{}{"device_no": "D001"})
	req, _ := http.NewRequest("POST", "/api/admin/devices", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	require.Equal(t, http.StatusOK, w.Code)

	log := waitForOperationLog(t, db, "module = ? AND action = ?", "device", "create")
	assert.Equal(t, int64(1), log.AdminID)
	require.NotNil(t, log.TargetType)
	assert.Equal(t, "device", *log.TargetType)
	assert.Nil(t, log.TargetID)

	statusBody, _ := json.Marshal(map[string]interface{}{"status": 2})
	req2, _ := http.NewRequest("PUT", "/api/admin/devices/123/status", bytes.NewBuffer(statusBody))
	req2.Header.Set("Content-Type", "application/json")
	w2 := httptest.NewRecorder()
	r.ServeHTTP(w2, req2)
	require.Equal(t, http.StatusOK, w2.Code)

	log2 := waitForOperationLog(t, db, "module = ? AND action = ? AND target_id = ?", "device", "update_status", 123)
	assert.Equal(t, int64(1), log2.AdminID)
}

