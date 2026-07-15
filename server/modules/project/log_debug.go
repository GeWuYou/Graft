package project

import (
	"os"
	"strings"

	"go.uber.org/zap"

	"graft/server/internal/logger/logsafe"
)

const (
	projectLogDebugEnvironmentKey           = "GRAFT_PROJECT_LOG_DEBUG"
	projectManagedCreateDebugEnvironmentKey = "GRAFT_PROJECT_MANAGED_CREATE_DEBUG"
)

// isProjectLogDebugEnabled determines whether project log diagnostics are enabled.
// It returns true when GRAFT_PROJECT_LOG_DEBUG is set to 1, true, yes, or on,
// ignoring surrounding whitespace and letter case.
func isProjectLogDebugEnabled() bool {
	value, ok := os.LookupEnv(projectLogDebugEnvironmentKey)
	if !ok {
		return false
	}
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "1", "true", "yes", "on":
		return true
	default:
		return false
	}
}

// logProjectLogDiagnostic emits bounded metadata only; log contents are intentionally excluded.
func (s *Service) logProjectLogDiagnostic(event string, fields ...zap.Field) {
	if s == nil || !isProjectLogDebugEnabled() {
		return
	}
	logsafe.Info(s.logger, "project log diagnostic", append([]zap.Field{zap.String("event", event)}, fields...)...)
}

// logManagedCreateDiagnostic emits bounded managed-create diagnostics without
// recording workspace file contents or other request payload bodies.
func (s *Service) logManagedCreateDiagnostic(event string, fields ...zap.Field) {
	if s == nil || !isProjectManagedCreateDebugEnabled() {
		return
	}
	logsafe.Info(s.logger, "managed project create diagnostic", append([]zap.Field{zap.String("event", event)}, fields...)...)
}

func isProjectManagedCreateDebugEnabled() bool {
	value, ok := os.LookupEnv(projectManagedCreateDebugEnvironmentKey)
	if !ok {
		return false
	}
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "1", "true", "yes", "on":
		return true
	default:
		return false
	}
}
