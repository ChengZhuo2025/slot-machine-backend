// Package user 提供用户服务
package user

import (
	"context"
	"fmt"
	"time"

	"gorm.io/gorm"

	"github.com/dumeirei/smart-locker-backend/internal/common/errors"
	"github.com/dumeirei/smart-locker-backend/internal/models"
	"github.com/dumeirei/smart-locker-backend/internal/repository"
)

// PointsService 积分服务
type PointsService struct {
	db              *gorm.DB
	userRepo        *repository.UserRepository
	memberLevelRepo *repository.MemberLevelRepository
}

// NewPointsService 创建积分服务
func NewPointsService(db *gorm.DB, userRepo *repository.UserRepository, memberLevelRepo *repository.MemberLevelRepository) *PointsService {
	return &PointsService{
		db:              db,
		userRepo:        userRepo,
		memberLevelRepo: memberLevelRepo,
	}
}

// PointsRecord 积分记录
type PointsRecord struct {
	ID          int64     `json:"id"`
	UserID      int64     `json:"user_id"`
	Type        string    `json:"type"`
	TypeName    string    `json:"type_name"`
	Points      int       `json:"points"`
	OrderNo     *string   `json:"order_no,omitempty"`
	Description string    `json:"description"`
	CreatedAt   time.Time `json:"created_at"`
}

// PointsInfo 积分信息
type PointsInfo struct {
	Points           int     `json:"points"`
	MemberLevelID    int64   `json:"member_level_id"`
	MemberLevelName  string  `json:"member_level_name"`
	Discount         float64 `json:"discount"`
	NextLevelName    *string `json:"next_level_name,omitempty"`
	NextLevelPoints  *int    `json:"next_level_points,omitempty"`
	PointsToNextLevel *int   `json:"points_to_next_level,omitempty"`
}

// 积分类型常量
const (
	PointsTypeConsume         = "consume"          // 消费获取
	PointsTypePackagePurchase = "package_purchase" // 套餐购买
	PointsTypeSignIn          = "sign_in"          // 签到
	PointsTypeActivity        = "activity"         // 活动赠送
	PointsTypeRefund          = "refund"           // 退款扣减
	PointsTypeExpired         = "expired"          // 积分过期
	PointsTypeExchange        = "exchange"         // 积分兑换
	PointsTypeAdmin           = "admin"            // 管理员调整
)

// GetPointsInfo 获取积分信息
func (s *PointsService) GetPointsInfo(ctx context.Context, userID int64) (*PointsInfo, error) {
	user, err := s.userRepo.GetByIDWithMemberLevel(ctx, userID)
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, errors.ErrUserNotFound
		}
		return nil, errors.ErrDatabaseError.WithError(err)
	}

	info := &PointsInfo{
		Points:          user.Points,
		MemberLevelID:   user.MemberLevelID,
		MemberLevelName: "普通会员",
		Discount:        1.0,
	}

	if user.MemberLevel != nil {
		info.MemberLevelName = user.MemberLevel.Name
		info.Discount = user.MemberLevel.Discount

		// 获取下一级会员信息
		nextLevel, err := s.memberLevelRepo.GetNextLevel(ctx, user.MemberLevel.Level)
		if err == nil && nextLevel != nil {
			info.NextLevelName = &nextLevel.Name
			info.NextLevelPoints = &nextLevel.MinPoints
			pointsToNext := nextLevel.MinPoints - user.Points
			if pointsToNext > 0 {
				info.PointsToNextLevel = &pointsToNext
			}
		}
	}

	return info, nil
}

// AddPoints 增加积分
func (s *PointsService) AddPoints(ctx context.Context, userID int64, points int, pointsType, description string, orderNo *string) error {
	if points <= 0 {
		return errors.ErrInvalidParams.WithMessage("积分必须大于0")
	}

	return s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		return s.AddPointsTx(ctx, tx, userID, points, pointsType, description, orderNo)
	})
}

