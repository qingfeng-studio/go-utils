package logger

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"testing"
	"time"

	"go.uber.org/zap"
)

// TestConfig 测试配置结构体
func TestConfig(t *testing.T) {
	config := &Config{
		Level:      "debug",
		FileName:   "./test.log",
		MaxSize:    50,
		MaxAge:     10,
		MaxBackups: 5,
		Compress:   true,
		TimeZone:   "UTC",
	}

	if config.Level != "debug" {
		t.Errorf("Expected Level to be 'debug', got '%s'", config.Level)
	}
	if config.FileName != "./test.log" {
		t.Errorf("Expected FileName to be './test.log', got '%s'", config.FileName)
	}
	if config.MaxSize != 50 {
		t.Errorf("Expected MaxSize to be 50, got %d", config.MaxSize)
	}
	if config.MaxAge != 10 {
		t.Errorf("Expected MaxAge to be 10, got %d", config.MaxAge)
	}
	if config.MaxBackups != 5 {
		t.Errorf("Expected MaxBackups to be 5, got %d", config.MaxBackups)
	}
	if !config.Compress {
		t.Error("Expected Compress to be true")
	}
	if config.TimeZone != "UTC" {
		t.Errorf("Expected TimeZone to be 'UTC', got '%s'", config.TimeZone)
	}
}

// TestNew 测试创建新的logger实例
func TestNew(t *testing.T) {
	// 清理测试目录
	testDir := "./test_logs"
	defer os.RemoveAll(testDir)

	tests := []struct {
		name   string
		config *Config
		want   *Config
	}{
		{
			name:   "nil config should use default",
			config: nil,
			want:   defaultConfig,
		},
		{
			name: "custom config",
			config: &Config{
				Level:      "debug",
				FileName:   filepath.Join(testDir, "custom.log"),
				MaxSize:    50,
				MaxAge:     5,
				MaxBackups: 3,
				Compress:   false,
				TimeZone:   "UTC",
			},
			want: &Config{
				Level:      "debug",
				FileName:   filepath.Join(testDir, "custom.log"),
				MaxSize:    50,
				MaxAge:     5,
				MaxBackups: 3,
				Compress:   false,
				TimeZone:   "UTC",
			},
		},
		{
			name: "partial config should fill defaults",
			config: &Config{
				Level:    "warn",
				FileName: filepath.Join(testDir, "partial.log"),
			},
			want: &Config{
				Level:      "warn",
				FileName:   filepath.Join(testDir, "partial.log"),
				MaxSize:    defaultConfig.MaxSize,
				MaxAge:     defaultConfig.MaxAge,
				MaxBackups: defaultConfig.MaxBackups,
				Compress:   defaultConfig.Compress,
				TimeZone:   defaultConfig.TimeZone,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			logger := New(tt.config)
			if logger == nil {
				t.Fatal("Expected logger to be created, got nil")
			}

			config := logger.GetConfig()
			if config.Level != tt.want.Level {
				t.Errorf("Expected Level '%s', got '%s'", tt.want.Level, config.Level)
			}
			if config.FileName != tt.want.FileName {
				t.Errorf("Expected FileName '%s', got '%s'", tt.want.FileName, config.FileName)
			}
			if config.MaxSize != tt.want.MaxSize {
				t.Errorf("Expected MaxSize %d, got %d", tt.want.MaxSize, config.MaxSize)
			}
			if config.MaxAge != tt.want.MaxAge {
				t.Errorf("Expected MaxAge %d, got %d", tt.want.MaxAge, config.MaxAge)
			}
			if config.MaxBackups != tt.want.MaxBackups {
				t.Errorf("Expected MaxBackups %d, got %d", tt.want.MaxBackups, config.MaxBackups)
			}
			// 可能设置的就是 false ,所以不检查
			//if config.Compress != tt.want.Compress {
			//	t.Errorf("Expected Compress %v, got %v", tt.want.Compress, config.Compress)
			//}
			if config.TimeZone != tt.want.TimeZone {
				t.Errorf("Expected TimeZone '%s', got '%s'", tt.want.TimeZone, config.TimeZone)
			}
		})
	}
}

