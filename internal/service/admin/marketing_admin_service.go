// Package admin 提供管理端服务
package admin

import (
	"context"
	"encoding/json"
	"time"

	"gorm.io/gorm"

	"github.com/dumeirei/smart-locker-backend/internal/models"
	"github.com/dumeirei/smart-locker-backend/internal/repository"
)

// MarketingAdminService 营销管理服务
type MarketingAdminService struct {
	db           *gorm.DB
	couponRepo   *repository.CouponRepository
	campaignRepo *repository.CampaignRepository
}

// NewMarketingAdminService 创建营销管理服务
func NewMarketingAdminService(db *gorm.DB, couponRepo *repository.CouponRepository, campaignRepo *repository.CampaignRepository) *MarketingAdminService {
	return &MarketingAdminService{
		db:           db,
		couponRepo:   couponRepo,
		campaignRepo: campaignRepo,
	}
}

// AdminCouponListRequest 管理端优惠券列表请求
type AdminCouponListRequest struct {
	Page           int
	PageSize       int
	Status         *int8
	Type           string
	ApplicableType string
	Keyword        string
}

// AdminCouponListResponse 管理端优惠券列表响应
type AdminCouponListResponse struct {
	List  []*AdminCouponItem `json:"list"`
	Total int64              `json:"total"`
}

// AdminCouponItem 管理端优惠券项
type AdminCouponItem struct {
	ID              int64      `json:"id"`
	Name            string     `json:"name"`
	Type            string     `json:"type"`
	TypeText        string     `json:"type_text"`
	Value           float64    `json:"value"`
	MinAmount       float64    `json:"min_amount"`
	MaxDiscount     *float64   `json:"max_discount,omitempty"`
	ApplicableScope string     `json:"applicable_scope"`
	ApplicableIDs   []int64    `json:"applicable_ids,omitempty"`
	StartTime       time.Time  `json:"start_time"`
	EndTime         time.Time  `json:"end_time"`
	ValidDays       *int       `json:"valid_days,omitempty"`
	TotalCount      int        `json:"total_count"`
	ReceivedCount   int        `json:"received_count"`
	UsedCount       int        `json:"used_count"`
	PerUserLimit    int        `json:"per_user_limit"`
	Description     *string    `json:"description,omitempty"`
	Status          int8       `json:"status"`
	StatusText      string     `json:"status_text"`
	CreatedAt       time.Time  `json:"created_at"`
}

// GetCouponList 获取优惠券列表（管理端）
func (s *MarketingAdminService) GetCouponList(ctx context.Context, req *AdminCouponListRequest) (*AdminCouponListResponse, error) {
	offset := (req.Page - 1) * req.PageSize

	params := repository.CouponListParams{
		Offset:         offset,
		Limit:          req.PageSize,
		Status:         req.Status,
		Type:           req.Type,
		ApplicableType: req.ApplicableType,
		Keyword:        req.Keyword,
	}

	coupons, total, err := s.couponRepo.List(ctx, params)
	if err != nil {
		return nil, err
	}

	list := make([]*AdminCouponItem, 0, len(coupons))
	for _, c := range coupons {
		item := s.buildAdminCouponItem(c)
		list = append(list, item)
	}

	return &AdminCouponListResponse{
		List:  list,
		Total: total,
	}, nil
}

// GetCouponDetail 获取优惠券详情（管理端）
func (s *MarketingAdminService) GetCouponDetail(ctx context.Context, couponID int64) (*AdminCouponItem, error) {
	coupon, err := s.couponRepo.GetByID(ctx, couponID)
	if err != nil {
		return nil, err
	}
	return s.buildAdminCouponItem(coupon), nil
}

// CreateCouponRequest 创建优惠券请求
type CreateCouponRequest struct {
	Name            string   `json:"name" binding:"required"`
	Type            string   `json:"type" binding:"required,oneof=fixed percent"`
	Value           float64  `json:"value" binding:"required,gt=0"`
	MinAmount       float64  `json:"min_amount"`
	MaxDiscount     *float64 `json:"max_discount"`
	TotalCount      int      `json:"total_count" binding:"required,gt=0"`
	PerUserLimit    int      `json:"per_user_limit" binding:"required,gt=0"`
	ApplicableScope string   `json:"applicable_scope" binding:"required,oneof=all category product"`
	ApplicableIDs   []int64  `json:"applicable_ids,omitempty"`
	StartTime       string   `json:"start_time" binding:"required"`
	EndTime         string   `json:"end_time" binding:"required"`
	ValidDays       *int     `json:"valid_days"`
	Description     *string  `json:"description"`
}

