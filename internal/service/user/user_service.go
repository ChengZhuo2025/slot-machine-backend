// Package user 提供用户服务
package user

import (
	"context"
	"time"

	"gorm.io/gorm"

	"smart-locker-backend/internal/common/errors"
	"smart-locker-backend/internal/models"
	"smart-locker-backend/internal/repository"
)

// UserService 用户服务
type UserService struct {
	db       *gorm.DB
	userRepo *repository.UserRepository
}

// NewUserService 创建用户服务
func NewUserService(db *gorm.DB, userRepo *repository.UserRepository) *UserService {
	return &UserService{
		db:       db,
		userRepo: userRepo,
	}
}

// UserProfile 用户详情
type UserProfile struct {
	ID            int64       `json:"id"`
	Phone         *string     `json:"phone,omitempty"`
	Nickname      string      `json:"nickname"`
	Avatar        *string     `json:"avatar,omitempty"`
	Gender        int8        `json:"gender"`
	Birthday      *time.Time  `json:"birthday,omitempty"`
	MemberLevelID int64       `json:"member_level_id"`
	MemberLevel   *MemberLevelInfo `json:"member_level,omitempty"`
	Points        int         `json:"points"`
	IsVerified    bool        `json:"is_verified"`
	CreatedAt     time.Time   `json:"created_at"`
}

// MemberLevelInfo 会员等级信息
type MemberLevelInfo struct {
	ID       int64   `json:"id"`
	Name     string  `json:"name"`
	Level    int     `json:"level"`
	Discount float64 `json:"discount"`
	Icon     *string `json:"icon,omitempty"`
}

// UpdateProfileRequest 更新用户信息请求
type UpdateProfileRequest struct {
	Nickname *string    `json:"nickname,omitempty"`
	Avatar   *string    `json:"avatar,omitempty"`
	Gender   *int8      `json:"gender,omitempty"`
	Birthday *time.Time `json:"birthday,omitempty"`
}

// GetProfile 获取用户详情
func (s *UserService) GetProfile(ctx context.Context, userID int64) (*UserProfile, error) {
	user, err := s.userRepo.GetByIDWithMemberLevel(ctx, userID)
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, errors.ErrUserNotFound
		}
		return nil, errors.ErrDatabaseError.WithError(err)
	}

	profile := &UserProfile{
		ID:            user.ID,
		Phone:         user.Phone,
		Nickname:      user.Nickname,
		Avatar:        user.Avatar,
		Gender:        user.Gender,
		Birthday:      user.Birthday,
		MemberLevelID: user.MemberLevelID,
		Points:        user.Points,
		IsVerified:    user.IsVerified,
		CreatedAt:     user.CreatedAt,
	}

	if user.MemberLevel != nil {
		profile.MemberLevel = &MemberLevelInfo{
			ID:       user.MemberLevel.ID,
			Name:     user.MemberLevel.Name,
			Level:    user.MemberLevel.Level,
			Discount: user.MemberLevel.Discount,
			Icon:     user.MemberLevel.Icon,
		}
	}

	return profile, nil
}

// UpdateProfile 更新用户信息
func (s *UserService) UpdateProfile(ctx context.Context, userID int64, req *UpdateProfileRequest) error {
	updates := make(map[string]interface{})

	if req.Nickname != nil {
		updates["nickname"] = *req.Nickname
	}
	if req.Avatar != nil {
		updates["avatar"] = *req.Avatar
	}
	if req.Gender != nil {
		updates["gender"] = *req.Gender
	}
	if req.Birthday != nil {
		updates["birthday"] = *req.Birthday
	}

	if len(updates) == 0 {
		return nil
	}

	if err := s.userRepo.UpdateFields(ctx, userID, updates); err != nil {
		return errors.ErrDatabaseError.WithError(err)
	}

	return nil
}

// GetMemberLevels 获取会员等级列表
func (s *UserService) GetMemberLevels(ctx context.Context) ([]*MemberLevelInfo, error) {
	var levels []*models.MemberLevel
	if err := s.db.WithContext(ctx).Order("level ASC").Find(&levels).Error; err != nil {
		return nil, errors.ErrDatabaseError.WithError(err)
	}

	result := make([]*MemberLevelInfo, len(levels))
	for i, level := range levels {
		result[i] = &MemberLevelInfo{
			ID:       level.ID,
			Name:     level.Name,
			Level:    level.Level,
			Discount: level.Discount,
			Icon:     level.Icon,
		}
	}

	return result, nil
}

// RealNameVerifyRequest 实名认证请求
type RealNameVerifyRequest struct {
	RealName string `json:"real_name" binding:"required"`
	IDCard   string `json:"id_card" binding:"required"`
}

// RealNameVerify 实名认证
func (s *UserService) RealNameVerify(ctx context.Context, userID int64, req *RealNameVerifyRequest) error {
	user, err := s.userRepo.GetByID(ctx, userID)
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return errors.ErrUserNotFound
		}
		return errors.ErrDatabaseError.WithError(err)
	}

	if user.IsVerified {
		return errors.ErrRealNameVerified
	}

	// TODO: 调用第三方实名认证接口验证
	// 这里暂时直接通过

	// 加密存储（实际应该使用 AES 加密）
	updates := map[string]interface{}{
		"real_name_encrypted": req.RealName, // 应该加密后存储
		"id_card_encrypted":   req.IDCard,   // 应该加密后存储
		"is_verified":         true,
	}

	if err := s.userRepo.UpdateFields(ctx, userID, updates); err != nil {
		return errors.ErrDatabaseError.WithError(err)
	}

	return nil
}

// GetPoints 获取用户积分
func (s *UserService) GetPoints(ctx context.Context, userID int64) (int, error) {
	user, err := s.userRepo.GetByID(ctx, userID)
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return 0, errors.ErrUserNotFound
		}
		return 0, errors.ErrDatabaseError.WithError(err)
	}

	return user.Points, nil
}