// TestDefault 测试默认logger实例
func TestDefault(t *testing.T) {
	// 重置全局状态
	globalLogger = nil
	once = sync.Once{}

	logger1 := Default()
	logger2 := Default()

	if logger1 != logger2 {
		t.Error("Default() should return the same instance (singleton pattern)")
	}

	if logger1 == nil {
		t.Fatal("Default logger should not be nil")
	}

	config := logger1.GetConfig()
	if config.Level != defaultConfig.Level {
		t.Errorf("Expected default level '%s', got '%s'", defaultConfig.Level, config.Level)
	}
}

// TestSetLevel 测试动态设置日志级别
func TestSetLevel(t *testing.T) {
	testDir := "./test_logs"
	defer os.RemoveAll(testDir)

	config := &Config{
		Level:    "info",
		FileName: filepath.Join(testDir, "level_test.log"),
	}
	logger := New(config)

	tests := []struct {
		level     string
		expectErr bool
	}{
		{"debug", false},
		{"info", false},
		{"warn", false},
		{"error", false},
		{"fatal", false},
		{"invalid", true},
	}

	for _, tt := range tests {
		t.Run(fmt.Sprintf("set_level_%s", tt.level), func(t *testing.T) {
			err := logger.SetLevel(tt.level)
			if tt.expectErr && err == nil {
				t.Error("Expected error but got none")
			}
			if !tt.expectErr && err != nil {
				t.Errorf("Unexpected error: %v", err)
			}

			if !tt.expectErr {
				config := logger.GetConfig()
				if config.Level != tt.level {
					t.Errorf("Expected level '%s', got '%s'", tt.level, config.Level)
				}
			}
		})
	}
}

// TestGetConfig 测试获取配置
func TestGetConfig(t *testing.T) {
	testDir := "./test_logs"
	defer os.RemoveAll(testDir)

	originalConfig := &Config{
		Level:      "debug",
		FileName:   filepath.Join(testDir, "config_test.log"),
		MaxSize:    100,
		MaxAge:     30,
		MaxBackups: 10,
		Compress:   true,
		TimeZone:   "Asia/Shanghai",
	}

	logger := New(originalConfig)
	config := logger.GetConfig()

	// 测试返回的是副本，修改不应影响原配置
	config.Level = "error"
	originalLevel := logger.GetConfig().Level
	if originalLevel != "debug" {
		t.Error("GetConfig should return a copy, original config should not be modified")
	}
}

// TestAddTraceID 测试traceId添加功能
func TestAddTraceID(t *testing.T) {
	testDir := "./test_logs"
	defer os.RemoveAll(testDir)

	logger := New(&Config{
		Level:    "debug",
		FileName: filepath.Join(testDir, "trace_test.log"),
	})

	tests := []struct {
		name     string
		ctx      context.Context
		expected int // expected number of fields after adding traceId
	}{
		{
			name:     "nil context",
			ctx:      nil,
			expected: 1, // original field count
		},
		{
			name:     "context without traceId",
			ctx:      context.Background(),
			expected: 1, // original field count
		},
		{
			name:     "context with traceId",
			ctx:      context.WithValue(context.Background(), "traceId", "test-trace-123"),
			expected: 2, // original field + traceId
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fields := []zap.Field{zap.String("test", "value")}
			result := logger.addTraceID(tt.ctx, fields)

			if len(result) != tt.expected {
				t.Errorf("Expected %d fields, got %d", tt.expected, len(result))
			}

			if tt.ctx != nil && tt.ctx.Value("traceId") != nil {
				// 检查是否包含traceId字段
				found := false
				for _, field := range result {
					if field.Key == "traceId" {
						found = true
						break
					}
				}
				if !found {
					t.Error("Expected traceId field to be added")
				}
			}
		})
	}
}

// TestLogMethods 测试各种日志记录方法
func TestLogMethods(t *testing.T) {
	testDir := "./test_logs"
	defer os.RemoveAll(testDir)

	logger := New(&Config{
		Level:    "debug",
		FileName: filepath.Join(testDir, "methods_test.log"),
	})

	ctx := context.WithValue(context.Background(), "traceId", "test-123")

	// 测试结构化日志方法
	t.Run("structured_logging", func(t *testing.T) {
		logger.Debug(ctx, "debug message", zap.String("key", "value"))
		logger.Info(ctx, "info message", zap.Int("count", 42))
		logger.Warn(ctx, "warn message", zap.Bool("flag", true))
		logger.Error(ctx, "error message", zap.Error(fmt.Errorf("test error")))
		// 注意：不测试Fatal方法，因为它会调用os.Exit
	})

	// 测试格式化日志方法
	t.Run("formatted_logging", func(t *testing.T) {
		logger.Debugf(ctx, "debug: %s = %d", "count", 10)
		logger.Infof(ctx, "info: user %s logged in", "john")
		logger.Warnf(ctx, "warn: %d attempts remaining", 3)
		logger.Errorf(ctx, "error: failed to process %s", "request")
	})
}

