// Package admin 管理端服务
package admin

import (
	"context"
	"time"

	"gorm.io/gorm"

	"github.com/dumeirei/smart-locker-backend/internal/models"
	"github.com/dumeirei/smart-locker-backend/internal/repository"
)

// UserAdminService 用户管理服务
type UserAdminService struct {
	userRepo *repository.UserRepository
	db       *gorm.DB
}

// NewUserAdminService 创建用户管理服务
func NewUserAdminService(db *gorm.DB, userRepo *repository.UserRepository) *UserAdminService {
	return &UserAdminService{
		userRepo: userRepo,
		db:       db,
	}
}

// UserListFilters 用户列表筛选条件
type UserListFilters struct {
	Phone         string
	Nickname      string
	Status        *int8
	MemberLevelID int64
	StartDate     *time.Time
	EndDate       *time.Time
}

// UserListResponse 用户列表响应
type UserListResponse struct {
	ID            int64                `json:"id"`
	Phone         string               `json:"phone"`
	Nickname      string               `json:"nickname"`
	Avatar        string               `json:"avatar"`
	Gender        int8                 `json:"gender"`
	MemberLevelID int64                `json:"member_level_id"`
	MemberLevel   *models.MemberLevel  `json:"member_level,omitempty"`
	Points        int                  `json:"points"`
	IsVerified    bool                 `json:"is_verified"`
	Status        int8                 `json:"status"`
	CreatedAt     time.Time            `json:"created_at"`
	Wallet        *WalletBrief         `json:"wallet,omitempty"`
}

// WalletBrief 钱包摘要
type WalletBrief struct {
	Balance       float64 `json:"balance"`
	FrozenBalance float64 `json:"frozen_balance"`
}

// List 获取用户列表
func (s *UserAdminService) List(ctx context.Context, page, pageSize int, filters *UserListFilters) ([]*UserListResponse, int64, error) {
	offset := (page - 1) * pageSize

	query := s.db.WithContext(ctx).Model(&models.User{})

	if filters != nil {
		if filters.Phone != "" {
			query = query.Where("phone LIKE ?", "%"+filters.Phone+"%")
		}
		if filters.Nickname != "" {
			query = query.Where("nickname LIKE ?", "%"+filters.Nickname+"%")
		}
		if filters.Status != nil {
			query = query.Where("status = ?", *filters.Status)
		}
		if filters.MemberLevelID > 0 {
			query = query.Where("member_level_id = ?", filters.MemberLevelID)
		}
		if filters.StartDate != nil {
			query = query.Where("created_at >= ?", *filters.StartDate)
		}
		if filters.EndDate != nil {
			query = query.Where("created_at <= ?", *filters.EndDate)
		}
	}

	var total int64
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	var users []*models.User
	if err := query.Preload("MemberLevel").Preload("Wallet").
		Order("id DESC").Offset(offset).Limit(pageSize).
		Find(&users).Error; err != nil {
		return nil, 0, err
	}

	results := make([]*UserListResponse, len(users))
	for i, user := range users {
		results[i] = s.toUserListResponse(user)
	}

	return results, total, nil
}

// GetByID 根据 ID 获取用户详情
func (s *UserAdminService) GetByID(ctx context.Context, id int64) (*models.User, error) {
	var user models.User
	err := s.db.WithContext(ctx).
		Preload("MemberLevel").
		Preload("Wallet").
		First(&user, id).Error
	if err != nil {
		return nil, err
	}
	return &user, nil
}

// UpdateUserRequest 更新用户请求
type UpdateUserRequest struct {
	Nickname      *string `json:"nickname"`
	Status        *int8   `json:"status"`
	MemberLevelID *int64  `json:"member_level_id"`
	Points        *int    `json:"points"`
}

// Update 更新用户
func (s *UserAdminService) Update(ctx context.Context, id int64, req *UpdateUserRequest) (*models.User, error) {
	user, err := s.userRepo.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}

	if req.Nickname != nil {
		user.Nickname = *req.Nickname
	}
	if req.Status != nil {
		user.Status = *req.Status
	}
	if req.MemberLevelID != nil {
		user.MemberLevelID = *req.MemberLevelID
	}
	if req.Points != nil {
		user.Points = *req.Points
	}

	if err := s.userRepo.Update(ctx, user); err != nil {
		return nil, err
	}

	return s.GetByID(ctx, id)
}

