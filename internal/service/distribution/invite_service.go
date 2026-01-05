// Package distribution 分销服务
package distribution

import (
	"context"
	"crypto/md5"
	"encoding/base64"
	"encoding/hex"
	"errors"
	"fmt"
	"net/url"
	"time"

	"github.com/dumeirei/smart-locker-backend/internal/models"
	"github.com/dumeirei/smart-locker-backend/internal/repository"
)

// InviteService 邀请服务
type InviteService struct {
	distributorRepo *repository.DistributorRepository
	baseURL         string // 邀请链接基础URL
}

// NewInviteService 创建邀请服务
func NewInviteService(distributorRepo *repository.DistributorRepository, baseURL string) *InviteService {
	if baseURL == "" {
		baseURL = "https://app.example.com"
	}
	return &InviteService{
		distributorRepo: distributorRepo,
		baseURL:         baseURL,
	}
}

// SetBaseURL 设置基础URL
func (s *InviteService) SetBaseURL(baseURL string) {
	s.baseURL = baseURL
}

// InviteInfo 邀请信息
type InviteInfo struct {
	InviteCode    string `json:"invite_code"`    // 邀请码
	InviteLink    string `json:"invite_link"`    // 邀请链接
	QRCodeURL     string `json:"qrcode_url"`     // 二维码URL
	ShortLink     string `json:"short_link"`     // 短链接
	PosterURL     string `json:"poster_url"`     // 邀请海报URL
	DistributorID int64  `json:"distributor_id"` // 分销商ID
	UserName      string `json:"user_name"`      // 分销商名称
}

// GenerateInviteInfo 生成邀请信息
func (s *InviteService) GenerateInviteInfo(ctx context.Context, distributorID int64) (*InviteInfo, error) {
	distributor, err := s.distributorRepo.GetByIDWithUser(ctx, distributorID)
	if err != nil {
		return nil, err
	}

	if distributor.Status != models.DistributorStatusApproved {
		return nil, errors.New("分销商尚未审核通过")
	}

	inviteLink := s.generateInviteLink(distributor.InviteCode)
	qrCodeURL := s.generateQRCodeURL(inviteLink)
	shortLink := s.generateShortLink(distributor.InviteCode)
	posterURL := s.generatePosterURL(distributor.InviteCode)

	userName := ""
	if distributor.User != nil {
		userName = distributor.User.Nickname
	}

	return &InviteInfo{
		InviteCode:    distributor.InviteCode,
		InviteLink:    inviteLink,
		QRCodeURL:     qrCodeURL,
		ShortLink:     shortLink,
		PosterURL:     posterURL,
		DistributorID: distributor.ID,
		UserName:      userName,
	}, nil
}

// generateInviteLink 生成邀请链接
func (s *InviteService) generateInviteLink(inviteCode string) string {
	return fmt.Sprintf("%s/invite/%s", s.baseURL, inviteCode)
}

// generateQRCodeURL 生成二维码URL
// 使用第三方二维码生成服务或自建服务
func (s *InviteService) generateQRCodeURL(link string) string {
	// 使用 URL 编码
	encodedLink := url.QueryEscape(link)
	// 这里可以使用自建的二维码服务或第三方服务
	return fmt.Sprintf("%s/api/qrcode?data=%s&size=200", s.baseURL, encodedLink)
}

// generateShortLink 生成短链接
func (s *InviteService) generateShortLink(inviteCode string) string {
	// 简单的短链接生成，实际可以使用专门的短链服务
	hash := md5.Sum([]byte(inviteCode + time.Now().Format("20060102")))
	shortCode := hex.EncodeToString(hash[:])[:8]
	return fmt.Sprintf("%s/s/%s", s.baseURL, shortCode)
}

// generatePosterURL 生成邀请海报URL
func (s *InviteService) generatePosterURL(inviteCode string) string {
	// 海报生成服务URL
	return fmt.Sprintf("%s/api/poster?code=%s", s.baseURL, inviteCode)
}

// ValidateInviteCode 验证邀请码是否有效
func (s *InviteService) ValidateInviteCode(ctx context.Context, inviteCode string) (*models.Distributor, error) {
	distributor, err := s.distributorRepo.GetByInviteCodeWithUser(ctx, inviteCode)
	if err != nil {
		return nil, errors.New("邀请码无效")
	}

	if distributor.Status != models.DistributorStatusApproved {
		return nil, errors.New("邀请人尚未通过审核")
	}

	return distributor, nil
}

// GetInviteCodeFromLink 从邀请链接中提取邀请码
func (s *InviteService) GetInviteCodeFromLink(link string) (string, error) {
	parsedURL, err := url.Parse(link)
	if err != nil {
		return "", errors.New("无效的邀请链接")
	}

	// 从路径中提取邀请码
	// 期望格式: /invite/{code} 或 /s/{code}
	path := parsedURL.Path
	if len(path) > 8 && path[:8] == "/invite/" {
		return path[8:], nil
	}
	if len(path) > 3 && path[:3] == "/s/" {
		// 短链接需要通过其他方式解析
		return "", errors.New("短链接请通过专门接口解析")
	}

	return "", errors.New("无效的邀请链接格式")
}

// ShareContent 分享内容
type ShareContent struct {
	Title       string `json:"title"`        // 分享标题
	Description string `json:"description"`  // 分享描述
	ImageURL    string `json:"image_url"`    // 分享图片
	Link        string `json:"link"`         // 分享链接
	WechatPath  string `json:"wechat_path"`  // 微信小程序路径
}

// GenerateShareContent 生成分享内容
func (s *InviteService) GenerateShareContent(ctx context.Context, distributorID int64) (*ShareContent, error) {
	inviteInfo, err := s.GenerateInviteInfo(ctx, distributorID)
	if err != nil {
		return nil, err
	}

	return &ShareContent{
		Title:       "邀请您加入，一起赚佣金",
		Description: fmt.Sprintf("%s邀请您加入平台，享受专属优惠", inviteInfo.UserName),
		ImageURL:    inviteInfo.PosterURL,
		Link:        inviteInfo.InviteLink,
		WechatPath:  fmt.Sprintf("/pages/register/index?code=%s", inviteInfo.InviteCode),
	}, nil
}

// TrackClick 记录邀请链接点击
// 可以用于统计分析
func (s *InviteService) TrackClick(ctx context.Context, inviteCode string, source string) error {
	// 验证邀请码有效性
	_, err := s.ValidateInviteCode(ctx, inviteCode)
	if err != nil {
		return err
	}

	// TODO: 记录点击统计
	// 可以存储到 Redis 或数据库进行统计分析

	return nil
}

// EncodeInviteCode 编码邀请码（用于URL）
func (s *InviteService) EncodeInviteCode(code string) string {
	return base64.URLEncoding.EncodeToString([]byte(code))
}

// DecodeInviteCode 解码邀请码
func (s *InviteService) DecodeInviteCode(encoded string) (string, error) {
	decoded, err := base64.URLEncoding.DecodeString(encoded)
	if err != nil {
		return "", errors.New("无效的邀请码")
	}
	return string(decoded), nil
}
