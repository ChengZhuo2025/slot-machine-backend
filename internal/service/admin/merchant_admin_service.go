// Package admin 提供管理员相关服务
package admin

import (
	"context"
	"errors"

	"gorm.io/gorm"

	"github.com/dumeirei/smart-locker-backend/internal/common/crypto"
	"github.com/dumeirei/smart-locker-backend/internal/models"
	"github.com/dumeirei/smart-locker-backend/internal/repository"
)

// MerchantAdminService 商户管理服务
type MerchantAdminService struct {
	merchantRepo *repository.MerchantRepository
	aes          *crypto.AES
}

// NewMerchantAdminService 创建商户管理服务
func NewMerchantAdminService(merchantRepo *repository.MerchantRepository, aes *crypto.AES) *MerchantAdminService {
	return &MerchantAdminService{
		merchantRepo: merchantRepo,
		aes:          aes,
	}
}

// 预定义错误
var (
	ErrMerchantNotFound   = errors.New("商户不存在")
	ErrMerchantNameExists = errors.New("商户名称已存在")
	ErrMerchantHasVenues  = errors.New("商户下有场地，无法删除")
)

// MerchantInfo 商户信息
type MerchantInfo struct {
	ID              int64                     `json:"id"`
	Name            string                    `json:"name"`
	ContactName     string                    `json:"contact_name"`
	ContactPhone    string                    `json:"contact_phone"`
	Address         *string                   `json:"address,omitempty"`
	BusinessLicense *string                   `json:"business_license,omitempty"`
	CommissionRate  float64                   `json:"commission_rate"`
	SettlementType  string                    `json:"settlement_type"`
	BankName        *string                   `json:"bank_name,omitempty"`
	BankAccount     *string                   `json:"bank_account,omitempty"` // 脱敏后的账号
	BankHolder      *string                   `json:"bank_holder,omitempty"`  // 脱敏后的持卡人
	Status          int8                      `json:"status"`
	Stats           *repository.MerchantStats `json:"stats,omitempty"`
}

// CreateMerchantRequest 创建商户请求
type CreateMerchantRequest struct {
	Name            string   `json:"name" binding:"required,max=100"`
	ContactName     string   `json:"contact_name" binding:"required,max=50"`
	ContactPhone    string   `json:"contact_phone" binding:"required,max=20"`
	Address         *string  `json:"address"`
	BusinessLicense *string  `json:"business_license"`
	CommissionRate  float64  `json:"commission_rate" binding:"min=0,max=1"`
	SettlementType  string   `json:"settlement_type" binding:"oneof=weekly monthly"`
	BankName        *string  `json:"bank_name"`
	BankAccount     *string  `json:"bank_account"`
	BankHolder      *string  `json:"bank_holder"`
}

// CreateMerchant 创建商户
func (s *MerchantAdminService) CreateMerchant(ctx context.Context, req *CreateMerchantRequest) (*models.Merchant, error) {
	// 检查名称是否存在
	exists, err := s.merchantRepo.ExistsByName(ctx, req.Name)
	if err != nil {
		return nil, err
	}
	if exists {
		return nil, ErrMerchantNameExists
	}

	commissionRate := req.CommissionRate
	if commissionRate == 0 {
		commissionRate = 0.2 // 默认 20% 分成
	}

	settlementType := req.SettlementType
	if settlementType == "" {
		settlementType = models.SettlementTypeMonthly
	}

	merchant := &models.Merchant{
		Name:           req.Name,
		ContactName:    req.ContactName,
		ContactPhone:   req.ContactPhone,
		Address:        req.Address,
		BusinessLicense: req.BusinessLicense,
		CommissionRate:  commissionRate,
		SettlementType:  settlementType,
		BankName:        req.BankName,
		Status:          models.MerchantStatusActive,
	}

	// 加密银行账号
	if req.BankAccount != nil && *req.BankAccount != "" && s.aes != nil {
		encrypted, err := s.aes.Encrypt(*req.BankAccount)
		if err != nil {
			return nil, err
		}
		merchant.BankAccountEncrypted = &encrypted
	}

	// 加密持卡人姓名
	if req.BankHolder != nil && *req.BankHolder != "" && s.aes != nil {
		encrypted, err := s.aes.Encrypt(*req.BankHolder)
		if err != nil {
			return nil, err
		}
		merchant.BankHolderEncrypted = &encrypted
	}

	if err := s.merchantRepo.Create(ctx, merchant); err != nil {
		return nil, err
	}

	return merchant, nil
}

// UpdateMerchantRequest 更新商户请求
type UpdateMerchantRequest struct {
	Name            string   `json:"name" binding:"required,max=100"`
	ContactName     string   `json:"contact_name" binding:"required,max=50"`
	ContactPhone    string   `json:"contact_phone" binding:"required,max=20"`
	Address         *string  `json:"address"`
	BusinessLicense *string  `json:"business_license"`
	CommissionRate  float64  `json:"commission_rate" binding:"min=0,max=1"`
	SettlementType  string   `json:"settlement_type" binding:"oneof=weekly monthly"`
	BankName        *string  `json:"bank_name"`
	BankAccount     *string  `json:"bank_account"`
	BankHolder      *string  `json:"bank_holder"`
}

