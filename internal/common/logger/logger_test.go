// Package logger 日志模块单元测试
package logger

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/dumeirei/smart-locker-backend/internal/common/config"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

// ==================== Init 函数测试 ====================

func TestInit_ConsoleFormat(t *testing.T) {
	cfg := &config.LoggerConfig{
		Level:  "debug",
		Format: "console",
		Output: "stdout",
		Caller: true,
	}

	err := Init(cfg)
	assert.NoError(t, err)
	assert.NotNil(t, log)
	assert.NotNil(t, sugar)
}

func TestInit_JSONFormat(t *testing.T) {
	cfg := &config.LoggerConfig{
		Level:  "info",
		Format: "json",
		Output: "stdout",
		Caller: false,
	}

	err := Init(cfg)
	assert.NoError(t, err)
	assert.NotNil(t, log)
	assert.NotNil(t, sugar)
}

func TestInit_FileOutput(t *testing.T) {
	// 创建临时目录
	tmpDir := t.TempDir()
	logFile := filepath.Join(tmpDir, "test.log")

	cfg := &config.LoggerConfig{
		Level:      "debug",
		Format:     "json",
		Output:     "file",
		FilePath:   logFile,
		MaxSize:    1,
		MaxBackups: 3,
		MaxAge:     7,
		Compress:   false,
		Caller:     true,
	}

	err := Init(cfg)
	assert.NoError(t, err)

	// 写入日志
	Info("test message")
	_ = Sync()

	// 验证文件创建
	_, err = os.Stat(logFile)
	assert.NoError(t, err)
}

func TestInit_AllLogLevels(t *testing.T) {
	levels := []string{"debug", "info", "warn", "error", "unknown"}

	for _, level := range levels {
		t.Run(level, func(t *testing.T) {
			cfg := &config.LoggerConfig{
				Level:  level,
				Format: "console",
				Output: "stdout",
			}
			err := Init(cfg)
			assert.NoError(t, err)
		})
	}
}

// ==================== getLogLevel 测试 ====================

func TestGetLogLevel(t *testing.T) {
	tests := []struct {
		name     string
		level    string
		expected zapcore.Level
	}{
		{"debug level", "debug", zapcore.DebugLevel},
		{"info level", "info", zapcore.InfoLevel},
		{"warn level", "warn", zapcore.WarnLevel},
		{"error level", "error", zapcore.ErrorLevel},
		{"default level", "invalid", zapcore.InfoLevel},
		{"empty level", "", zapcore.InfoLevel},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := getLogLevel(tt.level)
			assert.Equal(t, tt.expected, result)
		})
	}
}

// ==================== customTimeEncoder 测试 ====================

func TestCustomTimeEncoder(t *testing.T) {
	testTime := time.Date(2026, 1, 11, 15, 30, 45, 123000000, time.Local)

	arrEnc := &testArrayEncoder{}

	customTimeEncoder(testTime, arrEnc)

	expected := "2026-01-11 15:30:45.123"
	assert.Equal(t, expected, arrEnc.lastString)
}

// testArrayEncoder 用于测试的数组编码器
type testArrayEncoder struct {
	lastString string
}

func (e *testArrayEncoder) AppendString(s string) {
	e.lastString = s
}

func (e *testArrayEncoder) AppendBool(bool)              {}
func (e *testArrayEncoder) AppendByteString([]byte)      {}
func (e *testArrayEncoder) AppendComplex128(complex128)  {}
func (e *testArrayEncoder) AppendComplex64(complex64)    {}
func (e *testArrayEncoder) AppendDuration(time.Duration) {}
func (e *testArrayEncoder) AppendFloat32(float32)        {}
func (e *testArrayEncoder) AppendFloat64(float64)        {}
func (e *testArrayEncoder) AppendInt(int)                {}
func (e *testArrayEncoder) AppendInt16(int16)            {}
func (e *testArrayEncoder) AppendInt32(int32)            {}
func (e *testArrayEncoder) AppendInt64(int64)            {}
func (e *testArrayEncoder) AppendInt8(int8)              {}
func (e *testArrayEncoder) AppendTime(time.Time)         {}
func (e *testArrayEncoder) AppendUint(uint)              {}
func (e *testArrayEncoder) AppendUint16(uint16)          {}
func (e *testArrayEncoder) AppendUint32(uint32)          {}
func (e *testArrayEncoder) AppendUint64(uint64)          {}
func (e *testArrayEncoder) AppendUint8(uint8)            {}
func (e *testArrayEncoder) AppendUintptr(uintptr)        {}
func (e *testArrayEncoder) AppendReflected(interface{}) error {
	return nil
}
func (e *testArrayEncoder) AppendArray(zapcore.ArrayMarshaler) error {
	return nil
}
func (e *testArrayEncoder) AppendObject(zapcore.ObjectMarshaler) error {
	return nil
}

// ==================== GetLogger / GetSugar 测试 ====================

