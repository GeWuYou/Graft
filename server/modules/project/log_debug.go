package project

import (
	"graft/server/internal/config"
	"graft/server/internal/logger/logsafe"

	"go.uber.org/zap"
)

// WithDebugConfig 注入项目诊断使用的不可变 core 配置快照。
func WithDebugConfig(value config.ProjectConfig) ServiceOption {
	return serviceOptionFunc(func(s *Service) { s.debugConfig = value })
}

// logProjectLogDiagnostic 仅输出有界元数据，刻意排除日志正文。
func (s *Service) logProjectLogDiagnostic(event string, fields ...zap.Field) {
	if s == nil || !s.debugConfig.LogDebug {
		return
	}
	logsafe.Info(s.logger, "project log diagnostic", append([]zap.Field{zap.String("event", event)}, fields...)...)
}

// logManagedCreateDiagnostic 输出有界的受管创建诊断，不记录工作区文件内容或其它请求体。
func (s *Service) logManagedCreateDiagnostic(event string, fields ...zap.Field) {
	if s == nil || !s.debugConfig.ManagedCreateDebug {
		return
	}
	logsafe.Info(s.logger, "managed project create diagnostic", append([]zap.Field{zap.String("event", event)}, fields...)...)
}
