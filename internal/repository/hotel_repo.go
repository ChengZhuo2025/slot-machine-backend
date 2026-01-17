// Package repository 提供数据访问层
package repository

import (
	"context"
	"time"

	"gorm.io/gorm"

	"github.com/dumeirei/smart-locker-backend/internal/models"
)

// HotelRepository 酒店仓储
type HotelRepository struct {
	db *gorm.DB
}

// NewHotelRepository 创建酒店仓储
func NewHotelRepository(db *gorm.DB) *HotelRepository {
	return &HotelRepository{db: db}
}

// Create 创建酒店
func (r *HotelRepository) Create(ctx context.Context, hotel *models.Hotel) error {
	return r.db.WithContext(ctx).Create(hotel).Error
}

// GetByID 根据 ID 获取酒店
func (r *HotelRepository) GetByID(ctx context.Context, id int64) (*models.Hotel, error) {
	var hotel models.Hotel
	err := r.db.WithContext(ctx).First(&hotel, id).Error
	if err != nil {
		return nil, err
	}
	return &hotel, nil
}

// GetByIDWithRooms 根据 ID 获取酒店（包含房间）
func (r *HotelRepository) GetByIDWithRooms(ctx context.Context, id int64) (*models.Hotel, error) {
	var hotel models.Hotel
	err := r.db.WithContext(ctx).
		Preload("Rooms", func(db *gorm.DB) *gorm.DB {
			return db.Where("status = ?", models.RoomStatusActive).Order("room_no ASC")
		}).
		First(&hotel, id).Error
	if err != nil {
		return nil, err
	}
	return &hotel, nil
}

// Update 更新酒店
func (r *HotelRepository) Update(ctx context.Context, hotel *models.Hotel) error {
	return r.db.WithContext(ctx).Save(hotel).Error
}

// UpdateFields 更新指定字段
func (r *HotelRepository) UpdateFields(ctx context.Context, id int64, fields map[string]interface{}) error {
	return r.db.WithContext(ctx).Model(&models.Hotel{}).Where("id = ?", id).Updates(fields).Error
}

// UpdateStatus 更新酒店状态
func (r *HotelRepository) UpdateStatus(ctx context.Context, id int64, status int8) error {
	return r.db.WithContext(ctx).Model(&models.Hotel{}).Where("id = ?", id).Update("status", status).Error
}

// List 获取酒店列表
func (r *HotelRepository) List(ctx context.Context, offset, limit int, filters map[string]interface{}) ([]*models.Hotel, int64, error) {
	var hotels []*models.Hotel
	var total int64

	query := r.db.WithContext(ctx).Model(&models.Hotel{})

	// 应用过滤条件
	if name, ok := filters["name"].(string); ok && name != "" {
		query = query.Where("name LIKE ?", "%"+name+"%")
	}
	if city, ok := filters["city"].(string); ok && city != "" {
		query = query.Where("city = ?", city)
	}
	if district, ok := filters["district"].(string); ok && district != "" {
		query = query.Where("district = ?", district)
	}
	if province, ok := filters["province"].(string); ok && province != "" {
		query = query.Where("province = ?", province)
	}
	if starRating, ok := filters["star_rating"].(int); ok && starRating > 0 {
		query = query.Where("star_rating = ?", starRating)
	}
	if status, ok := filters["status"].(int8); ok {
		query = query.Where("status = ?", status)
	}

	// 统计总数
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	// 查询列表
	if err := query.Order("id DESC").Offset(offset).Limit(limit).Find(&hotels).Error; err != nil {
		return nil, 0, err
	}

	return hotels, total, nil
}

// ListActive 获取上架的酒店列表（用户端）
func (r *HotelRepository) ListActive(ctx context.Context, offset, limit int, filters map[string]interface{}) ([]*models.Hotel, int64, error) {
	filters["status"] = int8(models.HotelStatusActive)
	return r.List(ctx, offset, limit, filters)
}

// ListNearby 获取附近酒店列表（基于经纬度）
func (r *HotelRepository) ListNearby(ctx context.Context, longitude, latitude float64, radiusKm float64, limit int) ([]*models.Hotel, error) {
	var hotels []*models.Hotel

	// 使用 Haversine 公式计算距离
	query := r.db.WithContext(ctx).
		Select("*, (6371 * acos(cos(radians(?)) * cos(radians(latitude)) * cos(radians(longitude) - radians(?)) + sin(radians(?)) * sin(radians(latitude)))) AS distance", latitude, longitude, latitude).
		Where("status = ?", models.HotelStatusActive).
		Where("latitude IS NOT NULL AND longitude IS NOT NULL").
		Having("distance < ?", radiusKm).
		Order("distance ASC").
		Limit(limit)

	err := query.Find(&hotels).Error
	return hotels, err
}

