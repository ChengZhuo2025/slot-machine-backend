// Package qrcode 二维码生成功能单元测试
package qrcode

import (
	"bytes"
	"encoding/base64"
	"image/png"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// ==================== NewGenerator 测试 ====================

func TestNewGenerator_Default(t *testing.T) {
	gen := NewGenerator()
	assert.NotNil(t, gen)
	assert.Equal(t, 256, gen.size)
	assert.Equal(t, Medium, gen.recoveryLevel)
}

func TestNewGenerator_WithSize(t *testing.T) {
	sizes := []int{128, 256, 512, 1024}

	for _, size := range sizes {
		t.Run(string(rune(size)), func(t *testing.T) {
			gen := NewGenerator(WithSize(size))
			assert.Equal(t, size, gen.size)
		})
	}
}

func TestNewGenerator_WithRecoveryLevel(t *testing.T) {
	levels := []RecoveryLevel{Low, Medium, High, Highest}

	for _, level := range levels {
		t.Run(string(rune(level)), func(t *testing.T) {
			gen := NewGenerator(WithRecoveryLevel(level))
			assert.Equal(t, level, gen.recoveryLevel)
		})
	}
}

func TestNewGenerator_MultipleOptions(t *testing.T) {
	gen := NewGenerator(
		WithSize(512),
		WithRecoveryLevel(High),
	)
	assert.Equal(t, 512, gen.size)
	assert.Equal(t, High, gen.recoveryLevel)
}

// ==================== Generate 测试 ====================

func TestGenerator_Generate_Success(t *testing.T) {
	gen := NewGenerator()

	tests := []struct {
		name    string
		content string
	}{
		{"Simple text", "Hello, World!"},
		{"URL", "https://example.com"},
		{"Chinese", "你好世界"},
		{"Long text", strings.Repeat("测试", 100)},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			img, err := gen.Generate(tt.content)
			require.NoError(t, err)
			assert.NotNil(t, img)

			// 验证图片类型和尺寸
			bounds := img.Bounds()
			assert.Equal(t, 256, bounds.Dx())
			assert.Equal(t, 256, bounds.Dy())
		})
	}
}

func TestGenerator_Generate_DifferentSizes(t *testing.T) {
	content := "Test Content"
	sizes := []int{128, 256, 512}

	for _, size := range sizes {
		t.Run(string(rune(size)), func(t *testing.T) {
			gen := NewGenerator(WithSize(size))
			img, err := gen.Generate(content)
			require.NoError(t, err)

			bounds := img.Bounds()
			// 注意：实际生成的二维码可能不是精确的size，因为二维码有模块对齐等要求
			// 但应该接近指定的大小
			assert.Greater(t, bounds.Dx(), 0)
			assert.Greater(t, bounds.Dy(), 0)
		})
	}
}

// ==================== GeneratePNG 测试 ====================

func TestGenerator_GeneratePNG_Success(t *testing.T) {
	gen := NewGenerator()
	content := "https://example.com"

	data, err := gen.GeneratePNG(content)
	require.NoError(t, err)
	assert.NotEmpty(t, data)

	// 验证是有效的PNG
	img, err := png.Decode(bytes.NewReader(data))
	require.NoError(t, err)
	assert.NotNil(t, img)
}

func TestGenerator_GeneratePNG_DifferentContents(t *testing.T) {
	gen := NewGenerator()

	tests := []string{
		"Short",
		"https://example.com/very/long/url/path?param1=value1&param2=value2",
		"中文内容测试",
		"12345",
	}

	for _, content := range tests {
		t.Run(content, func(t *testing.T) {
			data, err := gen.GeneratePNG(content)
			require.NoError(t, err)
			assert.NotEmpty(t, data)

			// 验证PNG格式
			_, err = png.Decode(bytes.NewReader(data))
			require.NoError(t, err)
		})
	}
}

// ==================== GenerateBase64 测试 ====================

