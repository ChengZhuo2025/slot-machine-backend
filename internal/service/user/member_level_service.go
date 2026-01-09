// Package user 提供用户服务
package user

import (
	"context"

	"gorm.io/gorm"

	"github.com/dumeirei/smart-locker-backend/internal/common/errors"
	"github.com/dumeirei/smart-locker-backend/internal/models"
	"github.com/dumeirei/smart-locker-backend/internal/repository"
)

// MemberLevelService 会员等级服务
type MemberLevelService struct {
	db              *gorm.DB
	userRepo        *repository.UserRepository
	memberLevelRepo *repository.MemberLevelRepository
}

// NewMemberLevelService 创建会员等级服务
func NewMemberLevelService(db *gorm.DB, userRepo *repository.UserRepository, memberLevelRepo *repository.MemberLevelRepository) *MemberLevelService {
	return &MemberLevelService{
		db:              db,
		userRepo:        userRepo,
		memberLevelRepo: memberLevelRepo,
	}
}

// DetailedMemberLevelInfo 详细会员等级信息
type DetailedMemberLevelInfo struct {
	ID        int64                  `json:"id"`
	Name      string                 `json:"name"`
	Level     int                    `json:"level"`
	MinPoints int                    `json:"min_points"`
	Discount  float64                `json:"discount"`
	Benefits  map[string]interface{} `json:"benefits,omitempty"`
	Icon      *string                `json:"icon,omitempty"`
}

// UserMemberInfo 用户会员信息
type UserMemberInfo struct {
	UserID              int64                    `json:"user_id"`
	Points              int                      `json:"points"`
	CurrentLevel        *DetailedMemberLevelInfo `json:"current_level"`
	NextLevel           *DetailedMemberLevelInfo `json:"next_level,omitempty"`
	PointsToNextLevel   *int                     `json:"points_to_next_level,omitempty"`
	ProgressPercent     *float64                 `json:"progress_percent,omitempty"`
	MemberExpireAt      *string                  `json:"member_expire_at,omitempty"` // 付费会员过期时间
	IsPaidMember        bool                     `json:"is_paid_member"`
}

// GetAllLevels 获取所有会员等级
func (s *MemberLevelService) GetAllLevels(ctx context.Context) ([]*DetailedMemberLevelInfo, error) {
	levels, err := s.memberLevelRepo.GetAll(ctx)
	if err != nil {
		return nil, errors.ErrDatabaseError.WithError(err)
	}

	result := make([]*DetailedMemberLevelInfo, len(levels))
	for i, level := range levels {
		result[i] = s.toDetailedMemberLevelInfo(level)
	}

	return result, nil
}

// GetLevelByID 根据ID获取会员等级
func (s *MemberLevelService) GetLevelByID(ctx context.Context, id int64) (*DetailedMemberLevelInfo, error) {
	level, err := s.memberLevelRepo.GetByID(ctx, id)
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, errors.New(errors.ErrResourceNotFound.Code, "会员等级不存在")
		}
		return nil, errors.ErrDatabaseError.WithError(err)
	}

	return s.toDetailedMemberLevelInfo(level), nil
}

// GetUserMemberInfo 获取用户会员信息
func (s *MemberLevelService) GetUserMemberInfo(ctx context.Context, userID int64) (*UserMemberInfo, error) {
	user, err := s.userRepo.GetByIDWithMemberLevel(ctx, userID)
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, errors.ErrUserNotFound
		}
		return nil, errors.ErrDatabaseError.WithError(err)
	}

	info := &UserMemberInfo{
		UserID:       userID,
		Points:       user.Points,
		IsPaidMember: false,
	}

	// 当前会员等级
	if user.MemberLevel != nil {
		info.CurrentLevel = s.toDetailedMemberLevelInfo(user.MemberLevel)
	} else {
		// 获取默认等级
		defaultLevel, err := s.memberLevelRepo.GetDefaultLevel(ctx)
		if err == nil && defaultLevel != nil {
			info.CurrentLevel = s.toDetailedMemberLevelInfo(defaultLevel)
		}
	}

	// 获取下一级会员信息
	if info.CurrentLevel != nil {
		nextLevel, err := s.memberLevelRepo.GetNextLevel(ctx, info.CurrentLevel.Level)
		if err == nil && nextLevel != nil {
			info.NextLevel = s.toDetailedMemberLevelInfo(nextLevel)

			// 计算到下一级需要的积分
			pointsNeeded := nextLevel.MinPoints - user.Points
			if pointsNeeded > 0 {
				info.PointsToNextLevel = &pointsNeeded
			}

			// 计算进度百分比
			if info.CurrentLevel.MinPoints < nextLevel.MinPoints {
				progress := float64(user.Points-info.CurrentLevel.MinPoints) / float64(nextLevel.MinPoints-info.CurrentLevel.MinPoints) * 100
				if progress > 100 {
					progress = 100
				}
				if progress < 0 {
					progress = 0
				}
				info.ProgressPercent = &progress
			}
		}
	}

	return info, nil
}