// ListByCity 获取城市下的酒店列表
func (r *HotelRepository) ListByCity(ctx context.Context, city string, offset, limit int) ([]*models.Hotel, int64, error) {
	var hotels []*models.Hotel
	var total int64

	query := r.db.WithContext(ctx).Model(&models.Hotel{}).
		Where("city = ?", city).
		Where("status = ?", models.HotelStatusActive)

	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	if err := query.Order("star_rating DESC, id DESC").Offset(offset).Limit(limit).Find(&hotels).Error; err != nil {
		return nil, 0, err
	}

	return hotels, total, nil
}

// GetCities 获取所有城市列表
func (r *HotelRepository) GetCities(ctx context.Context) ([]string, error) {
	var cities []string
	err := r.db.WithContext(ctx).Model(&models.Hotel{}).
		Where("status = ?", models.HotelStatusActive).
		Distinct("city").
		Pluck("city", &cities).Error
	return cities, err
}

// ExistsByName 检查酒店名称是否存在
func (r *HotelRepository) ExistsByName(ctx context.Context, name string) (bool, error) {
	var count int64
	err := r.db.WithContext(ctx).Model(&models.Hotel{}).
		Where("name = ?", name).
		Count(&count).Error
	return count > 0, err
}

// Delete 删除酒店（软删除）
func (r *HotelRepository) Delete(ctx context.Context, id int64) error {
	return r.db.WithContext(ctx).Delete(&models.Hotel{}, id).Error
}

// GetRoomCount 获取酒店下的房间数量
func (r *HotelRepository) GetRoomCount(ctx context.Context, hotelID int64) (int64, error) {
	var count int64
	err := r.db.WithContext(ctx).Model(&models.Room{}).
		Where("hotel_id = ?", hotelID).
		Where("status = ?", models.RoomStatusActive).
		Count(&count).Error
	return count, err
}

// GetAvailableRoomCount 获取酒店下的可用房间数量
func (r *HotelRepository) GetAvailableRoomCount(ctx context.Context, hotelID int64) (int64, error) {
	var count int64
	err := r.db.WithContext(ctx).Model(&models.Room{}).
		Where("hotel_id = ?", hotelID).
		Where("status = ?", models.RoomStatusActive).
		Count(&count).Error
	return count, err
}

// Search 搜索酒店
func (r *HotelRepository) Search(ctx context.Context, keyword string, offset, limit int) ([]*models.Hotel, int64, error) {
	var hotels []*models.Hotel
	var total int64

	query := r.db.WithContext(ctx).Model(&models.Hotel{}).
		Where("status = ?", models.HotelStatusActive).
		Where("name LIKE ? OR address LIKE ? OR description LIKE ?",
			"%"+keyword+"%", "%"+keyword+"%", "%"+keyword+"%")

	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	if err := query.Order("star_rating DESC, id DESC").Offset(offset).Limit(limit).Find(&hotels).Error; err != nil {
		return nil, 0, err
	}

	return hotels, total, nil
}

// ListRecommended 获取推荐酒店列表
func (r *HotelRepository) ListRecommended(ctx context.Context, limit int) ([]*models.Hotel, error) {
	var hotels []*models.Hotel
	err := r.db.WithContext(ctx).
		Where("is_recommended = ?", true).
		Where("status = ?", models.HotelStatusActive).
		Order("recommend_score DESC, id DESC").
		Limit(limit).
		Find(&hotels).Error
	return hotels, err
}

// SetRecommended 设置酒店推荐状态
func (r *HotelRepository) SetRecommended(ctx context.Context, id int64, isRecommended bool, score int) error {
	fields := map[string]interface{}{
		"is_recommended":  isRecommended,
		"recommend_score": score,
	}
	if isRecommended {
		now := time.Now()
		fields["recommended_at"] = &now
	} else {
		fields["recommended_at"] = nil
	}
	return r.db.WithContext(ctx).Model(&models.Hotel{}).Where("id = ?", id).Updates(fields).Error
}