// UpdateMerchant 更新商户
func (s *MerchantAdminService) UpdateMerchant(ctx context.Context, id int64, req *UpdateMerchantRequest) error {
	merchant, err := s.merchantRepo.GetByID(ctx, id)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return ErrMerchantNotFound
		}
		return err
	}

	// 检查名称是否与其他商户冲突
	if merchant.Name != req.Name {
		exists, err := s.merchantRepo.ExistsByNameExcludeID(ctx, req.Name, id)
		if err != nil {
			return err
		}
		if exists {
			return ErrMerchantNameExists
		}
	}

	merchant.Name = req.Name
	merchant.ContactName = req.ContactName
	merchant.ContactPhone = req.ContactPhone
	merchant.Address = req.Address
	merchant.BusinessLicense = req.BusinessLicense
	merchant.CommissionRate = req.CommissionRate
	merchant.SettlementType = req.SettlementType
	merchant.BankName = req.BankName

	// 更新银行账号（如果提供了新值）
	if req.BankAccount != nil && *req.BankAccount != "" && s.aes != nil {
		encrypted, err := s.aes.Encrypt(*req.BankAccount)
		if err != nil {
			return err
		}
		merchant.BankAccountEncrypted = &encrypted
	}

	// 更新持卡人姓名（如果提供了新值）
	if req.BankHolder != nil && *req.BankHolder != "" && s.aes != nil {
		encrypted, err := s.aes.Encrypt(*req.BankHolder)
		if err != nil {
			return err
		}
		merchant.BankHolderEncrypted = &encrypted
	}

	return s.merchantRepo.Update(ctx, merchant)
}

// UpdateMerchantStatus 更新商户状态
func (s *MerchantAdminService) UpdateMerchantStatus(ctx context.Context, id int64, status int8) error {
	_, err := s.merchantRepo.GetByID(ctx, id)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return ErrMerchantNotFound
		}
		return err
	}

	return s.merchantRepo.UpdateStatus(ctx, id, status)
}

// DeleteMerchant 删除商户
func (s *MerchantAdminService) DeleteMerchant(ctx context.Context, id int64) error {
	// 检查商户是否存在
	_, err := s.merchantRepo.GetByID(ctx, id)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return ErrMerchantNotFound
		}
		return err
	}

	// 检查商户下是否有场地
	count, err := s.merchantRepo.CountVenues(ctx, id)
	if err != nil {
		return err
	}
	if count > 0 {
		return ErrMerchantHasVenues
	}

	return s.merchantRepo.Delete(ctx, id)
}

// GetMerchant 获取商户详情
func (s *MerchantAdminService) GetMerchant(ctx context.Context, id int64) (*MerchantInfo, error) {
	merchant, err := s.merchantRepo.GetByID(ctx, id)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrMerchantNotFound
		}
		return nil, err
	}

	// 获取统计信息
	stats, _ := s.merchantRepo.GetStatistics(ctx, id)

	return s.toMerchantInfo(merchant, stats), nil
}

// ListMerchants 获取商户列表
func (s *MerchantAdminService) ListMerchants(ctx context.Context, offset, limit int, filters map[string]interface{}) ([]*MerchantInfo, int64, error) {
	merchants, total, err := s.merchantRepo.List(ctx, offset, limit, filters)
	if err != nil {
		return nil, 0, err
	}

	infos := make([]*MerchantInfo, 0, len(merchants))
	for _, m := range merchants {
		stats, _ := s.merchantRepo.GetStatistics(ctx, m.ID)
		infos = append(infos, s.toMerchantInfo(m, stats))
	}

	return infos, total, nil
}

// ListAllMerchants 获取所有商户（用于下拉选择）
func (s *MerchantAdminService) ListAllMerchants(ctx context.Context) ([]*models.Merchant, error) {
	return s.merchantRepo.ListAll(ctx)
}

// toMerchantInfo 转换为商户信息
func (s *MerchantAdminService) toMerchantInfo(merchant *models.Merchant, stats *repository.MerchantStats) *MerchantInfo {
	info := &MerchantInfo{
		ID:              merchant.ID,
		Name:            merchant.Name,
		ContactName:     merchant.ContactName,
		ContactPhone:    merchant.ContactPhone,
		Address:         merchant.Address,
		BusinessLicense: merchant.BusinessLicense,
		CommissionRate:  merchant.CommissionRate,
		SettlementType:  merchant.SettlementType,
		BankName:        merchant.BankName,
		Status:          merchant.Status,
		Stats:           stats,
	}

	// 解密并脱敏银行账号
	if merchant.BankAccountEncrypted != nil && s.aes != nil {
		if decrypted, err := s.aes.Decrypt(*merchant.BankAccountEncrypted); err == nil {
			masked := crypto.MaskBankCard(decrypted)
			info.BankAccount = &masked
		}
	}

	// 解密并脱敏持卡人姓名
	if merchant.BankHolderEncrypted != nil && s.aes != nil {
		if decrypted, err := s.aes.Decrypt(*merchant.BankHolderEncrypted); err == nil {
			// 姓名脱敏：保留第一个字，其余用*
			if len(decrypted) > 0 {
				runes := []rune(decrypted)
				if len(runes) > 1 {
					masked := string(runes[0])
					for i := 1; i < len(runes); i++ {
						masked += "*"
					}
					info.BankHolder = &masked
				} else {
					info.BankHolder = &decrypted
				}
			}
		}
	}

	return info
}
