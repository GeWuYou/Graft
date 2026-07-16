package project

import (
	"graft/server/internal/config"
	"graft/server/internal/logger/logsafe"

	"go.uber.org/zap"
)

// WithDebugConfig injects the immutable core configuration snapshot used by project diagnostics.
func WithDebugConfig(value config.ProjectConfig) ServiceOption {
	return serviceOptionFunc(func(s *Service) { s.debugConfig = value })
}

// logProjectLogDiagnostic emits bounded metadata only; log contents are intentionally excluded.
func (s *Service) logProjectLogDiagnostic(event string, fields ...zap.Field) {
	if s == nil || !s.debugConfig.LogDebug {
		return
	}
	logsafe.Info(s.logger, "project log diagnostic", append([]zap.Field{zap.String("event", event)}, fields...)...)
}

// logManagedCreateDiagnostic emits bounded managed-create diagnostics without
// recording workspace file contents or other request payload bodies.
func (s *Service) logManagedCreateDiagnostic(event string, fields ...zap.Field) {
	if s == nil || !s.debugConfig.ManagedCreateDebug {
		return
	}
	logsafe.Info(s.logger, "managed project create diagnostic", append([]zap.Field{zap.String("event", event)}, fields...)...)
}
