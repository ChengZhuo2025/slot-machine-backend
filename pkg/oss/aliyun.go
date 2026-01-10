// Package oss 对象存储服务
package oss

import (
	"context"
	"crypto/md5"
	"encoding/hex"
	"fmt"
	"io"
	"net/http"
	"path"
	"strings"
	"time"

	"github.com/aliyun/aliyun-oss-go-sdk/oss"
)

// Uploader 上传器接口
type Uploader interface {
	Upload(ctx context.Context, objectKey string, reader io.Reader) (string, error)
	UploadFile(ctx context.Context, objectKey, filePath string) (string, error)
	Delete(ctx context.Context, objectKey string) error
	GetURL(objectKey string) string
	GetSignedURL(objectKey string, expires time.Duration) (string, error)
}

// AliyunConfig 阿里云 OSS 配置
type AliyunConfig struct {
	Endpoint        string
	AccessKeyID     string
	AccessKeySecret string
	BucketName      string
	Domain          string // 自定义域名（可选）
	BasePath        string // 基础路径，如 "uploads/"
}

// AliyunUploader 阿里云 OSS 上传器
type AliyunUploader struct {
	client   *oss.Client
	bucket   *oss.Bucket
	config   *AliyunConfig
}

// NewAliyunUploader 创建阿里云 OSS 上传器
func NewAliyunUploader(config *AliyunConfig) (*AliyunUploader, error) {
	client, err := oss.New(config.Endpoint, config.AccessKeyID, config.AccessKeySecret)
	if err != nil {
		return nil, fmt.Errorf("创建 OSS 客户端失败: %v", err)
	}

	bucket, err := client.Bucket(config.BucketName)
	if err != nil {
		return nil, fmt.Errorf("获取 Bucket 失败: %v", err)
	}

	return &AliyunUploader{
		client: client,
		bucket: bucket,
		config: config,
	}, nil
}

// Upload 上传文件
func (u *AliyunUploader) Upload(ctx context.Context, objectKey string, reader io.Reader) (string, error) {
	fullKey := u.getFullKey(objectKey)

	err := u.bucket.PutObject(fullKey, reader)
	if err != nil {
		return "", fmt.Errorf("上传文件失败: %v", err)
	}

	return u.GetURL(objectKey), nil
}

// UploadFile 上传本地文件
func (u *AliyunUploader) UploadFile(ctx context.Context, objectKey, filePath string) (string, error) {
	fullKey := u.getFullKey(objectKey)

	err := u.bucket.PutObjectFromFile(fullKey, filePath)
	if err != nil {
		return "", fmt.Errorf("上传文件失败: %v", err)
	}

	return u.GetURL(objectKey), nil
}

// Delete 删除文件
func (u *AliyunUploader) Delete(ctx context.Context, objectKey string) error {
	fullKey := u.getFullKey(objectKey)
	return u.bucket.DeleteObject(fullKey)
}

// GetURL 获取文件 URL
func (u *AliyunUploader) GetURL(objectKey string) string {
	fullKey := u.getFullKey(objectKey)

	if u.config.Domain != "" {
		return fmt.Sprintf("%s/%s", strings.TrimSuffix(u.config.Domain, "/"), fullKey)
	}

	return fmt.Sprintf("https://%s.%s/%s", u.config.BucketName, u.config.Endpoint, fullKey)
}

// GetSignedURL 获取带签名的临时 URL
func (u *AliyunUploader) GetSignedURL(objectKey string, expires time.Duration) (string, error) {
	fullKey := u.getFullKey(objectKey)
	return u.bucket.SignURL(fullKey, oss.HTTPGet, int64(expires.Seconds()))
}

// getFullKey 获取完整的对象键
func (u *AliyunUploader) getFullKey(objectKey string) string {
	if u.config.BasePath == "" {
		return objectKey
	}
	return path.Join(u.config.BasePath, objectKey)
}

