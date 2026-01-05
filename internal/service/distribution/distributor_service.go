// Package distribution 分销服务
package distribution

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"strings"
	"time"

	"gorm.io/gorm"

	"github.com/dumeirei/smart-locker-backend/internal/models"
	"github.com/dumeirei/smart-locker-backend/internal/repository"
)

// DistributorService 分销商服务
type DistributorService struct {
	distributorRepo *repository.DistributorRepository
	userRepo        *repository.UserRepository
	db              *gorm.DB
}

// NewDistributorService 创建分销商服务
func NewDistributorService(
	distributorRepo *repository.DistributorRepository,
	userRepo *repository.UserRepository,
	db *gorm.DB,
) *DistributorService {
	return &DistributorService{
		distributorRepo: distributorRepo,
		userRepo:        userRepo,
		db:              db,
	}
}

// ApplyRequest 申请成为分销商请求
type ApplyRequest struct {
	UserID     int64   `json:"user_id"`
	InviteCode *string `json:"invite_code,omitempty"` // 上级邀请码
}

// ApplyResponse 申请成为分销商响应
type ApplyResponse struct {
	Distributor *models.Distributor `json:"distributor"`
	Message     string              `json:"message"`
}

// Apply 申请成为分销商
func (s *DistributorService) Apply(ctx context.Context, req *ApplyRequest) (*ApplyResponse, error) {
	// 检查用户是否存在
	user, err := s.userRepo.GetByID(ctx, req.UserID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, errors.New("用户不存在")
		}
		return nil, err
	}

	// 检查是否已经是分销商
	exists, err := s.distributorRepo.ExistsByUserID(ctx, req.UserID)
	if err != nil {
		return nil, err
	}
	if exists {
		return nil, errors.New("您已经是分销商了")
	}

	// 查找上级分销商
	var parentID *int64
	if req.InviteCode != nil && *req.InviteCode != "" {
		parent, err := s.distributorRepo.GetByInviteCode(ctx, *req.InviteCode)
		if err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return nil, errors.New("邀请码无效")
			}
			return nil, err
		}
		// 检查上级是否已审核通过
		if parent.Status != models.DistributorStatusApproved {
			return nil, errors.New("邀请人尚未通过审核")
		}
		// 检查是否形成循环（不能邀请自己的上级）
		if parent.UserID == req.UserID {
			return nil, errors.New("不能填写自己的邀请码")
		}
		parentID = &parent.ID
	} else if user.ReferrerID != nil {
		// 如果用户注册时有推荐人，尝试查找推荐人是否是分销商
		referrerDistributor, err := s.distributorRepo.GetByUserID(ctx, *user.ReferrerID)
		if err == nil && referrerDistributor.Status == models.DistributorStatusApproved {
			parentID = &referrerDistributor.ID
		}
	}

	// 生成唯一邀请码
	inviteCode, err := s.generateInviteCode(ctx)
	if err != nil {
		return nil, err
	}

	// 确定层级
	level := models.DistributorLevelDirect
	if parentID != nil {
		parent, _ := s.distributorRepo.GetByID(ctx, *parentID)
		if parent != nil && parent.Level >= models.DistributorLevelIndirect {
			// 只支持2级分销
			level = models.DistributorLevelIndirect
		}
	}

	// 创建分销商
	distributor := &models.Distributor{
		UserID:     req.UserID,
		ParentID:   parentID,
		Level:      level,
		InviteCode: inviteCode,
		Status:     models.DistributorStatusPending,
	}

	if err := s.distributorRepo.Create(ctx, distributor); err != nil {
		return nil, err
	}

	return &ApplyResponse{
		Distributor: distributor,
		Message:     "申请已提交，请等待审核",
	}, nil
}

// generateInviteCode 生成唯一邀请码
func (s *DistributorService) generateInviteCode(ctx context.Context) (string, error) {
	for i := 0; i < 10; i++ {
		code := s.randomCode(8)
		exists, err := s.distributorRepo.ExistsByInviteCode(ctx, code)
		if err != nil {
			return "", err
		}
		if !exists {
			return code, nil
		}
	}
	return "", errors.New("生成邀请码失败，请重试")
}

