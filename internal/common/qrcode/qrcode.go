// Package qrcode 提供二维码生成功能
package qrcode

import (
	"bytes"
	"encoding/base64"
	"fmt"
	"image"
	"image/png"
	"io"
	"os"
	"path/filepath"

	"github.com/skip2/go-qrcode"
)

// RecoveryLevel 纠错级别
type RecoveryLevel int

const (
	// Low 7% 纠错
	Low RecoveryLevel = iota
	// Medium 15% 纠错
	Medium
	// High 25% 纠错
	High
	// Highest 30% 纠错
	Highest
)

// Generator 二维码生成器
type Generator struct {
	size          int           // 二维码尺寸（像素）
	recoveryLevel RecoveryLevel // 纠错级别
}

// Option 生成器选项
type Option func(*Generator)

// WithSize 设置二维码尺寸
func WithSize(size int) Option {
	return func(g *Generator) {
		g.size = size
	}
}

// WithRecoveryLevel 设置纠错级别
func WithRecoveryLevel(level RecoveryLevel) Option {
	return func(g *Generator) {
		g.recoveryLevel = level
	}
}

// NewGenerator 创建二维码生成器
func NewGenerator(opts ...Option) *Generator {
	g := &Generator{
		size:          256,
		recoveryLevel: Medium,
	}
	for _, opt := range opts {
		opt(g)
	}
	return g
}

// toQRCodeLevel 转换纠错级别
func (g *Generator) toQRCodeLevel() qrcode.RecoveryLevel {
	switch g.recoveryLevel {
	case Low:
		return qrcode.Low
	case Medium:
		return qrcode.Medium
	case High:
		return qrcode.High
	case Highest:
		return qrcode.Highest
	default:
		return qrcode.Medium
	}
}

// Generate 生成二维码图片
func (g *Generator) Generate(content string) (image.Image, error) {
	qr, err := qrcode.New(content, g.toQRCodeLevel())
	if err != nil {
		return nil, fmt.Errorf("创建二维码失败: %w", err)
	}
	return qr.Image(g.size), nil
}

// GeneratePNG 生成 PNG 格式二维码
func (g *Generator) GeneratePNG(content string) ([]byte, error) {
	return qrcode.Encode(content, g.toQRCodeLevel(), g.size)
}

// GenerateBase64 生成 Base64 编码的二维码
func (g *Generator) GenerateBase64(content string) (string, error) {
	data, err := g.GeneratePNG(content)
	if err != nil {
		return "", err
	}
	return base64.StdEncoding.EncodeToString(data), nil
}

// GenerateDataURL 生成 Data URL 格式的二维码
func (g *Generator) GenerateDataURL(content string) (string, error) {
	b64, err := g.GenerateBase64(content)
	if err != nil {
		return "", err
	}
	return "data:image/png;base64," + b64, nil
}

// WriteToFile 将二维码写入文件
func (g *Generator) WriteToFile(content, filePath string) error {
	dir := filepath.Dir(filePath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("创建目录失败: %w", err)
	}

	data, err := g.GeneratePNG(content)
	if err != nil {
		return err
	}

	return os.WriteFile(filePath, data, 0644)
}

// WriteToWriter 将二维码写入 Writer
func (g *Generator) WriteToWriter(content string, w io.Writer) error {
	img, err := g.Generate(content)
	if err != nil {
		return err
	}
	return png.Encode(w, img)
}

// GenerateToBuffer 生成二维码到缓冲区
func (g *Generator) GenerateToBuffer(content string) (*bytes.Buffer, error) {
	buf := new(bytes.Buffer)
	if err := g.WriteToWriter(content, buf); err != nil {
		return nil, err
	}
	return buf, nil
}

// BatchGenerate 批量生成二维码
func (g *Generator) BatchGenerate(contents []string) (map[string][]byte, error) {
	results := make(map[string][]byte, len(contents))
	for _, content := range contents {
		data, err := g.GeneratePNG(content)
		if err != nil {
			return nil, fmt.Errorf("生成二维码 %s 失败: %w", content, err)
		}
		results[content] = data
	}
	return results, nil
}
