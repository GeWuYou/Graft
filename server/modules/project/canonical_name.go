package project

import (
	"fmt"
	"strings"

	projectcompose "graft/server/modules/project/compose"
)

// validateExplicitCanonicalProjectName 校验显式提供的规范项目名，并返回去除首尾空白后的值。
// 规范项目名必须以小写字母或数字开头，且仅允许小写字母、数字、下划线和连字符。
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