// randomCode 生成随机码
func (s *DistributorService) randomCode(length int) string {
	bytes := make([]byte, length/2+1)
	rand.Read(bytes)
	code := strings.ToUpper(hex.EncodeToString(bytes))
	return code[:length]
}

// GetByUserID 根据用户 ID 获取分销商信息
func (s *DistributorService) GetByUserID(ctx context.Context, userID int64) (*models.Distributor, error) {
	distributor, err := s.distributorRepo.GetByUserID(ctx, userID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, errors.New("您还不是分销商")
		}
		return nil, err
	}
	return distributor, nil
}

// GetByID 根据 ID 获取分销商信息
func (s *DistributorService) GetByID(ctx context.Context, id int64) (*models.Distributor, error) {
	return s.distributorRepo.GetByIDWithUser(ctx, id)
}

// GetByInviteCode 根据邀请码获取分销商信息
func (s *DistributorService) GetByInviteCode(ctx context.Context, inviteCode string) (*models.Distributor, error) {
	return s.distributorRepo.GetByInviteCodeWithUser(ctx, inviteCode)
}

// TeamStats 团队统计
type TeamStats struct {
	TeamCount       int     `json:"team_count"`         // 团队总人数
	DirectCount     int     `json:"direct_count"`       // 直推人数
	IndirectCount   int     `json:"indirect_count"`     // 间推人数
	TotalCommission float64 `json:"total_commission"`   // 累计佣金
	MonthCommission float64 `json:"month_commission"`   // 本月佣金
}

// GetTeamStats 获取团队统计
func (s *DistributorService) GetTeamStats(ctx context.Context, distributorID int64) (*TeamStats, error) {
	distributor, err := s.distributorRepo.GetByID(ctx, distributorID)
	if err != nil {
		return nil, err
	}

	// 计算间推人数 = 团队总人数 - 直推人数
	indirectCount := distributor.TeamCount - distributor.DirectCount
	if indirectCount < 0 {
		indirectCount = 0
	}

	return &TeamStats{
		TeamCount:       distributor.TeamCount,
		DirectCount:     distributor.DirectCount,
		IndirectCount:   indirectCount,
		TotalCommission: distributor.TotalCommission,
	}, nil
}

// GetTeamMembers 获取团队成员列表
func (s *DistributorService) GetTeamMembers(ctx context.Context, distributorID int64, offset, limit int, memberType string) ([]*models.Distributor, int64, error) {
	switch memberType {
	case "direct":
		return s.distributorRepo.GetDirectMembers(ctx, distributorID, offset, limit)
	case "all":
		return s.distributorRepo.GetTeamMembers(ctx, distributorID, offset, limit)
	default:
		return s.distributorRepo.GetDirectMembers(ctx, distributorID, offset, limit)
	}
}

// GetDirectMembers 获取直推成员列表
func (s *DistributorService) GetDirectMembers(ctx context.Context, distributorID int64, offset, limit int) ([]*models.Distributor, int64, error) {
	return s.distributorRepo.GetDirectMembers(ctx, distributorID, offset, limit)
}

// DashboardData 仪表盘数据
type DashboardData struct {
	TotalCommission     float64 `json:"total_commission"`      // 累计佣金
	AvailableCommission float64 `json:"available_commission"`  // 可提现佣金
	FrozenCommission    float64 `json:"frozen_commission"`     // 冻结佣金
	WithdrawnCommission float64 `json:"withdrawn_commission"`  // 已提现佣金
	TeamCount           int     `json:"team_count"`            // 团队人数
	DirectCount         int     `json:"direct_count"`          // 直推人数
	TodayCommission     float64 `json:"today_commission"`      // 今日佣金
	MonthCommission     float64 `json:"month_commission"`      // 本月佣金
	InviteCode          string  `json:"invite_code"`           // 邀请码
	InviteLink          string  `json:"invite_link"`           // 邀请链接
	Level               int     `json:"level"`                 // 分销层级
	Status              int     `json:"status"`                // 状态
}