func TestGetLogger_LazyInit(t *testing.T) {
	// 重置全局变量
	log = nil
	sugar = nil

	logger := GetLogger()
	assert.NotNil(t, logger)

	// 再次获取应该返回相同实例
	logger2 := GetLogger()
	assert.Equal(t, logger, logger2)
}

func TestGetSugar_LazyInit(t *testing.T) {
	// 重置全局变量
	log = nil
	sugar = nil

	sugarLogger := GetSugar()
	assert.NotNil(t, sugarLogger)

	// 再次获取应该返回相同实例
	sugarLogger2 := GetSugar()
	assert.Equal(t, sugarLogger, sugarLogger2)
}

// ==================== Sync 测试 ====================

func TestSync_WithNilLogger(t *testing.T) {
	log = nil
	err := Sync()
	assert.NoError(t, err)
}

func TestSync_WithInitializedLogger(t *testing.T) {
	cfg := &config.LoggerConfig{
		Level:  "info",
		Format: "console",
		Output: "stdout",
	}
	err := Init(cfg)
	require.NoError(t, err)

	err = Sync()
	// Sync 可能因为 stdout 不支持同步而返回错误，这是正常的
	// 我们只验证不会 panic
}

// ==================== 日志函数测试 ====================

func TestLogFunctions(t *testing.T) {
	cfg := &config.LoggerConfig{
		Level:  "debug",
		Format: "console",
		Output: "stdout",
	}
	err := Init(cfg)
	require.NoError(t, err)

	// 测试各种日志函数不会 panic
	t.Run("Debug", func(t *testing.T) {
		assert.NotPanics(t, func() {
			Debug("debug message", String("key", "value"))
		})
	})

	t.Run("Info", func(t *testing.T) {
		assert.NotPanics(t, func() {
			Info("info message", Int("count", 10))
		})
	})

	t.Run("Warn", func(t *testing.T) {
		assert.NotPanics(t, func() {
			Warn("warn message", Bool("flag", true))
		})
	})

	t.Run("Error", func(t *testing.T) {
		assert.NotPanics(t, func() {
			Error("error message", Err(nil))
		})
	})
}

func TestFormattedLogFunctions(t *testing.T) {
	cfg := &config.LoggerConfig{
		Level:  "debug",
		Format: "console",
		Output: "stdout",
	}
	err := Init(cfg)
	require.NoError(t, err)

	t.Run("Debugf", func(t *testing.T) {
		assert.NotPanics(t, func() {
			Debugf("debug %s %d", "test", 123)
		})
	})

	t.Run("Infof", func(t *testing.T) {
		assert.NotPanics(t, func() {
			Infof("info %s", "test")
		})
	})

	t.Run("Warnf", func(t *testing.T) {
		assert.NotPanics(t, func() {
			Warnf("warn %v", map[string]int{"a": 1})
		})
	})

	t.Run("Errorf", func(t *testing.T) {
		assert.NotPanics(t, func() {
			Errorf("error %s", "test")
		})
	})
}

// ==================== With / WithFields / Named 测试 ====================

func TestWith(t *testing.T) {
	cfg := &config.LoggerConfig{
		Level:  "info",
		Format: "console",
		Output: "stdout",
	}
	err := Init(cfg)
	require.NoError(t, err)

	childLogger := With(String("service", "test"))
	assert.NotNil(t, childLogger)
	assert.IsType(t, &zap.Logger{}, childLogger)
}

func TestWithFields(t *testing.T) {
	cfg := &config.LoggerConfig{
		Level:  "info",
		Format: "console",
		Output: "stdout",
	}
	err := Init(cfg)
	require.NoError(t, err)

	childSugar := WithFields("key1", "value1", "key2", 123)
	assert.NotNil(t, childSugar)
	assert.IsType(t, &zap.SugaredLogger{}, childSugar)
}

func TestNamed(t *testing.T) {
	cfg := &config.LoggerConfig{
		Level:  "info",
		Format: "console",
		Output: "stdout",
	}
	err := Init(cfg)
	require.NoError(t, err)

	namedLogger := Named("test-service")
	assert.NotNil(t, namedLogger)
}

// ==================== 字段构造函数测试 ====================

func TestFieldConstructors(t *testing.T) {
	tests := []struct {
		name  string
		field zap.Field
	}{
		{"RequestID", RequestID("req-123")},
		{"UserID", UserID(12345)},
		{"AdminID", AdminID(999)},
		{"DeviceID", DeviceID(100)},
		{"OrderNo", OrderNo("ORD20260111001")},
		{"Module", Module("payment")},
		{"Action", Action("create")},
		{"Latency", Latency(100 * time.Millisecond)},
		{"StatusCode", StatusCode(200)},
		{"Method", Method("POST")},
		{"Path", Path("/api/v1/orders")},
		{"IP", IP("192.168.1.1")},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.NotEmpty(t, tt.field.Key)
		})
	}
}