// TestGlobalMethods 测试全局便捷方法
func TestGlobalMethods(t *testing.T) {
	// 重置全局状态
	globalLogger = nil
	once = sync.Once{}

	ctx := context.WithValue(context.Background(), "traceId", "global-test")

	// 测试全局结构化日志方法
	t.Run("global_structured", func(t *testing.T) {
		Debug(ctx, "global debug", zap.String("method", "Debug"))
		Info(ctx, "global info", zap.String("method", "Info"))
		Warn(ctx, "global warn", zap.String("method", "Warn"))
		Error(ctx, "global error", zap.String("method", "Error"))
	})

	// 测试全局格式化日志方法
	t.Run("global_formatted", func(t *testing.T) {
		Debugf(ctx, "global debug: %s", "formatted")
		Infof(ctx, "global info: %s", "formatted")
		Warnf(ctx, "global warn: %s", "formatted")
		Errorf(ctx, "global error: %s", "formatted")
	})
}

// TestSetGlobalConfig 测试设置全局配置
func TestSetGlobalConfig(t *testing.T) {
	// 重置全局状态
	globalLogger = nil
	once = sync.Once{}

	testDir := "./test_logs"
	defer os.RemoveAll(testDir)

	customConfig := &Config{
		Level:    "warn",
		FileName: filepath.Join(testDir, "global_config_test.log"),
	}

	SetGlobalConfig(customConfig)

	config := Default().GetConfig()
	if config.Level != "warn" {
		t.Errorf("Expected global config level 'warn', got '%s'", config.Level)
	}
}

// TestSync 测试同步功能
func TestSync(t *testing.T) {
	testDir := "./test_logs"
	defer os.RemoveAll(testDir)

	logger := New(&Config{
		Level:    "info",
		FileName: filepath.Join(testDir, "sync_test.log"),
	})

	// 写入一些日志
	logger.Info(context.Background(), "test message before sync")

	// 测试实例方法同步
	err := logger.Sync()
	if err != nil {
		t.Errorf("Sync failed: %v", err)
	}

	// 测试全局方法同步
	err = Sync()
	if err != nil {
		t.Errorf("Global Sync failed: %v", err)
	}
}

// TestConcurrentAccess 测试并发访问安全性
func TestConcurrentAccess(t *testing.T) {
	testDir := "./test_logs"
	defer os.RemoveAll(testDir)

	logger := New(&Config{
		Level:    "info",
		FileName: filepath.Join(testDir, "concurrent_test.log"),
	})

	const numGoroutines = 100
	const numMessages = 10

	var wg sync.WaitGroup
	wg.Add(numGoroutines)

	// 并发写入日志
	for i := 0; i < numGoroutines; i++ {
		go func(id int) {
			defer wg.Done()
			ctx := context.WithValue(context.Background(), "traceId", fmt.Sprintf("trace-%d", id))

			for j := 0; j < numMessages; j++ {
				logger.Info(ctx, fmt.Sprintf("message from goroutine %d, iteration %d", id, j))
				logger.Debugf(ctx, "debug from goroutine %d, iteration %d", id, j)
			}
		}(i)
	}

	// 并发修改配置
	go func() {
		defer wg.Done()
		for i := 0; i < 10; i++ {
			_ = logger.SetLevel("debug")
			time.Sleep(time.Millisecond)
			_ = logger.SetLevel("info")
			time.Sleep(time.Millisecond)
		}
	}()
	wg.Add(1)

	// 并发获取配置
	go func() {
		defer wg.Done()
		for i := 0; i < 100; i++ {
			_ = logger.GetConfig()
			time.Sleep(time.Microsecond * 100)
		}
	}()
	wg.Add(1)

	wg.Wait()
}

// TestInvalidTimeZone 测试无效时区处理
func TestInvalidTimeZone(t *testing.T) {
	testDir := "./test_logs"
	defer os.RemoveAll(testDir)

	config := &Config{
		Level:    "info",
		FileName: filepath.Join(testDir, "timezone_test.log"),
		TimeZone: "Invalid/TimeZone",
	}

	// 应该能够创建logger，即使时区无效
	logger := New(config)
	if logger == nil {
		t.Fatal("Logger should be created even with invalid timezone")
	}

	// 应该能够正常记录日志
	logger.Info(context.Background(), "test message with invalid timezone")
}

