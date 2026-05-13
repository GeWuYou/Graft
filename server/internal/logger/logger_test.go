package logger

import (
	"testing"

	"go.uber.org/zap"

	"graft/server/internal/config"
)

// TestNewUsesConfiguredLogLevel 验证日志预留口子至少会把配置中的级别装配
// 到统一 logger 上，避免 runtime 与插件读取到不同的日志阈值。
func TestNewUsesConfiguredLogLevel(t *testing.T) {
	cfg := &config.Config{
		App: config.AppConfig{
			Name: "graft",
			Env:  "test",
		},
		Log: config.LogConfig{
			Level: "debug",
		},
	}

	logger, err := New(cfg)
	if err != nil {
		t.Fatalf("new logger: %v", err)
	}
	t.Cleanup(func() {
		_ = Close(logger)
	})

	if !logger.Core().Enabled(zap.DebugLevel) {
		t.Fatal("expected debug level to be enabled")
	}
}

// TestNewRejectsInvalidLogLevel 验证非法日志级别会在 runtime 装配前直接失败。
func TestNewRejectsInvalidLogLevel(t *testing.T) {
	cfg := &config.Config{
		App: config.AppConfig{
			Name: "graft",
			Env:  "test",
		},
		Log: config.LogConfig{
			Level: "definitely-not-a-level",
		},
	}

	if _, err := New(cfg); err == nil {
		t.Fatal("expected invalid log level error")
	}
}
