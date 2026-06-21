package httpx

import (
	"fmt"
	"regexp"
	"strings"
)

const accessLogRedactedValue = "[REDACTED]"
const accessLogSplitPairParts = 2

var sensitiveAccessLogPatterns = []*regexp.Regexp{
	regexp.MustCompile(`(?i)(authorization\s*:\s*)[^\r\n]+`),
	regexp.MustCompile(`(?i)(authorization\s*=\s*)[^\r\n]+`),
	regexp.MustCompile(`(?i)(cookie\s*:\s*)[^\r\n]+`),
	regexp.MustCompile(`(?i)(\"?(?:password|passwd|pwd|token|secret|authorization|cookie|set-cookie|access_token|refresh_token|client_secret|api_key)\"?\s*[:=]\s*)\"?[^",;\s]+\"?`),
}

func sanitizeAccessLogPath(path string) string {
	return sanitizeAccessLogFreeText(path)
}

func sanitizeAccessLogRoute(route string) string {
	return sanitizeAccessLogFreeText(route)
}

func sanitizeAccessLogFreeText(value string) string {
	sanitized := strings.TrimSpace(value)
	for _, pattern := range sensitiveAccessLogPatterns {
		sanitized = pattern.ReplaceAllStringFunc(sanitized, func(match string) string {
			parts := strings.SplitN(match, ":", accessLogSplitPairParts)
			if len(parts) == accessLogSplitPairParts && strings.Contains(match, ":") {
				return fmt.Sprintf("%s: %s", strings.TrimSpace(parts[0]), accessLogRedactedValue)
			}

			parts = strings.SplitN(match, "=", accessLogSplitPairParts)
			if len(parts) == accessLogSplitPairParts {
				return fmt.Sprintf("%s=%s", strings.TrimSpace(parts[0]), accessLogRedactedValue)
			}

			return accessLogRedactedValue
		})
	}

	return sanitized
}