// AddPointsTx 在已有事务中增加积分
func (s *PointsService) AddPointsTx(ctx context.Context, tx *gorm.DB, userID int64, points int, pointsType, description string, orderNo *string) error {
	if points <= 0 {
		return errors.ErrInvalidParams.WithMessage("积分必须大于0")
	}

	// 增加积分
	if err := tx.WithContext(ctx).Model(&models.User{}).
		Where("id = ?", userID).
		UpdateColumn("points", gorm.Expr("points + ?", points)).Error; err != nil {
		return errors.ErrDatabaseError.WithError(err)
	}

	// 记录积分变动（使用钱包交易表记录，或者创建专门的积分记录表）
	// 这里使用备注记录积分变动信息
	record := &models.WalletTransaction{
		UserID:  userID,
		Type:    fmt.Sprintf("points_%s", pointsType),
		Amount:  float64(points),
		OrderNo: orderNo,
		Remark:  &description,
	}
	if err := tx.WithContext(ctx).Create(record).Error; err != nil {
		return errors.ErrDatabaseError.WithError(err)
	}

	// 检查并更新会员等级
	return s.checkAndUpgradeLevelTx(ctx, tx, userID)
}

// DeductPoints 扣减积分
func (s *PointsService) DeductPoints(ctx context.Context, userID int64, points int, pointsType, description string, orderNo *string) error {
	if points <= 0 {
		return errors.ErrInvalidParams.WithMessage("积分必须大于0")
	}

	return s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		return s.DeductPointsTx(ctx, tx, userID, points, pointsType, description, orderNo)
	})
}

// DeductPointsTx 在已有事务中扣减积分
func (s *PointsService) DeductPointsTx(ctx context.Context, tx *gorm.DB, userID int64, points int, pointsType, description string, orderNo *string) error {
	if points <= 0 {
		return errors.ErrInvalidParams.WithMessage("积分必须大于0")
	}

	// 扣减积分（确保积分不为负）
	result := tx.WithContext(ctx).Model(&models.User{}).
		Where("id = ? AND points >= ?", userID, points).
		UpdateColumn("points", gorm.Expr("points - ?", points))
	if result.Error != nil {
		return errors.ErrDatabaseError.WithError(result.Error)
	}
	if result.RowsAffected == 0 {
		return errors.New(errors.ErrOperationFailed.Code, "积分不足")
	}

	// 记录积分变动
	record := &models.WalletTransaction{
		UserID:  userID,
		Type:    fmt.Sprintf("points_%s", pointsType),
		Amount:  float64(-points),
		OrderNo: orderNo,
		Remark:  &description,
	}
	if err := tx.WithContext(ctx).Create(record).Error; err != nil {
		return errors.ErrDatabaseError.WithError(err)
	}

	return nil
}

// checkAndUpgradeLevelTx 检查并升级会员等级
func (s *PointsService) checkAndUpgradeLevelTx(ctx context.Context, tx *gorm.DB, userID int64) error {
	// 获取用户当前积分
	var user models.User
	if err := tx.WithContext(ctx).Select("id, points, member_level_id").First(&user, userID).Error; err != nil {
		return err
	}

	// 获取用户应该对应的会员等级
	var targetLevel models.MemberLevel
	if err := tx.WithContext(ctx).
		Where("min_points <= ?", user.Points).
		Order("min_points DESC").
		First(&targetLevel).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			// 没有匹配的等级，使用默认等级
			return nil
		}
		return err
	}

	// 如果需要升级
	if targetLevel.ID != user.MemberLevelID {
		if err := tx.WithContext(ctx).Model(&models.User{}).
			Where("id = ?", userID).
			Update("member_level_id", targetLevel.ID).Error; err != nil {
			return err
		}
	}

	return nil
}

