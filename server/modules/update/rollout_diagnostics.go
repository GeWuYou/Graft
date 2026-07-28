package update

import (
	"errors"
	"net/http"
	"regexp"
	"strings"
)

const (
	rolloutFailureInvalidTarget            = "PLATFORM_UPDATE_INVALID_TARGET"
	rolloutFailureCatalogStale             = "PLATFORM_UPDATE_CATALOG_STALE"
	rolloutFailureInstallationUnavailable  = "PLATFORM_UPDATE_INSTALLATION_UNAVAILABLE"
	rolloutFailureSourceVersionUnsupported = "PLATFORM_UPDATE_SOURCE_VERSION_UNSUPPORTED"
	rolloutFailureComposeCandidateInvalid  = "PLATFORM_UPDATE_COMPOSE_CANDIDATE_INVALID"
	rolloutFailureComposePreflightFailed   = "PLATFORM_UPDATE_COMPOSE_PREFLIGHT_FAILED"
	rolloutFailureOperationStartFailed     = "PLATFORM_UPDATE_OPERATION_START_FAILED"
	rolloutFailureRunnerTerminal           = "PLATFORM_UPDATE_RUNNER_TERMINAL_FAILED"
)

type rolloutStartFailure struct {
	code        string
	stage       string
	operationID string
	cause       error
}

func (e *rolloutStartFailure) Error() string {
	if e == nil || e.cause == nil {
		return "update rollout start failed"
	}
	return e.cause.Error()
}

func (e *rolloutStartFailure) Unwrap() error {
	if e == nil {
		return nil
	}
	return e.cause
}

func newRolloutStartFailure(code, stage, operationID string, cause error) error {
	return &rolloutStartFailure{code: code, stage: stage, operationID: operationID, cause: cause}
}

func rolloutFailureDetails(err error) (code, stage, operationID string) {
	var failure *rolloutStartFailure
	if errors.As(err, &failure) && failure != nil {
		return failure.code, failure.stage, failure.operationID
	}
	return rolloutFailureOperationStartFailed, "internal", ""
}

func rolloutFailureMessageKey(code string) string {
	switch code {
	case rolloutFailureInvalidTarget:
		return "update.operation.start.invalid_target"
	case rolloutFailureCatalogStale:
		return "update.operation.start.catalog_stale"
	case rolloutFailureInstallationUnavailable:
		return "update.operation.start.installation_unavailable"
	case rolloutFailureSourceVersionUnsupported:
		return "update.operation.start.source_version_unsupported"
	case rolloutFailureComposeCandidateInvalid:
		return "update.operation.start.compose_candidate_invalid"
	case rolloutFailureComposePreflightFailed:
		return "update.operation.start.compose_preflight_failed"
	default:
		return "update.operation.start.operation_start_failed"
	}
}

func rolloutFailureHTTPStatus(code string) int {
	switch code {
	case rolloutFailureInvalidTarget:
		return http.StatusBadRequest
	case rolloutFailureCatalogStale, rolloutFailureInstallationUnavailable, rolloutFailureSourceVersionUnsupported, rolloutFailureComposeCandidateInvalid, rolloutFailureComposePreflightFailed:
		return http.StatusPreconditionFailed
	default:
		return http.StatusInternalServerError
	}
}

var rolloutSensitiveValuePattern = regexp.MustCompile(`(?i)(\b(?:password|passwd|pwd|token|secret|authorization|cookie|access_token|refresh_token|client_secret|api_key)\b\s*[:=]\s*)(?:"[^"]*"|'[^']*'|(?:bearer|basic)\s+[^,;\s]+|[^,;\s]+)`)
var rolloutDSNUserInfoPattern = regexp.MustCompile(`([a-z][a-z0-9+.-]*://[^\s/@:]+:)[^@\s/]+(@)`)

// sanitizeRolloutError 保留错误链可检索的文字，同时剔除常见凭证赋值与连接串密码。
func sanitizeRolloutError(err error) string {
	if err == nil {
		return ""
	}
	value := rolloutSensitiveValuePattern.ReplaceAllString(err.Error(), "${1}[REDACTED]")
	return rolloutDSNUserInfoPattern.ReplaceAllString(value, "${1}[REDACTED]${2}")
}

func rolloutFailureResponseData(code string) map[string]string {
	return map[string]string{"reason": strings.TrimSpace(code)}
}
