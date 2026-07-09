package project

import (
	"fmt"
	"regexp"
	"strings"
)

var canonicalProjectNamePattern = regexp.MustCompile(`^[a-z0-9][a-z0-9_-]*$`)

// validateExplicitCanonicalProjectName trims surrounding whitespace and validates a canonical project name.
// It accepts names beginning with a lowercase letter or digit and containing only lowercase letters,
// digits, underscores, and hyphens.
func validateExplicitCanonicalProjectName(value string) (string, error) {
	normalized := strings.TrimSpace(value)
	if normalized == "" {
		return "", errProjectInvalidCanonicalName
	}
	if !canonicalProjectNamePattern.MatchString(normalized) {
		return "", fmt.Errorf("%w: invalid canonical project name", errProjectInvalidCanonicalName)
	}
	return normalized, nil
}
