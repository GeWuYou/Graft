// Copyright (c) 2025-2026 GeWuYou
// SPDX-License-Identifier: Apache-2.0

package logger

import (
	"errors"
	"fmt"
	"os"
	"strings"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"

	"graft/server/internal/config"
)

const (
	bootstrapAppName  = "graft"
	bootstrapAppEnv   = "local"
	bootstrapLogLevel = "info"
)

// New 按运行时配置创建统一的结构化日志实例。
//
// local 与 test 环境默认使用更适合本地排查的 console 编码，其它环境
// 保持生产配置，避免模块自行决定日志编码或级别。
func New(cfg *config.Config) (*zap.Logger, error) {
	if cfg == nil {
		return nil, errors.New("config is required")
	}

	logger, err := buildLogger(
		strings.TrimSpace(cfg.App.Name),
		strings.TrimSpace(cfg.App.Env),
		strings.TrimSpace(cfg.Log.Level),
		cfg.Log.Format,
		cfg.Log.Color,
	)
	if err != nil {
		return nil, err
	}

	// runtime logger 是仓库内应用日志的唯一主基线；同步替换 zap 全局 logger，
	// 让需要全局入口的基础设施不会绕开同一后端。
	zap.ReplaceGlobals(logger)
	return logger, nil
}

// NewBootstrap 创建不依赖完整 runtime config 的早期 CLI logger。
func NewBootstrap() *zap.Logger {
	appEnv := strings.TrimSpace(os.Getenv(config.EnvAppEnv))
	if appEnv == "" {
		appEnv = bootstrapAppEnv
	}
	logLevel := strings.TrimSpace(os.Getenv(config.EnvLogLevel))
	if logLevel == "" {
		logLevel = bootstrapLogLevel
	}
	logFormat := config.LogFormat(strings.TrimSpace(os.Getenv(config.EnvLogFormat)))
	logColor := config.LogColor(strings.TrimSpace(os.Getenv(config.EnvLogColor)))

	logger, err := buildLogger(bootstrapAppName, appEnv, logLevel, logFormat, logColor)
	if err != nil {
		fallback := zap.NewNop()
		zap.ReplaceGlobals(fallback)
		return fallback
	}

	zap.ReplaceGlobals(logger)
	return logger
}

func buildLogger(
	appName string,
	appEnv string,
	logLevel string,
	logFormat config.LogFormat,
	logColor config.LogColor,
) (*zap.Logger, error) {
	level, err := zap.ParseAtomicLevel(logLevel)
	if err != nil {
		return nil, fmt.Errorf("parse log level %q: %w", logLevel, err)
	}

	zapConfig := buildZapConfig(appEnv, logFormat, logColor)
	zapConfig.Level = level

	logger, err := zapConfig.Build(
		zap.AddCaller(),
		zap.Fields(
			zap.String("app", firstNonEmpty(appName, bootstrapAppName)),
			zap.String("env", firstNonEmpty(appEnv, bootstrapAppEnv)),
		),
	)
	if err != nil {
		return nil, fmt.Errorf("build logger: %w", err)
	}

	return logger, nil
}

func buildZapConfig(appEnv string, logFormat config.LogFormat, logColor config.LogColor) zap.Config {
	effectiveFormat := config.ResolveLogFormat(appEnv, logFormat)
	if effectiveFormat == config.LogFormatConsole {
		zapConfig := zap.NewDevelopmentConfig()
		zapConfig.Encoding = string(config.LogFormatConsole)
		zapConfig.EncoderConfig.EncodeLevel = zapcore.CapitalLevelEncoder
		if config.ResolveLogColor(appEnv, logFormat, logColor) {
			zapConfig.EncoderConfig.EncodeLevel = zapcore.CapitalColorLevelEncoder
		}
		return zapConfig
	}

	zapConfig := zap.NewProductionConfig()
	zapConfig.Encoding = string(config.LogFormatJSON)
	return zapConfig
}

// Close 刷新日志缓冲并忽略标准输出场景下的已知 Sync 噪声。
//
// 某些本地终端或测试环境会让 `Sync` 返回无害的文件描述符错误；这里
// 统一收敛这些细节，避免调用方把正常关闭误判为失败。
func Close(logger *zap.Logger) error {
	if logger == nil {
		return nil
	}

	if err := logger.Sync(); err != nil && !isIgnorableSyncError(err) {
		return fmt.Errorf("sync logger: %w", err)
	}

	return nil
}

func isIgnorableSyncError(err error) bool {
	if err == nil {
		return false
	}

	message := strings.ToLower(err.Error())
	return strings.Contains(message, "invalid argument") ||
		strings.Contains(message, "bad file descriptor") ||
		strings.Contains(message, "inappropriate ioctl")
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if trimmed := strings.TrimSpace(value); trimmed != "" {
			return trimmed
		}
	}
	return ""
}
