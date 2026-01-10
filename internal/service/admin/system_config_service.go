// Package admin 管理端服务
package admin

import (
	"context"
	"encoding/json"
	"strconv"

	"github.com/dumeirei/smart-locker-backend/internal/models"
	"github.com/dumeirei/smart-locker-backend/internal/repository"
)

// SystemConfigService 系统配置服务
type SystemConfigService struct {
	configRepo *repository.SystemConfigRepository
}

// NewSystemConfigService 创建系统配置服务
func NewSystemConfigService(configRepo *repository.SystemConfigRepository) *SystemConfigService {
	return &SystemConfigService{configRepo: configRepo}
}

// CreateConfigRequest 创建配置请求
type CreateConfigRequest struct {
	Group       string `json:"group" binding:"required"`
	Key         string `json:"key" binding:"required"`
	Value       string `json:"value" binding:"required"`
	Type        string `json:"type"` // string/number/boolean/json
	Description string `json:"description"`
	IsPublic    bool   `json:"is_public"`
}

// UpdateConfigRequest 更新配置请求
type UpdateConfigRequest struct {
	Value       string  `json:"value"`
	Type        *string `json:"type"`
	Description *string `json:"description"`
	IsPublic    *bool   `json:"is_public"`
}

// ConfigResponse 配置响应
type ConfigResponse struct {
	ID          int64       `json:"id"`
	Group       string      `json:"group"`
	Key         string      `json:"key"`
	Value       interface{} `json:"value"`
	Type        string      `json:"type"`
	Description string      `json:"description"`
	IsPublic    bool        `json:"is_public"`
}

// Create 创建配置
func (s *SystemConfigService) Create(ctx context.Context, req *CreateConfigRequest) (*models.SystemConfig, error) {
	// 检查是否已存在
	exists, err := s.configRepo.ExistsByGroupAndKey(ctx, req.Group, req.Key)
	if err != nil {
		return nil, err
	}
	if exists {
		return nil, ErrConfigAlreadyExists
	}

	configType := req.Type
	if configType == "" {
		configType = models.ConfigTypeString
	}

	config := &models.SystemConfig{
		Group:    req.Group,
		Key:      req.Key,
		Value:    req.Value,
		Type:     configType,
		IsPublic: req.IsPublic,
	}
	if req.Description != "" {
		config.Description = &req.Description
	}

	if err := s.configRepo.Create(ctx, config); err != nil {
		return nil, err
	}

	return config, nil
}

// GetByID 根据 ID 获取配置
func (s *SystemConfigService) GetByID(ctx context.Context, id int64) (*models.SystemConfig, error) {
	return s.configRepo.GetByID(ctx, id)
}

// GetByGroupAndKey 根据分组和键获取配置
func (s *SystemConfigService) GetByGroupAndKey(ctx context.Context, group, key string) (*models.SystemConfig, error) {
	return s.configRepo.GetByGroupAndKey(ctx, group, key)
}

// GetByGroup 获取分组下的所有配置
func (s *SystemConfigService) GetByGroup(ctx context.Context, group string) ([]*models.SystemConfig, error) {
	return s.configRepo.GetByGroup(ctx, group)
}

// Update 更新配置
func (s *SystemConfigService) Update(ctx context.Context, id int64, req *UpdateConfigRequest) (*models.SystemConfig, error) {
	config, err := s.configRepo.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}

	if req.Value != "" {
		config.Value = req.Value
	}
	if req.Type != nil {
		config.Type = *req.Type
	}
	if req.Description != nil {
		config.Description = req.Description
	}
	if req.IsPublic != nil {
		config.IsPublic = *req.IsPublic
	}

	if err := s.configRepo.Update(ctx, config); err != nil {
		return nil, err
	}

	return config, nil
}

// Delete 删除配置
func (s *SystemConfigService) Delete(ctx context.Context, id int64) error {
	return s.configRepo.Delete(ctx, id)
}

// List 获取配置列表
func (s *SystemConfigService) List(ctx context.Context, page, pageSize int, group, keyword string, isPublic *bool) ([]*models.SystemConfig, int64, error) {
	offset := (page - 1) * pageSize
	filters := &repository.SystemConfigListFilters{
		Group:    group,
		Keyword:  keyword,
		IsPublic: isPublic,
	}
	return s.configRepo.List(ctx, offset, pageSize, filters)
}

