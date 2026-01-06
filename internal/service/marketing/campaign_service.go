// Package marketing 提供营销相关服务
package marketing

import (
	"context"
	"encoding/json"
	"time"

	"github.com/dumeirei/smart-locker-backend/internal/models"
	"github.com/dumeirei/smart-locker-backend/internal/repository"
)

// CampaignService 活动服务
type CampaignService struct {
	campaignRepo *repository.CampaignRepository
}

// NewCampaignService 创建活动服务
func NewCampaignService(campaignRepo *repository.CampaignRepository) *CampaignService {
	return &CampaignService{
		campaignRepo: campaignRepo,
	}
}

// CampaignListRequest 活动列表请求
type CampaignListRequest struct {
	Page     int
	PageSize int
}

// CampaignListResponse 活动列表响应
type CampaignListResponse struct {
	List  []*CampaignItem `json:"list"`
	Total int64           `json:"total"`
}

// CampaignItem 活动项
type CampaignItem struct {
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
	IsActive    bool            `json:"is_active"`
}

// GetCampaignList 获取活动列表（用户端）
func (s *CampaignService) GetCampaignList(ctx context.Context, req *CampaignListRequest) (*CampaignListResponse, error) {
	offset := (req.Page - 1) * req.PageSize

	campaigns, total, err := s.campaignRepo.ListActive(ctx, offset, req.PageSize)
	if err != nil {
		return nil, err
	}

	now := time.Now()
	list := make([]*CampaignItem, 0, len(campaigns))
	for _, c := range campaigns {
		item := s.buildCampaignItem(c, now)
		list = append(list, item)
	}

	return &CampaignListResponse{
		List:  list,
		Total: total,
	}, nil
}

// GetCampaignDetail 获取活动详情
func (s *CampaignService) GetCampaignDetail(ctx context.Context, campaignID int64) (*CampaignItem, error) {
	campaign, err := s.campaignRepo.GetByID(ctx, campaignID)
	if err != nil {
		return nil, err
	}

	now := time.Now()
	return s.buildCampaignItem(campaign, now), nil
}

// GetCampaignsByType 根据类型获取活动列表
func (s *CampaignService) GetCampaignsByType(ctx context.Context, campaignType string) ([]*CampaignItem, error) {
	campaigns, err := s.campaignRepo.ListByType(ctx, campaignType)
	if err != nil {
		return nil, err
	}

	now := time.Now()
	list := make([]*CampaignItem, 0, len(campaigns))
	for _, c := range campaigns {
		item := s.buildCampaignItem(c, now)
		list = append(list, item)
	}

	return list, nil
}

// DiscountRule 满减规则
type DiscountRule struct {
	MinAmount float64 `json:"min_amount"` // 满足金额
	Discount  float64 `json:"discount"`   // 优惠金额
}

// CalculateDiscountCampaign 计算满减活动优惠
func (s *CampaignService) CalculateDiscountCampaign(ctx context.Context, orderAmount float64) (float64, *models.Campaign, error) {
	campaign, err := s.campaignRepo.GetActiveByType(ctx, models.CampaignTypeDiscount)
	if err != nil {
		// 无活动时返回0优惠
		return 0, nil, nil
	}

	// 解析规则
	var rules []DiscountRule
	if campaign.Rules != nil {
		// models.JSON 是 map 结构；兼容 {"rules":[...]} 或直接存数组（若未来 JSON 类型扩展）。
		if rawRules, ok := campaign.Rules["rules"]; ok {
			b, err := json.Marshal(rawRules)
			if err != nil {
				return 0, nil, err
			}
			if err := json.Unmarshal(b, &rules); err != nil {
				return 0, nil, err
			}
		} else if err := campaign.Rules.Unmarshal(&rules); err != nil {
			return 0, nil, err
		}
	}

	// 计算优惠金额（取最大满减档位）
	var maxDiscount float64
	for _, rule := range rules {
		if orderAmount >= rule.MinAmount && rule.Discount > maxDiscount {
			maxDiscount = rule.Discount
		}
	}

	if maxDiscount > 0 {
		return maxDiscount, campaign, nil
	}
	return 0, nil, nil
}

// buildCampaignItem 构建活动项
func (s *CampaignService) buildCampaignItem(c *models.Campaign, now time.Time) *CampaignItem {
	item := &CampaignItem{
		ID:          c.ID,
		Name:        c.Name,
		Type:        c.Type,
		Description: c.Description,
		Image:       c.Image,
		StartTime:   c.StartTime,
		EndTime:     c.EndTime,
		Status:      c.Status,
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

	// 设置状态文本和活动状态
	if c.Status == models.CampaignStatusDisabled {
		item.StatusText = "已禁用"
		item.IsActive = false
	} else if now.Before(c.StartTime) {
		item.StatusText = "未开始"
		item.IsActive = false
	} else if now.After(c.EndTime) {
		item.StatusText = "已结束"
		item.IsActive = false
	} else {
		item.StatusText = "进行中"
		item.IsActive = true
	}

	// 处理规则（转换为 JSON）
	if c.Rules != nil {
		rulesBytes, _ := json.Marshal(c.Rules)
		item.Rules = rulesBytes
	}

	return item
}