// GetDashboard 获取仪表盘数据
func (s *DistributorService) GetDashboard(ctx context.Context, distributorID int64) (*DashboardData, error) {
	distributor, err := s.distributorRepo.GetByID(ctx, distributorID)
	if err != nil {
		return nil, err
	}

	return &DashboardData{
		TotalCommission:     distributor.TotalCommission,
		AvailableCommission: distributor.AvailableCommission,
		FrozenCommission:    distributor.FrozenCommission,
		WithdrawnCommission: distributor.WithdrawnCommission,
		TeamCount:           distributor.TeamCount,
		DirectCount:         distributor.DirectCount,
		InviteCode:          distributor.InviteCode,
		InviteLink:          s.generateInviteLink(distributor.InviteCode),
		Level:               distributor.Level,
		Status:              distributor.Status,
	}, nil
}

// generateInviteLink 生成邀请链接
func (s *DistributorService) generateInviteLink(inviteCode string) string {
	// 这里应该从配置读取域名，简化处理使用占位符
	return "https://app.example.com/invite/" + inviteCode
}

// ApproveRequest 审核请求
type ApproveRequest struct {
	DistributorID int64 `json:"distributor_id"`
	OperatorID    int64 `json:"operator_id"`
	Approved      bool  `json:"approved"`
	Reason        string `json:"reason,omitempty"` // 拒绝原因
}

// Approve 审核分销商申请
func (s *DistributorService) Approve(ctx context.Context, req *ApproveRequest) error {
	distributor, err := s.distributorRepo.GetByID(ctx, req.DistributorID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return errors.New("分销商不存在")
		}
		return err
	}

	if distributor.Status != models.DistributorStatusPending {
		return errors.New("该申请已处理")
	}

	now := time.Now()

	return s.db.Transaction(func(tx *gorm.DB) error {
		var status int
		if req.Approved {
			status = models.DistributorStatusApproved
		} else {
			status = models.DistributorStatusRejected
		}

		// 更新分销商状态
		updates := map[string]interface{}{
			"status":      status,
			"approved_at": now,
			"approved_by": req.OperatorID,
		}
		if err := tx.Model(&models.Distributor{}).Where("id = ?", req.DistributorID).Updates(updates).Error; err != nil {
			return err
		}

		// 如果审核通过，更新上级的团队人数
		if req.Approved && distributor.ParentID != nil {
			// 更新直接上级的直推人数
			if err := tx.Model(&models.Distributor{}).
				Where("id = ?", *distributor.ParentID).
				UpdateColumn("direct_count", gorm.Expr("direct_count + 1")).Error; err != nil {
				return err
			}

			// 更新所有上级的团队人数
			parentID := distributor.ParentID
			for parentID != nil {
				if err := tx.Model(&models.Distributor{}).
					Where("id = ?", *parentID).
					UpdateColumn("team_count", gorm.Expr("team_count + 1")).Error; err != nil {
					return err
				}

				// 获取上级的上级
				var parent models.Distributor
				if err := tx.First(&parent, *parentID).Error; err != nil {
					break
				}
				parentID = parent.ParentID
			}
		}

		return nil
	})
}

// GetPendingList 获取待审核列表
func (s *DistributorService) GetPendingList(ctx context.Context, offset, limit int) ([]*models.Distributor, int64, error) {
	return s.distributorRepo.GetPendingList(ctx, offset, limit)
}

// List 获取分销商列表
func (s *DistributorService) List(ctx context.Context, offset, limit int, filters map[string]interface{}) ([]*models.Distributor, int64, error) {
	return s.distributorRepo.List(ctx, offset, limit, filters)
}

// GetTopDistributors 获取佣金排行榜
func (s *DistributorService) GetTopDistributors(ctx context.Context, limit int) ([]*models.Distributor, error) {
	return s.distributorRepo.GetTopDistributors(ctx, limit)
}

// CheckIsDistributor 检查用户是否是分销商
func (s *DistributorService) CheckIsDistributor(ctx context.Context, userID int64) (bool, error) {
	return s.distributorRepo.ExistsByUserID(ctx, userID)
}

// GetApprovedDistributorByUserID 获取已审核通过的分销商
func (s *DistributorService) GetApprovedDistributorByUserID(ctx context.Context, userID int64) (*models.Distributor, error) {
	distributor, err := s.distributorRepo.GetByUserID(ctx, userID)
	if err != nil {
		return nil, err
	}
	if distributor.Status != models.DistributorStatusApproved {
		return nil, errors.New("分销商尚未审核通过")
	}
	return distributor, nil
}