func TestGenerator_GenerateBase64_Success(t *testing.T) {
	gen := NewGenerator()
	content := "Test content"

	b64, err := gen.GenerateBase64(content)
	require.NoError(t, err)
	assert.NotEmpty(t, b64)

	// 验证是有效的base64
	data, err := base64.StdEncoding.DecodeString(b64)
	require.NoError(t, err)
	assert.NotEmpty(t, data)

	// 验证解码后是有效的PNG
	_, err = png.Decode(bytes.NewReader(data))
	require.NoError(t, err)
}

// ==================== GenerateDataURL 测试 ====================

func TestGenerator_GenerateDataURL_Success(t *testing.T) {
	gen := NewGenerator()
	content := "Test content"

	dataURL, err := gen.GenerateDataURL(content)
	require.NoError(t, err)
	assert.NotEmpty(t, dataURL)

	// 验证格式
	assert.True(t, strings.HasPrefix(dataURL, "data:image/png;base64,"))

	// 提取base64部分
	b64 := strings.TrimPrefix(dataURL, "data:image/png;base64,")
	data, err := base64.StdEncoding.DecodeString(b64)
	require.NoError(t, err)

	// 验证是有效的PNG
	_, err = png.Decode(bytes.NewReader(data))
	require.NoError(t, err)
}

// ==================== WriteToFile 测试 ====================

func TestGenerator_WriteToFile_Success(t *testing.T) {
	gen := NewGenerator()
	content := "Test content"

	// 创建临时目录
	tempDir := t.TempDir()
	filePath := filepath.Join(tempDir, "test_qr.png")

	err := gen.WriteToFile(content, filePath)
	require.NoError(t, err)

	// 验证文件存在
	_, err = os.Stat(filePath)
	require.NoError(t, err)

	// 验证文件内容
	data, err := os.ReadFile(filePath)
	require.NoError(t, err)
	assert.NotEmpty(t, data)

	// 验证是有效的PNG
	_, err = png.Decode(bytes.NewReader(data))
	require.NoError(t, err)
}

func TestGenerator_WriteToFile_CreateDirectory(t *testing.T) {
	gen := NewGenerator()
	content := "Test content"

	// 创建临时目录
	tempDir := t.TempDir()
	// 使用不存在的子目录
	filePath := filepath.Join(tempDir, "subdir", "nested", "test_qr.png")

	err := gen.WriteToFile(content, filePath)
	require.NoError(t, err)

	// 验证目录和文件都被创建
	_, err = os.Stat(filePath)
	require.NoError(t, err)
}

// ==================== WriteToWriter 测试 ====================

func TestGenerator_WriteToWriter_Success(t *testing.T) {
	gen := NewGenerator()
	content := "Test content"

	var buf bytes.Buffer
	err := gen.WriteToWriter(content, &buf)
	require.NoError(t, err)
	assert.NotEmpty(t, buf.Bytes())

	// 验证是有效的PNG
	_, err = png.Decode(&buf)
	require.NoError(t, err)
}

// ==================== GenerateToBuffer 测试 ====================

func TestGenerator_GenerateToBuffer_Success(t *testing.T) {
	gen := NewGenerator()
	content := "Test content"

	buf, err := gen.GenerateToBuffer(content)
	require.NoError(t, err)
	assert.NotNil(t, buf)
	assert.NotEmpty(t, buf.Bytes())

	// 验证是有效的PNG
	_, err = png.Decode(buf)
	require.NoError(t, err)
}

// ==================== BatchGenerate 测试 ====================

func TestGenerator_BatchGenerate_Success(t *testing.T) {
	gen := NewGenerator()

	contents := []string{
		"Content 1",
		"Content 2",
		"Content 3",
	}

	results, err := gen.BatchGenerate(contents)
	require.NoError(t, err)
	assert.Equal(t, len(contents), len(results))

	for _, content := range contents {
		data, ok := results[content]
		assert.True(t, ok, "应该包含 %s 的结果", content)
		assert.NotEmpty(t, data)

		// 验证是有效的PNG
		_, err := png.Decode(bytes.NewReader(data))
		require.NoError(t, err, "生成的二维码应该是有效的PNG: %s", content)
	}
}

func TestGenerator_BatchGenerate_Empty(t *testing.T) {
	gen := NewGenerator()

	results, err := gen.BatchGenerate([]string{})
	require.NoError(t, err)
	assert.Empty(t, results)
}

