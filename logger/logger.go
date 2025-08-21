package logger

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	lumberjack "gopkg.in/natefinch/lumberjack.v2"
)

// Config 日志配置结构体
type Config struct {
	Level      string `json:"level" yaml:"level"`           // 日志级别
	FileName   string `json:"filename" yaml:"filename"`     // 日志文件名
	MaxSize    int    `json:"maxsize" yaml:"maxsize"`       // 日志文件最大大小(MB)
	MaxAge     int    `json:"maxage" yaml:"maxage"`         // 日志文件最大保存天数
	MaxBackups int    `json:"maxbackups" yaml:"maxbackups"` // 最大备份文件数量
	Compress   bool   `json:"compress" yaml:"compress"`     // 是否压缩备份文件
	TimeZone   string `json:"timezone" yaml:"timezone"`     // 时区，默认"Asia/Shanghai"
}

// Logger 日志器结构体
type Logger struct {
	logger *zap.Logger
	config *Config
	level  zap.AtomicLevel
	mu     sync.RWMutex
}

// 默认配置
var defaultConfig = &Config{
	Level:      "info",
	FileName:   "./logs/app.log",
	MaxSize:    100,
	MaxAge:     7,
	MaxBackups: 10,
	Compress:   true,
	TimeZone:   "Asia/Shanghai",
}

// 全局logger实例
var globalLogger *Logger
var once sync.Once

// Default 获取默认logger实例（使用默认配置）
func Default() *Logger {
	if globalLogger != nil {
		return globalLogger
	}
	once.Do(func() {
		if globalLogger == nil {
			globalLogger = New(defaultConfig)
		}
	})
	return globalLogger
}

// New 创建新的logger实例
func New(config *Config) *Logger {
	// 当调用方传入 nil 时，不直接引用 defaultConfig 指针，而是拷贝一份值。这样后续对 config 进行的填充不会污染全局的默认配置实例，避免副作用
	if config == nil {
		cfg := *defaultConfig
		config = &cfg
	}

	// 填充默认值
	if config.Level == "" {
		config.Level = defaultConfig.Level
	}
	if config.FileName == "" {
		config.FileName = defaultConfig.FileName
	}
	if config.MaxSize == 0 {
		config.MaxSize = defaultConfig.MaxSize
	}
	if config.MaxAge == 0 {
		config.MaxAge = defaultConfig.MaxAge
	}
	if config.MaxBackups == 0 {
		config.MaxBackups = defaultConfig.MaxBackups
	}
	if !config.Compress {
		config.Compress = defaultConfig.Compress
	}
	if config.TimeZone == "" {
		config.TimeZone = defaultConfig.TimeZone
	}

	logger := &Logger{
		config: config,
	}

	if err := logger.init(); err != nil {
		// 如果初始化失败，使用基本的控制台logger
		logger.logger, _ = zap.NewDevelopment()
	}

	return logger
}

// init 初始化zap logger
func (l *Logger) init() error {
	// 编码器配置
	encoderCfg := zap.NewProductionEncoderConfig()
	encoderCfg.TimeKey = "time"
	encoderCfg.LevelKey = "level"
	encoderCfg.NameKey = "logger"
	encoderCfg.CallerKey = "caller"
	encoderCfg.MessageKey = "msg"
	encoderCfg.StacktraceKey = "stacktrace"
	encoderCfg.LineEnding = zapcore.DefaultLineEnding
	encoderCfg.EncodeLevel = zapcore.CapitalLevelEncoder
	encoderCfg.EncodeDuration = zapcore.SecondsDurationEncoder
	encoderCfg.EncodeCaller = zapcore.ShortCallerEncoder

	// 设置时区
	loc, err := time.LoadLocation(l.config.TimeZone)
	if err != nil {
		loc = time.Local // 如果时区设置失败，使用本地时区
	}

	// 自定义时间编码器，应用时区
	encoderCfg.EncodeTime = func(t time.Time, enc zapcore.PrimitiveArrayEncoder) {
		enc.AppendString(t.In(loc).Format("2006-01-02 15:04:05.000"))
	}

	// 确保日志目录存在
	dir := filepath.Dir(l.config.FileName)
	if dir != "." && dir != "" {
		_ = os.MkdirAll(dir, 0o755)
	}
	// Lumberjack 日志分割器
	writer := zapcore.AddSync(&lumberjack.Logger{
		Filename:   l.config.FileName,
		MaxSize:    l.config.MaxSize,
		MaxAge:     l.config.MaxAge,
		MaxBackups: l.config.MaxBackups,
		Compress:   l.config.Compress,
	})

	// 初始化日志级别（使用可动态调整的 AtomicLevel）
	l.level = zap.NewAtomicLevel()
	if err := l.level.UnmarshalText([]byte(l.config.Level)); err != nil {
		l.level.SetLevel(zap.InfoLevel) // 默认info级别
	}

	// 同步写入
	core := zapcore.NewCore(
		zapcore.NewJSONEncoder(encoderCfg),
		zapcore.NewMultiWriteSyncer(zapcore.AddSync(os.Stdout), writer),
		l.level,
	)

	l.logger = zap.New(core, zap.AddCaller(), zap.AddCallerSkip(1), zap.AddStacktrace(zap.ErrorLevel))
	return nil
}

