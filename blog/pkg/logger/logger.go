package logger

import (
	"blog/pkg/config"
	"os"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"gopkg.in/natefinch/lumberjack.v2"
)

const (
	colorRed    = "\033[31m"
	colorGreen  = "\033[32m"
	colorYellow = "\033[33m"
	colorBlue   = "\033[34m"
	colorReset  = "\033[0m"
)

var (
	log   *zap.Logger
	sugar *zap.SugaredLogger
)

// CustomLevelEncoder 自定义日志级别编码器，实现控制台颜色显示
func CustomLevelEncoder(l zapcore.Level, enc zapcore.PrimitiveArrayEncoder) {
	var color string
	switch l {
	case zapcore.DebugLevel:
		color = colorBlue
	case zapcore.InfoLevel:
		color = colorGreen
	case zapcore.WarnLevel:
		color = colorYellow
	case zapcore.ErrorLevel:
		color = colorRed
	default:
		color = colorReset
	}
	enc.AppendString(color + l.CapitalString() + colorReset)
}

// Init 初始化日志系统
func Init(cfg *config.LogConfig) error {
	// 基础编码器配置
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
		EncodeTime:     zapcore.ISO8601TimeEncoder,
		EncodeDuration: zapcore.SecondsDurationEncoder,
		EncodeCaller:   zapcore.ShortCallerEncoder,
	}

	// 日志级别
	level := zapcore.InfoLevel
	switch cfg.Level {
	case "debug":
		level = zapcore.DebugLevel
	case "info":
		level = zapcore.InfoLevel
	case "warn":
		level = zapcore.WarnLevel
	case "error":
		level = zapcore.ErrorLevel
	}

	// 文件输出配置
	fileWriter := &lumberjack.Logger{
		Filename:   cfg.Filename,
		MaxSize:    cfg.MaxSize,
		MaxBackups: cfg.MaxBackups,
		MaxAge:     cfg.MaxAge,
		Compress:   cfg.Compress,
	}

	// 创建文件核心 (JSON 格式，无颜色)
	fileCore := zapcore.NewCore(
		zapcore.NewJSONEncoder(encoderConfig),
		zapcore.AddSync(fileWriter),
		level,
	)

	// 创建控制台核心 (Console 格式，带颜色)
	consoleEncoderConfig := encoderConfig
	consoleEncoderConfig.EncodeLevel = CustomLevelEncoder
	consoleCore := zapcore.NewCore(
		zapcore.NewConsoleEncoder(consoleEncoderConfig),
		zapcore.AddSync(os.Stdout),
		level,
	)

	// 合并核心
	core := zapcore.NewTee(fileCore, consoleCore)

	// 创建 logger
	log = zap.New(core, zap.AddCaller(), zap.AddCallerSkip(1), zap.AddStacktrace(zapcore.ErrorLevel))
	sugar = log.Sugar()

	return nil
}

// Debug 调试日志
func Debug(msg string, fields ...zap.Field) {
	log.Debug(msg, fields...)
}

// Info 信息日志
func Info(msg string, fields ...zap.Field) {
	log.Info(msg, fields...)
}

// Warn 警告日志
func Warn(msg string, fields ...zap.Field) {
	log.Warn(msg, fields...)
}

// Error 错误日志
func Error(msg string, fields ...zap.Field) {
	log.Error(msg, fields...)
}

// Fatal 致命错误日志
func Fatal(msg string, fields ...zap.Field) {
	log.Fatal(msg, fields...)
}

// Debugf 格式化调试日志
func Debugf(format string, args ...interface{}) {
	sugar.Debugf(format, args...)
}

// Infof 格式化信息日志
func Infof(format string, args ...interface{}) {
	sugar.Infof(format, args...)
}

// Warnf 格式化警告日志
func Warnf(format string, args ...interface{}) {
	sugar.Warnf(format, args...)
}

// Errorf 格式化错误日志
func Errorf(format string, args ...interface{}) {
	sugar.Errorf(format, args...)
}

// Fatalf 格式化致命错误日志
func Fatalf(format string, args ...interface{}) {
	sugar.Fatalf(format, args...)
}

// With 创建带字段的 logger
func With(fields ...zap.Field) *zap.Logger {
	return log.With(fields...)
}

// Sync 同步日志缓冲区
func Sync() error {
	return log.Sync()
}

// GetLogger 获取原始 logger
func GetLogger() *zap.Logger {
	return log
}

// GetSugaredLogger 获取 sugared logger
func GetSugaredLogger() *zap.SugaredLogger {
	return sugar
}
