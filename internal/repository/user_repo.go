// Package repository 提供数据访问层
package repository

import (
	"context"

	"gorm.io/gorm"

	"github.com/dumeirei/smart-locker-backend/internal/models"
)

// UserRepository 用户仓储
type UserRepository struct {
	db *gorm.DB
}

// NewUserRepository 创建用户仓储
func NewUserRepository(db *gorm.DB) *UserRepository {
	return &UserRepository{db: db}
}

// Create 创建用户
func (r *UserRepository) Create(ctx context.Context, user *models.User) error {
	return r.db.WithContext(ctx).Create(user).Error
}

// GetByID 根据 ID 获取用户
func (r *UserRepository) GetByID(ctx context.Context, id int64) (*models.User, error) {
	var user models.User
	err := r.db.WithContext(ctx).First(&user, id).Error
	if err != nil {
		return nil, err
	}
	return &user, nil
}

// GetByIDWithWallet 根据 ID 获取用户（包含钱包）
func (r *UserRepository) GetByIDWithWallet(ctx context.Context, id int64) (*models.User, error) {
	var user models.User
	err := r.db.WithContext(ctx).Preload("Wallet").First(&user, id).Error
	if err != nil {
		return nil, err
	}
	return &user, nil
}

// GetByIDWithMemberLevel 根据 ID 获取用户（包含会员等级）
func (r *UserRepository) GetByIDWithMemberLevel(ctx context.Context, id int64) (*models.User, error) {
	var user models.User
	err := r.db.WithContext(ctx).Preload("MemberLevel").First(&user, id).Error
	if err != nil {
		return nil, err
	}
	return &user, nil
}

// GetByPhone 根据手机号获取用户
func (r *UserRepository) GetByPhone(ctx context.Context, phone string) (*models.User, error) {
	var user models.User
	err := r.db.WithContext(ctx).Where("phone = ?", phone).First(&user).Error
	if err != nil {
		return nil, err
	}
	return &user, nil
}

// GetByOpenID 根据 OpenID 获取用户
func (r *UserRepository) GetByOpenID(ctx context.Context, openID string) (*models.User, error) {
	var user models.User
	err := r.db.WithContext(ctx).Where("openid = ?", openID).First(&user).Error
	if err != nil {
		return nil, err
	}
	return &user, nil
}

// GetByInviteCode 根据邀请码获取用户（通过分销商表）
func (r *UserRepository) GetByInviteCode(ctx context.Context, inviteCode string) (*models.User, error) {
	var distributor models.Distributor
	err := r.db.WithContext(ctx).Where("invite_code = ?", inviteCode).First(&distributor).Error
	if err != nil {
		return nil, err
	}

	var user models.User
	err = r.db.WithContext(ctx).First(&user, distributor.UserID).Error
	if err != nil {
		return nil, err
	}
	return &user, nil
}

// Update 更新用户
func (r *UserRepository) Update(ctx context.Context, user *models.User) error {
	return r.db.WithContext(ctx).Save(user).Error
}

// UpdateFields 更新指定字段
func (r *UserRepository) UpdateFields(ctx context.Context, id int64, fields map[string]interface{}) error {
	return r.db.WithContext(ctx).Model(&models.User{}).Where("id = ?", id).Updates(fields).Error
}

// UpdateStatus 更新用户状态
func (r *UserRepository) UpdateStatus(ctx context.Context, id int64, status int8) error {
	return r.db.WithContext(ctx).Model(&models.User{}).Where("id = ?", id).Update("status", status).Error
}

// List 获取用户列表
func (r *UserRepository) List(ctx context.Context, offset, limit int, filters map[string]interface{}) ([]*models.User, int64, error) {
	var users []*models.User
	var total int64

	query := r.db.WithContext(ctx).Model(&models.User{})

	// 应用过滤条件
	if phone, ok := filters["phone"].(string); ok && phone != "" {
		query = query.Where("phone LIKE ?", "%"+phone+"%")
	}
	if status, ok := filters["status"].(int8); ok {
		query = query.Where("status = ?", status)
	}
	if memberLevelID, ok := filters["member_level_id"].(int64); ok && memberLevelID > 0 {
		query = query.Where("member_level_id = ?", memberLevelID)
	}

	// 统计总数
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	// 查询列表
	if err := query.Order("id DESC").Offset(offset).Limit(limit).Find(&users).Error; err != nil {
		return nil, 0, err
	}

	return users, total, nil
}

// ExistsByPhone 检查手机号是否存在
func (r *UserRepository) ExistsByPhone(ctx context.Context, phone string) (bool, error) {
	var count int64
	err := r.db.WithContext(ctx).Model(&models.User{}).Where("phone = ?", phone).Count(&count).Error
	return count > 0, err
}

// ExistsByOpenID 检查 OpenID 是否存在
func (r *UserRepository) ExistsByOpenID(ctx context.Context, openID string) (bool, error) {
	var count int64
	err := r.db.WithContext(ctx).Model(&models.User{}).Where("openid = ?", openID).Count(&count).Error
	return count > 0, err
}

// GetReferrals 获取推荐的用户列表
func (r *UserRepository) GetReferrals(ctx context.Context, referrerID int64, offset, limit int) ([]*models.User, int64, error) {
	var users []*models.User
	var total int64

	query := r.db.WithContext(ctx).Model(&models.User{}).Where("referrer_id = ?", referrerID)

	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	if err := query.Order("id DESC").Offset(offset).Limit(limit).Find(&users).Error; err != nil {
		return nil, 0, err
	}

	return users, total, nil
}

// AddPoints 增加积分
func (r *UserRepository) AddPoints(ctx context.Context, userID int64, points int) error {
	return r.db.WithContext(ctx).Model(&models.User{}).
		Where("id = ?", userID).
		UpdateColumn("points", gorm.Expr("points + ?", points)).
		Error
}

// DeductPoints 扣减积分
func (r *UserRepository) DeductPoints(ctx context.Context, userID int64, points int) error {
	result := r.db.WithContext(ctx).Model(&models.User{}).
		Where("id = ? AND points >= ?", userID, points).
		UpdateColumn("points", gorm.Expr("points - ?", points))
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return gorm.ErrRecordNotFound
	}
	return nil
}