// addTraceID 添加traceId到字段中
func (l *Logger) addTraceID(ctx context.Context, fields []zap.Field) []zap.Field {
	if ctx != nil {
		if traceId := ctx.Value("traceId"); traceId != nil {
			fields = append(fields, zap.String("traceId", fmt.Sprint(traceId)))
		}
	}
	return fields
}

// Info 记录info级别日志
func (l *Logger) Info(ctx context.Context, msg string, fields ...zap.Field) {
	fields = l.addTraceID(ctx, fields)
	l.logger.Info(msg, fields...)
}

// Error 记录error级别日志
func (l *Logger) Error(ctx context.Context, msg string, fields ...zap.Field) {
	fields = l.addTraceID(ctx, fields)
	l.logger.Error(msg, fields...)
}

// Debug 记录debug级别日志
func (l *Logger) Debug(ctx context.Context, msg string, fields ...zap.Field) {
	fields = l.addTraceID(ctx, fields)
	l.logger.Debug(msg, fields...)
}

// Warn 记录warn级别日志
func (l *Logger) Warn(ctx context.Context, msg string, fields ...zap.Field) {
	fields = l.addTraceID(ctx, fields)
	l.logger.Warn(msg, fields...)
}

// Fatal 记录fatal级别日志
func (l *Logger) Fatal(ctx context.Context, msg string, fields ...zap.Field) {
	fields = l.addTraceID(ctx, fields)
	l.logger.Fatal(msg, fields...)
}

// Infof 格式化记录info级别日志
func (l *Logger) Infof(ctx context.Context, msg string, args ...interface{}) {
	sugar := l.logger.Sugar()
	if ctx != nil {
		if traceId := ctx.Value("traceId"); traceId != nil {
			sugar = sugar.With("traceId", fmt.Sprint(traceId))
		}
	}
	sugar.Infof(msg, args...)
}

// Errorf 格式化记录error级别日志
func (l *Logger) Errorf(ctx context.Context, msg string, args ...interface{}) {
	sugar := l.logger.Sugar()
	if ctx != nil {
		if traceId := ctx.Value("traceId"); traceId != nil {
			sugar = sugar.With("traceId", fmt.Sprint(traceId))
		}
	}
	sugar.Errorf(msg, args...)
}

// Debugf 格式化记录debug级别日志
func (l *Logger) Debugf(ctx context.Context, msg string, args ...interface{}) {
	sugar := l.logger.Sugar()
	if ctx != nil {
		if traceId := ctx.Value("traceId"); traceId != nil {
			sugar = sugar.With("traceId", fmt.Sprint(traceId))
		}
	}
	sugar.Debugf(msg, args...)
}

// Warnf 格式化记录warn级别日志
func (l *Logger) Warnf(ctx context.Context, msg string, args ...interface{}) {
	sugar := l.logger.Sugar()
	if ctx != nil {
		if traceId := ctx.Value("traceId"); traceId != nil {
			sugar = sugar.With("traceId", fmt.Sprint(traceId))
		}
	}
	sugar.Warnf(msg, args...)
}

// Sync 同步日志缓冲区
func (l *Logger) Sync() error {
	if err := l.logger.Sync(); err != nil {
		// 容忍 stdout/stderr 的平台差异性错误，但保留其它真实错误
		msg := err.Error()
		if strings.Contains(msg, "bad file descriptor") ||
			strings.Contains(msg, "/dev/stdout") ||
			strings.Contains(msg, "invalid argument") {
			return nil
		}
		return err
	}
	return nil
}

// SetLevel 动态设置日志级别
func (l *Logger) SetLevel(level string) error {
	l.mu.Lock()
	defer l.mu.Unlock()

	if err := l.level.UnmarshalText([]byte(level)); err != nil {
		return err
	}
	l.config.Level = level
	return nil
}

// GetConfig 获取当前配置
func (l *Logger) GetConfig() *Config {
	l.mu.RLock()
	defer l.mu.RUnlock()

	// 返回配置的副本
	config := *l.config
	return &config
}

// 全局便捷方法，使用默认logger实例
func Info(ctx context.Context, msg string, fields ...zap.Field) {
	Default().Info(ctx, msg, fields...)
}

func Error(ctx context.Context, msg string, fields ...zap.Field) {
	Default().Error(ctx, msg, fields...)
}

func Debug(ctx context.Context, msg string, fields ...zap.Field) {
	Default().Debug(ctx, msg, fields...)
}

func Warn(ctx context.Context, msg string, fields ...zap.Field) {
	Default().Warn(ctx, msg, fields...)
}

func Infof(ctx context.Context, msg string, args ...interface{}) {
	Default().Infof(ctx, msg, args...)
}

func Errorf(ctx context.Context, msg string, args ...interface{}) {
	Default().Errorf(ctx, msg, args...)
}

func Debugf(ctx context.Context, msg string, args ...interface{}) {
	Default().Debugf(ctx, msg, args...)
}

func Warnf(ctx context.Context, msg string, args ...interface{}) {
	Default().Warnf(ctx, msg, args...)
}

// SetGlobalConfig 设置全局logger配置
func SetGlobalConfig(config *Config) {
	globalLogger = New(config)
}

// Sync 同步全局logger缓冲区
func Sync() error {
	return Default().Sync()
}
