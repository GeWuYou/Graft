package project

import (
	"os"
	"strings"

	"go.uber.org/zap"

	"graft/server/internal/logger/logsafe"
)

const projectLogDebugEnvironmentKey = "GRAFT_PROJECT_LOG_DEBUG"

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
	logsafe.Info(s.logger, "project log diagnostic: "+event, fields...)
}