// CreateCoupon 创建优惠券
func (s *MarketingAdminService) CreateCoupon(ctx context.Context, req *CreateCouponRequest) (*models.Coupon, error) {
	startTime, err := time.ParseInLocation("2006-01-02 15:04:05", req.StartTime, time.Local)
	if err != nil {
		return nil, err
	}
	endTime, err := time.ParseInLocation("2006-01-02 15:04:05", req.EndTime, time.Local)
	if err != nil {
		return nil, err
	}

	coupon := &models.Coupon{
		Name:            req.Name,
		Type:            req.Type,
		Value:           req.Value,
		MinAmount:       req.MinAmount,
		MaxDiscount:     req.MaxDiscount,
		TotalCount:      req.TotalCount,
		PerUserLimit:    req.PerUserLimit,
		ApplicableScope: req.ApplicableScope,
		StartTime:       startTime,
		EndTime:         endTime,
		ValidDays:       req.ValidDays,
		Description:     req.Description,
		Status:          models.CouponStatusActive,
	}

	// 处理适用范围
	if len(req.ApplicableIDs) > 0 {
		applicableIDsMap := make(models.JSON)
		applicableIDsMap["ids"] = req.ApplicableIDs
		coupon.ApplicableIDs = applicableIDsMap
	}

	if err := s.couponRepo.Create(ctx, coupon); err != nil {
		return nil, err
	}

	return coupon, nil
}

// UpdateCouponRequest 更新优惠券请求
type UpdateCouponRequest struct {
	Name            *string  `json:"name"`
	Type            *string  `json:"type"`
	Value           *float64 `json:"value"`
	MinAmount       *float64 `json:"min_amount"`
	MaxDiscount     *float64 `json:"max_discount"`
	TotalCount      *int     `json:"total_count"`
	PerUserLimit    *int     `json:"per_user_limit"`
	ApplicableScope *string  `json:"applicable_scope"`
	ApplicableIDs   []int64  `json:"applicable_ids"`
	StartTime       *string  `json:"start_time"`
	EndTime         *string  `json:"end_time"`
	ValidDays       *int     `json:"valid_days"`
	Description     *string  `json:"description"`
	Status          *int8    `json:"status"`
}

// UpdateCoupon 更新优惠券
func (s *MarketingAdminService) UpdateCoupon(ctx context.Context, couponID int64, req *UpdateCouponRequest) error {
	fields := make(map[string]interface{})

	if req.Name != nil {
		fields["name"] = *req.Name
	}
	if req.Type != nil {
		fields["type"] = *req.Type
	}
	if req.Value != nil {
		fields["value"] = *req.Value
	}
	if req.MinAmount != nil {
		fields["min_amount"] = *req.MinAmount
	}
	if req.MaxDiscount != nil {
		fields["max_discount"] = *req.MaxDiscount
	}
	if req.TotalCount != nil {
		fields["total_count"] = *req.TotalCount
	}
	if req.PerUserLimit != nil {
		fields["per_user_limit"] = *req.PerUserLimit
	}
	if req.ApplicableScope != nil {
		fields["applicable_scope"] = *req.ApplicableScope
	}
	if len(req.ApplicableIDs) > 0 {
		ids, _ := json.Marshal(req.ApplicableIDs)
		fields["applicable_ids"] = ids
	}
	if req.StartTime != nil {
		startTime, err := time.ParseInLocation("2006-01-02 15:04:05", *req.StartTime, time.Local)
		if err != nil {
			return err
		}
		fields["start_time"] = startTime
	}
	if req.EndTime != nil {
		endTime, err := time.ParseInLocation("2006-01-02 15:04:05", *req.EndTime, time.Local)
		if err != nil {
			return err
		}
		fields["end_time"] = endTime
	}
	if req.ValidDays != nil {
		fields["valid_days"] = *req.ValidDays
	}
	if req.Description != nil {
		fields["description"] = *req.Description
	}
	if req.Status != nil {
		fields["status"] = *req.Status
	}

	if len(fields) == 0 {
		return nil
	}

	return s.couponRepo.UpdateFields(ctx, couponID, fields)
}

// UpdateCouponStatus 更新优惠券状态
func (s *MarketingAdminService) UpdateCouponStatus(ctx context.Context, couponID int64, status int8) error {
	return s.couponRepo.UpdateFields(ctx, couponID, map[string]interface{}{
		"status": status,
	})
}

// DeleteCoupon 删除优惠券
func (s *MarketingAdminService) DeleteCoupon(ctx context.Context, couponID int64) error {
	return s.couponRepo.Delete(ctx, couponID)
}

// AdminCampaignListRequest 管理端活动列表请求
type AdminCampaignListRequest struct {
	Page     int
	PageSize int
	Status   *int8
	Type     string
	Keyword  string
}

