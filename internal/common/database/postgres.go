// Package database 提供数据库连接和管理功能
package database

import (
	"context"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/dumeirei/smart-locker-backend/internal/common/config"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

var db *gorm.DB

// Init 初始化数据库连接
func Init(cfg *config.DatabaseConfig) (*gorm.DB, error) {
	var err error

	// 配置 GORM 日志
	gormLogger := logger.New(
		log.New(os.Stdout, "\r\n", log.LstdFlags),
		logger.Config{
			SlowThreshold:             time.Duration(cfg.SlowThreshold) * time.Millisecond,
			LogLevel:                  getLogLevel(cfg.LogMode),
			IgnoreRecordNotFoundError: true,
			Colorful:                  true,
		},
	)

	// 连接数据库
	db, err = gorm.Open(postgres.Open(cfg.DSN()), &gorm.Config{
		Logger:                                   gormLogger,
		DisableForeignKeyConstraintWhenMigrating: true,
		PrepareStmt:                              true,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to connect database: %w", err)
	}

	// 获取底层 *sql.DB
	sqlDB, err := db.DB()
	if err != nil {
		return nil, fmt.Errorf("failed to get sql.DB: %w", err)
	}

	// 配置连接池
	sqlDB.SetMaxIdleConns(cfg.MaxIdleConns)
	sqlDB.SetMaxOpenConns(cfg.MaxOpenConns)
	sqlDB.SetConnMaxLifetime(time.Duration(cfg.ConnMaxLifetime) * time.Minute)

	// 测试连接
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err = sqlDB.PingContext(ctx); err != nil {
		return nil, fmt.Errorf("failed to ping database: %w", err)
	}

	return db, nil
}

// GetDB 获取数据库实例
func GetDB() *gorm.DB {
	return db
}

// Close 关闭数据库连接
func Close() error {
	if db != nil {
		sqlDB, err := db.DB()
		if err != nil {
			return err
		}
		return sqlDB.Close()
	}
	return nil
}

// Transaction 执行事务
func Transaction(fn func(tx *gorm.DB) error) error {
	return db.Transaction(fn)
}

// WithContext 返回带 context 的数据库实例
func WithContext(ctx context.Context) *gorm.DB {
	return db.WithContext(ctx)
}

// getLogLevel 获取日志级别
func getLogLevel(logMode bool) logger.LogLevel {
	if logMode {
		return logger.Info
	}
	return logger.Silent
}

// Paginate GORM 分页作用域
func Paginate(page, pageSize int) func(db *gorm.DB) *gorm.DB {
	return func(db *gorm.DB) *gorm.DB {
		if page <= 0 {
			page = 1
		}
		if pageSize <= 0 {
			pageSize = 10
		}
		if pageSize > 100 {
			pageSize = 100
		}
		offset := (page - 1) * pageSize
		return db.Offset(offset).Limit(pageSize)
	}
}

// OrderByCreatedDesc 按创建时间降序
func OrderByCreatedDesc(db *gorm.DB) *gorm.DB {
	return db.Order("created_at DESC")
}

// OrderByUpdatedDesc 按更新时间降序
func OrderByUpdatedDesc(db *gorm.DB) *gorm.DB {
	return db.Order("updated_at DESC")
}
