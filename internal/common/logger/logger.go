// Package logger 提供结构化日志功能
package logger

import (
	"os"
	"time"

	"github.com/dumeirei/smart-locker-backend/internal/common/config"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"gopkg.in/natefinch/lumberjack.v2"
)

var (
	log   *zap.Logger
	sugar *zap.SugaredLogger
)

// Init 初始化日志
func Init(cfg *config.LoggerConfig) error {
	// 设置日志级别
	level := getLogLevel(cfg.Level)

	// 设置编码器配置
	encoderConfig := zapcore.EncoderConfig{
		TimeKey:        "time",
		LevelKey:       "level",
		NameKey:        "logger",
		CallerKey:      "caller",
		FunctionKey:    zapcore.OmitKey,
		MessageKey:     "msg",
		StacktraceKey:  "stacktrace",
		LineEnding:     zapcore.DefaultLineEnding,
		EncodeLevel:    zapcore.LowercaseLevelEncoder,
		EncodeTime:     customTimeEncoder,
		EncodeDuration: zapcore.SecondsDurationEncoder,
		EncodeCaller:   zapcore.ShortCallerEncoder,
	}

	// 创建编码器
	var encoder zapcore.Encoder
	if cfg.Format == "json" {
		encoder = zapcore.NewJSONEncoder(encoderConfig)
	} else {
		encoderConfig.EncodeLevel = zapcore.CapitalColorLevelEncoder
		encoder = zapcore.NewConsoleEncoder(encoderConfig)
	}

	// 创建写入器
	var writers []zapcore.WriteSyncer

	// 标准输出
	if cfg.Output == "stdout" || cfg.Output == "" {
		writers = append(writers, zapcore.AddSync(os.Stdout))
	}

	// 文件输出
	if cfg.FilePath != "" && cfg.Output != "stdout" {
		fileWriter := &lumberjack.Logger{
			Filename:   cfg.FilePath,
			MaxSize:    cfg.MaxSize,
			MaxBackups: cfg.MaxBackups,
			MaxAge:     cfg.MaxAge,
			Compress:   cfg.Compress,
			LocalTime:  true,
		}
		writers = append(writers, zapcore.AddSync(fileWriter))
	}

	// 创建核心
	core := zapcore.NewCore(
		encoder,
		zapcore.NewMultiWriteSyncer(writers...),
		level,
	)

	// 创建日志器
	options := []zap.Option{
		zap.AddStacktrace(zapcore.ErrorLevel),
	}
	if cfg.Caller {
		options = append(options, zap.AddCaller(), zap.AddCallerSkip(1))
	}

	log = zap.New(core, options...)
	sugar = log.Sugar()

	return nil
}

// customTimeEncoder 自定义时间编码器
func customTimeEncoder(t time.Time, enc zapcore.PrimitiveArrayEncoder) {
	enc.AppendString(t.Format("2006-01-02 15:04:05.000"))
}

// getLogLevel 获取日志级别
func getLogLevel(level string) zapcore.Level {
	switch level {
	case "debug":
		return zapcore.DebugLevel
	case "info":
		return zapcore.InfoLevel
	case "warn":
		return zapcore.WarnLevel
	case "error":
		return zapcore.ErrorLevel
	default:
		return zapcore.InfoLevel
	}
}

// GetLogger 获取原始日志器
func GetLogger() *zap.Logger {
	if log == nil {
		log, _ = zap.NewDevelopment()
		sugar = log.Sugar()
	}
	return log
}

// GetSugar 获取 Sugar 日志器
func GetSugar() *zap.SugaredLogger {
	if sugar == nil {
		log, _ = zap.NewDevelopment()
		sugar = log.Sugar()
	}
	return sugar
}

// Sync 同步日志
func Sync() error {
	if log != nil {
		return log.Sync()
	}
	return nil
}

// Debug 调试日志
func Debug(msg string, fields ...zap.Field) {
	GetLogger().Debug(msg, fields...)
}

// Info 信息日志
func Info(msg string, fields ...zap.Field) {
	GetLogger().Info(msg, fields...)
}

// Warn 警告日志
func Warn(msg string, fields ...zap.Field) {
	GetLogger().Warn(msg, fields...)
}

// Error 错误日志
func Error(msg string, fields ...zap.Field) {
	GetLogger().Error(msg, fields...)
}

// Fatal 致命错误日志
func Fatal(msg string, fields ...zap.Field) {
	GetLogger().Fatal(msg, fields...)
}

// Panic 恐慌日志
func Panic(msg string, fields ...zap.Field) {
	GetLogger().Panic(msg, fields...)
}

// Debugf 格式化调试日志
func Debugf(template string, args ...interface{}) {
	GetSugar().Debugf(template, args...)
}

// Infof 格式化信息日志
func Infof(template string, args ...interface{}) {
	GetSugar().Infof(template, args...)
}

// Warnf 格式化警告日志
func Warnf(template string, args ...interface{}) {
	GetSugar().Warnf(template, args...)
}

// Errorf 格式化错误日志
func Errorf(template string, args ...interface{}) {
	GetSugar().Errorf(template, args...)
}

// Fatalf 格式化致命错误日志
func Fatalf(template string, args ...interface{}) {
	GetSugar().Fatalf(template, args...)
}

// Panicf 格式化恐慌日志
func Panicf(template string, args ...interface{}) {
	GetSugar().Panicf(template, args...)
}

// With 返回带有字段的日志器
func With(fields ...zap.Field) *zap.Logger {
	return GetLogger().With(fields...)
}

// WithFields 返回带有字段的 Sugar 日志器
func WithFields(args ...interface{}) *zap.SugaredLogger {
	return GetSugar().With(args...)
}

// Named 返回命名日志器
func Named(name string) *zap.Logger {
	return GetLogger().Named(name)
}

// 常用字段构造函数
var (
	String   = zap.String
	Int      = zap.Int
	Int64    = zap.Int64
	Uint64   = zap.Uint64
	Float64  = zap.Float64
	Bool     = zap.Bool
	Any      = zap.Any
	Err      = zap.Error
	Duration = zap.Duration
	Time     = zap.Time
)

// RequestID 请求ID字段
func RequestID(id string) zap.Field {
	return zap.String("request_id", id)
}

// UserID 用户ID字段
func UserID(id int64) zap.Field {
	return zap.Int64("user_id", id)
}

// AdminID 管理员ID字段
func AdminID(id int64) zap.Field {
	return zap.Int64("admin_id", id)
}

// DeviceID 设备ID字段
func DeviceID(id int64) zap.Field {
	return zap.Int64("device_id", id)
}

// OrderNo 订单号字段
func OrderNo(no string) zap.Field {
	return zap.String("order_no", no)
}

// Module 模块字段
func Module(name string) zap.Field {
	return zap.String("module", name)
}

// Action 操作字段
func Action(name string) zap.Field {
	return zap.String("action", name)
}

// Latency 延迟字段
func Latency(d time.Duration) zap.Field {
	return zap.Duration("latency", d)
}

// StatusCode HTTP状态码字段
func StatusCode(code int) zap.Field {
	return zap.Int("status_code", code)
}

// Method HTTP方法字段
func Method(method string) zap.Field {
	return zap.String("method", method)
}

// Path 路径字段
func Path(path string) zap.Field {
	return zap.String("path", path)
}

// IP IP地址字段
func IP(ip string) zap.Field {
	return zap.String("ip", ip)
}
