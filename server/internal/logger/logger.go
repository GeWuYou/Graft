package logger

import (
	"errors"
	"fmt"
	"strings"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"

	"graft/server/internal/config"
)

// New 按运行时配置创建统一的结构化日志实例。
//
// local 与 test 环境默认使用更适合本地排查的 console 编码，其它环境
// 保持生产配置，避免插件自行决定日志编码或级别。
func New(cfg *config.Config) (*zap.Logger, error) {
	if cfg == nil {
		return nil, errors.New("config is required")
	}

	zapConfig := zap.NewProductionConfig()
	if cfg.App.Env == "local" || cfg.App.Env == "test" {
		zapConfig = zap.NewDevelopmentConfig()
		zapConfig.Encoding = "console"
		zapConfig.EncoderConfig.EncodeLevel = zapcore.CapitalLevelEncoder
	}

	level, err := zap.ParseAtomicLevel(strings.TrimSpace(cfg.Log.Level))
	if err != nil {
		return nil, fmt.Errorf("parse log level %q: %w", cfg.Log.Level, err)
	}
	zapConfig.Level = level

	logger, err := zapConfig.Build(
		zap.AddCaller(),
		zap.Fields(
			zap.String("app", cfg.App.Name),
			zap.String("env", cfg.App.Env),
		),
	)
	if err != nil {
		return nil, fmt.Errorf("build logger: %w", err)
	}

	return logger, nil
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
