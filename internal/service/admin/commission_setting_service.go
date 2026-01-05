// Package admin 提供管理端相关服务
package admin

import (
	"context"
	"errors"

	"gorm.io/gorm"

	"github.com/dumeirei/smart-locker-backend/internal/models"
)

// CommissionSettingService 佣金设置服务
type CommissionSettingService struct {
	db *gorm.DB
}

// NewCommissionSettingService 创建佣金设置服务
func NewCommissionSettingService(db *gorm.DB) *CommissionSettingService {
	return &CommissionSettingService{db: db}
}

// CommissionConfig 佣金配置
type CommissionConfig struct {
	DirectRate   float64 `json:"direct_rate"`   // 直推佣金比例
	IndirectRate float64 `json:"indirect_rate"` // 间推佣金比例
	MinWithdraw  float64 `json:"min_withdraw"`  // 最低提现金额
	WithdrawFee  float64 `json:"withdraw_fee"`  // 提现手续费比例
	SettleDelay  int     `json:"settle_delay"`  // 结算延迟天数
}

// DefaultCommissionConfig 默认佣金配置
var DefaultCommissionConfig = CommissionConfig{
	DirectRate:   0.10,  // 10%
	IndirectRate: 0.05,  // 5%
	MinWithdraw:  10.0,  // 10元
	WithdrawFee:  0.006, // 0.6%
	SettleDelay:  7,     // 7天
}

// GetConfig 获取佣金配置
func (s *CommissionSettingService) GetConfig(ctx context.Context) (*CommissionConfig, error) {
	var setting models.CommissionSetting
	err := s.db.WithContext(ctx).Where("is_active = ?", true).First(&setting).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			// 返回默认配置
			return &DefaultCommissionConfig, nil
		}
		return nil, err
	}

	return &CommissionConfig{
		DirectRate:   setting.DirectRate,
		IndirectRate: setting.IndirectRate,
		MinWithdraw:  setting.MinWithdraw,
		WithdrawFee:  setting.WithdrawFee,
		SettleDelay:  setting.SettleDelay,
	}, nil
}

// UpdateConfig 更新佣金配置
func (s *CommissionSettingService) UpdateConfig(ctx context.Context, config *CommissionConfig) error {
	// 验证配置
	if config.DirectRate < 0 || config.DirectRate > 1 {
		return errors.New("直推佣金比例必须在0-100%之间")
	}
	if config.IndirectRate < 0 || config.IndirectRate > 1 {
		return errors.New("间推佣金比例必须在0-100%之间")
	}
	if config.DirectRate+config.IndirectRate > 0.5 {
		return errors.New("佣金比例总和不能超过50%")
	}
	if config.MinWithdraw < 0 {
		return errors.New("最低提现金额不能为负数")
	}
	if config.WithdrawFee < 0 || config.WithdrawFee > 1 {
		return errors.New("提现手续费比例必须在0-100%之间")
	}
	if config.SettleDelay < 0 {
		return errors.New("结算延迟天数不能为负数")
	}

	return s.db.Transaction(func(tx *gorm.DB) error {
		// 先将所有配置设为非活跃
		if err := tx.Model(&models.CommissionSetting{}).
			Where("is_active = ?", true).
			Update("is_active", false).Error; err != nil {
			return err
		}

		// 创建新配置
		setting := &models.CommissionSetting{
			DirectRate:   config.DirectRate,
			IndirectRate: config.IndirectRate,
			MinWithdraw:  config.MinWithdraw,
			WithdrawFee:  config.WithdrawFee,
			SettleDelay:  config.SettleDelay,
			IsActive:     true,
		}

		return tx.Create(setting).Error
	})
}

// GetConfigHistory 获取配置历史记录
func (s *CommissionSettingService) GetConfigHistory(ctx context.Context, offset, limit int) ([]*models.CommissionSetting, int64, error) {
	var settings []*models.CommissionSetting
	var total int64

	if err := s.db.WithContext(ctx).Model(&models.CommissionSetting{}).Count(&total).Error; err != nil {
		return nil, 0, err
	}

	if err := s.db.WithContext(ctx).Order("id DESC").Offset(offset).Limit(limit).Find(&settings).Error; err != nil {
		return nil, 0, err
	}

	return settings, total, nil
}

// InitDefaultConfig 初始化默认配置
func (s *CommissionSettingService) InitDefaultConfig(ctx context.Context) error {
	var count int64
	if err := s.db.WithContext(ctx).Model(&models.CommissionSetting{}).Count(&count).Error; err != nil {
		return err
	}

	if count > 0 {
		return nil // 已有配置
	}

	setting := &models.CommissionSetting{
		DirectRate:   DefaultCommissionConfig.DirectRate,
		IndirectRate: DefaultCommissionConfig.IndirectRate,
		MinWithdraw:  DefaultCommissionConfig.MinWithdraw,
		WithdrawFee:  DefaultCommissionConfig.WithdrawFee,
		SettleDelay:  DefaultCommissionConfig.SettleDelay,
		IsActive:     true,
	}

	return s.db.WithContext(ctx).Create(setting).Error
}