// UpdateStatus 更新用户状态
func (s *UserAdminService) UpdateStatus(ctx context.Context, id int64, status int8) error {
	return s.userRepo.UpdateStatus(ctx, id, status)
}

// Enable 启用用户
func (s *UserAdminService) Enable(ctx context.Context, id int64) error {
	return s.UpdateStatus(ctx, id, models.UserStatusActive)
}

// Disable 禁用用户
func (s *UserAdminService) Disable(ctx context.Context, id int64) error {
	return s.UpdateStatus(ctx, id, models.UserStatusDisabled)
}

// AdjustPoints 调整积分
func (s *UserAdminService) AdjustPoints(ctx context.Context, id int64, points int, remark string) error {
	return s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var user models.User
		if err := tx.First(&user, id).Error; err != nil {
			return err
		}

		newPoints := user.Points + points
		if newPoints < 0 {
			newPoints = 0
		}

		return tx.Model(&user).Update("points", newPoints).Error
	})
}

// UserStatistics 用户统计
type UserStatistics struct {
	TotalUsers      int64            `json:"total_users"`
	TodayNewUsers   int64            `json:"today_new_users"`
	ActiveUsers     int64            `json:"active_users"`      // 7天内活跃
	VerifiedUsers   int64            `json:"verified_users"`
	DisabledUsers   int64            `json:"disabled_users"`
	LevelDistribution []LevelCount   `json:"level_distribution"`
}

// LevelCount 等级分布
type LevelCount struct {
	LevelID   int64  `json:"level_id"`
	LevelName string `json:"level_name"`
	Count     int64  `json:"count"`
}

// GetStatistics 获取用户统计
func (s *UserAdminService) GetStatistics(ctx context.Context) (*UserStatistics, error) {
	stats := &UserStatistics{}

	now := time.Now()
	today := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())
	weekAgo := now.Add(-7 * 24 * time.Hour)

	// 总用户数
	s.db.WithContext(ctx).Model(&models.User{}).Count(&stats.TotalUsers)

	// 今日新增
	s.db.WithContext(ctx).Model(&models.User{}).
		Where("created_at >= ?", today).
		Count(&stats.TodayNewUsers)

	// 活跃用户
	s.db.WithContext(ctx).Model(&models.User{}).
		Where("updated_at >= ?", weekAgo).
		Count(&stats.ActiveUsers)

	// 实名用户
	s.db.WithContext(ctx).Model(&models.User{}).
		Where("is_verified = ?", true).
		Count(&stats.VerifiedUsers)

	// 禁用用户
	s.db.WithContext(ctx).Model(&models.User{}).
		Where("status = ?", models.UserStatusDisabled).
		Count(&stats.DisabledUsers)

	// 等级分布
	var levelCounts []struct {
		LevelID   int64
		LevelName string
		Count     int64
	}
	s.db.WithContext(ctx).Table("users u").
		Select("u.member_level_id as level_id, ml.name as level_name, COUNT(*) as count").
		Joins("LEFT JOIN member_levels ml ON u.member_level_id = ml.id").
		Group("u.member_level_id, ml.name").
		Order("u.member_level_id ASC").
		Find(&levelCounts)

	stats.LevelDistribution = make([]LevelCount, len(levelCounts))
	for i, lc := range levelCounts {
		stats.LevelDistribution[i] = LevelCount{
			LevelID:   lc.LevelID,
			LevelName: lc.LevelName,
			Count:     lc.Count,
		}
	}

	return stats, nil
}

// toUserListResponse 转换为列表响应
func (s *UserAdminService) toUserListResponse(user *models.User) *UserListResponse {
	resp := &UserListResponse{
		ID:            user.ID,
		Nickname:      user.Nickname,
		Gender:        user.Gender,
		MemberLevelID: user.MemberLevelID,
		MemberLevel:   user.MemberLevel,
		Points:        user.Points,
		IsVerified:    user.IsVerified,
		Status:        user.Status,
		CreatedAt:     user.CreatedAt,
	}

	if user.Phone != nil {
		resp.Phone = *user.Phone
	}
	if user.Avatar != nil {
		resp.Avatar = *user.Avatar
	}
	if user.Wallet != nil {
		resp.Wallet = &WalletBrief{
			Balance:       user.Wallet.Balance,
			FrozenBalance: user.Wallet.FrozenBalance,
		}
	}

	return resp
}
