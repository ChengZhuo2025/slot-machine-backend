// Package upload 提供文件上传服务
package upload

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"mime/multipart"

	"github.com/dumeirei/smart-locker-backend/internal/common/errors"
	"github.com/dumeirei/smart-locker-backend/internal/repository"
	"github.com/dumeirei/smart-locker-backend/pkg/oss"
)

// UploadService 上传服务
type UploadService struct {
	uploader oss.Uploader
	userRepo *repository.UserRepository
}

// NewUploadService 创建上传服务
func NewUploadService(uploader oss.Uploader, userRepo *repository.UserRepository) *UploadService {
	return &UploadService{
		uploader: uploader,
		userRepo: userRepo,
	}
}

const (
	// MaxImageSize 通用图片最大大小（10MB）
	MaxImageSize = 10 * 1024 * 1024
	// MaxAvatarSize 头像最大大小（5MB）
	MaxAvatarSize = 5 * 1024 * 1024
)

// UploadImageRequest 上传图片请求
type UploadImageRequest struct {
	File     *multipart.FileHeader
	FileType string // 文件类型：avatar, hotel, room, product 等
}

// UploadImageResponse 上传图片响应
type UploadImageResponse struct {
	URL      string `json:"url"`
	FileName string `json:"file_name"`
	Size     int64  `json:"size"`
}

// UploadImage 上传图片（通用）
func (s *UploadService) UploadImage(ctx context.Context, req *UploadImageRequest) (*UploadImageResponse, error) {
	if req.File == nil {
		return nil, errors.ErrInvalidParams.WithMessage("请选择要上传的文件")
	}

	// 检查文件大小
	if req.File.Size > MaxImageSize {
		return nil, errors.ErrInvalidParams.WithMessage(fmt.Sprintf("图片大小不能超过 %dMB", MaxImageSize/(1024*1024)))
	}

	// 打开文件
	file, err := req.File.Open()
	if err != nil {
		return nil, errors.ErrOperationFailed.WithMessage("无法打开文件").WithError(err)
	}
	defer file.Close()

	// 读取文件内容到 buffer
	buf := new(bytes.Buffer)
	if _, err := io.Copy(buf, file); err != nil {
		return nil, errors.ErrOperationFailed.WithMessage("读取文件失败").WithError(err)
	}

	// 验证图片文件（检查 magic bytes）
	reader := bytes.NewReader(buf.Bytes())
	if err := oss.ValidateImageFile(req.File.Filename, req.File.Size, reader); err != nil {
		return nil, errors.ErrInvalidParams.WithMessage("文件格式不正确：仅支持 jpg/jpeg/png/gif/webp 格式").WithError(err)
	}

	// 生成对象键
	fileType := req.FileType
	if fileType == "" {
		fileType = "images"
	}
	objectKey := oss.GenerateObjectKey(fileType, req.File.Filename)

	// 上传到 OSS
	reader.Seek(0, io.SeekStart) // 重置读取位置
	url, err := s.uploader.Upload(ctx, objectKey, reader)
	if err != nil {
		return nil, errors.ErrOperationFailed.WithMessage("上传文件失败").WithError(err)
	}

	return &UploadImageResponse{
		URL:      url,
		FileName: req.File.Filename,
		Size:     req.File.Size,
	}, nil
}

// UploadAvatarRequest 上传头像请求
type UploadAvatarRequest struct {
	UserID int64
	File   *multipart.FileHeader
}

// UploadAvatar 上传头像并更新用户信息
func (s *UploadService) UploadAvatar(ctx context.Context, req *UploadAvatarRequest) (*UploadImageResponse, error) {
	if req.File == nil {
		return nil, errors.ErrInvalidParams.WithMessage("请选择要上传的文件")
	}

	// 检查文件大小（头像限制为 5MB）
	if req.File.Size > MaxAvatarSize {
		return nil, errors.ErrInvalidParams.WithMessage(fmt.Sprintf("头像大小不能超过 %dMB", MaxAvatarSize/(1024*1024)))
	}

	// 验证用户存在
	user, err := s.userRepo.GetByID(ctx, req.UserID)
	if err != nil {
		return nil, errors.ErrUserNotFound.WithError(err)
	}

	// 打开文件
	file, err := req.File.Open()
	if err != nil {
		return nil, errors.ErrOperationFailed.WithMessage("无法打开文件").WithError(err)
	}
	defer file.Close()

	// 读取文件内容到 buffer
	buf := new(bytes.Buffer)
	if _, err := io.Copy(buf, file); err != nil {
		return nil, errors.ErrOperationFailed.WithMessage("读取文件失败").WithError(err)
	}

	// 验证图片文件（检查 magic bytes）
	reader := bytes.NewReader(buf.Bytes())
	if err := oss.ValidateImageFile(req.File.Filename, req.File.Size, reader); err != nil {
		return nil, errors.ErrInvalidParams.WithMessage("文件格式不正确：仅支持 jpg/jpeg/png/gif/webp 格式").WithError(err)
	}

	// 生成对象键
	objectKey := oss.GenerateObjectKey("avatar", req.File.Filename)

	// 上传到 OSS
	reader.Seek(0, io.SeekStart) // 重置读取位置
	url, err := s.uploader.Upload(ctx, objectKey, reader)
	if err != nil {
		return nil, errors.ErrOperationFailed.WithMessage("上传文件失败").WithError(err)
	}

	// 更新用户头像字段
	fields := map[string]interface{}{
		"avatar": url,
	}
	if err := s.userRepo.UpdateFields(ctx, user.ID, fields); err != nil {
		return nil, errors.ErrDatabaseError.WithMessage("更新用户头像失败").WithError(err)
	}

	return &UploadImageResponse{
		URL:      url,
		FileName: req.File.Filename,
		Size:     req.File.Size,
	}, nil
}