// CheckAndUpgradeLevel 检查并升级会员等级
func (s *MemberLevelService) CheckAndUpgradeLevel(ctx context.Context, userID int64) (bool, *DetailedMemberLevelInfo, error) {
	user, err := s.userRepo.GetByID(ctx, userID)
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return false, nil, errors.ErrUserNotFound
		}
		return false, nil, errors.ErrDatabaseError.WithError(err)
	}

	// 根据积分获取应该对应的等级
	targetLevel, err := s.memberLevelRepo.GetByMinPoints(ctx, user.Points)
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return false, nil, nil
		}
		return false, nil, errors.ErrDatabaseError.WithError(err)
	}

	// 如果等级变化了，更新用户等级
	if targetLevel.ID != user.MemberLevelID {
		if err := s.userRepo.UpdateFields(ctx, userID, map[string]interface{}{
			"member_level_id": targetLevel.ID,
		}); err != nil {
			return false, nil, errors.ErrDatabaseError.WithError(err)
		}
		return true, s.toDetailedMemberLevelInfo(targetLevel), nil
	}

	return false, nil, nil
}

// CheckAndUpgradeLevelTx 在事务中检查并升级会员等级
func (s *MemberLevelService) CheckAndUpgradeLevelTx(ctx context.Context, tx *gorm.DB, userID int64) (bool, error) {
	// 获取用户当前积分
	var user models.User
	if err := tx.WithContext(ctx).Select("id, points, member_level_id").First(&user, userID).Error; err != nil {
		return false, err
	}

	// 获取用户应该对应的会员等级
	var targetLevel models.MemberLevel
	if err := tx.WithContext(ctx).
		Where("min_points <= ?", user.Points).
		Order("min_points DESC").
		First(&targetLevel).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return false, nil
		}
		return false, err
	}

	// 如果需要升级
	if targetLevel.ID != user.MemberLevelID {
		if err := tx.WithContext(ctx).Model(&models.User{}).
			Where("id = ?", userID).
			Update("member_level_id", targetLevel.ID).Error; err != nil {
			return false, err
		}
		return true, nil
	}

	return false, nil
}

// GetDiscount 获取用户会员折扣
func (s *MemberLevelService) GetDiscount(ctx context.Context, userID int64) (float64, error) {
	user, err := s.userRepo.GetByIDWithMemberLevel(ctx, userID)
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return 1.0, nil // 未找到用户，返回无折扣
		}
		return 1.0, errors.ErrDatabaseError.WithError(err)
	}

	if user.MemberLevel != nil {
		return user.MemberLevel.Discount, nil
	}

	return 1.0, nil
}

// toDetailedMemberLevelInfo 转换为详细会员等级信息
func (s *MemberLevelService) toDetailedMemberLevelInfo(level *models.MemberLevel) *DetailedMemberLevelInfo {
	if level == nil {
		return nil
	}

	info := &DetailedMemberLevelInfo{
		ID:        level.ID,
		Name:      level.Name,
		Level:     level.Level,
		MinPoints: level.MinPoints,
		Discount:  level.Discount,
		Icon:      level.Icon,
	}

	if level.Benefits != nil {
		info.Benefits = level.Benefits
	}

	return info
}