// GetPointsHistory 获取积分历史记录
func (s *PointsService) GetPointsHistory(ctx context.Context, userID int64, offset, limit int, pointsType string) ([]*PointsRecord, int64, error) {
	var transactions []*models.WalletTransaction
	var total int64

	query := s.db.WithContext(ctx).Model(&models.WalletTransaction{}).
		Where("user_id = ? AND type LIKE 'points_%'", userID)

	if pointsType != "" {
		query = query.Where("type = ?", fmt.Sprintf("points_%s", pointsType))
	}

	if err := query.Count(&total).Error; err != nil {
		return nil, 0, errors.ErrDatabaseError.WithError(err)
	}

	if err := query.Order("id DESC").Offset(offset).Limit(limit).Find(&transactions).Error; err != nil {
		return nil, 0, errors.ErrDatabaseError.WithError(err)
	}

	records := make([]*PointsRecord, len(transactions))
	for i, tx := range transactions {
		records[i] = &PointsRecord{
			ID:          tx.ID,
			UserID:      tx.UserID,
			Type:        tx.Type,
			TypeName:    s.getPointsTypeName(tx.Type),
			Points:      int(tx.Amount),
			OrderNo:     tx.OrderNo,
			Description: s.getDescription(tx.Remark),
			CreatedAt:   tx.CreatedAt,
		}
	}

	return records, total, nil
}

// getPointsTypeName 获取积分类型名称
func (s *PointsService) getPointsTypeName(txType string) string {
	switch txType {
	case "points_consume":
		return "消费获取"
	case "points_package_purchase":
		return "套餐购买"
	case "points_sign_in":
		return "签到"
	case "points_activity":
		return "活动赠送"
	case "points_refund":
		return "退款扣减"
	case "points_expired":
		return "积分过期"
	case "points_exchange":
		return "积分兑换"
	case "points_admin":
		return "管理员调整"
	default:
		return "其他"
	}
}

// getDescription 获取描述
func (s *PointsService) getDescription(remark *string) string {
	if remark != nil {
		return *remark
	}
	return ""
}

// CalculatePointsByAmount 根据消费金额计算积分（每消费1元获得1积分）
func (s *PointsService) CalculatePointsByAmount(amount float64) int {
	return int(amount)
}

// AddConsumePoints 添加消费积分
func (s *PointsService) AddConsumePoints(ctx context.Context, userID int64, amount float64, orderNo string) error {
	points := s.CalculatePointsByAmount(amount)
	if points <= 0 {
		return nil
	}

	description := fmt.Sprintf("消费%.2f元获得积分", amount)
	return s.AddPoints(ctx, userID, points, PointsTypeConsume, description, &orderNo)
}

// AddConsumePointsTx 在事务中添加消费积分
func (s *PointsService) AddConsumePointsTx(ctx context.Context, tx *gorm.DB, userID int64, amount float64, orderNo string) error {
	points := s.CalculatePointsByAmount(amount)
	if points <= 0 {
		return nil
	}

	description := fmt.Sprintf("消费%.2f元获得积分", amount)
	return s.AddPointsTx(ctx, tx, userID, points, PointsTypeConsume, description, &orderNo)
}

// RefundPoints 退款扣减积分
func (s *PointsService) RefundPoints(ctx context.Context, userID int64, amount float64, orderNo string) error {
	points := s.CalculatePointsByAmount(amount)
	if points <= 0 {
		return nil
	}

	description := fmt.Sprintf("订单退款扣减积分")
	return s.DeductPoints(ctx, userID, points, PointsTypeRefund, description, &orderNo)
}

// RefundPointsTx 在事务中退款扣减积分
func (s *PointsService) RefundPointsTx(ctx context.Context, tx *gorm.DB, userID int64, amount float64, orderNo string) error {
	points := s.CalculatePointsByAmount(amount)
	if points <= 0 {
		return nil
	}

	description := fmt.Sprintf("订单退款扣减积分")
	return s.DeductPointsTx(ctx, tx, userID, points, PointsTypeRefund, description, &orderNo)
}