// AdminCampaignListResponse 管理端活动列表响应
type AdminCampaignListResponse struct {
	List  []*AdminCampaignItem `json:"list"`
	Total int64                `json:"total"`
}

// AdminCampaignItem 管理端活动项
type AdminCampaignItem struct {
	ID          int64           `json:"id"`
	Name        string          `json:"name"`
	Type        string          `json:"type"`
	TypeText    string          `json:"type_text"`
	Description *string         `json:"description,omitempty"`
	Image       *string         `json:"image,omitempty"`
	Rules       json.RawMessage `json:"rules,omitempty"`
	StartTime   time.Time       `json:"start_time"`
	EndTime     time.Time       `json:"end_time"`
	Status      int8            `json:"status"`
	StatusText  string          `json:"status_text"`
	CreatedAt   time.Time       `json:"created_at"`
	UpdatedAt   time.Time       `json:"updated_at"`
}

// GetCampaignList 获取活动列表（管理端）
func (s *MarketingAdminService) GetCampaignList(ctx context.Context, req *AdminCampaignListRequest) (*AdminCampaignListResponse, error) {
	offset := (req.Page - 1) * req.PageSize

	params := repository.CampaignListParams{
		Offset:  offset,
		Limit:   req.PageSize,
		Status:  req.Status,
		Type:    req.Type,
		Keyword: req.Keyword,
	}

	campaigns, total, err := s.campaignRepo.List(ctx, params)
	if err != nil {
		return nil, err
	}

	list := make([]*AdminCampaignItem, 0, len(campaigns))
	for _, c := range campaigns {
		item := s.buildAdminCampaignItem(c)
		list = append(list, item)
	}

	return &AdminCampaignListResponse{
		List:  list,
		Total: total,
	}, nil
}

// GetCampaignDetail 获取活动详情（管理端）
func (s *MarketingAdminService) GetCampaignDetail(ctx context.Context, campaignID int64) (*AdminCampaignItem, error) {
	campaign, err := s.campaignRepo.GetByID(ctx, campaignID)
	if err != nil {
		return nil, err
	}
	return s.buildAdminCampaignItem(campaign), nil
}

// CreateCampaignRequest 创建活动请求
type CreateCampaignRequest struct {
	Name        string          `json:"name" binding:"required"`
	Type        string          `json:"type" binding:"required"`
	Description *string         `json:"description"`
	Image       *string         `json:"image"`
	Rules       json.RawMessage `json:"rules"`
	StartTime   string          `json:"start_time" binding:"required"`
	EndTime     string          `json:"end_time" binding:"required"`
}

// CreateCampaign 创建活动
func (s *MarketingAdminService) CreateCampaign(ctx context.Context, req *CreateCampaignRequest) (*models.Campaign, error) {
	startTime, err := time.ParseInLocation("2006-01-02 15:04:05", req.StartTime, time.Local)
	if err != nil {
		return nil, err
	}
	endTime, err := time.ParseInLocation("2006-01-02 15:04:05", req.EndTime, time.Local)
	if err != nil {
		return nil, err
	}

	campaign := &models.Campaign{
		Name:        req.Name,
		Type:        req.Type,
		Description: req.Description,
		Image:       req.Image,
		StartTime:   startTime,
		EndTime:     endTime,
		Status:      models.CampaignStatusActive,
	}

	// 处理规则
	if req.Rules != nil {
		var rulesMap models.JSON
		if err := json.Unmarshal(req.Rules, &rulesMap); err == nil {
			campaign.Rules = rulesMap
		}
	}

	if err := s.campaignRepo.Create(ctx, campaign); err != nil {
		return nil, err
	}

	return campaign, nil
}

// UpdateCampaignRequest 更新活动请求
type UpdateCampaignRequest struct {
	Name        *string         `json:"name"`
	Type        *string         `json:"type"`
	Description *string         `json:"description"`
	Image       *string         `json:"image"`
	Rules       json.RawMessage `json:"rules"`
	StartTime   *string         `json:"start_time"`
	EndTime     *string         `json:"end_time"`
	Status      *int8           `json:"status"`
}