// GenerateObjectKey 生成对象键
func GenerateObjectKey(prefix, filename string) string {
	ext := path.Ext(filename)
	now := time.Now()

	// 使用时间戳和原文件名生成唯一键
	hash := md5.Sum([]byte(fmt.Sprintf("%s_%d_%s", filename, now.UnixNano(), time.Now().Format(time.RFC3339Nano))))
	hashStr := hex.EncodeToString(hash[:])[:16]

	return fmt.Sprintf("%s/%s/%s%s",
		prefix,
		now.Format("2006/01/02"),
		hashStr,
		ext,
	)
}

// GetContentType 根据文件扩展名获取 Content-Type
func GetContentType(filename string) string {
	ext := strings.ToLower(path.Ext(filename))
	contentTypes := map[string]string{
		".jpg":  "image/jpeg",
		".jpeg": "image/jpeg",
		".png":  "image/png",
		".gif":  "image/gif",
		".webp": "image/webp",
		".svg":  "image/svg+xml",
		".pdf":  "application/pdf",
		".doc":  "application/msword",
		".docx": "application/vnd.openxmlformats-officedocument.wordprocessingml.document",
		".xls":  "application/vnd.ms-excel",
		".xlsx": "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet",
		".mp4":  "video/mp4",
		".mp3":  "audio/mpeg",
		".zip":  "application/zip",
		".txt":  "text/plain",
		".json": "application/json",
	}

	if ct, ok := contentTypes[ext]; ok {
		return ct
	}
	return "application/octet-stream"
}

// ValidateImageFile 验证图片文件
func ValidateImageFile(filename string, maxSize int64, reader io.Reader) error {
	// 检查文件扩展名
	ext := strings.ToLower(path.Ext(filename))
	validExts := map[string]bool{
		".jpg": true, ".jpeg": true, ".png": true, ".gif": true, ".webp": true,
	}
	if !validExts[ext] {
		return fmt.Errorf("不支持的图片格式: %s", ext)
	}

	// 读取文件头判断真实类型
	header := make([]byte, 512)
	n, err := reader.Read(header)
	if err != nil && err != io.EOF {
		return fmt.Errorf("读取文件失败: %v", err)
	}

	contentType := http.DetectContentType(header[:n])
	if !strings.HasPrefix(contentType, "image/") {
		return fmt.Errorf("文件不是有效的图片")
	}

	return nil
}

// MockUploader 模拟上传器（用于开发/测试）
type MockUploader struct {
	Files map[string][]byte
}

// NewMockUploader 创建模拟上传器
func NewMockUploader() *MockUploader {
	return &MockUploader{
		Files: make(map[string][]byte),
	}
}

// Upload 模拟上传
func (u *MockUploader) Upload(ctx context.Context, objectKey string, reader io.Reader) (string, error) {
	data, err := io.ReadAll(reader)
	if err != nil {
		return "", err
	}
	u.Files[objectKey] = data
	return u.GetURL(objectKey), nil
}

// UploadFile 模拟上传本地文件
func (u *MockUploader) UploadFile(ctx context.Context, objectKey, filePath string) (string, error) {
	u.Files[objectKey] = []byte(filePath) // 仅存储路径
	return u.GetURL(objectKey), nil
}

// Delete 模拟删除
func (u *MockUploader) Delete(ctx context.Context, objectKey string) error {
	delete(u.Files, objectKey)
	return nil
}

// GetURL 获取模拟 URL
func (u *MockUploader) GetURL(objectKey string) string {
	return fmt.Sprintf("https://mock-oss.example.com/%s", objectKey)
}

// GetSignedURL 获取模拟签名 URL
func (u *MockUploader) GetSignedURL(objectKey string, expires time.Duration) (string, error) {
	return fmt.Sprintf("%s?expires=%d", u.GetURL(objectKey), time.Now().Add(expires).Unix()), nil
}