// GetPublicConfigs 获取所有公开配置
func (s *SystemConfigService) GetPublicConfigs(ctx context.Context) (map[string]map[string]interface{}, error) {
	configs, err := s.configRepo.GetPublicConfigs(ctx)
	if err != nil {
		return nil, err
	}

	result := make(map[string]map[string]interface{})
	for _, config := range configs {
		if result[config.Group] == nil {
			result[config.Group] = make(map[string]interface{})
		}
		result[config.Group][config.Key] = s.parseValue(config)
	}

	return result, nil
}

// GetAllGroups 获取所有配置分组
func (s *SystemConfigService) GetAllGroups(ctx context.Context) ([]string, error) {
	return s.configRepo.GetAllGroups(ctx)
}

// GetString 获取字符串配置值
func (s *SystemConfigService) GetString(ctx context.Context, group, key, defaultValue string) string {
	config, err := s.configRepo.GetByGroupAndKey(ctx, group, key)
	if err != nil {
		return defaultValue
	}
	return config.Value
}

// GetInt 获取整数配置值
func (s *SystemConfigService) GetInt(ctx context.Context, group, key string, defaultValue int) int {
	config, err := s.configRepo.GetByGroupAndKey(ctx, group, key)
	if err != nil {
		return defaultValue
	}
	val, err := strconv.Atoi(config.Value)
	if err != nil {
		return defaultValue
	}
	return val
}

// GetFloat 获取浮点数配置值
func (s *SystemConfigService) GetFloat(ctx context.Context, group, key string, defaultValue float64) float64 {
	config, err := s.configRepo.GetByGroupAndKey(ctx, group, key)
	if err != nil {
		return defaultValue
	}
	val, err := strconv.ParseFloat(config.Value, 64)
	if err != nil {
		return defaultValue
	}
	return val
}

// GetBool 获取布尔配置值
func (s *SystemConfigService) GetBool(ctx context.Context, group, key string, defaultValue bool) bool {
	config, err := s.configRepo.GetByGroupAndKey(ctx, group, key)
	if err != nil {
		return defaultValue
	}
	val, err := strconv.ParseBool(config.Value)
	if err != nil {
		return defaultValue
	}
	return val
}

// GetJSON 获取 JSON 配置值
func (s *SystemConfigService) GetJSON(ctx context.Context, group, key string, target interface{}) error {
	config, err := s.configRepo.GetByGroupAndKey(ctx, group, key)
	if err != nil {
		return err
	}
	return json.Unmarshal([]byte(config.Value), target)
}

// parseValue 解析配置值
func (s *SystemConfigService) parseValue(config *models.SystemConfig) interface{} {
	switch config.Type {
	case models.ConfigTypeNumber:
		if val, err := strconv.ParseFloat(config.Value, 64); err == nil {
			return val
		}
	case models.ConfigTypeBoolean:
		if val, err := strconv.ParseBool(config.Value); err == nil {
			return val
		}
	case models.ConfigTypeJSON:
		var val interface{}
		if err := json.Unmarshal([]byte(config.Value), &val); err == nil {
			return val
		}
	}
	return config.Value
}

// BatchUpdateRequest 批量更新请求
type BatchUpdateRequest struct {
	Configs []struct {
		Group string `json:"group" binding:"required"`
		Key   string `json:"key" binding:"required"`
		Value string `json:"value" binding:"required"`
	} `json:"configs" binding:"required"`
}

// BatchUpdate 批量更新配置
func (s *SystemConfigService) BatchUpdate(ctx context.Context, req *BatchUpdateRequest) error {
	configs := make([]*models.SystemConfig, len(req.Configs))
	for i, c := range req.Configs {
		configs[i] = &models.SystemConfig{
			Group: c.Group,
			Key:   c.Key,
			Value: c.Value,
		}
	}
	return s.configRepo.BatchUpsert(ctx, configs)
}

// 错误定义
var (
	ErrConfigAlreadyExists = &ServiceError{Code: "CONFIG_EXISTS", Message: "配置已存在"}
)

// ServiceError 服务错误
type ServiceError struct {
	Code    string
	Message string
}

func (e *ServiceError) Error() string {
	return e.Message
}