// UpdateCampaign 更新活动
func (s *MarketingAdminService) UpdateCampaign(ctx context.Context, campaignID int64, req *UpdateCampaignRequest) error {
	fields := make(map[string]interface{})

	if req.Name != nil {
		fields["name"] = *req.Name
	}
	if req.Type != nil {
		fields["type"] = *req.Type
	}
	if req.Description != nil {
		fields["description"] = *req.Description
	}
	if req.Image != nil {
		fields["image"] = *req.Image
	}
	if req.Rules != nil {
		fields["rules"] = req.Rules
	}
	if req.StartTime != nil {
		startTime, err := time.ParseInLocation("2006-01-02 15:04:05", *req.StartTime, time.Local)
		if err != nil {
			return err
		}
		fields["start_time"] = startTime
	}
	if req.EndTime != nil {
		endTime, err := time.ParseInLocation("2006-01-02 15:04:05", *req.EndTime, time.Local)
		if err != nil {
			return err
		}
		fields["end_time"] = endTime
	}
	if req.Status != nil {
		fields["status"] = *req.Status
	}

	if len(fields) == 0 {
		return nil
	}

	return s.campaignRepo.UpdateFields(ctx, campaignID, fields)
}

// UpdateCampaignStatus 更新活动状态
func (s *MarketingAdminService) UpdateCampaignStatus(ctx context.Context, campaignID int64, status int8) error {
	return s.campaignRepo.UpdateStatus(ctx, campaignID, status)
}

// DeleteCampaign 删除活动
func (s *MarketingAdminService) DeleteCampaign(ctx context.Context, campaignID int64) error {
	return s.campaignRepo.Delete(ctx, campaignID)
}

// buildAdminCouponItem 构建管理端优惠券项
func (s *MarketingAdminService) buildAdminCouponItem(c *models.Coupon) *AdminCouponItem {
	item := &AdminCouponItem{
		ID:              c.ID,
		Name:            c.Name,
		Type:            c.Type,
		Value:           c.Value,
		MinAmount:       c.MinAmount,
		MaxDiscount:     c.MaxDiscount,
		ApplicableScope: c.ApplicableScope,
		StartTime:       c.StartTime,
		EndTime:         c.EndTime,
		ValidDays:       c.ValidDays,
		TotalCount:      c.TotalCount,
		ReceivedCount:   c.ReceivedCount,
		UsedCount:       c.UsedCount,
		PerUserLimit:    c.PerUserLimit,
		Description:     c.Description,
		Status:          c.Status,
		CreatedAt:       c.CreatedAt,
	}

	// 设置类型文本
	switch c.Type {
	case models.CouponTypeFixed:
		item.TypeText = "固定金额"
	case models.CouponTypePercent:
		item.TypeText = "百分比折扣"
	}

	// 解析适用ID
	if c.ApplicableIDs != nil {
		if ids, ok := c.ApplicableIDs["ids"]; ok {
			if idsSlice, ok := ids.([]interface{}); ok {
				for _, id := range idsSlice {
					if idFloat, ok := id.(float64); ok {
						item.ApplicableIDs = append(item.ApplicableIDs, int64(idFloat))
					}
				}
			}
		}
	}

	// 设置状态文本
	now := time.Now()
	if c.Status == models.CouponStatusDisabled {
		item.StatusText = "已禁用"
	} else if now.Before(c.StartTime) {
		item.StatusText = "未开始"
	} else if now.After(c.EndTime) {
		item.StatusText = "已结束"
	} else if c.ReceivedCount >= c.TotalCount {
		item.StatusText = "已领完"
	} else {
		item.StatusText = "进行中"
	}

	return item
}

// buildAdminCampaignItem 构建管理端活动项
func (s *MarketingAdminService) buildAdminCampaignItem(c *models.Campaign) *AdminCampaignItem {
	item := &AdminCampaignItem{
		ID:          c.ID,
		Name:        c.Name,
		Type:        c.Type,
		Description: c.Description,
		Image:       c.Image,
		StartTime:   c.StartTime,
		EndTime:     c.EndTime,
		Status:      c.Status,
		CreatedAt:   c.CreatedAt,
		UpdatedAt:   c.UpdatedAt,
	}

	// 设置类型文本
	switch c.Type {
	case models.CampaignTypeDiscount:
		item.TypeText = "满减"
	case models.CampaignTypeGift:
		item.TypeText = "满赠"
	case models.CampaignTypeFlashSale:
		item.TypeText = "秒杀"
	case models.CampaignTypeGroupBuy:
		item.TypeText = "团购"
	default:
		item.TypeText = "活动"
	}

	// 处理规则
	if c.Rules != nil {
		rulesBytes, _ := json.Marshal(c.Rules)
		item.Rules = rulesBytes
	}

	// 设置状态文本
	now := time.Now()
	if c.Status == models.CampaignStatusDisabled {
		item.StatusText = "已禁用"
	} else if now.Before(c.StartTime) {
		item.StatusText = "未开始"
	} else if now.After(c.EndTime) {
		item.StatusText = "已结束"
	} else {
		item.StatusText = "进行中"
	}

	return item
}