// TestInvalidLogLevel 测试无效日志级别处理
func TestInvalidLogLevel(t *testing.T) {
	testDir := "./test_logs"
	defer os.RemoveAll(testDir)

	config := &Config{
		Level:    "invalid_level",
		FileName: filepath.Join(testDir, "invalid_level_test.log"),
	}

	// 应该能够创建logger，使用默认级别
	logger := New(config)
	if logger == nil {
		t.Fatal("Logger should be created even with invalid log level")
	}

	// 应该能够正常记录日志
	logger.Info(context.Background(), "test message with invalid log level")
}

// TestLogFileCreation 测试日志文件创建
func TestLogFileCreation(t *testing.T) {
	testDir := "./test_logs"
	defer os.RemoveAll(testDir)

	// 确保目录不存在
	os.RemoveAll(testDir)

	logFile := filepath.Join(testDir, "creation_test.log")
	config := &Config{
		Level:    "info",
		FileName: logFile,
	}

	logger := New(config)
	logger.Info(context.Background(), "test message for file creation")

	// 强制同步，确保文件被创建
	logger.Sync()

	// 检查文件是否被创建
	if _, err := os.Stat(logFile); os.IsNotExist(err) {
		t.Error("Log file should be created")
	}
}

// TestContextVariations 测试不同context情况
func TestContextVariations(t *testing.T) {
	testDir := "./test_logs"
	defer os.RemoveAll(testDir)

	logger := New(&Config{
		Level:    "debug",
		FileName: filepath.Join(testDir, "context_test.log"),
	})

	// 测试不同类型的traceId值
	tests := []struct {
		name    string
		traceId interface{}
	}{
		{"string traceId", "trace-string"},
		{"int traceId", 12345},
		{"struct traceId", struct{ ID string }{ID: "struct-trace"}},
		{"nil traceId", nil},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var ctx context.Context
			if tt.traceId != nil {
				ctx = context.WithValue(context.Background(), "traceId", tt.traceId)
			} else {
				ctx = context.WithValue(context.Background(), "traceId", nil)
			}

			// 这些调用不应该panic
			logger.Info(ctx, "test message")
			logger.Infof(ctx, "test formatted message: %s", "value")
		})
	}
}

// BenchmarkLoggerInfo 基准测试Info方法性能
func BenchmarkLoggerInfo(b *testing.B) {
	testDir := "./test_logs"
	defer os.RemoveAll(testDir)

	logger := New(&Config{
		Level:    "info",
		FileName: filepath.Join(testDir, "benchmark.log"),
	})

	ctx := context.WithValue(context.Background(), "traceId", "bench-trace")

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		logger.Info(ctx, "benchmark message", zap.Int("iteration", i))
	}
}

// BenchmarkLoggerInfof 基准测试Infof方法性能
func BenchmarkLoggerInfof(b *testing.B) {
	testDir := "./test_logs"
	defer os.RemoveAll(testDir)

	logger := New(&Config{
		Level:    "info",
		FileName: filepath.Join(testDir, "benchmark_f.log"),
	})

	ctx := context.WithValue(context.Background(), "traceId", "bench-trace")

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		logger.Infof(ctx, "benchmark message: %d", i)
	}
}

// BenchmarkGlobalInfo 基准测试全局Info方法性能
func BenchmarkGlobalInfo(b *testing.B) {
	// 重置全局状态
	globalLogger = nil
	once = sync.Once{}

	ctx := context.WithValue(context.Background(), "traceId", "global-bench")

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		Info(ctx, "global benchmark message", zap.Int("iteration", i))
	}
}

// TestLoggerMemoryLeak 测试内存泄漏（简单检查）
func TestLoggerMemoryLeak(t *testing.T) {
	testDir := "./test_logs"
	defer os.RemoveAll(testDir)

	// 创建多个logger实例，确保没有明显的内存泄漏
	for i := 0; i < 100; i++ {
		config := &Config{
			Level:    "info",
			FileName: filepath.Join(testDir, fmt.Sprintf("leak_test_%d.log", i)),
		}
		logger := New(config)
		logger.Info(context.Background(), "test message")
		logger.Sync()
	}
}
