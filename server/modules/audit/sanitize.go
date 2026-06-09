// Copyright (c) 2025-2026 GeWuYou
// SPDX-License-Identifier: Apache-2.0

package audit

import (
	"fmt"
	"regexp"
	"strings"
)

const redactedValue = "[REDACTED]"
const splitPairParts = 2

var sensitiveFreeTextPatterns = []*regexp.Regexp{
	regexp.MustCompile(`(?i)(authorization\s*:\s*bearer\s+)\S+`),
	regexp.MustCompile(`(?i)(cookie\s*:\s*)[^\r\n]+`),
	regexp.MustCompile(`(?i)(\"?(?:password|passwd|pwd|token|secret|authorization|cookie|set-cookie|access_token|refresh_token|client_secret|api_key)\"?\s*[:=]\s*)\"?[^",;\s]+\"?`),
}

func sanitizeMetadataValue(input any) any {
	switch typed := input.(type) {
	case map[string]any:
		return sanitizeMetadataObject(typed)
	case []any:
		items := make([]any, 0, len(typed))
		for _, item := range typed {
			items = append(items, sanitizeMetadataValue(item))
		}
		return items
	case string:
		return sanitizeFreeText(typed)
	case nil:
		return nil
	default:
		return typed
	}
}

func sanitizeMetadataObject(input map[string]any) map[string]any {
	output := make(map[string]any, len(input))
	for key, value := range input {
		if isSensitiveKey(key) {
			output[key] = redactedValue
			continue
		}
		output[key] = sanitizeMetadataValue(value)
	}
	return output
}

func sanitizeFreeText(value string) string {
	sanitized := strings.TrimSpace(value)
	for _, pattern := range sensitiveFreeTextPatterns {
		sanitized = pattern.ReplaceAllStringFunc(sanitized, func(match string) string {
			parts := strings.SplitN(match, ":", splitPairParts)
			if len(parts) == 2 && strings.Contains(match, ":") {
				return fmt.Sprintf("%s: %s", strings.TrimSpace(parts[0]), redactedValue)
			}
			parts = strings.SplitN(match, "=", splitPairParts)
			if len(parts) == splitPairParts {
				return fmt.Sprintf("%s=%s", strings.TrimSpace(parts[0]), redactedValue)
			}
			return redactedValue
		})
	}
	return sanitized
}

func isSensitiveKey(key string) bool {
	normalized := strings.ToLower(strings.TrimSpace(key))
	for _, fragment := range []string{
		"password",
		"passwd",
		"pwd",
		"token",
		"secret",
		"authorization",
		"cookie",
		"credential",
		"api_key",
		"apikey",
	} {
		if strings.Contains(normalized, fragment) {
			return true
		}
	}
	return false
}
