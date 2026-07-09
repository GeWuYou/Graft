package project

import (
	"fmt"
	"strings"

	projectcompose "graft/server/modules/project/compose"
)

// validateExplicitCanonicalProjectName trims surrounding whitespace and validates a canonical project name.
// It accepts names beginning with a lowercase letter or digit and containing only lowercase letters,
// digits, underscores, and hyphens.
func validateExplicitCanonicalProjectName(value string) (string, error) {
	normalized := strings.TrimSpace(value)
	if normalized == "" {
		return "", errProjectInvalidCanonicalName
	}
	if !projectcompose.IsValidCanonicalProjectName(normalized) {
		return "", fmt.Errorf("%w: invalid canonical project name", errProjectInvalidCanonicalName)
	}
	return normalized, nil
}