func TestFieldConstructorValues(t *testing.T) {
	t.Run("RequestID", func(t *testing.T) {
		field := RequestID("test-id")
		assert.Equal(t, "request_id", field.Key)
		assert.Equal(t, "test-id", field.String)
	})

	t.Run("UserID", func(t *testing.T) {
		field := UserID(12345)
		assert.Equal(t, "user_id", field.Key)
		assert.Equal(t, int64(12345), field.Integer)
	})

	t.Run("AdminID", func(t *testing.T) {
		field := AdminID(999)
		assert.Equal(t, "admin_id", field.Key)
		assert.Equal(t, int64(999), field.Integer)
	})

	t.Run("DeviceID", func(t *testing.T) {
		field := DeviceID(100)
		assert.Equal(t, "device_id", field.Key)
		assert.Equal(t, int64(100), field.Integer)
	})

	t.Run("OrderNo", func(t *testing.T) {
		field := OrderNo("ORD123")
		assert.Equal(t, "order_no", field.Key)
		assert.Equal(t, "ORD123", field.String)
	})

	t.Run("Module", func(t *testing.T) {
		field := Module("auth")
		assert.Equal(t, "module", field.Key)
		assert.Equal(t, "auth", field.String)
	})

	t.Run("Action", func(t *testing.T) {
		field := Action("login")
		assert.Equal(t, "action", field.Key)
		assert.Equal(t, "login", field.String)
	})

	t.Run("StatusCode", func(t *testing.T) {
		field := StatusCode(404)
		assert.Equal(t, "status_code", field.Key)
		assert.Equal(t, int64(404), field.Integer)
	})

	t.Run("Method", func(t *testing.T) {
		field := Method("GET")
		assert.Equal(t, "method", field.Key)
		assert.Equal(t, "GET", field.String)
	})

	t.Run("Path", func(t *testing.T) {
		field := Path("/api/health")
		assert.Equal(t, "path", field.Key)
		assert.Equal(t, "/api/health", field.String)
	})

	t.Run("IP", func(t *testing.T) {
		field := IP("10.0.0.1")
		assert.Equal(t, "ip", field.Key)
		assert.Equal(t, "10.0.0.1", field.String)
	})
}

// ==================== zap 字段别名测试 ====================

func TestZapFieldAliases(t *testing.T) {
	// 验证别名正确指向 zap 函数
	assert.Equal(t, zap.String("k", "v"), String("k", "v"))
	assert.Equal(t, zap.Int("k", 1), Int("k", 1))
	assert.Equal(t, zap.Int64("k", 100), Int64("k", 100))
	assert.Equal(t, zap.Uint64("k", 200), Uint64("k", 200))
	assert.Equal(t, zap.Float64("k", 1.5), Float64("k", 1.5))
	assert.Equal(t, zap.Bool("k", true), Bool("k", true))
	assert.Equal(t, zap.Duration("k", time.Second), Duration("k", time.Second))
}

// ==================== JSON 日志格式验证 ====================

func TestJSONLogFormat(t *testing.T) {
	// 创建临时日志文件
	tmpDir := t.TempDir()
	logFile := filepath.Join(tmpDir, "json.log")

	cfg := &config.LoggerConfig{
		Level:    "info",
		Format:   "json",
		Output:   "file",
		FilePath: logFile,
		Caller:   false,
	}

	err := Init(cfg)
	require.NoError(t, err)

	// 写入日志
	Info("test json log", String("key", "value"), Int("count", 42))
	_ = Sync()

	// 读取并验证 JSON 格式
	content, err := os.ReadFile(logFile)
	require.NoError(t, err)

	// 解析 JSON
	var logEntry map[string]interface{}
	lines := strings.Split(strings.TrimSpace(string(content)), "\n")
	require.NotEmpty(t, lines)

	err = json.Unmarshal([]byte(lines[0]), &logEntry)
	assert.NoError(t, err)

	// 验证字段存在
	assert.Equal(t, "test json log", logEntry["msg"])
	assert.Equal(t, "value", logEntry["key"])
	assert.Equal(t, float64(42), logEntry["count"])
	assert.Equal(t, "info", logEntry["level"])
	assert.Contains(t, logEntry, "time")
}

// ==================== 日志级别过滤测试 ====================

func TestLogLevelFiltering(t *testing.T) {
	tmpDir := t.TempDir()
	logFile := filepath.Join(tmpDir, "level.log")

	// 设置为 warn 级别
	cfg := &config.LoggerConfig{
		Level:    "warn",
		Format:   "json",
		Output:   "file",
		FilePath: logFile,
	}

	err := Init(cfg)
	require.NoError(t, err)

	// 写入不同级别的日志
	Debug("debug message")
	Info("info message")
	Warn("warn message")
	Error("error message")
	_ = Sync()

	// 读取日志文件
	content, err := os.ReadFile(logFile)
	require.NoError(t, err)

	logContent := string(content)

	// debug 和 info 应该被过滤
	assert.NotContains(t, logContent, "debug message")
	assert.NotContains(t, logContent, "info message")

	// warn 和 error 应该存在
	assert.Contains(t, logContent, "warn message")
	assert.Contains(t, logContent, "error message")
}
