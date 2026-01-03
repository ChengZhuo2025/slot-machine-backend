// Package admin 提供管理员相关服务
package admin

import (
	"context"
	"fmt"

	"github.com/dumeirei/smart-locker-backend/internal/common/qrcode"
	"github.com/dumeirei/smart-locker-backend/internal/repository"
)

// DeviceQRCodeService 设备二维码服务
type DeviceQRCodeService struct {
	deviceRepo *repository.DeviceRepository
	qrGen      *qrcode.Generator
	baseURL    string
}

// NewDeviceQRCodeService 创建设备二维码服务
func NewDeviceQRCodeService(deviceRepo *repository.DeviceRepository, baseURL string) *DeviceQRCodeService {
	return &DeviceQRCodeService{
		deviceRepo: deviceRepo,
		qrGen:      qrcode.NewGenerator(qrcode.WithSize(300), qrcode.WithRecoveryLevel(qrcode.High)),
		baseURL:    baseURL,
	}
}

// QRCodeInfo 二维码信息
type QRCodeInfo struct {
	DeviceID   int64  `json:"device_id"`
	DeviceNo   string `json:"device_no"`
	DeviceName string `json:"device_name"`
	QRCodeURL  string `json:"qr_code_url"`
	DataURL    string `json:"data_url"`
}

// GenerateQRCode 为设备生成二维码
func (s *DeviceQRCodeService) GenerateQRCode(ctx context.Context, deviceID int64) (*QRCodeInfo, error) {
	device, err := s.deviceRepo.GetByID(ctx, deviceID)
	if err != nil {
		return nil, ErrDeviceNotFound
	}

	// 生成二维码内容 URL
	qrCodeURL := s.generateQRCodeURL(device.DeviceNo)

	// 生成 Data URL 格式的二维码图片
	dataURL, err := s.qrGen.GenerateDataURL(qrCodeURL)
	if err != nil {
		return nil, fmt.Errorf("生成二维码失败: %w", err)
	}

	// 更新设备的二维码字段
	if err := s.deviceRepo.UpdateQRCode(ctx, deviceID, qrCodeURL); err != nil {
		return nil, err
	}

	return &QRCodeInfo{
		DeviceID:   device.ID,
		DeviceNo:   device.DeviceNo,
		DeviceName: device.Name,
		QRCodeURL:  qrCodeURL,
		DataURL:    dataURL,
	}, nil
}

// GetQRCodeImage 获取二维码图片（PNG 格式）
func (s *DeviceQRCodeService) GetQRCodeImage(ctx context.Context, deviceID int64) ([]byte, error) {
	device, err := s.deviceRepo.GetByID(ctx, deviceID)
	if err != nil {
		return nil, ErrDeviceNotFound
	}

	qrCodeURL := device.QRCode
	if qrCodeURL == "" {
		qrCodeURL = s.generateQRCodeURL(device.DeviceNo)
	}

	return s.qrGen.GeneratePNG(qrCodeURL)
}

// GetQRCodeDataURL 获取二维码 Data URL
func (s *DeviceQRCodeService) GetQRCodeDataURL(ctx context.Context, deviceID int64) (string, error) {
	device, err := s.deviceRepo.GetByID(ctx, deviceID)
	if err != nil {
		return "", ErrDeviceNotFound
	}

	qrCodeURL := device.QRCode
	if qrCodeURL == "" {
		qrCodeURL = s.generateQRCodeURL(device.DeviceNo)
	}

	return s.qrGen.GenerateDataURL(qrCodeURL)
}

// BatchGenerateQRCodes 批量生成二维码
func (s *DeviceQRCodeService) BatchGenerateQRCodes(ctx context.Context, deviceIDs []int64) ([]*QRCodeInfo, error) {
	results := make([]*QRCodeInfo, 0, len(deviceIDs))

	for _, deviceID := range deviceIDs {
		info, err := s.GenerateQRCode(ctx, deviceID)
		if err != nil {
			continue
		}
		results = append(results, info)
	}

	return results, nil
}

// RegenerateQRCode 重新生成二维码
func (s *DeviceQRCodeService) RegenerateQRCode(ctx context.Context, deviceID int64) (*QRCodeInfo, error) {
	return s.GenerateQRCode(ctx, deviceID)
}

// generateQRCodeURL 生成二维码 URL
func (s *DeviceQRCodeService) generateQRCodeURL(deviceNo string) string {
	return fmt.Sprintf("%s/scan/%s", s.baseURL, deviceNo)
}

// BatchDownloadQRCodes 批量下载二维码
type BatchQRCodeResult struct {
	DeviceNo string `json:"device_no"`
	Name     string `json:"name"`
	Data     []byte `json:"data"`
}

// BatchDownload 批量下载二维码图片
func (s *DeviceQRCodeService) BatchDownload(ctx context.Context, deviceIDs []int64) ([]*BatchQRCodeResult, error) {
	results := make([]*BatchQRCodeResult, 0, len(deviceIDs))

	for _, deviceID := range deviceIDs {
		device, err := s.deviceRepo.GetByID(ctx, deviceID)
		if err != nil {
			continue
		}

		qrCodeURL := device.QRCode
		if qrCodeURL == "" {
			qrCodeURL = s.generateQRCodeURL(device.DeviceNo)
		}

		data, err := s.qrGen.GeneratePNG(qrCodeURL)
		if err != nil {
			continue
		}

		results = append(results, &BatchQRCodeResult{
			DeviceNo: device.DeviceNo,
			Name:     device.Name,
			Data:     data,
		})
	}

	return results, nil
}