// ==================== RecoveryLevel 测试 ====================

func TestGenerator_DifferentRecoveryLevels(t *testing.T) {
	content := "Test content"
	levels := []RecoveryLevel{Low, Medium, High, Highest}

	for _, level := range levels {
		t.Run(string(rune(level)), func(t *testing.T) {
			gen := NewGenerator(WithRecoveryLevel(level))
			data, err := gen.GeneratePNG(content)
			require.NoError(t, err)
			assert.NotEmpty(t, data)

			// 验证PNG有效
			_, err = png.Decode(bytes.NewReader(data))
			require.NoError(t, err)
		})
	}
}

// ==================== 边界条件测试 ====================

func TestGenerator_EmptyContent(t *testing.T) {
	gen := NewGenerator()

	// 空内容应该返回错误（底层库不支持空内容）
	img, err := gen.Generate("")
	assert.Error(t, err)
	assert.Nil(t, img)
	assert.Contains(t, err.Error(), "no data to encode")
}

func TestGenerator_VeryLongContent(t *testing.T) {
	gen := NewGenerator()

	// 生成很长的内容
	longContent := strings.Repeat("Long content test. ", 100)

	img, err := gen.Generate(longContent)
	require.NoError(t, err)
	assert.NotNil(t, img)
}

func TestGenerator_SpecialCharacters(t *testing.T) {
	gen := NewGenerator()

	contents := []string{
		"!@#$%^&*()",
		"<html>test</html>",
		"{\"key\": \"value\"}",
		"Line1\nLine2\nLine3",
		"Tab\tSeparated\tValues",
	}

	for _, content := range contents {
		t.Run(content, func(t *testing.T) {
			img, err := gen.Generate(content)
			require.NoError(t, err)
			assert.NotNil(t, img)
		})
	}
}

// ==================== 一致性测试 ====================

func TestGenerator_ConsistentOutput(t *testing.T) {
	gen := NewGenerator()
	content := "Consistent test"

	// 多次生成相同内容应该产生相同的二维码
	data1, err := gen.GeneratePNG(content)
	require.NoError(t, err)

	data2, err := gen.GeneratePNG(content)
	require.NoError(t, err)

	assert.Equal(t, data1, data2, "相同内容应该生成相同的二维码")
}

func TestGenerator_DifferentContentsDifferentOutput(t *testing.T) {
	gen := NewGenerator()

	data1, err := gen.GeneratePNG("Content A")
	require.NoError(t, err)

	data2, err := gen.GeneratePNG("Content B")
	require.NoError(t, err)

	assert.NotEqual(t, data1, data2, "不同内容应该生成不同的二维码")
}

// ==================== 图片属性测试 ====================

func TestGenerator_ImageIsSquare(t *testing.T) {
	gen := NewGenerator()

	img, err := gen.Generate("Square test")
	require.NoError(t, err)

	bounds := img.Bounds()
	assert.Equal(t, bounds.Dx(), bounds.Dy(), "二维码应该是正方形")
}

func TestGenerator_ImageIsNotNil(t *testing.T) {
	gen := NewGenerator()

	img, err := gen.Generate("Test")
	require.NoError(t, err)
	assert.NotNil(t, img)

	// 验证可以访问像素
	at := img.At(0, 0)
	assert.NotNil(t, at)
}

// ==================== 性能测试 ====================

func BenchmarkGenerate(b *testing.B) {
	gen := NewGenerator()
	content := "https://example.com/test"

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = gen.Generate(content)
	}
}

func BenchmarkGeneratePNG(b *testing.B) {
	gen := NewGenerator()
	content := "https://example.com/test"

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = gen.GeneratePNG(content)
	}
}

func BenchmarkGenerateBase64(b *testing.B) {
	gen := NewGenerator()
	content := "https://example.com/test"

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = gen.GenerateBase64(content)
	}
}

func BenchmarkBatchGenerate(b *testing.B) {
	gen := NewGenerator()
	contents := []string{
		"Content 1",
		"Content 2",
		"Content 3",
		"Content 4",
		"Content 5",
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = gen.BatchGenerate(contents)
	}
}
