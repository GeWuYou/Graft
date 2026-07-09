package project

import (
	"fmt"
	"regexp"
	"strings"
)

var canonicalProjectNamePattern = regexp.MustCompile(`^[a-z0-9][a-z0-9_-]*$`)

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
